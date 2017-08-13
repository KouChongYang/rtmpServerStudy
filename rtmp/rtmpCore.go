package rtmp

import (
	"net/url"
	"net"
	"fmt"
	"time"
	"bufio"
	"github.com/nareix/bits/pio"
	"context"
	"io"
	"encoding/hex"
	"sync"
	"container/list"
)

/* RTMP message types */
var(
	Debug = true
)

type sessionIndex map[string](*Session)
type sessionIndexStruct struct{
	sessionIndex *sessionIndex
	sync.RWMutex
}

type Server struct{
	RtmpAddr          string
	HttpAddr 	  string
	HandlePublish func(*Session)
	HandlePlay    func(*Session)
	HandleConn    func(*Session)
}

type Session struct {
	context           context.Context
	SubList  	  *list.List
	App               *string
	cancel            context.CancelFunc
	URL               *url.URL
	TcUrl		  *string
	isServer          bool
	isPlay            bool
	isPublish         bool
	connected         bool
	ackn              uint32
	readAckSize       uint32
	avmsgsid         uint32
	publishing      bool
	//状态机
	stage             int
	//client
	netconn           net.Conn
	readcsmap         map[uint32]*chunkStream
	writeMaxChunkSize int
	readMaxChunkSize  int
	chunkHeaderBuf    []byte
	writebuf          []byte
	readbuf           []byte
	bufr              *bufio.Reader
	bufw              *bufio.Writer
	commandtransid 	  float64
	gotmsg            bool
	gotcommand        bool
	metaversion       int
	eventtype         uint16
	ackSize           uint32
}

const(
	stageHandshakeStart = iota
	stageHandshakeDone
	stageSessionDone
)
const (
	prepareReading = iota + 1
	prepareWriting
)

const chunkHeaderLength = 12 + 4


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
	session.context , session.cancel = context.WithCancel(context.Background())
	//
	session.bufr = bufio.NewReaderSize(netconn, pio.RecommendBufioSize)
	session.bufw = bufio.NewWriterSize(netconn, pio.RecommendBufioSize)
	session.writebuf = make([]byte, 4096)
	session.readbuf = make([]byte, 4096)
	session.chunkHeaderBuf = make([]byte,chunkHeaderLength)
	return session
}


func (self *Session) GetWriteBuf(n int) []byte {
	if len(self.writebuf) < n {
		self.writebuf = make([]byte, n)
	}
	return self.writebuf
}

func (self *Session) fillChunk3Header(b[]byte,csid uint32,timestamp uint32)(n int){
	b[n] = (byte(csid) & 0x3f) | 0xC0
	n++
	if timestamp >= 0xffffff {
		pio.PutU32BE(b[n:], timestamp)
		n += 4
	}
	// always has header
	return
}

func (self *Session) fillChunk0Header(b []byte, csid uint32, timestamp uint32, msgtypeid uint8, msgsid uint32, msgdatalen int) (n int) {
	//  0                   1                   2                   3
	//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |                   timestamp                   |message length |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |     message length (cont)     |message type id| msg stream id |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |           message stream id (cont)            |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	//
	//       Figure 9 Chunk Message Header – Type 0

	b[n] = byte(csid) & 0x3f
	n++
	if timestamp <0xffffff {
		pio.PutU24BE(b[n:], uint32(timestamp))
	}else{
		pio.PutU24BE(b[n:], uint32(0xffffff))
	}
	n += 3
	pio.PutU24BE(b[n:], uint32(msgdatalen))
	n += 3
	b[n] = msgtypeid
	n++
	pio.PutU32LE(b[n:], msgsid)
	n += 4

	if timestamp >= 0xffffff {
		pio.PutU32BE(b[n:], timestamp)
		n += 4
	}
	if Debug {
		fmt.Printf("rtmp: write chunk msgdatalen=%d msgsid=%d\n", msgdatalen, msgsid)
	}

	return
}

