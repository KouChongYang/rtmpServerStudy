package rtmp

import (
	"net/url"
	"net"
	"fmt"
	"time"
	"bufio"
	"github.com/nareix/bits/pio"
)

var(
	Debug = 1
)
type Server struct{
	RtmpAddr          string
	HttpAddr 	  string
	HandlePublish func(*Session)
	HandlePlay    func(*Session)
	HandleConn    func(*Session)
}

type Session struct {
	URL             	*url.URL
	isServerSession 	bool
	isPlay 			bool
	isPublish 		bool
	//状态机
	stage int
	//client
	netconn 	  	net.Conn
	readcsmap         	map[uint32]*chunkStream
	writeMaxChunkSize 	int
	readMaxChunkSize  	int
	writebuf 		[]byte
	readbuf  		[]byte
	bufr *bufio.Reader
	bufw *bufio.Writer
}

const(
	stageHandshakeStart = iota
	stageHandshakeDone
	stageCommandDone
	stageCodecDataDone
)

type chunkStream struct {
	timenow     uint32
	timedelta   uint32
	hastimeext  bool
	msgsid      uint32
	msgtypeid   uint8
	msgdatalen  uint32
	msgdataleft uint32
	msghdrtype  uint8
	msgdata     []byte
}

func NewSesion(netconn net.Conn) *Session {
	session := &Session{}
	session.netconn = netconn
	session.readcsmap = make(map[uint32]*chunkStream)
	session.readMaxChunkSize = 128
	session.writeMaxChunkSize = 128

	//
	session.bufr = bufio.NewReaderSize(netconn, pio.RecommendBufioSize)
	session.bufw = bufio.NewWriterSize(netconn, pio.RecommendBufioSize)
	session.writebuf = make([]byte, 4096)
	session.readbuf = make([]byte, 4096)
	return session
}

func (self *Session) prepare(stage int, flags int) (err error) {
	for self.stage < stage {
		switch self.stage {
		case stageHandshakeStart:
			if self.isServerSession {
				if err = self.handshakeServer(); err != nil {
					return
				}
			} else {
				if err = self.handshakeClient(); err != nil {
					return
				}
			}

		case stageHandshakeDone:
			if self.isServerSession {
				if err = self.readConnect(); err != nil {
					return
				}
			} else {
				if flags == prepareReading {
					if err = self.connectPlay(); err != nil {
						return
					}
				} else {
					if err = self.connectPublish(); err != nil {
						return
					}
				}
			}

		case stageCommandDone:
			if flags == prepareReading {
				if err = self.probe(); err != nil {
					return
				}
			} else {
				err = fmt.Errorf("rtmp: call WriteHeader() before WritePacket()")
				return
			}
		}
	}
	return
}

func (self *Server) SessionHandle(session *Session) (err error) {
	if self.HandleConn != nil {
		self.HandleConn(session)
	} else {
		if err = session.prepare(stageCommandDone, 0); err != nil {
			return
		}

		if session.isPlay {
			if self.HandlePlay != nil {
				self.HandlePlay(session)
			}
		} else if session.isPublish {
			if self.HandlePublish != nil {
				self.HandlePublish(session)
			}
		}
	}

	return
}

func (self *Server) ListenAndServe() (err error) {
	addr := self.RtmpAddr
	if addr == "" {
		addr = ":1935"
	}
	var tcpaddr *net.TCPAddr
	if tcpaddr, err = net.ResolveTCPAddr("tcp", addr); err != nil {
		err = fmt.Errorf("rtmp: ListenAndServe: %s", err)
		return
	}

	var listener *net.TCPListener
	if listener, err = net.ListenTCP("tcp", tcpaddr); err != nil {
		return
	}

	if Debug {
		fmt.Println("rtmp: server: listening on", addr)
	}

	for {
		var netconn net.Conn
		var tempDelay time.Duration

		netconn, e := listener.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				fmt.Printf("rtmp: Accept error: %v; retrying in %v\n", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			if Debug {
				fmt.Printf("rtmp: Accept error:%v\n",e)
			}
			return
		}
		tempDelay = 0

		if Debug {
			fmt.Println("rtmp: server: accepted")
		}

		session := NewSesion(netconn)
		session.isServerSession = true
		go func() {
			err := self.SessionHandle(session)
			if Debug {
				fmt.Println("rtmp: server: client closed err:", err)
			}
		}()
	}
}