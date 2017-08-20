package rtmp

import (
	"time"
	"net/url"
	"net"
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

func Dial(uri string) (conn *Session, err error) {
	return DialTimeout(uri, 0)
}

func DialTimeout(uri string, timeout time.Duration) (session *Session, err error) {
	var u *url.URL
	if u, err = ParseURL(uri); err != nil {
		return
	}

	dailer := net.Dialer{Timeout: timeout}
	var netconn net.Conn
	if netconn, err = dailer.Dial("tcp", u.Host); err != nil {
		return
	}

	session = NewSesion(netconn)
	session.URL = u
	session.ClientSessionPrepare(stageSessionDone,prepareWriting)
	return
}

func (session *Session)AutoRelay(uri string ,timeout time.Duration,retryTime int)(err error){
	stage := stageHandshakeStart

	for i:=0;i< retryTime;i++{
		switch stage {
		case	stageHandshakeStart:
			if err = session.handshakeClient(); err != nil {
				stage = stageHandshakeStart
			}
		case stageHandshakeDone:
		}
	}
	return err
}


func  (self *Session)connectPlay()(err error){
	return err
}