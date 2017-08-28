package flv

import (
	"bufio"
	"fmt"

	"github.com/nareix/bits/pio"

	"io"
	"rtmpServerStudy/av"
	"rtmpServerStudy/flv/flvio"

	"rtmpServerStudy/aacParse"
	"rtmpServerStudy/amf"
	"rtmpServerStudy/h264Parse"
	"encoding/hex"
)

var MaxProbePacketCount = 20

func NewMetadataByStreams(streams []av.CodecData) (metadata amf.AMFMap, err error) {
	metadata = amf.AMFMap{}

	for _, _stream := range streams {
		typ := _stream.Type()
		switch {
		case typ.IsVideo():
			stream := _stream.(av.VideoCodecData)
			switch typ {
			case av.H264:
				metadata["videocodecid"] = flvio.VIDEO_H264

			default:
				err = fmt.Errorf("flv: metadata: unsupported video codecType=%v", stream.Type())
				return
			}
			metadata["width"] = stream.Width()
			metadata["height"] = stream.Height()
			metadata["displayWidth"] = stream.Width()
			metadata["displayHeight"] = stream.Height()
		case typ.IsAudio():
			stream := _stream.(av.AudioCodecData)
			switch typ {
			case av.AAC:
				metadata["audiocodecid"] = flvio.SOUND_AAC

			case av.SPEEX:
				metadata["audiocodecid"] = flvio.SOUND_SPEEX

			default:
				err = fmt.Errorf("flv: metadata: unsupported audio codecType=%v", stream.Type())
				return
			}
			metadata["audiosamplerate"] = stream.SampleRate()
		}
	}

	return
}

type Prober struct {
	HasAudio, HasVideo             bool
	GotAudio, GotVideo             bool
	VideoStreamIdx, AudioStreamIdx int
	PushedCount                    int
	Streams                        []av.CodecData
	CachedPkts                     []av.Packet
}

func (self *Prober) Empty() bool {
	return len(self.CachedPkts) == 0
}

func (self *Prober) PopPacket() av.Packet {
	pkt := self.CachedPkts[0]
	self.CachedPkts = self.CachedPkts[1:]
	return pkt
}

func CodecDataToTag(stream av.CodecData) (_tag *flvio.Tag, ok bool, err error) {
	_tag = new(flvio.Tag)
	switch stream.Type() {
	case av.H264:
		fmt.Println("head:h264")
		h264 := stream.(h264parser.CodecData)
		_tag.Type = flvio.TAG_VIDEO
		_tag.AVCPacketType = flvio.AVC_SEQHDR
		_tag.CodecID = flvio.VIDEO_H264
		_tag.Data =  h264.AVCDecoderConfRecordBytes()
		ok = true

	case av.NELLYMOSER:
	case av.SPEEX:

	case av.AAC:
		aac := stream.(aacparser.CodecData)
		_tag.Type = flvio.TAG_AUDIO
		_tag.SoundFormat =    flvio.SOUND_AAC
		_tag.SoundRate = flvio.SOUND_44Khz
		_tag.AACPacketType = flvio.AAC_SEQHDR
		_tag.Data =  aac.MPEG4AudioConfigBytes()

		fmt.Println(hex.Dump(_tag.Data))
		switch aac.SampleFormat().BytesPerSample() {
		case 1:
			_tag.SoundSize = flvio.SOUND_8BIT
		default:
			_tag.SoundSize = flvio.SOUND_16BIT
		}
		switch aac.ChannelLayout().Count() {
		case 1:
			_tag.SoundType = flvio.SOUND_MONO
		case 2:
			_tag.SoundType = flvio.SOUND_STEREO
		}
		ok = true
	default:
		err = fmt.Errorf("flv: unspported codecType=%v", stream.Type())
		return
	}
	return
}

