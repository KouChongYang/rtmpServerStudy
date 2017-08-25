package rtmp

import (
	"container/list"
	"fmt"
	"rtmpServerStudy/aacParse"
	"rtmpServerStudy/av"
	"rtmpServerStudy/flv/flvio"
	"rtmpServerStudy/h264Parse"
	"encoding/hex"
)

func RtmpMsgDecodeVideoHandler(session *Session, timestamp uint32, msgsid uint32, msgtypeid uint8, msgdata []byte) (err error) {

	if msgtypeid != RtmpMsgVideo {
		return
	}
	if len(msgdata) == 0 {
		return
	}
	tag := new(flvio.Tag)
	tag.Type = flvio.TAG_VIDEO
	var n int
	if n, err = tag.ParseHeader(msgdata); err != nil {
		fmt.Println("parse frame hare err ")
		return
	}
	if tag.CodecID == flvio.VIDEO_H264 {
		if !(tag.FrameType == flvio.FRAME_INTER || tag.FrameType == flvio.FRAME_KEY) {
			fmt.Println("parse frame err fomat is err")
			return
		}
		tag.Data = msgdata[n:]
		var stream h264parser.CodecData
		switch tag.AVCPacketType {
		case flvio.AVC_SEQHDR:
			fmt.Println("find avc seqhdr")
			fmt.Println("=======================h264====")
			fmt.Println(hex.Dump(tag.Data))
			fmt.Println("================================")
			if stream, err = h264parser.NewCodecDataFromAVCDecoderConfRecord(tag.Data); err != nil {
				err = fmt.Errorf("flv: h264 seqhdr invalid")
				fmt.Println("++++++++++++++++++err h264 err")
				return
			}
			session.Lock()
			session.vCodec = &stream
			fmt.Println("=======================h264fffffffffffffffffffffffff====")
			fmt.Println(hex.Dump(session.vCodec.Record))
			fmt.Println("================================")
			session.Unlock()
		case flvio.AVC_NALU:
			b := tag.Data
			nalus, _ := h264parser.SplitNALUs(b)
			var sps, pps [][]byte
			for _, nalu := range nalus {
				if len(nalu) > 0 {
					naltype := nalu[0] & 0x1f
					switch {
					case naltype == 7:
						sps = append(sps, nalu)
					case naltype == 8:
						pps = append(pps, nalu)
					}
				}
			}
			if len(sps) > 0 && len(pps) > 0 {
				if stream, err = h264parser.NewCodecDataFromSPSAndPPS(sps, pps); err != nil {
					return
				}
				session.Lock()
				session.vCodec = &stream
				session.Unlock()
			}
		}
		//
	}

	var pkt *av.Packet
	pkt, _ = TagToPacket(tag, int32(timestamp), msgdata)
	//this is a long time lock may be something err must
	//session.CursorList.Lock()
	var next *list.Element
	CursorList := session.CursorList.GetList()
	pkt.GopIsKeyFrame = pkt.IsKeyFrame

	flag := 0
	for i := 0; i < MAXREGISTERCHANNEL; i++ {
		select {
		case registerSession, ok := <-session.RegisterChannel:
			if ok {
				CursorList.PushBack(registerSession)
			} else {
				//some log
				break
			}
		default:
			flag = 1
			break
		}
		if flag == 1 {
			break
		}
	}

	for e := CursorList.Front(); e != nil; {
		switch value1 := e.Value.(type) {
		case *Session:
			cursorSession := value1
			if !cursorSession.isClosed {
				if cursorSession.CurQue.RingBufferPut(pkt) != 0 {
					//fmt.Println("the cursorsession ring is full so drop the messg")
				}
				cursorSession.cond.Signal()
				next = e.Next()
			} else {
				next = e.Next()
				CursorList.Remove(e)
				e = next
			}

		}
	}
	session.rtmpUpdateGopCache(pkt)
	//session.CursorList.Unlock()
	return
}