func (self *Session) flushWrite() (err error) {
	if err = self.bufw.Flush(); err != nil {
		return
	}
	return
}

func (self *chunkStream) Start() {
	self.msgdataleft = self.msgdatalen
	self.msgdata = make([]byte, self.msgdatalen)
}

func (self *Session) readChunk() (err error) {

	b := self.readbuf
	n := 0
	if _, err = io.ReadFull(self.bufr, b[:1]); err != nil {
		return
	}
	header := b[0]
	n += 1

	var fmtTpye uint8
	var csid uint32

	fmtTpye = header >> 6

	csid = uint32(header) & 0x3f
	switch csid {
	default: // Chunk basic header 1
	case 0: // Chunk basic header 2
		if _, err = io.ReadFull(self.bufr, b[:1]); err != nil {
			return
		}
		n += 1
		csid = uint32(b[0]) + 64
	case 1: // Chunk basic header 3
		if _, err = io.ReadFull(self.bufr, b[:2]); err != nil {
			return
		}
		n += 2
		csid = uint32(pio.U16BE(b)) + 64
	}

	cs := self.readcsmap[csid]
	if cs == nil {
		cs = &chunkStream{}
		self.readcsmap[csid] = cs
	}

	var timestamp uint32

	switch fmtTpye {
	case 0:
		//  0                   1                   2                   3
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                   timestamp                   |message length |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |     message length (cont)     |message type id| msg stream id |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |           message stream id (cont)            |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//
		//       Figure 9 Chunk Message Header – Type 0
		if cs.msgdataleft != 0 {
			err = fmt.Errorf("rtmp: chunk msgdataleft=%d invalid", cs.msgdataleft)
			return
		}
		h := b[:11]
		if _, err = io.ReadFull(self.bufr, h); err != nil {
			return
		}
		n += len(h)
		timestamp = pio.U24BE(h[0:3])
		cs.msghdrtype = fmtTpye
		cs.msgdatalen = pio.U24BE(h[3:6])
		cs.msgtypeid = h[6]
		cs.msgsid = pio.U32LE(h[7:11])
		if timestamp == 0xffffff {
			if _, err = io.ReadFull(self.bufr, b[:4]); err != nil {
				return
			}
			n += 4
			timestamp = pio.U32BE(b)
			cs.hastimeext = true
		} else {
			cs.hastimeext = false
		}
		cs.timenow = timestamp
		cs.Start()

	case 1:
		//  0                   1                   2                   3
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                timestamp delta                |message length |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |     message length (cont)     |message type id|
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//
		//       Figure 10 Chunk Message Header – Type 1
		if cs.msgdataleft != 0 {
			err = fmt.Errorf("rtmp: chunk msgdataleft=%d invalid", cs.msgdataleft)
			return
		}
		h := b[:7]
		if _, err = io.ReadFull(self.bufr, h); err != nil {
			return
		}
		n += len(h)
		timestamp = pio.U24BE(h[0:3])
		cs.msghdrtype = fmtTpye
		cs.msgdatalen = pio.U24BE(h[3:6])
		cs.msgtypeid = h[6]
		if timestamp == 0xffffff {
			if _, err = io.ReadFull(self.bufr, b[:4]); err != nil {
				return
			}
			n += 4
			timestamp = pio.U32BE(b)
			cs.hastimeext = true
		} else {
			cs.hastimeext = false
		}
		cs.timedelta = timestamp
		cs.timenow += timestamp
		cs.Start()

	case 2:
		//  0                   1                   2
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                timestamp delta                |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		//
		//       Figure 11 Chunk Message Header – Type 2
		if cs.msgdataleft != 0 {
			err = fmt.Errorf("rtmp: chunk msgdataleft=%d invalid", cs.msgdataleft)
			return
		}
		h := b[:3]
		if _, err = io.ReadFull(self.bufr, h); err != nil {
			return
		}
		n += len(h)
		cs.msghdrtype = fmtTpye
		timestamp = pio.U24BE(h[0:3])
		if timestamp == 0xffffff {
			if _, err = io.ReadFull(self.bufr, b[:4]); err != nil {
				return
			}
			n += 4
			timestamp = pio.U32BE(b)
			cs.hastimeext = true
		} else {
			cs.hastimeext = false
		}
		cs.timedelta = timestamp
		cs.timenow += timestamp
		cs.Start()

	case 3:
		if cs.msgdataleft == 0 {
			switch cs.msghdrtype {
			case 0:
				if cs.hastimeext {
					if _, err = io.ReadFull(self.bufr, b[:4]); err != nil {
						return
					}
					n += 4
					timestamp = pio.U32BE(b)
					cs.timenow = timestamp
				}
			case 1, 2:
				if cs.hastimeext {
					if _, err = io.ReadFull(self.bufr, b[:4]); err != nil {
						return
					}
					n += 4
					timestamp = pio.U32BE(b)
				} else {
					timestamp = cs.timedelta
				}
				cs.timenow += timestamp
			}
			cs.Start()
		}

	default:
		err = fmt.Errorf("rtmp: invalid chunk msg header type=%d", fmtTpye)
		return
	}

	size := int(cs.msgdataleft)
	if size > self.readMaxChunkSize {
		size = self.readMaxChunkSize
	}

	off := cs.msgdatalen - cs.msgdataleft
	buf := cs.msgdata[off : int(off)+size]
	if _, err = io.ReadFull(self.bufr, buf); err != nil {
		return
	}

	n += len(buf)
	cs.msgdataleft -= uint32(size)

	if Debug {
		fmt.Printf("rtmp: chunk msgsid=%d msgtypeid=%d msghdrtype=%d len=%d left=%d\n",
			cs.msgsid, cs.msgtypeid, cs.msghdrtype, cs.msgdatalen, cs.msgdataleft)
	}

	if cs.msgdataleft == 0 {
		if Debug {
			fmt.Println("rtmp: chunk data")
			fmt.Print(hex.Dump(cs.msgdata))
		}
		if RtmpMsgHandles[cs.msgtypeid] != nil {
			if err = RtmpMsgHandles[cs.msgtypeid](self,cs.timenow, cs.msgsid, cs.msgtypeid, cs.msgdata);err!=nil {
				return
			}
		}
	}

	self.ackn += uint32(n)
	if self.readAckSize != 0 && self.ackn > self.readAckSize {
		if err = self.writeRtmpMsgAck(self.ackn); err != nil {
			return
		}
		self.ackn = 0
	}

	return
}

