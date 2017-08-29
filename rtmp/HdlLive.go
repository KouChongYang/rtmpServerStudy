package rtmp
import (
	"net/http"
	"fmt"
	"strings"
	"rtmpServerStudy/AvQue"
	"sync"
	"time"
	"io"
	"rtmpServerStudy/av"
	"rtmpServerStudy/flv"
	"rtmpServerStudy/flv/flvio"
	"github.com/gorilla/mux"
)

type writeFlusher struct {
	httpflusher http.Flusher
	io.Writer
}

func (self writeFlusher) Flush() error {
	self.httpflusher.Flush()
	return nil
}

//r just write log
func (self *Session) hdlSendHead(w * flv.Muxer, r *http.Request) (err error) {

	var streams []av.CodecData

	if self.aCodec == nil && self.vCodec == nil {
		return
	}
	if self.aCodec != nil {
		streams = append(streams, *self.aCodec)
	}
	if self.vCodec != nil {
		streams = append(streams, *self.vCodec)
	}
	w.WriteHeader(streams)
	return
}

func PacketToTag(pkt *av.Packet) (tag *flvio.Tag, timestamp int32) {
	tag = new(flvio.Tag)

	switch pkt.PacketType {
	case RtmpMsgAudio:
		tag.Type = flvio.TAG_AUDIO
	case RtmpMsgVideo:
		tag.Type  = flvio.TAG_VIDEO
	}
	timestamp = flvio.TimeToTs(pkt.Time)
	return
}

func (self *Session) hdlSendGop(w * flv.Muxer, r *http.Request) (err error) {
	if self.GopCache == nil {
		return
	}
	for pkt := self.GopCache.RingBufferGet(); pkt != nil; {
		tag,ts := PacketToTag(pkt)
		if err = flvio.WriteTag(w.GetMuxerWrite(), tag, ts,w.B); err != nil {
			return
		}
		if err != nil {
			self.GopCache = nil
			return err
		}
		pkt = self.GopCache.RingBufferGet();
	}
	self.GopCache = nil
	return
}

func (self *Session) hdlSendAvPackets(w * flv.Muxer, r *http.Request) (err error) {
	for {
		pkt := self.CurQue.RingBufferGet()
		select {
		case <-self.context.Done():
		// here publish may over so play is over
			fmt.Println("the publisher is close")
			self.isClosed = true
			return
		default:

		// 没有结束 ... 执行 ...
		}

		if pkt == nil && self.isClosed  != true {
			select {
			case <-self.PacketAck:
			case <-time.After(time.Second * MAXREADTIMEOUT):
			}
		}
		if self.pubSession.isClosed == true{
			self.isClosed = true
		}
		if pkt != nil {
			tag,ts := PacketToTag(pkt)
			if err = flvio.WriteTag(w.GetMuxerWrite(),tag, ts,w.B); err != nil {
				return
			}
		}
	}
	return
}

func HDLHandler(w http.ResponseWriter, r *http.Request){
	fmt.Println(r.URL.Path)
	itmes:=strings.Split(r.URL.Path, ".flv")
	hashPath:=itmes[0]
	fmt.Println(hashPath)
	name := mux.Vars(r)["name"]
	app := mux.Vars(r)["app"]
	fmt.Println(name,app)
	pubSession:= RtmpSessionGet(hashPath)
	if pubSession != nil {
		session:=new(Session)
		session.CursorList = AvQue.NewPublist()
		session.lock = &sync.RWMutex{}
		session.PacketAck = make(chan bool, 1)
		session.CurQue = AvQue.RingBufferCreate(10)
		//onpublish handler
		select {
		case pubSession.RegisterChannel <- session:
		case <-time.After(time.Second * MAXREADTIMEOUT):
		//may be is err
		}
		session.pubSession = pubSession
		//copy gop,codec here all new play Competitive the publishing lock
		pubSession.RLock()
		session.aCodec = pubSession.aCodec
		session.vCodecData = pubSession.vCodecData
		session.aCodecData = pubSession.aCodecData
		session.vCodec = pubSession.vCodec
		//copy all gop just ptr copy
		session.GopCache = pubSession.GopCache.GopCopy()
		pubSession.RUnlock()
		/*Cache-Control: no-cache
		Content-Type: video/x-flv
		Connection: close
		Expires: -1
		Pragma: no-cache*/
		w.Header().Set("Content-Type", "video/x-flv")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("Pragma","no-cache")
		w.Header().Set("Cache-Control","no-cache")
		w.WriteHeader(200)

		flusher := w.(http.Flusher)
		flusher.Flush()
		session.bufw = &writeFlusher{httpflusher: flusher, Writer: w}
		muxer := flv.NewMuxerWriteFlusher(writeFlusher{httpflusher: flusher, Writer: w})
		//send audio,video head and meta
		if err := session.hdlSendHead(muxer,r); err != nil {
			session.isClosed = true
			flusher.Flush()
			return
		}
		//send gop for first screen
		if err := session.hdlSendGop(muxer,r); err != nil {
			session.isClosed = true
			flusher.Flush()
			return
		}
		if err := session.hdlSendAvPackets(muxer,r); err != nil {
			session.isClosed = true
			flusher.Flush()
			return
		}
		flusher.Flush()
	}else{
		//hdl relay or rtmp relay must add
	}
}