func RtmpMsgDecodeAudioHandler(session *Session, timestamp uint32, msgsid uint32, msgtypeid uint8, msgdata []byte) (err error) {

	if msgtypeid != RtmpMsgAudio {
		return
	}
	if len(msgdata) == 0 {
		return
	}
	tag := new(flvio.Tag)
	tag.Type = flvio.TAG_AUDIO
	var n int
	if n, err = tag.ParseHeader(msgdata); err != nil {
		fmt.Println("parse frame hare err ")
		return
	}

	switch tag.SoundFormat {
	case flvio.SOUND_AAC:
		tag.Data = msgdata[n:]
		switch tag.AACPacketType {
		case flvio.AAC_SEQHDR:
			fmt.Println("find acc seqhdr")
			var stream aacparser.CodecData
			fmt.Println("=======================aac====")
			fmt.Println(hex.Dump(tag.Data))
			fmt.Println("================================")

			if stream, err = aacparser.NewCodecDataFromMPEG4AudioConfigBytes(tag.Data); err != nil {
				err = fmt.Errorf("flv: aac seqhdr invalid")
				fmt.Println(err)
				return
			}
			session.Lock()
			session.aCodec = &stream
			session.Unlock()
		}
	}
	var pkt *av.Packet
	pkt, _ = TagToPacket(tag, int32(timestamp), msgdata)
	//this is a long time lock may be something err must
	//session.CursorList.Lock()
	if session.audioAfterLastVideoCnt > audioAfterLastVideoCnt {
		pkt.GopIsKeyFrame = true
	}
	var next *list.Element
	CursorList := session.CursorList.GetList()

	flag := 0
	for i := 0; i < MAXREGISTERCHANNEL; i++ {
		select {
		case registerSession, ok := <-session.RegisterChannel:
			if ok {
				CursorList.PushBack(registerSession)
			} else {
				//some log
				break
			}
		default:
			flag = 1
			break
		}
		if flag == 1 {
			break
		}
	}

	for e := CursorList.Front(); e != nil; {
		switch value1 := e.Value.(type) {
		case *Session:
			cursorSession := value1
			if !cursorSession.isClosed {
				//jumst put may be the ring is full ,when the ring is full ,drop the pkt
				if cursorSession.CurQue.RingBufferPut(pkt) != 0 {
					//fmt.Println("the cursorsession ring is full so drop the messg")
				}
				cursorSession.cond.Signal()
				e = e.Next()
			} else {
				next = e.Next()
				CursorList.Remove(e)
				e = next
			}
		}
	}
	session.rtmpUpdateGopCache(pkt)
	//session.CursorList.Unlock()
	return
}


func (session *Session) rtmpUpdateGopCache(pkt *av.Packet) (err error) {

	if session.vCodec == nil {
		return
	}

	if (*session.vCodec).Type() != av.H264 {
		return
	}

	//the first pkt must keyframe
	if session.curgopcount == 0 && pkt.IsKeyFrame != true {
		return
	}

	session.Lock()
	des:= session.GopCache.RingBufferABSPut(pkt)
	if des == nil{
		session.GopCache.RingBufferCleanGop()
		session.curgopcount = 0
		session.audioAfterLastVideoCnt = 0
	}else{
		session.GopCache = des
	}
	if pkt.IsKeyFrame {
		session.curgopcount++
	}
	if pkt.PacketType == flvio.TAG_AUDIO {
		session.audioAfterLastVideoCnt++
	} else {
		session.audioAfterLastVideoCnt = 0
	}

	for session.curgopcount >= session.maxgopcount  {
		session.GopCache.RingBufferCleanOldGop()
		session.curgopcount--
		if session.curgopcount <= session.maxgopcount {
			break
		}
	}
	//
	if session.audioAfterLastVideoCnt > audioAfterLastVideoCnt {
		session.GopCache.RingBufferCleanGop()
		session.curgopcount = 0
		session.audioAfterLastVideoCnt = 0
	}
	//println("shrink", self.curgopcount, self.maxgopcount, self.buf.Head, self.buf.Tail, "count", self.buf.Count, "size", self.buf.Size)
	session.Unlock()
	return
}

func TagToPacket(tag *flvio.Tag, timestamp int32, b []byte) (pkt *av.Packet, ok bool) {
	pkt = new(av.Packet)
	switch tag.Type {
	case flvio.TAG_VIDEO:
		switch tag.AVCPacketType {
		case flvio.AVC_NALU:
			pkt.CompositionTime = flvio.TsToTime(tag.CompositionTime)
			pkt.IsKeyFrame = tag.FrameType == flvio.FRAME_KEY
		}
	case flvio.TAG_AUDIO:
	}
	pkt.PacketType = tag.Type
	pkt.Data = b
	pkt.Time = flvio.TsToTime(timestamp)
	return
}
