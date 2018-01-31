package rtmp
import (
	"net/http"
	"fmt"
	//"strings"
	"rtmpServerStudy/AvQue"
	"sync"
	"time"
	"io"
	"rtmpServerStudy/av"
	"rtmpServerStudy/flv"
	"rtmpServerStudy/flv/flvio"
	"github.com/gorilla/mux"
	"net/url"
	"rtmpServerStudy/timer"
	//"rtmpServerStudy/amf"
	//"github.com/aws/aws-sdk-go/aws/client/metadata"
	"strings"
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

	w.WriteHeader(streams,self.metaData)
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
	tag.Data = pkt.Data
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
		var tag *flvio.Tag
		var ok bool

		metaversion:=self.pubSession.metaversion
		if self.metaversion != metaversion {

			self.pubSession.RLock()
			metaData := self.pubSession.metaData
			self.pubSession.RUnlock()

			if tag, ok = flv.MetadeToTag("onMetaData", metaData); err != nil {
				self.metaversion = metaversion
				continue
			}

			if ok {
				if err = flvio.WriteTag(w.GetMuxerWrite(), tag, 0, w.B); err != nil {
					return
				}
			}

			self.metaversion = metaversion
		}

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
			t := timer.GlobalTimerPool.Get(time.Second * MAXREADTIMEOUT)
			select {
			case <-self.PacketAck:
			case <-t.C:
			}
			timer.GlobalTimerPool.Put(t)
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
	//itmes:=strings.Split(r.URL.Path, ".flv")
	host :=r.Host
	m, _ := url.ParseQuery(r.URL.RawQuery)
	if len(m["vhost"])>0{
		host = m["vhost"][0]
	}
	if _,PlayOk:=Gconfig.UserConf.PlayDomain[host];PlayOk == false{
		w.WriteHeader(404)
	}

	h := strings.Split(host, ":")
	if  len(h)>0{
		host = h[0]
	}

	//hashPath:=itmes[0]
	//fmt.Println(hashPath)
	name := mux.Vars(r)["name"]
	app := mux.Vars(r)["app"]
	fmt.Println(name,app)
	StreamAnchor := name + ":" + Gconfig.UserConf.PlayDomain[host].UniqueName + ":" + app
	pubSession:= RtmpSessionGet(StreamAnchor)
	if pubSession != nil {
		session:=new(Session)
		session.CursorList = AvQue.NewPublist()
		session.lock = &sync.RWMutex{}
		session.PacketAck = make(chan bool, 1)
		session.CurQue = AvQue.RingBufferCreate(10)
		session.context, session.cancel = pubSession.context, pubSession.cancel
		//onpublish handler
		t := timer.GlobalTimerPool.Get(time.Second * MAXREADTIMEOUT)
		select {
		case pubSession.RegisterChannel <- session:
		case <-t.C:
		//may be is err
		}
		timer.GlobalTimerPool.Put(t)

		session.pubSession = pubSession
		session.StreamAnchor = StreamAnchor
		session.StreamId = name
		session.App = app
		session.Vhost = host

		//copy gop,codec here all new play Competitive the publishing lock
		pubSession.RLock()
		session.updatedGop = true
		session.aCodec = pubSession.aCodec
		session.vCodecData = pubSession.vCodecData
		session.aCodecData = pubSession.aCodecData
		session.vCodec = pubSession.vCodec
		//copy all gop just ptr copy
		session.metaversion = pubSession.metaversion
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
