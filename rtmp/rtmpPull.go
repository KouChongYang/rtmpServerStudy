package rtmp

import (
	"fmt"
	"time"
	"net/url"
	"runtime"
	"net"
	"rtmpServerStudy/timer"
	"rtmpServerStudy/log"
)

const (
	ConnectStage = iota
	PublishStage
	PlayStage
)

func rtmpClientPullProxy(srcSession *Session,network,host,desUrl string,stage int) {

	var self *Session
	var url1 *url.URL
	var err error
	url1 ,err = url.Parse(desUrl)
	if err != nil {
		log.Log.Info(fmt.Sprintf("%s rtmp pull proxy parse url :%s err: %s",
			srcSession.LogFormat(),desUrl,err.Error()))
		return
	}

	log.Log.Info(fmt.Sprintf("%s rtmp pull proxy start desurl:%s deshost:%s",
		srcSession.LogFormat(),desUrl,host))

	proxyStage := stageClientConnect
	defer func() {
		log.Log.Info(fmt.Sprintf("%s rtmp auto push close the desurl:%s",
			srcSession.LogFormat(),desUrl))

		if self != nil {
			self.rtmpCloseSessionHanler()
		}
		if err := recover(); err != nil  {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			log.Log.Info(fmt.Sprintf("%s rtmp panic err pull proxy recover err :%s",
				srcSession.LogFormat(),string(buf)))
		}
	}()
	isBreak := true
	connectErrTimes:=0

	for (proxyStage < stage ) && srcSession.isClosed != true && isBreak{

		switch proxyStage {
			case stageClientConnect:
				var netConn net.Conn
				if netConn, err = DialTimeout(network,host,time.Duration(Gconfig.RtmpServer.SendTimeOut)*time.Second); err != nil {
					if connectErrTimes > 3{
						log.Log.Info(fmt.Sprintf("%s rtmp pull proxy dialtimeout host:%s err: %s",
							srcSession.LogFormat(),host,err.Error()))
						return
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
				tcpConn, ok := netConn.(*net.TCPConn)
				if !ok {
					//error handle
				}
				tcpConn.SetNoDelay(true)
				self.RemoteAddr = tcpConn.RemoteAddr().String()
				proxyStage++

			case stageHandshakeStart:
				if err = self.handshakeClient(); err != nil {
					log.Log.Info(fmt.Sprintf("%s rtmp pull proxy handshake err: %s",
						srcSession.LogFormat(),err.Error()))
					return
				}
				proxyStage++
			case stageHandshakeDone:
				if err = self.connectPublish(); err != nil {
					log.Log.Info(fmt.Sprintf("%s rtmp pull proxy connectPublish err: %s",
						srcSession.LogFormat(),err.Error()))
					return
				}
				proxyStage++
			case stageSessionDone:
				proxyStage++
				isBreak = false
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
	log.Log.Info(fmt.Sprintf("%s rtmp pull connectPublish send rtmp connect cmd",
		self.LogFormat()))
	if err=self.writeConnect(connectpath);err != nil{
		return err
	}
	transid := 2
	// > createStream()

	log.Log.Info(fmt.Sprintf("%s rtmp pull connectPublish sedn rtmp createstream cmd",
		self.LogFormat()))

	//create stream
	if err = self.writeCommandMsg(3, 0, "createStream", transid, nil); err != nil {
		return
	}
	if err = self.flushWrite(); err != nil {
		return
	}
	transid++

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

	log.Log.Info(fmt.Sprintf("%s rtmp pull connectPublish send rtmp publish cmd",
		self.LogFormat()))
//5
	if err = self.writeCommandMsg(8, self.avmsgsid, "publish", transid, nil, publishpath,"live"); err != nil {
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
		//self.resultCheckStage = StageOnstatusChecked
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

