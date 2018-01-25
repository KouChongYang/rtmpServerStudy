package ts

import (
	"fmt"
	"rtmpServerStudy/av"
	"rtmpServerStudy/aacParse"
	"rtmpServerStudy/h264Parse"
	"rtmpServerStudy/ts/tsio"
	"io"
//	"time"
	//"encoding/hex"
	"bufio"
	"github.com/nareix/bits/pio"
)

//now just support aac h264 for ts
var CodecTypes = []av.CodecType{av.H264, av.AAC}

type Muxer struct {
	w                        io.WriteCloser
	bufw       		 *bufio.Writer
	//streams                  []*Stream
	astream *Stream
	vstream *Stream
	PaddingToMakeCounterCont bool

	psidata []byte
	peshdr  []byte
	tshdr   []byte
	adtshdr []byte
	datav   [][]byte
	nalus   [][]byte

	tswpat, tswpmt *tsio.TSWriter
}


func NewMuxer(w io.WriteCloser) *Muxer {
	return &Muxer{
		w:       w,
		bufw: 	 bufio.NewWriterSize(w, pio.RecommendBufioSize),
		psidata: make([]byte, 188),
		peshdr:  make([]byte, tsio.MaxPESHeaderLength),
		tshdr:   make([]byte, tsio.MaxTSHeaderLength),
		adtshdr: make([]byte, aacparser.ADTSHeaderLength),
		nalus:   make([][]byte, 16),
		datav:   make([][]byte, 16),
		tswpmt:  tsio.NewTSWriter(tsio.PMT_PID),
		tswpat:  tsio.NewTSWriter(tsio.PAT_PID),
	}
}


const	(
	videoPid = uint16(0x100)
	audioPid = uint16(0x101)
)
//new stream
func (self *Muxer) newStream(codec av.CodecData) (err error) {
	ok := false
	for _, c := range CodecTypes {
		if codec.Type() == c {
			ok = true
			break
		}
	}
	if !ok {
		err = fmt.Errorf("ts: codec type=%s is not supported", codec.Type())
		return
	}
	switch codec.Type() {
	case av.H264:
		pid:=videoPid
		stream := &Stream{
			muxer:     self,
			CodecData: codec,
			pid:       pid,
			tsw:       tsio.NewTSWriter(pid),
		}
		self.vstream = stream
	case av.NELLYMOSER:
	case av.SPEEX:
	case av.AAC:
		pid:=audioPid
		stream := &Stream{
			muxer:     self,
			CodecData: codec,
			pid:       pid,
			tsw:       tsio.NewTSWriter(pid),
		}
		self.astream = stream
	default:
		err = fmt.Errorf("ts.Unspported.CodecType(%v)", codec.Type())
		return
	}
	return
}

func (self *Muxer) writePaddingTSPackets(tsw *tsio.TSWriter) (err error) {
	for tsw.ContinuityCounter&0xf != 0x0 {
		if err = tsw.WritePackets(self.bufw, self.datav[:0], 0, false, true); err != nil {
			return
		}
	}
	return
}

func (self *Muxer) WriteTrailer() (err error) {

	if self.PaddingToMakeCounterCont {
		if self.astream != nil {
			if err = self.writePaddingTSPackets(self.astream.tsw); err != nil {
				return
			}
		}
		if self.vstream != nil {
			if err = self.writePaddingTSPackets(self.vstream.tsw); err != nil {
				return
			}
		}
	}
	self.bufw.Flush()
	return
}

func (self *Muxer) SetWriter(w io.WriteCloser) {
	self.w = w
	self.bufw = bufio.NewWriterSize(w, pio.RecommendBufioSize)
	return
}