func PacketToTag(pkt av.Packet, stream av.CodecData) (tag flvio.Tag, timestamp int32) {
	switch stream.Type() {
	case av.H264:
		tag = flvio.Tag{
			Type:            flvio.TAG_VIDEO,
			AVCPacketType:   flvio.AVC_NALU,
			CodecID:         flvio.VIDEO_H264,
			Data:            pkt.Data,
			CompositionTime: flvio.TimeToTs(pkt.CompositionTime),
		}
		if pkt.IsKeyFrame {
			tag.FrameType = flvio.FRAME_KEY
		} else {
			tag.FrameType = flvio.FRAME_INTER
		}

	case av.AAC:
		tag = flvio.Tag{
			Type:          flvio.TAG_AUDIO,
			SoundFormat:   flvio.SOUND_AAC,
			SoundRate:     flvio.SOUND_44Khz,
			AACPacketType: flvio.AAC_RAW,
			Data:          pkt.Data,
		}
		astream := stream.(av.AudioCodecData)
		switch astream.SampleFormat().BytesPerSample() {
		case 1:
			tag.SoundSize = flvio.SOUND_8BIT
		default:
			tag.SoundSize = flvio.SOUND_16BIT
		}
		switch astream.ChannelLayout().Count() {
		case 1:
			tag.SoundType = flvio.SOUND_MONO
		case 2:
			tag.SoundType = flvio.SOUND_STEREO
		}

	case av.SPEEX:
		tag = flvio.Tag{
			Type:        flvio.TAG_AUDIO,
			SoundFormat: flvio.SOUND_SPEEX,
			Data:        pkt.Data,
		}

	case av.NELLYMOSER:
		tag = flvio.Tag{
			Type:        flvio.TAG_AUDIO,
			SoundFormat: flvio.SOUND_NELLYMOSER,
			Data:        pkt.Data,
		}
	}

	timestamp = flvio.TimeToTs(pkt.Time)
	return
}

type Muxer struct {
	bufw    writeFlusher
	B       []byte
	streams []av.CodecData
}

type writeFlusher interface {
	io.Writer
	Flush() error
}

func NewMuxerWriteFlusher(w writeFlusher) *Muxer {
	return &Muxer{
		bufw: w,
		B:    make([]byte, 256),
	}
}

func NewMuxer(w io.Writer) *Muxer {
	return NewMuxerWriteFlusher(bufio.NewWriterSize(w, pio.RecommendBufioSize))
}

var CodecTypes = []av.CodecType{av.H264, av.AAC, av.SPEEX}

func MetadeToTag(args ...interface{}) (_tag *flvio.Tag, ok bool) {
	size := 0
	for _, arg := range args {
		size += amf.LenAMF0Val(arg)
	}

	b := make([]byte, size)
	n := 0

	for _, arg := range args {
		n += amf.FillAMF0Val(b[n:], arg)
	}

	_tag = new(flvio.Tag)
	_tag.Type = flvio.TAG_SCRIPTDATA
	_tag.Data =  b

	ok = true
	return
}

func (self *Muxer) WriteHeader(streams []av.CodecData) (err error) {
	var flags uint8
	flags |= flvio.FILE_HAS_VIDEO
	flags |= flvio.FILE_HAS_AUDIO

	n := flvio.FillFileHeader(self.B, flags)
	if _, err = self.bufw.Write(self.B[:n]); err != nil {
		return
	}

	metadata := amf.AMFMap{}


	var tag *flvio.Tag
	var ok bool
	if tag, ok = MetadeToTag("onMetaData", metadata); err != nil {

	}

	if ok {
		if err = flvio.WriteTag(self.bufw, tag, 0, self.B); err != nil {
			return
		}
	}

	for _, stream := range streams {
		var tag flvio.Tag
		var ok bool
		if tag, ok, err = CodecDataToTag(stream); err != nil {
			return
		}
		tag.NoHead = true
		if ok {
			if err = flvio.WriteTag(self.bufw, tag, 0, self.B); err != nil {
				return
			}
		}
	}
	self.streams = streams
	return
}