func (self *Session)rtmpReadMsgCycle()(err error) {
	for {
		if err = self.readChunk();err != nil{
			return err
		}
	}
	return
}

func (self *Session)rtmpCloseSessionHanler(){

}

func  (self *Session)connectPlay()(err error){
	return err
}

func (self *Session)connectPublish()(err error){
	return err
}

func (self *Session) ClientSessionPrepare(stage,flags int)(err error){
	for self.stage < stage {
		switch self.stage {
		case stageHandshakeStart:
			if err = self.handshakeClient(); err != nil {
				return
			}
		case stageHandshakeDone:
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

	}
	return
}

func (self *Session) ServerSessionPrepare(stage int, flags int) (err error) {

	for self.stage < stage {
		switch self.stage {
		//first handshake
		case stageHandshakeStart:
			if err = self.handshakeServer(); err != nil {
				return
			}
		case stageHandshakeDone:
			err = self.rtmpReadMsgCycle()
		case stageSessionDone:
			//some thing close handler
			self.rtmpCloseSessionHanler()
		}
	}
	return
}

func (self *Server) ServerHandle(session *Session) (err error) {

	if err = session.ServerSessionPrepare(stageSessionDone, 0); err != nil {
			return
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
		session.isServer = true
		go func() {
			err := self.ServerHandle(session)
			if Debug {
				fmt.Println("rtmp: server: client closed err:", err)
			}
		}()
	}
}