func (self *Muxer) WritePATPMT() (err error) {
	pat := tsio.PAT{
		Entries: []tsio.PATEntry{
			{ProgramNumber: 1, ProgramMapPID: tsio.PMT_PID},
		},
	}
	patlen := pat.Marshal(self.psidata[tsio.PSIHeaderLength:])
	n := tsio.FillPSI(self.psidata, tsio.TableIdPAT, tsio.TableExtPAT, patlen)
	self.datav[0] = self.psidata[:n]
	if err = self.tswpat.WritePackets(self.bufw, self.datav[:1], 0, false, true); err != nil {
		return
	}

	var elemStreams []tsio.ElementaryStreamInfo

	//aac
	elemStreams = append(elemStreams, tsio.ElementaryStreamInfo{
		StreamType:    tsio.ElementaryStreamTypeAdtsAAC,
		ElementaryPID: self.astream.pid,
	})

	//h264
	elemStreams = append(elemStreams, tsio.ElementaryStreamInfo{
		StreamType:    tsio.ElementaryStreamTypeH264,
		ElementaryPID: self.vstream.pid,
	})

	pmt := tsio.PMT{
		PCRPID:                0x100,
		ElementaryStreamInfos: elemStreams,
	}

	pmtlen := pmt.Len()
	if pmtlen+tsio.PSIHeaderLength > len(self.psidata) {
		err = fmt.Errorf("ts.PMT.Too.Large")
		return
	}

	pmt.Marshal(self.psidata[tsio.PSIHeaderLength:])
	n = tsio.FillPSI(self.psidata, tsio.TableIdPMT, tsio.TableExtPMT, pmtlen)
	self.datav[0] = self.psidata[:n]
	if err = self.tswpmt.WritePackets(self.bufw, self.datav[:1], 0, false, true); err != nil {
		return
	}

	return
}

func (self *Muxer) WriteHeader() (err error) {

	self.vstream = &Stream{
		muxer:     self,
		pid:       videoPid,
		tsw:       tsio.NewTSWriter(videoPid),
	}

	self.astream = &Stream{
		muxer:     self,
		pid:       audioPid,
		tsw:       tsio.NewTSWriter(audioPid),
	}

	//write pat pmt
	if err = self.WritePATPMT(); err != nil {
		return
	}
	return
}

//const NGX_RTMP_HLS_DELAY = 63000
func (self *Muxer)WriteAudioPacket(pkts []*av.Packet,Cstream av.CodecData,pts uint64)(err error){

	datav:=make([][]byte,(len(pkts)+1)*2+1)
	audioLen:=0
	j:=1

	for i,_:= range pkts {
		codec := Cstream.(*aacparser.CodecData)
		aacparser.FillADTSHeader(self.adtshdr, codec.Config, 1024, len(pkts[i].Data[pkts[i].DataPos:]))
		datav[j] = self.adtshdr
		j++
		datav[j] = pkts[i].Data[pkts[i].DataPos:]
		j++
		audioLen+=len(self.adtshdr) + len(pkts[i].Data[pkts[i].DataPos:])
	}
	pts1:=audioTsToTs(pts)
	// pes heaer
	n := tsio.FillPESHeader(self.peshdr, tsio.StreamIdAAC,audioLen ,pts1, 0)
	datav[0] = self.peshdr[:n]

	// packet
	if err = self.astream.tsw.WritePackets(self.bufw, datav[:j],0, true, false); err != nil {
		return
	}
	return
}

func (self *Muxer) WritePacket(pkt *av.Packet,Cstream av.CodecData) (err error) {

	switch Cstream.Type() {

	case av.H264:
		codec := Cstream.(*h264parser.CodecData)

		nalus := self.nalus[:0]
		if pkt.IsKeyFrame {
			nalus = append(nalus, codec.SPS())
			nalus = append(nalus, codec.PPS())
		}

		pktnalus, _ := h264parser.SplitNALUs(pkt.Data[pkt.DataPos:])
		for _, pktnalu:= range pktnalus {
			nalus = append(nalus, pktnalu)
		}

		datav := self.datav[:1]
		for i, nalu := range nalus {
			if i == 0 {
				datav = append(datav, h264parser.AUDBytes)
			} else {
				datav = append(datav, h264parser.StartCodeBytes)
			}
			datav = append(datav, nalu)
		}
		pts:=tsio.TimeToTs(pkt.Time+pkt.CompositionTime)
		dts:=tsio.TimeToTs(pkt.Time)
		n := tsio.FillPESHeader(self.peshdr, tsio.StreamIdH264, -1, pts, dts)
		datav[0] = self.peshdr[:n]
		var pcr uint64
		if pkt.IsKeyFrame{
			pcr =tsio.TimeToTs(pkt.Time)
		}else{
			pcr = uint64(0)
		}
		if err = self.vstream.tsw.WritePackets(self.bufw, datav, pcr, pkt.IsKeyFrame, false); err != nil {
			return
		}
	}
	return
}
