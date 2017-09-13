package rtmp

import (
	"net"
	"fmt"
	"time"
)

const (
	ConnectStage = iota
	PublishStage
	PlayStage
)
//hash pull rtmp trunk to right server

func rtmpPush(srcsession *Session,network, host ,url string)(err error){
	var netConn net.Conn
	if netConn,err=Dial(network,host); err != nil{
		return
	}
	session:=NewSesion(netConn)
	session.network = network
	session.Host = host
	session.pubSession = srcsession
	session.stage = stageHandshakeStart
	go ClientSessionPrepare(session,stageSessionDone,preparePullWriting)
	return
}

func (self *Session) connectPublish() (err error) {
	connectpath, publishpath := SplitPath(self.URL)

	//write connect
	self.OnStatusStage = ConnectStage
	if err=self.writeConnect(connectpath);err != nil{
		return err
	}
	transid := 2
	// > createStream()
	if Debug {
		fmt.Printf("rtmp: > createStream()\n")
	}

	//create stream
	if err = self.writeCommandMsg(3, 0, "createStream", transid, nil); err != nil {
		return
	}
	if err = self.flushWrite(); err != nil {
		return
	}
	transid++

	if Debug {
		fmt.Printf("rtmp: > publish('%s')\n", publishpath)
	}
	//check create stream
	CreatStreamOk:=false
	for i:= 0;i<15;i++{
		if err = self.readChunk(RtmpMsgHandles); err != nil {
			if err.Error() == "NetConnection.CreateStream.Success" {
				CreatStreamOk = true
				err = nil
				break
			}
			return err
		}
	}

	if CreatStreamOk == false {
		err = fmt.Errorf("NetConnection.Connect.err")
		return
	}
	self.OnStatusStage++
	self.rtmpCmdHandler["_result"] =CheckCreateStreamResult

	if err = self.writeCommandMsg(8, self.avmsgsid, "publish", transid, nil, publishpath); err != nil {
		return
	}
	transid++
	if err = self.flushWrite(); err != nil {
		return
	}

	self.rtmpCmdHandler["onStatus"] =CheckOnStatus
	publishOk:=false
	for i:= 0;i<15;i++{
		if err = self.readChunk(RtmpMsgHandles); err != nil {
			if err.Error() == "NetConnection.Onstatus.Success" {
				publishOk = true
				err = nil
				break
			}
			return err
		}
	}

	if publishOk != true{
		err = fmt.Errorf("NetConnection.Publish.err")
		return
	}
	if self.pubSession.isClosed != true {
		select {
		case self.pubSession.RegisterChannel <- self:
		case <-time.After(time.Second * MAXREADTIMEOUT):
		//may be is err
		}
	}else{
		err = fmt.Errorf("EOF")
		return
	}
	pubSession := self.pubSession
	pubSession.RLock()
	self.aCodec = pubSession.aCodec
	self.vCodecData = pubSession.vCodecData
	self.aCodecData = pubSession.aCodecData
	self.vCodec = pubSession.vCodec
	//copy all gop just ptr copy
	self.GopCache = pubSession.GopCache.GopCopy()
	pubSession.RUnlock()

	self.context, self.cancel = pubSession.context, pubSession.cancel
	//send audio,video head and meta
	if err = self.rtmpSendHead(); err != nil {
		self.isClosed = true
		return err
	}
	//send gop for first screen
	if err = self.rtmpSendGop(); err != nil {
		self.isClosed = true
		return err
	}
	err = self.RtmpSendAvPackets()
	self.isClosed = true
	self.stage = stageSessionDone
	self.stage++
	return err
}

