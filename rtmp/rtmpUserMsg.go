package rtmp

func RtmpUserStreamBeginHandler (sesion *Session,timestamp uint32, msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}

func RtmpUserStreamEofHandler (sesion *Session,timestamp uint32, msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}

func RtmpUserStreamDRYHandler (sesion *Session,timestamp uint32, msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}

func RtmpUserSetBufLenHandler (sesion *Session,timestamp uint32, msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}

func RtmpUserRecordedHandler (sesion *Session,timestamp uint32, msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}

func RtmpUserPingRequestHandler (sesion *Session,timestamp uint32, msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}

func RtmpUserPingResponseHandler (sesion *Session,timestamp uint32, msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}

func RtmpUserUnknownHandler (sesion *Session,timestamp uint32, msgsid uint32, msgtypeid uint8, msgdata []byte) (err error){

}
