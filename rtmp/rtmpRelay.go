package rtmp

import (
	"net"
	"net/url"
	"time"
	"fmt"
	"rtmpServerStudy/amf"
	"runtime"
	//"github.com/aws/aws-sdk-go/aws/session"
	"context"
	"rtmpServerStudy/AvQue"
)

func rtmpClientRelayProxy(network,host,vhost,App,streamId,desUrl string,stage int) (err error) {

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
			fmt.Println("rtmp: panic ClientSessionPrepare %v: %v\n%s", self.netconn.RemoteAddr(), err, string(buf))
		}
	}()
	isBreak := true
	for isBreak {
		for (proxyStage < stage ) && isBreak{
			switch proxyStage {
			case stageClientConnect:
				var netConn net.Conn
				if netConn, err = Dial(network,host); err != nil {
					time.Sleep(1*time.Second)
					continue
				}
				self = NewSesion(netConn)
				self.network = network
				self.netconn = netConn
				self.URL = url1
				self.Vhost = vhost
				self.StreamId = streamId
				self.App = App
				proxyStage++
			case stageHandshakeStart:
				if err = self.handshakeClient(); err != nil {
					proxyStage = stageClientConnect
					time.Sleep(1*time.Second)
					continue
				}
				proxyStage++
			case stageHandshakeDone:
				if err = self.connectPlay(); err != nil {
					if self.rtmpCheckErr(err) != true{
						return err
					}
					proxyStage = stageClientConnect
					time.Sleep(1*time.Second)
					continue
				}
			case stageSessionDone:
				isBreak = false
			}

		}
	}
	return
}

func (self *Session) connectPlay() (err error) {
	//write connect
	connectpath, playpath := SplitPath(self.URL)
	fmt.Println(playpath)
	fmt.Println(connectpath)
	fmt.Println(self.URL.RawQuery)
	fmt.Println(self.URL.Path)
	fmt.Println(self.Vhost,self.App,self.StreamId)
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
		fmt.Printf("rtmp: > play('%s')\n", playpath)
	}
	self.rtmpCmdHandler["_result"] =CheckCreateStreamResult
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
	self.OnStatusStage = PlayStage
	self.rtmpCmdHandler["_result"] =CheckCreateStreamResult

	if err = self.writeCommandMsg(8, self.avmsgsid, "play", 0, nil, playpath); err != nil {
		return
	}
	if err = self.flushWrite(); err != nil {
		return
	}
	transid++


	playOk :=false
	for i:= 0;i<15;i++{
		if err = self.readChunk(RtmpMsgHandles); err != nil {
			if err.Error() == "NetConnection.Onstatus.Success" {
				playOk = true
				err = nil
				break
			}
			return err
		}
	}

	if playOk != true{
		err = fmt.Errorf("NetConnection.Play.err")
		return
	}
	self.StreamAnchor = self.StreamId + ":" + Gconfig.UserConf.PlayDomain[self.Vhost].UniqueName + ":" + self.App
	self.context, self.cancel = context.WithCancel(context.Background())
	self.GopCache = AvQue.RingBufferCreate(8)
	self.RegisterChannel = make(chan *Session, MAXREGISTERCHANNEL)
	ok := RtmpSessionPush(self)
	if !ok {
		err = fmt.Errorf("Already publishing")
	}
	self.publishing = true
	err = self.rtmpReadMsgCycle()

	return err
}

func (self *Session) writeConnect(path string) (err error) {
	if err = self.writeBasicConf(); err != nil {
		return
	}
	// > connect("app")
	if Debug {
		fmt.Printf("rtmp: > connect('%s') host=%s\n", path, self.URL.Host)
	}
	if err = self.writeCommandMsg(3, 0, "connect", 1,
		amf.AMFMap{
			"app":           path,
			"flashVer":      "Golang 1.8 rtmp server",
			"tcUrl":         getTcUrl(self.URL),
			"audioCodecs":   3575,
			"videoCodecs":   252,
			"videoFunction": 1,
		},
	); err != nil {
		return
	}

	if err = self.flushWrite(); err != nil {
		return
	}

	self.rtmpCmdHandler["_result"] =CheckConnectResult

	connectOk:=false
	for i:= 0;i<15;i++{
		if err = self.readChunk(RtmpMsgHandles); err != nil {
			if err.Error() == "NetConnection.Connect.Success" {
				connectOk = true
				err = nil
				break
			}
			return err
		}
	}

	if connectOk == false {
		err = fmt.Errorf("NetConnection.Connect.err")
		return
	}
	return
}