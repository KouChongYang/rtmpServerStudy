package rtmp

import (
	"fmt"
	"rtmpServerStudy/utils/bits/pio"
	"rtmpServerStudy/log"
)

func RtmpUserStreamBeginHandler(session *Session, timestamp uint32, msgsid uint32, msgtypeid uint8, msgdata []byte) (err error) {

	log.Log.Info(fmt.Sprintf(" %s %s stream begin\n",
			session.LogFormat(),session.TcUrl))
	//do something your self
	return
}

func RtmpUserStreamEofHandler(session *Session, timestamp uint32, msgsid uint32, msgtypeid uint8, msgdata []byte) (err error) {

	log.Log.Info(fmt.Sprintf("%s %s stream eof",
			session.LogFormat(), session.TcUrl))
	return
	//do something your self
}

func RtmpUserSetBufLenHandler(session *Session, timestamp uint32, msgsid uint32, msgtypeid uint8, msgdata []byte) (err error) {

	log.Log.Info(fmt.Sprintf("%s %s RtmpUserSetBufLenHandler",
				session.LogFormat(), session.TcUrl))
	return
	//do something your self
}

func (self *Session) sendSetPingResponse(msgsid uint32, timestamp uint32) (err error) {
	b := self.GetWriteBuf(chunkHeaderLength + 6)
	n := self.fillChunk0Header(b, 2, 0, RtmpMsgUser, 0, 6)
	pio.PutU16BE(b[n:], RtmpUserPingResponse)
	n += 2
	pio.PutU32BE(b[n:], timestamp)
	n += 4
	_, err = self.bufw.Write(b[:n])
	return
}

func RtmpUserPingRequestHandler(session *Session, timestamp uint32, msgsid uint32, msgtypeid uint8, msgdata []byte) (err error) {
	time := pio.U32BE(msgdata[2:])
	err = session.sendSetPingResponse(msgsid, time)
	return
}

func RtmpUserPingResponseHandler(session *Session, timestamp uint32, msgsid uint32, msgtypeid uint8, msgdata []byte) (err error) {
	return
}

func RtmpUserUnknownHandler(session *Session, timestamp uint32, msgsid uint32, msgtypeid uint8, msgdata []byte) (err error) {
	fmt.Printf("%s %s RtmpUserSetBufLenHandler\n", session.App, session.TcUrl)
	return
	//do something your self
}
