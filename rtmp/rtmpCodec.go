package rtmp

import (
	"container/list"
	"fmt"
	"rtmpServerStudy/aacParse"
	"rtmpServerStudy/av"
	"rtmpServerStudy/flv/flvio"
	"rtmpServerStudy/h264Parse"
	"rtmpServerStudy/h265Parse"
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
	AvHeader:=false
	dataPos:=n

	switch tag.CodecID {
	case flvio.VIDEO_H264:
		if !(tag.FrameType == flvio.FRAME_INTER || tag.FrameType == flvio.FRAME_KEY) {
			fmt.Println("parse frame err fomat is err")
			return
		}
		tag.Data = msgdata[n:]
		var stream h264parser.CodecData
		switch tag.AVCPacketType {
		case flvio.AVC_SEQHDR:
			fmt.Println("find avc seqhdr")
			if stream, err = h264parser.NewCodecDataFromAVCDecoderConfRecord(tag.Data); err != nil {
				return
			}
			session.Lock()
			session.vCodec = stream
			session.vCodecData = msgdata
			session.Unlock()
			AvHeader = true

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
				session.vCodec = stream
				session.Unlock()
			}
		}
	case flvio.VIDEO_H265:
		if !(tag.FrameType == flvio.FRAME_INTER || tag.FrameType == flvio.FRAME_KEY) {
			fmt.Println("parse frame err fomat is err")
			return
		}
		tag.Data = msgdata[n:]
		var stream h265parser.CodecData
		switch tag.AVCPacketType {
		case flvio.AVC_SEQHDR:
			fmt.Println("find avc seqhdr")
			if stream, err = h265parser.NewCodecDataFromAVCDecoderConfRecord(tag.Data); err != nil {
				return
			}
			session.Lock()
			session.vCodec = stream
			session.vCodecData = msgdata
			session.Unlock()
			AvHeader = true
		}
	}

	var pkt *av.Packet
	pkt, _ = TagToPacket(tag, int32(timestamp), msgdata)
	pkt.DataPos = dataPos
	//this is a long time lock may be something err must
	//every chunk check the register

	session.Lock()
	//session.updatedGop == true
	session.rtmpUpdateGopCache(pkt)
	session.ReadRegister()
	//session.updatedGop == true
	session.Unlock()


	var next *list.Element
	CursorList := session.CursorList.GetList()
	pkt.GopIsKeyFrame = pkt.IsKeyFrame
	for e := CursorList.Front(); e != nil; {
		switch value1 := e.Value.(type) {
		case *Session:
			cursorSession := value1
			if !cursorSession.isClosed {
				if cursorSession.needUpPkt == true  {
					if cursorSession.CurQue.RingBufferPut(pkt) != 0 {
						//fmt.Println("the cursorsession ring is full so drop the messg")
					}
					select {
					case cursorSession.PacketAck <- true:
					default:
					}
				}else{
					cursorSession.needUpPkt = true
				}
				e = e.Next()
			} else {
				next = e.Next()
				CursorList.Remove(e)
				e = next
			}

		}
	}

	if AvHeader == true {
		return
	}

	//startTime:=time.Now()

	if session.IsSelf == true {
		RecordHandler(session, session.vCodec, pkt)
	}
	//dis := time.Now().Sub(startTime).Nanoseconds()/1000
	//fmt.Println(dis)
	return
}

func (self *Session)ReadRegister(){
	if self.publishing != true{
		return
	}
	flag := 0
	CursorList := self.CursorList.GetList()
	for i := 0; i < MAXREGISTERCHANNEL; i++ {
		select {
		case registerSession, ok := <-self.RegisterChannel:
			if ok {
				if registerSession.updatedGop == true{
					registerSession.needUpPkt = true
				}
				CursorList.PushBack(registerSession)
			} else {
				//some log
				//may be register session is close
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
	AvHeader:=false
	dataPos:=n
	switch tag.SoundFormat {
	case flvio.SOUND_AAC:
		tag.Data = msgdata[n:]
		switch tag.AACPacketType {
		case flvio.AAC_SEQHDR:
			if len(tag.Data)==0{
				return
			}
			fmt.Println("find acc seqhdr")
			var stream aacparser.CodecData
			if stream, err = aacparser.NewCodecDataFromMPEG4AudioConfigBytes(tag.Data); err != nil {
				return
			}

			session.Lock()
			session.aCodec = stream
			session.aCodecData = msgdata
			session.Unlock()
			AvHeader = true
		}
	}
	var pkt *av.Packet
	pkt, _ = TagToPacket(tag, int32(timestamp), msgdata)
	pkt.DataPos = dataPos
	//this is a long time lock may be something err must
	//session.CursorList.Lock()
	if session.audioAfterLastVideoCnt > audioAfterLastVideoCnt {
		pkt.GopIsKeyFrame = true
	}
	session.Lock()
	//session.updatedGop == true
	session.rtmpUpdateGopCache(pkt)
	session.ReadRegister()
	//session.updatedGop == true
	session.Unlock()

	var next *list.Element
	CursorList := session.CursorList.GetList()
	for e := CursorList.Front(); e != nil; {
		switch value1 := e.Value.(type) {
		case *Session:
			cursorSession := value1
			if !cursorSession.isClosed {
				if cursorSession.needUpPkt == true {
					//jumst put may be the ring is full ,when the ring is full ,drop the pkt
					if cursorSession.CurQue.RingBufferPut(pkt) != 0 {
						//fmt.Println("the cursorsession ring is full so drop the messg")
					}
					//just ack
					select {
					case cursorSession.PacketAck <- true:
					default:
					}
				}else{
					cursorSession.needUpPkt = true
				}

				e = e.Next()
			} else {
				next = e.Next()
				CursorList.Remove(e)
				e = next
			}
		}
	}

	if AvHeader == true {
		return
	}
	//just hash push record
	if session.IsSelf == true {
		RecordHandler(session, session.aCodec, pkt)
	}
	return
}


func (session *Session) rtmpUpdateGopCache(pkt *av.Packet) (err error) {

	if session.vCodec == nil {
		return
	}

	if (session.vCodec).Type() != av.H264  &&  (session.vCodec).Type() != av.H265{
		return
	}

	//the first pkt must keyframe
	if session.curgopcount == 0 && pkt.IsKeyFrame != true {
		return
	}

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
