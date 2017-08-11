package rtmp

import (
	"fmt"
	"github.com/nareix/bits/pio"
	"rtmpServerStudy/amf"
)

// recv peer set chunk  size
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

//send our chunk size to peer
func (self *Session) writeSetChunkSize(size int) (err error) {
	self.writeMaxChunkSize = size
	b := self.GetWriteBuf(chunkHeaderLength + 4)
	n := self.fillChunk0Header(b, 2, 0, RtmpMsgChunkSize, 0, 4)
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
	n := self.fillChunk0Header(b, 2, 0, RtmpMsgAbort, 0, 4)
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
	n := self.fillChunk0Header(b, 2, 0, RtmpMsgAck, 0, 4)
	pio.PutU32BE(b[n:], seqnum)
	n += 4
	_, err = self.bufw.Write(b[:n])
	return
}

func RtmpMsgUserEventHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){
	msgLen := len(msgdata)
	if msgLen < 2 {
		err = fmt.Errorf("rtmp: short packet of UserControl the msgLen:%d",msgLen)
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
	n := self.fillChunk0Header(b, 2, 0, RtmpMsgAckSize, 0, 4)
	pio.PutU32BE(b[n:], size)
	n += 4
	_, err = self.bufw.Write(b[:n])
	return
}

func RtmpMsgBandwidthHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){
	return
}

func (self *Session) writeSetPeerBandwidth(acksize uint32, limittype uint8) (err error) {
	b := self.GetWriteBuf(chunkHeaderLength + 5)
	n := self.fillChunk0Header(b, 2, 0, RtmpMsgBandwidth, 0, 5)
	pio.PutU32BE(b[n:], acksize)
	n += 4
	//0,1,2
	b[n] = limittype
	n++
	_, err = self.bufw.Write(b[:n])
	return
}

func RtmpMsgAudioHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}

func RtmpMsgVideoHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}

func (self *Session) handleCommandMsgAMF0(b []byte) (n int, err error) {
	var name, transid, obj interface{}
	var size int

	if name, size, err = amf.ParseAMF0Val(b[n:]); err != nil {
		return
	}

	n += size
	if transid, size, err = amf.ParseAMF0Val(b[n:]); err != nil {
		return
	}
	n += size
	if obj, size, err = amf.ParseAMF0Val(b[n:]); err != nil {
		return
	}
	n += size

	var ok bool
	if commandname, ok := name.(string); !ok {
		err = fmt.Errorf("rtmp: CommandMsgAMF0 command is not string")
		return
	}

	commandtransid, _ := transid.(float64)
	commandobj, _ := obj.(amf.AMFMap)
	commandparams := []interface{}{}

	for n < len(b) {
		if obj, size, err = amf.ParseAMF0Val(b[n:]); err != nil {
			return
		}
		n += size
		commandparams = append(commandparams, obj)
	}
	if n < len(b) {
		err = fmt.Errorf("rtmp: CommandMsgAMF0 left bytes=%d", len(b)-n)
		return
	}

	return
}


func RtmpMsgAmf3Handler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){
	msgLen := len(msgdata)
	if msgLen < 1 {
		err = fmt.Errorf("rtmp: short packet of CommandMsgAMF3 the msgLen:%d",msgLen)
		return
	}
	// skip first byte
	if _, err = session.handleCommandMsgAMF0(msgdata[1:]); err != nil {
		return
	}
	return
}

func RtmpMsgAmfHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){
	/* AMF command names come with string type, but shared object names
  * come without type */
	if _, err = session.handleCommandMsgAMF0(msgdata); err != nil {
		return
	}
}

func RrmpMsgAggregateHandler(session *Session,timestamp uint32,
				msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}