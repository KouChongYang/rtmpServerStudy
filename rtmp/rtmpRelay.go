package rtmp

import (
	"net"
	"net/url"
	"time"
	"fmt"
	"rtmpServerStudy/amf"
)

func ParseURL(uri string) (u *url.URL, err error) {
	if u, err = url.Parse(uri); err != nil {
		return
	}
	if _, _, serr := net.SplitHostPort(u.Host); serr != nil {
		u.Host += ":1935"
	}
	return
}

func (session *Session) AutoRelay(uri string, timeout time.Duration, retryTime int) (err error) {
	stage := stageHandshakeStart
	for i := 0; i < retryTime; i++ {
		switch stage {
		case stageHandshakeStart:
			if err = session.handshakeClient(); err != nil {
				stage = stageHandshakeStart
			}
		case stageHandshakeDone:
		}
	}
	return err
}

func (self *Session) connectPlay() (err error) {
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
			"flashVer":      "kingsoft paly",
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