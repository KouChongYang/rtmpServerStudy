package rtmp

import (
	"fmt"
	"github.com/nareix/bits/pio"
)

// parse peer set chunk  size
func RtmpMsgChunkSizeHandler (session *Session,timeStamp uint32,
				msgSID uint32, msgtypeid uint8, msgdata []byte) (err error) {
	msgLen := len(msgdata)
	if msgLen < 4 {
		err = fmt.Errorf("rtmp: short packet of SetChunkSize the len:%d",msgLen)
		return
	}
	session.readMaxChunkSize = int(pio.U32BE(msgdata))
	return

}

func (self *Session) writeSetChunkSize(size int) (err error) {
	self.writeMaxChunkSize = size
	b := self.GetWriteBuf(chunkHeaderLength + 4)
	n := self.fillChunkHeader(b, 2, 0, RtmpMsgChunkSize, 0, 4)
	pio.PutU32BE(b[n:], uint32(size))
	n += 4
	_, err = self.bufw.Write(b[:n])
	return
}

func RtmpMsgAbortHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){
	//something messg log
	return
}

func (self *Session) writeRtmpMsgAbort(msgsid uint32) (err error) {

	b := self.GetWriteBuf(chunkHeaderLength + 4)
	n := self.fillChunkHeader(b, 2, 0, RtmpMsgAbort, 0, 4)
	pio.PutU32BE(b[n:], uint32(msgsid))
	n += 4
	_, err = self.bufw.Write(b[:n])
	return
}

func RtmpMsgAckHanldler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){
	return
}

func (self *Session) writeRtmpMsgAck(seqnum uint32) (err error) {
	b := self.GetWriteBuf(chunkHeaderLength + 4)
	n := self.fillChunkHeader(b, 2, 0, RtmpMsgAck, 0, 4)
	pio.PutU32BE(b[n:], seqnum)
	n += 4
	_, err = self.bufw.Write(b[:n])
	return
}

func RtmpMsgUserEventHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){
	if len(msgdata) < 2 {
		err = fmt.Errorf("rtmp: short packet of UserControl")
		return
	}
	session.eventtype = pio.U16BE(msgdata)
	return RtmpControlMsgHandles[session.eventtype]
}

func RtmpMsgAckSizeHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

	msgLen := len(msgdata)
	if msgLen < 4 {
		err = fmt.Errorf("rtmp: short packet of SetChunkSize the len:%d",msgLen)
		return
	}
	session.ackSize = int(pio.U32BE(msgdata))
	return
}

func (self *Session) writeWindowAckSize(size uint32) (err error) {
	b := self.GetWriteBuf(chunkHeaderLength + 4)
	n := self.fillChunkHeader(b, 2, 0, RtmpMsgAckSize, 0, 4)
	pio.PutU32BE(b[n:], size)
	n += 4
	_, err = self.bufw.Write(b[:n])
	return
}

func RtmpMsgBandwidthHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){
	return

}

func RtmpMsgEdgeHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}

func RtmpMsgAudioHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}

func RtmpMsgVideoHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}

func RtmpMsgAmf3MetaHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}

func RtmpMsgAmf3SharedHandler(sesion *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}

func RtmpMsgAmf3CMDHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}

func RtmpMsgAmfMetaHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}

func RtmpMsgAmfSharedHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}

func RtmpMsgAmfCMDHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}

func RrmpMsgAggregateHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}