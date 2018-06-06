package rtmp

import (
	"fmt"
	"time"
	"net/url"
	"runtime"
	"net"
	"rtmpServerStudy/timer"
)

const (
	ConnectStage = iota
	PublishStage
	PlayStage
)

func rtmpClientPullProxy(srcSession *Session,network,host,desUrl string,stage int) (err error) {

	var self *Session
	var url1 *url.URL
	url1 ,err = url.Parse(desUrl)
	proxyStage := stageClientConnect
	defer func() {
		if self != nil {
			self.rtmpCloseSessionHanler()
		}
		if err := recover(); err != nil  {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			fmt.Printf("rtmp: panic ClientSessionPrepare %v: %v\n%s", self.netconn.RemoteAddr(), err, string(buf))
		}
	}()
	isBreak := true
	connectErrTimes:=0
	for srcSession.isClosed != true{
		for (proxyStage < stage ) && srcSession.isClosed != true && isBreak{
			switch proxyStage {
			case stageClientConnect:
				var netConn net.Conn
				if netConn, err = Dial(network,host); err != nil {
					if connectErrTimes > 3{
						return err
					}
					connectErrTimes++
					time.Sleep(1*time.Second)
					continue
				}
				connectErrTimes = 0
				self = NewSsesion(netConn)
				self.network = network
				self.netconn = netConn
				self.URL = url1
				self.pubSession = srcSession
				proxyStage++
			case stageHandshakeStart:
				if err = self.handshakeClient(); err != nil {
					fmt.Printf("handshakeerr:%s\n",err)
					return err
				}
				proxyStage++
			case stageHandshakeDone:
				if err = self.connectPublish(); err != nil {
					if err.Error() == "NetStream.Publish.Bad"{
						return err
					}else{
						fmt.Println(err)
					}

					proxyStage = stageClientConnect
					time.Sleep(1*time.Second)
					continue
				}
			case stageSessionDone:
				proxyStage++
				isBreak = false
			}

		}
	}
	return
}

//hash pull rtmp trunk to right server
func (self *Session) connectPublish() (err error) {

	connectpath, publishpath := SplitPath(self.URL)

	//write connect
	self.OnStatusStage = ConnectStage
	self.isPull = true
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
	self.rtmpCmdHandler["_result"] =CheckCreateStreamResult
	//check create stream
	CreatStreamOk:=false
	for i:= 0;i<5;i++{
		if err = self.readChunk(RtmpMsgHandles); err != nil {
			return err
		}
		if self.resultCheckStage == StageCreateStreamResultChecked {
			CreatStreamOk = true
			break
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

	publishOk:=false
	for i:= 0;i<5;i++{
		if err = self.readChunk(RtmpMsgHandles); err != nil {
			return err
		}
		if self.resultCheckStage == StageOnstatusChecked{
			publishOk = true
			break
		}
	}
	if publishOk != true{
		err = fmt.Errorf("NetStream.Publish.Bad")
		return
	}

	if self.pubSession.isClosed != true {
		t := timer.GlobalTimerPool.Get(time.Second * MAXREADTIMEOUT)
		select {
		case self.pubSession.RegisterChannel <- self:
		case <-t.C:
		//may be is err
		}
		timer.GlobalTimerPool.Put(t)

	}else{
		err = fmt.Errorf("EOF")
		return
	}

	pubSession := self.pubSession
	pubSession.RLock()
	self.updatedGop = true
	self.aCodec = pubSession.aCodec
	self.vCodecData = pubSession.vCodecData
	self.aCodecData = pubSession.aCodecData
	self.vCodec = pubSession.vCodec
	self.metaData = pubSession.metaData
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

