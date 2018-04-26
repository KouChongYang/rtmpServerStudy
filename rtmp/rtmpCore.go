package rtmp

import (
	"bufio"
	"context"
	"fmt"
	"rtmpServerStudy/utils/bits/pio"
	"io"
	"net"
	"net/url"
	"time"
	//"encoding/hex"
	"sync"
	"rtmpServerStudy/AvQue"
	//"rtmpServerStudy/aacParse"
	"rtmpServerStudy/flv/flvio"
	"strings"
	"rtmpServerStudy/av"
	//"encoding/hex"
	"hash/fnv"
	"runtime"
	"os"
	"rtmpServerStudy/config"
	"rtmpServerStudy/amf"
	"github.com/lucas-clemente/quic-go"
	"crypto/tls"
	"crypto/rsa"
	"crypto/x509"
	"crypto/rand"
	"math/big"
	"encoding/pem"
	"github.com/xtaci/kcp-go"
)

/* RTMP message types */
var (
	Debug = false
	EXTTIME = true
	FlvRecord = true
	HlsRecord = true
)

var Gconfig *config.RtmpServerCnf

const (
	MAXREGISTERCHANNEL     = 512
	audioAfterLastVideoCnt = 115
	MAXREADTIMEOUT = 60
	HashMapFactors = 101
)

type sessionIndex map[string](*Session)
type sessionIndexStruct struct {
	sessionIndex sessionIndex
	sync.RWMutex
}


var PublishingSessionMap [HashMapFactors]sessionIndexStruct

type relaySessionIndex map[string]bool
type relaySessionIndexStruct struct {
	sessionIndex relaySessionIndex
	sync.RWMutex
}

var RelaySessionMap [HashMapFactors]relaySessionIndexStruct

type Server struct {
	RtmpAddr      []string
	HttpAddr      []string
	QuicAddr      string
	KcpAddr       string
	done          chan bool
	HandlePublish func(*Session)
	HandlePlay    func(*Session)
	HandleConn    func(*Session)
}

type Session struct {
	sync.RWMutex
	lock                   *sync.RWMutex
	context                context.Context
	CursorList             *AvQue.CursorList
	GopCache               *AvQue.AvRingbuffer
	RecodeCachedPkts       []av.Packet
	pubSession 	       *Session
	rtmpCmdHandler         RtmpCmdHandle
	selfPush               bool
	relayPush              bool
	UserCnf                config.App
	Vhost                  string
	RecordMuxerCnf         []*RecordMuxerInfo//hls,flv,other
	maxgopcount            int
	audioAfterLastVideoCnt int
	CurQue                 *AvQue.AvRingbuffer
	vCodec            av.CodecData
	vCodecData        []byte
	aCodec            av.CodecData
	aCodecData        []byte
	RegisterChannel   chan *Session
	PacketAck         chan bool
	curgopcount       int
	QuicOn            bool
	QuicConn quic.Stream


	App               string
	StreamId          string
	StreamAnchor      string
	uniqueName 	  string

	cancel            context.CancelFunc
	URL               *url.URL
	TcUrl             string
	isClosed          bool
	isServer          bool
	isPlay            bool
	isPull            bool
	connected         bool
	ackn              uint32
	readAckSize       uint32
	avmsgsid          uint32
	publishing        bool
	playing           bool
	isRelay           bool
	//状态机
	stage             int
	resultCheckStage  int
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
	commandtransid    float64
	gotmsg            bool
	gotcommand        bool
	metaversion       int
	metaData         amf.AMFMap
	eventtype         uint16
	ackSize           uint32
	pushIp            string
	network           string
	Host              string
	OnStatusStage     int
	updatedGop        bool
	needUpPkt         bool

	//record 时间 创建目录用
	recordTime        time.Time
	//hls 直播录制ts状态信息
	hlsLiveRecordInfo hlsLiveRecordInfo
	flvReordInfo  flvReordInfo
}

const (
	stageClientConnect = iota
	stageHandshakeStart
	stageHandshakeDone
	stageCommandDone
	stageSessionDone
)

const (
	StageConnectResultChecked = iota + 1
	StageCreateStreamResultChecked
	StageOnstatusChecked
)

