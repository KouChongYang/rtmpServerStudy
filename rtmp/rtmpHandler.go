package rtmp

import (
	"rtmpServerStudy/amf"

)

func (self *Session) writeDataMsg(csid, msgsid uint32, args ...interface{}) (err error) {
	return self.writeAMF0Msg(RtmpMsgAmfCMD, csid, msgsid, args...)
}

func (self *Session) writeCommandMsg(csid, msgsid uint32, args ...interface{}) (err error) {
	return self.writeAMF0Msg(RtmpMsgAmfCMD, csid, msgsid, args...)
}

func (self *Session) DoSend(b []byte, csid uint32, timestamp int32, msgtypeid uint8, msgsid uint32, msgdatalen int)(n int ,err error){

	bh := self.GetWriteBuf(chunkHeaderLength)

	pos:=0
	sn:=0
	last:=self.writeMaxChunkSize
	end:= msgdatalen

	for msgdatalen > 0{
		if pos == 0 {
			n := self.fillChunk0Header(bh, csid, timestamp, msgtypeid, msgsid, msgdatalen)
			_, err = self.bufw.Write(bh[:n])
		}else{
			n := self.fillChunk3Header(bh, csid, timestamp)
			_, err = self.bufw.Write(bh[:n])
		}
		if msgdatalen>self.writeMaxChunkSize {
			if sn, err = self.bufw.Write(b[pos:last]);err {
				return err
			}

			pos += sn
			last += sn
			msgdatalen -= sn
			continue
		}

		if sn, err = self.bufw.Write(b[pos:end]);err {
			return err
		}
	}
	return err
}

func (self *Session) writeAMF0Msg(msgtypeid uint8, csid, msgsid uint32, args ...interface{}) (err error) {

	size := 0
	for _, arg := range args {
		size += amf.LenAMF0Val(arg)
	}
	b := self.GetWriteBuf(size)
	n:=0
	for _, arg := range args {
		n += amf.FillAMF0Val(b[n:], arg)
	}
	_,err = self.DoSend(b,csid,0,msgtypeid,msgsid,size)
	return
}

func (self *Session) writeBasicConf() (err error) {
	// > SetChunkSize
	if err = self.writeSetChunkSize(self.writeMaxChunkSize); err != nil {
		return
	}
	// > WindowAckSize
	if err = self.writeWindowAckSize(5000000); err != nil {
		return
	}
	// > SetPeerBandwidth

	if err = self.writeSetPeerBandwidth(5000000, 2); err != nil {
		return
	}
	return
}