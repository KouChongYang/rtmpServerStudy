package flv

import (
	"bufio"
	"fmt"
	"time"
	"github.com/nareix/bits/pio"

	"io"
	"rtmpServerStudy/av"
	"rtmpServerStudy/flv/flvio"

	"rtmpServerStudy/amf"
	"rtmpServerStudy/h264Parse"
	"rtmpServerStudy/aacParse"
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


func (self *Prober) addPacket(payload []byte, timedelta time.Duration,tag flvio.Tag) {

	pkt := av.Packet{
	}

	switch tag.Type {
	case flvio.TAG_VIDEO:
		pkt.Idx = int8(self.VideoStreamIdx)
		switch tag.AVCPacketType {
		case flvio.AVC_NALU:
			pkt.Data = payload
			pkt.CompositionTime = flvio.TsToTime(tag.CompositionTime)
			pkt.IsKeyFrame = tag.FrameType == flvio.FRAME_KEY
		}
	}
	self.CachedPkts = append(self.CachedPkts, pkt)
}

func (self *Prober) Empty() bool {
	return len(self.CachedPkts) == 0
}

func (self *Prober) PopPacket() av.Packet {
	pkt := self.CachedPkts[0]
	self.CachedPkts = self.CachedPkts[1:]
	return pkt
}

func CodecDataToTag(stream av.CodecData) (_tag flvio.Tag, ok bool, err error) {
	switch stream.Type() {
	case av.H264:
		h264 := stream.(h264parser.CodecData)
		tag := flvio.Tag{
			Type:          flvio.TAG_VIDEO,
			AVCPacketType: flvio.AVC_SEQHDR,
			CodecID:       flvio.VIDEO_H264,
			Data:          h264.AVCDecoderConfRecordBytes(),
			FrameType:     flvio.FRAME_KEY,
		}
		ok = true
		_tag = tag

	case av.NELLYMOSER:
	case av.SPEEX:

	case av.AAC:
		aac := stream.(aacparser.CodecData)
		tag := flvio.Tag{
			Type:          flvio.TAG_AUDIO,
			SoundFormat:   flvio.SOUND_AAC,
			SoundRate:     flvio.SOUND_44Khz,
			AACPacketType: flvio.AAC_SEQHDR,
			Data:          aac.MPEG4AudioConfigBytes(),
		}
		switch aac.SampleFormat().BytesPerSample() {
		case 1:
			tag.SoundSize = flvio.SOUND_8BIT
		default:
			tag.SoundSize = flvio.SOUND_16BIT
		}
		switch aac.ChannelLayout().Count() {
		case 1:
			tag.SoundType = flvio.SOUND_MONO
		case 2:
			tag.SoundType = flvio.SOUND_STEREO
		}
		ok = true
		_tag = tag

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
	b       []byte
	streams []av.CodecData
}

type writeFlusher interface {
	io.Writer
	Flush() error
}

func NewMuxerWriteFlusher(w writeFlusher) *Muxer {
	return &Muxer{
		bufw: w,
		b:    make([]byte, 256),
	}
}

func NewMuxer(w io.Writer) *Muxer {
	return NewMuxerWriteFlusher(bufio.NewWriterSize(w, pio.RecommendBufioSize))
}

var CodecTypes = []av.CodecType{av.H264, av.AAC, av.SPEEX}

func MetadeToTag(args ...interface{})(_tag flvio.Tag,ok bool){
	size := 0
	for _, arg := range args {
		size += amf.LenAMF0Val(arg)
	}

	b := make([]byte, size)
	n := 0
	//n := self.fillChunkHeader(b, csid, 0, msgtypeid, msgsid, size)
	for _, arg := range args {
		n += amf.FillAMF0Val(b[n:], arg)
	}
	tag := flvio.Tag{
		Type: flvio.TAG_SCRIPTDATA,
		Data: b,
	}
	ok = true
	_tag = tag
	return
}

func (self *Muxer) WriteHeader(streams []av.CodecData) (err error) {
	var flags uint8
	fmt.Println("***************************34567891*****************************")
	for _, stream := range streams {
		if stream.Type().IsVideo() {
			flags |= flvio.FILE_HAS_VIDEO
		} else if stream.Type().IsAudio() {
			flags |= flvio.FILE_HAS_AUDIO
		}
	}
	fmt.Println("***************************34567892*****************************")

	n := flvio.FillFileHeader(self.b, flags)
	if _, err = self.bufw.Write(self.b[:n]); err != nil {
		return
	}
	fmt.Println("***************************34567893*****************************")
	metadata := amf.AMFMap{}
	metadata["stream-id"] = "1234567894"
	metadata["version"] = "golang v1.8.0"

	var tag flvio.Tag
	var ok bool
	if tag, ok = MetadeToTag("onMetaData", metadata); err != nil {
		fmt.Println("***************************34567895*****************************")
	}
	fmt.Println("***************************34567896*****************************")
	if ok {
		if err = flvio.WriteTag(self.bufw, tag, 0, self.b); err != nil {
			return
		}
	}
	fmt.Println("***************************34567897*****************************")

	for _, stream := range streams {
		var tag flvio.Tag
		var ok bool
		if tag, ok, err = CodecDataToTag(stream); err != nil {
			fmt.Println("***************************34567898*****************************")
			return
		}
		if ok {
			if err = flvio.WriteTag(self.bufw, tag, 0, self.b); err != nil {
				fmt.Println("***************************34567899*****************************")
				return
			}
		}
	}
	fmt.Println("***************************345678910*****************************")

	self.streams = streams
	return
}

func (self *Muxer) WritePacket(pkt av.Packet) (err error) {
	stream := self.streams[pkt.Idx]
	tag, timestamp := PacketToTag(pkt, stream)

	if err = flvio.WriteTag(self.bufw, tag, timestamp, self.b); err != nil {
		return
	}
	return
}

func (self *Muxer) WriteTrailer() (err error) {
	if err = self.bufw.Flush(); err != nil {
		return
	}
	return
}