const (
	preparePlayReading = iota + 1
	preparePullWriting
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


func init() {
	for i := 0; i < HashMapFactors; i++ {
		PublishingSessionMap[i].sessionIndex = make(sessionIndex)
		RelaySessionMap[i].sessionIndex = make(relaySessionIndex)
	}
}

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func RtmpSessionGet(path string)(session *Session){
	i:=hash(path)%HashMapFactors
	PublishingSessionMap[i].RLock()
	defer 	PublishingSessionMap[i].RUnlock()
	pubSession, ok := PublishingSessionMap[i].sessionIndex[path]
	if ok {
		return pubSession
	}else{
		return nil
	}
	return
}

//
func RtmpSessionPush(session *Session) bool{
	path:= session.StreamAnchor
	i:=hash(path)%HashMapFactors
	PublishingSessionMap[i].Lock()
	defer 	PublishingSessionMap[i].Unlock()
	if _, ok:= PublishingSessionMap[i].sessionIndex[path]; ok{
		return false
	}else{
		PublishingSessionMap[i].sessionIndex[path] = session
	}
	return true
}

func RtmpSessionDel(session *Session) {
	path:= session.StreamAnchor
	i:=hash(path)%HashMapFactors
	PublishingSessionMap[i].Lock()
	delete(PublishingSessionMap[i].sessionIndex,path)
	PublishingSessionMap[i].Unlock()
}

func NewSsesion(netconn net.Conn) *Session {
	session := &Session{}
	session.netconn = netconn
	session.readcsmap = make(map[uint32]*chunkStream)
	session.readMaxChunkSize = 128
	session.writeMaxChunkSize = 128
	session.CursorList = AvQue.NewPublist()
	session.maxgopcount = 2
	session.rtmpCmdHandler = newRtmpCmdHandler()
	session.lock = &sync.RWMutex{}
	session.stage = stageHandshakeStart
	session.metaData = amf.AMFMap{}
	//just for regist cursor session
	//session.RegisterChannel = make(chan *Session, MAXREGISTERCHANNEL)
	//true register ok ,false register false

	session.PacketAck = make(chan bool, 1)

	//this maybe
	//session.context , session.cancel = context.WithCancel(context.Background())
	//
	session.bufr = bufio.NewReaderSize(netconn, pio.RecommendBufioSize)
	session.bufw = bufio.NewWriterSize(netconn, pio.RecommendBufioSize)
	session.writebuf = make([]byte, 4096)
	session.readbuf = make([]byte, 4096)
	session.chunkHeaderBuf = make([]byte, chunkHeaderLength)
	//session.GopCache = AvQue.RingBufferCreate(8)
	session.CurQue = AvQue.RingBufferCreate(10) //
	return session
}

func (self *Session) GetWriteBuf(n int) []byte {
	if len(self.writebuf) < n {
		self.writebuf = make([]byte, n)
	}
	return self.writebuf
}

func (self *Session) fillChunk3Header(b []byte, csid uint32, timestamp uint32) (n int) {
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
	if timestamp < 0xffffff {
		pio.PutU24BE(b[n:], uint32(timestamp))
	} else {
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

func (self *Session) readChunk(hands RtmpMsgHandle) (err error) {

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
		csid = uint32(b[1]) + 64
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
			err = fmt.Errorf("Rtmp.ReadChunk.MsgdataLeft(%d invalid)", cs.msgdataleft)
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
			err = fmt.Errorf("Rtmp.ReadChunk.MsgdataLeft(%d invalid)", cs.msgdataleft)
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
			err = fmt.Errorf("Rtmp.ReadChunk.MsgdataLeft(%d invalid)", cs.msgdataleft)
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
		}else{
			if cs.hastimeext {
				tmpb, tmperr := self.bufr.Peek(4)
				if tmperr != nil {
					return tmperr
				}
				timestamp = pio.U32BE(tmpb)
				if timestamp == cs.timenow {
					self.bufr.Discard(4)
				}
			}
		}

	default:
		err = fmt.Errorf("Rtmp.Invalid.ChunkMsg.Header(type=%d)", fmtTpye)
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
			//fmt.Print(hex.Dump(cs.msgdata))
		}

		if hands[cs.msgtypeid] != nil {
			if err = hands[cs.msgtypeid](self, cs.timenow, cs.msgsid, cs.msgtypeid, cs.msgdata); err != nil {
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

func (self *Session) rtmpReadCmdMsgCycle() (err error) {
	for {
		if err = self.readChunk(RtmpMsgHandles); err != nil {
			return err
		}
		if self.publishing || self.playing {
			return
		}
	}
	return
}

func (self *Session) rtmpReadMsgCycle() (err error) {
	for {
		if err = self.readChunk(RtmpMsgHandles); err != nil {
			return err
		}
	}
	return
}

func (self *Session) rtmpClosePlaySession(){
	self.isClosed = true
	self.GopCache = nil
	self.aCodec = nil
	self.vCodec = nil
	self.context = nil

	self.netconn.Close()
	//some
}

func (self *Session) writeAVTag(tag *flvio.Tag, ts int32) (err error) {
	var msgtypeid uint8
	var csid uint32

	switch tag.Type {
	case flvio.TAG_AUDIO:
		msgtypeid = RtmpMsgAudio
		csid = 6
	case flvio.TAG_VIDEO:
		msgtypeid = RtmpMsgVideo
		csid = 7
	}
	n := 0
	n, err = self.DoSend(tag.Data, csid, uint32(ts), msgtypeid, self.avmsgsid, len(tag.Data))
	fmt.Println("send byte :%d", n)
	return
}

func (self *Session) writeAVPacket(packet *av.Packet) (err error) {
	var msgtypeid uint8
	var csid uint32

	switch packet.PacketType {
	case RtmpMsgAudio:
		msgtypeid = RtmpMsgAudio
		csid = 6
	case RtmpMsgVideo:
		msgtypeid = RtmpMsgVideo
		csid = 7
	}
	//n := 0
	ts := flvio.TimeToTs(packet.Time)

	//DoSend(b []byte, csid uint32, timestamp uint32, msgtypeid uint8, msgsid uint32, msgdatalen int)(n int ,err error){
	_, err = self.DoSend(packet.Data, csid, uint32(ts), msgtypeid, self.avmsgsid, len(packet.Data))
	//fmt.Println("send byte :%d", n)
	return
}

func (self *Session)CodecDataToTag(stream av.CodecData) (tag *flvio.Tag, ok bool, err error) {
	tag = new(flvio.Tag)
	switch stream.Type() {
	case av.H264:
		fmt.Println("head:h264")
		tag.Type = flvio.TAG_VIDEO
		tag.AVCPacketType = flvio.AVC_SEQHDR
		tag.CodecID = flvio.VIDEO_H264
		tag.Data = self.vCodecData
		ok = true
		tag = tag
	case av.H265:
		fmt.Println("head:h265")
		tag.Type = flvio.TAG_VIDEO
		tag.AVCPacketType = flvio.AVC_SEQHDR
		tag.CodecID = flvio.VIDEO_H265
		tag.Data = self.vCodecData
		ok = true
		tag = tag
	case av.AAC:
		tag.Type = flvio.TAG_AUDIO
		tag.SoundFormat =    flvio.SOUND_AAC
		tag.SoundRate = flvio.SOUND_44Khz
		tag.AACPacketType = flvio.AAC_SEQHDR
		tag.Data = self.aCodecData
		ok = true
	default:
		err = fmt.Errorf("Rtmp.Unspported.CodecType.%v", stream.Type())
		return
	}
	return
}

func DialTimeout(network,host string ,timeout time.Duration) (netconn net.Conn,err error) {
	dailer := net.Dialer{Timeout: timeout}

	if netconn, err = dailer.Dial(network, host); err != nil {
		return
	}

	return
}

func Dial(network,host string) (netconn net.Conn,err error) {
	return DialTimeout(network,host,5*time.Second)
}

//rtmp://tmp.socket0/live/streamid
//rtmp://test.uplive.com/live/streamid
//rtmp://127.0.0.1/live?vhost=test.uplive.com/123

func (self *Session)rtmpCheckErr(err error) bool{
	return true
}

func (self *Server) ServerHandle(session *Session) (err error) {

	if err = session.ServerSession(stageSessionDone); err != nil {
		return
	}
	return
}

func createUnixSocket(addr *net.UnixAddr) (l net.Listener, err error) {
	l, err = net.ListenUnix("unix", addr)

	if err != nil {
		return
	}

	fi, err := os.Stat(addr.String())
	if err != nil {
		return
	}

	err = os.Chmod(addr.String(), fi.Mode()|0060)
	return
}

func (srv *Server) socketListen(addr string) (net.Listener, error) {
	var err error
	var l net.Listener
	if addr == "" {
		addr = ":1935"
	}
	if strings.HasPrefix(addr, "/") {
		var laddr *net.UnixAddr
		if laddr, err = net.ResolveUnixAddr("unix", addr); err != nil {
			return nil, err
		}
		if l, err = createUnixSocket(laddr); err != nil {
			// Unix-domain-socket already exists, try to connect to it to
			// see if it still is usedb by another process
			if _, err = net.Dial("unix", addr); err != nil {
				if err = os.Remove(addr); err != nil {
					return nil, err
				}
				if l, err = createUnixSocket(laddr); err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("another process seems to be listening on %s already", addr)
			}
		}
	} else {
		var laddr *net.TCPAddr
		if laddr, err = net.ResolveTCPAddr("tcp", addr); err != nil {
			return nil, err
		}
		if l, err = net.ListenTCP("tcp", laddr); err != nil {
			return nil, err
		}
	}

	return l, nil

}



func (self *Server)ListenAndServersStart(){

	self.done = make(chan bool)
	//rtmp server start
	for _, addr := range self.RtmpAddr {
		go self.rtmpServeStart(addr)
	}

	if len(self.KcpAddr)>0{
		fmt.Println(self.KcpAddr)
		go self.rtmpKcpServerStart(self.KcpAddr)
	}

	if len(self.QuicAddr) >0 {
		fmt.Println(self.QuicAddr)
		go self.rtmpQuicServerStart(self.QuicAddr)
	}


	//http server start
	for _, addr :=  range self.HttpAddr{
		go self.httpServerStart(addr)
	}
	<-self.done
}

func NewServer(file string) (err error,server *Server){
	server = new(Server)
	if err,Gconfig = config.ParseConfig(file);err != nil{
		return
	}
	server.RtmpAddr ,server.HttpAddr,server.QuicAddr,server.KcpAddr =
		Gconfig.RtmpServer.RtmpListen,Gconfig.RtmpServer.HttpListen ,Gconfig.RtmpServer.QuicListen,Gconfig.RtmpServer.KcpListen
	return
}

func (self *Server) rtmpServeStart(addr string,) (err error) {

	if addr == "" {
		addr = ":1935"
	}

	defer func(){
		self.done <- false
	}()

	var listener net.Listener
	if listener,err = self.socketListen(addr); err != nil {
		return err
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
				fmt.Printf("rtmp: Accept error:%v\n", e)
			}
			return e
		}
		tempDelay = 0

		if Debug {
			fmt.Println("rtmp: server: accepted")
		}
		tcpConn, ok := netconn.(*net.TCPConn)
		if !ok {
			//error handle
		}
		tcpConn.SetNoDelay(true)
		session := NewSsesion(netconn)
		session.isServer = true
		go func() {
			defer func() {
				if err := recover(); err != nil  {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					session.rtmpCloseSessionHanler()
					fmt.Println("rtmp: panic serving %v: %v\n%s", session.netconn.RemoteAddr(), err, string(buf))
				}
			}()

			err := self.ServerHandle(session)
			if Debug {
				fmt.Println("rtmp: server: client closed err:", err)
			}
		}()
	}
}

func NewQuicSesion(netconn quic.Stream) *Session {
	session := &Session{}
	session.readcsmap = make(map[uint32]*chunkStream)
	session.readMaxChunkSize = 128
	session.writeMaxChunkSize = 4096
	session.CursorList = AvQue.NewPublist()
	session.maxgopcount = 2
	session.rtmpCmdHandler = newRtmpCmdHandler()
	session.lock = &sync.RWMutex{}
	session.stage = stageHandshakeStart
	session.metaData = amf.AMFMap{}
	session.QuicOn = true
	session.QuicConn = netconn
	//just for regist cursor session
	//session.RegisterChannel = make(chan *Session, MAXREGISTERCHANNEL)
	//true register ok ,false register false

	session.PacketAck = make(chan bool, 1)

	//this maybe
	//session.context , session.cancel = context.WithCancel(context.Background())
	//
	session.bufr = bufio.NewReaderSize(netconn, 4096)
	session.bufw = bufio.NewWriterSize(netconn, 4096)
	//session.bufr = bufio.NewReaderSize(netconn, pio.RecommendBufioSize)
	session.writebuf = make([]byte, 4096)
	session.readbuf = make([]byte, 4096)
	session.chunkHeaderBuf = make([]byte, chunkHeaderLength)
	//session.GopCache = AvQue.RingBufferCreate(8)
	session.CurQue = AvQue.RingBufferCreate(10) //
	return session
}

func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{Certificates: []tls.Certificate{tlsCert}}
}

//quic
func (self * Server)rtmpQuicServerStart(addr string)(err error) {

	listener, err := quic.ListenAddr(addr, generateTLSConfig(), nil)
	if err != nil {
		return err
	}

	for {
		sess, err := listener.Accept()
		if err != nil {
			return err
		}
		stream, err := sess.AcceptStream()
		if err != nil {
			sess.Close(err)
			fmt.Println("udp accept err:",err)
			continue
		}
		session := NewQuicSesion(stream)
		session.isServer = true
		go func() {
			defer func() {
				if err := recover(); err != nil  {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					session.rtmpCloseSessionHanler()
					fmt.Println("rtmp: panic serving %v: %v\n%s", session.netconn.RemoteAddr(), err, string(buf))
				}
			}()

			err := self.ServerHandle(session)
			if Debug {
				fmt.Println("rtmp: server: client closed err:", err)
			}
		}()
	}
}

//kcp
func (self * Server)rtmpKcpServerStart(addr string)(err error) {

	/*var pass = pbkdf2.Key(key, []byte(SALT), 4096, 32, sha1.New)
	block, _ := kcp.NewSalsa20BlockCrypt(pass)
    */
	//listener, err := kcp.ListenWithOptions(addr, nil, 10, 3)
	//no key no fec
	listener, err := kcp.ListenWithOptions(addr, nil, -1, -1)
	if err != nil {
		fmt.Println(err)
	}
	kcplistener := listener
	kcplistener.SetReadBuffer(4 * 1024 * 1024)
	kcplistener.SetWriteBuffer(4 * 1024 * 1024)
	kcplistener.SetDSCP(46)
	for {
		s, err := listener.Accept()
		if err != nil {
			return err
		}

		// coverage test
		s.(*kcp.UDPSession).SetReadBuffer(4 * 1024 * 1024)
		s.(*kcp.UDPSession).SetWriteBuffer(4 * 1024 * 1024)
		session := NewSsesion(s)
		session.isServer = true
		go func() {
			defer func() {
				if err := recover(); err != nil  {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					session.rtmpCloseSessionHanler()
					fmt.Println("rtmp: panic serving %v: %v\n%s", session.netconn.RemoteAddr(), err, string(buf))
				}
			}()

			err := self.ServerHandle(session)
			if Debug {
				fmt.Println("rtmp: server: client closed err:", err)
			}
		}()

	}
}


func SplitPath(u *url.URL) (app, stream string) {
	pathsegs := strings.SplitN(u.RequestURI(), "/", 3)
	if len(pathsegs) > 1 {
		app = pathsegs[1]
	}
	if len(pathsegs) > 2 {
		stream = pathsegs[2]
	}
	return
}

func getTcUrl(u *url.URL) string {
	app, _ := SplitPath(u)
	nu := *u
	nu.Path = "/" + app
	return u.Scheme +"://" + u.Host + nu.Path
}

func createURL(tcurl, app, play string) (u *url.URL) {
	ps := strings.Split(app+"/"+play, "/")
	out := []string{""}
	for _, s := range ps {
		if len(s) > 0 {
			out = append(out, s)
		}
	}
	if len(out) < 2 {
		out = append(out, "")
	}
	path := strings.Join(out, "/")
	u, _ = url.ParseRequestURI(path)

	if tcurl != "" {
		tu, _ := url.Parse(tcurl)
		if tu != nil {
			u.Host = tu.Host
			u.Scheme = tu.Scheme
		}
	}
	return
}
