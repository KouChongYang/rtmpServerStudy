package rtmp

import (
	"rtmpServerStudy/av"
	"os"
	"fmt"
	"time"
	"rtmpServerStudy/ts"
	"rtmpServerStudy/flv/flvio"
//	"rtmpServerStudy/aacParse"
	"strings"
//	"rtmpServerStudy/ts/tsio"
//	"rtmpServerStudy/aacParse"
	"io/ioutil"
	"net/http"
	"net/url"
	"github.com/gorilla/mux"
)

//hls点播
func hlsRecordOnPublish(*Session){
	//是否开启hls点播
	//初始化相关目录
	//初始化muxer
	return
}

//hls直播
func hlsLiveRecordOnPublish(self *Session){
	if self.UserCnf.RecodeHls != 1{
		return
	}

	if len(self.UserCnf.RecodeHlsPath) < 0{
		self.UserCnf.RecodeHlsPath = BasePath + "/hls/"
	}

	/*
	if GDefaultPath[len(GDefaultPath)-1] != '/' {
		GDefaultPath = GDefaultPath + "/"
	}
	*/
	if self.UserCnf.RecodeHlsPath[len(self.UserCnf.RecodeHlsPath)-1] !='/'{
		self.UserCnf.RecodeHlsPath = self.UserCnf.RecodeHlsPath + "/"
	}

	// /data/hls/test/app/
	self.UserCnf.RecodeHlsPath = fmt.Sprintf("%s%s/%s/%s/",self.UserCnf.RecodeHlsPath,self.uniqueName,self.App,self.StreamId)
	err:=os.MkdirAll(self.UserCnf.RecodeHlsPath,0666)
	if err != nil{
		fmt.Printf("%s\n",err.Error())
		return
	}
	return
}


func hlsRecordOnPublishDone(self *Session) {
	//是否开启
	//释放空间
}

func hlsLiveRecordOnPublishDone(self *Session){

}
/*
注意点，保证每一个ts的首帧为i帧
1.按时间来切片，如果配置的时间为5s，判断是否是i帧，如果是i帧，并且pkt.time - this-ts.first-pkt.time >5s 则切割ts
如果没有配置切片时间默认值为5s，如果时间戳不变，则
///切片逻辑，如果没有配置按时间来切片，按照gop来切，一个gop为一个ts一
 */
func hlsRecord(self *Session,stream av.CodecData,pkt *av.Packet){
	//判断是否该切片（如果有音频，又有视频帧，以视频帧为主）
	//判断是否是I帧，如果是I帧，并且上一帧为
}

//


type  hlsLiveRecordInfo struct {
	//
	muxer            *ts.Muxer
	lastAudioTs      time.Duration
	lastVideoTs      time.Duration
	lastTs           time.Duration
	audioCachedPkts  [](*av.Packet)
	tsBackFileName   string
	tsName 		 string
	m3u8BackFileName string
	//是否该切片
	force            bool
	duration 	 float32
	aframeNum      	uint64

	audioPts 	time.Duration
	audioBaseTime   time.Duration
	m3u8Box *m3u8Box
	seqNum uint64
}

//打开新的文件
func hlsLiveRecordOpenFragment(self *Session,stream av.CodecData,pkt *av.Packet){

	nowTime:=time.Now().UnixNano()/1000000
	err:=os.MkdirAll(self.UserCnf.RecodeHlsPath,0666)
	self.hlsLiveRecordInfo.tsBackFileName = fmt.Sprintf("%s%d.tsbak",self.UserCnf.RecodeHlsPath,nowTime)
	self.hlsLiveRecordInfo.tsName = fmt.Sprintf("%d.ts",nowTime)
	fmt.Println(self.hlsLiveRecordInfo.tsBackFileName)
	f1, err := FileCreate(self.hlsLiveRecordInfo.tsBackFileName)
	if err != nil {
		fmt.Printf("create ts file %s err the err is %s\n",self.hlsLiveRecordInfo.tsBackFileName,err.Error())
	}

	//重置文件
	self.hlsLiveRecordInfo.muxer.SetWriter(f1)
	//写pat pmt ts header
	self.hlsLiveRecordInfo.muxer.WriteHeader()

	//self.hlsLiveRecordInfo.lasetTs = pkt.Time
	if pkt.PacketType == RtmpMsgAudio {
		self.hlsLiveRecordInfo.lastAudioTs = pkt.Time
	} else {
		self.hlsLiveRecordInfo.lastVideoTs = pkt.Time
	}
	self.hlsLiveRecordInfo.lastTs =  pkt.Time
	self.hlsLiveRecordInfo.seqNum++
}

func hlsLiveRecordCloseFragment(self *Session,stream av.CodecData,pkt *av.Packet){
	self.hlsLiveRecordInfo.muxer.WriteTrailer()
	dstkey := strings.Replace(self.hlsLiveRecordInfo.tsBackFileName, ".tsbak", ".ts", 1)
	os.Rename(self.hlsLiveRecordInfo.tsBackFileName, dstkey)
	tsitem := NewTSItem(self.hlsLiveRecordInfo.tsName,self.hlsLiveRecordInfo.duration,self.hlsLiveRecordInfo.seqNum)
	//写m3u8
	self.hlsLiveRecordInfo.m3u8Box.SetItem(tsitem)
	b,err:=self.hlsLiveRecordInfo.m3u8Box.GenM3U8PlayList()
	if err != nil{
		fmt.Println(err)
		return
	}

	err = ioutil.WriteFile(self.hlsLiveRecordInfo.m3u8BackFileName, b, 0666)
	if err != nil{
		fmt.Println(err)
	}

	//self.hlsLiveRecordInfo.tsBackFileName = fmt.Sprintf("%s%d.tsbak",self.UserCnf.RecodeHlsPath,time.Now().UnixNano()/1000000)
}

func hlsLiveUpdateFragment(self *Session,stream av.CodecData,pkt *av.Packet,flush_rate uint64){

	cutting := 0

	//大于5s 切片
	self.hlsLiveRecordInfo.duration = float32(flvio.TimeToTs(pkt.Time - self.hlsLiveRecordInfo.lastTs))/(1000.0)
	if self.hlsLiveRecordInfo.duration > 5.0 {
		cutting = 1
	}else{
		return
	}

	//需要切割
	if cutting == 1 {
		hlsLiveRecordCloseFragment(self,stream,pkt)
		hlsLiveRecordOpenFragment(self,stream,pkt)
	}

	//see see nginx
	return
}

func hlsVedioRecord(self *Session,stream av.CodecData,pkt *av.Packet){
	//no body
	if len(pkt.Data[pkt.DataPos:])<=0{
		return
	}
	//关键帧判断是否需求切割
	if pkt.IsKeyFrame {
		hlsLiveUpdateFragment(self ,stream,pkt,1)
	}
	//将vedio 写入文件
	self.hlsLiveRecordInfo.muxer.WritePacket(pkt,stream)
	return
}
/*
func hlsAudioRecord(self *Session,stream av.CodecData,pkt *av.Packet){
	//no body
	if len(pkt.Data[pkt.DataPos:])<=0{
		return
	}
	//关键帧判断是否需求切割
	if pkt.IsKeyFrame {
		hlsLiveUpdateFragment(self ,stream,pkt,1)
	}
	//将vedio 写入文件
	self.hlsLiveRecordInfo.muxer.WritePacket(pkt,stream)
	return
}*/

func hlsAudioRecord(self *Session,stream av.CodecData,pkt *av.Packet){
	//no body
	if len(pkt.Data[pkt.DataPos:])<=0 {
		return
	}
	//判断cacheNum
	cacheNum := len(self.hlsLiveRecordInfo.audioCachedPkts)

	pts := (pkt.Time)

	//判读是否切片，如果需要就切片
	hlsLiveUpdateFragment(self ,stream,pkt,2)

	//缓存太多音频，估计音频有问题
	if cacheNum >= 1024 {
		fmt.Printf("hls vod: too many audio frame the num:%d", cacheNum)
		self.hlsLiveRecordInfo.muxer.WriteAudioPacket(self.hlsLiveRecordInfo.audioCachedPkts,self.aCodec,self.hlsLiveRecordInfo.audioPts)
		self.hlsLiveRecordInfo.audioCachedPkts = make([](*av.Packet),0,10)
		self.hlsLiveRecordInfo.audioPts = pts
		return
	}
	//
	//缓存音频
	self.hlsLiveRecordInfo.audioCachedPkts = append(self.hlsLiveRecordInfo.audioCachedPkts,pkt)
	if pts - self.hlsLiveRecordInfo.audioPts > 720{
		self.hlsLiveRecordInfo.muxer.WriteAudioPacket(self.hlsLiveRecordInfo.audioCachedPkts,self.aCodec,self.hlsLiveRecordInfo.audioPts)
		self.hlsLiveRecordInfo.audioCachedPkts = make([](*av.Packet),0,10)
		self.hlsLiveRecordInfo.audioPts = pts
	}
	/*if len(self.hlsLiveRecordInfo.audioCachedPkts)>0{
		self.hlsLiveRecordInfo.aframeNum++
		return
	}

	//更新pts 只有缓存第一个音频时才需要更新pts（其他缓存的音频参考该pts）
	self.hlsLiveRecordInfo.audioPts = uint64(pts)

	codec := stream.(aacparser.CodecData)
	if codec.SampleFormat() <=0{
		return
	}

	est_pts := self.hlsLiveRecordInfo.audioBaseTime + self.hlsLiveRecordInfo.aframeNum * 90000 * 1024 /
		uint64(codec.SampleRate())

	//
	dpts := int64(est_pts - pts)

	//pts
	if (dpts <= 2 * 90) && (dpts >= (2 * -90)){
		self.hlsLiveRecordInfo.aframeNum++
		self.hlsLiveRecordInfo.audioPts = est_pts
		return
	}

	self.hlsLiveRecordInfo.audioBaseTime = pts
	self.hlsLiveRecordInfo.aframeNum  = 1*/

	return
}

func hlsLiveRecord(self *Session,stream av.CodecData,pkt *av.Packet){
	//init
	if self.hlsLiveRecordInfo.muxer == nil {
		self.hlsLiveRecordInfo.audioCachedPkts = make([]*av.Packet,0,1024)
		nowTime:=time.Now().UnixNano()/1000000
		self.hlsLiveRecordInfo.tsBackFileName = fmt.Sprintf("%s%d.tsbak",self.UserCnf.RecodeHlsPath,nowTime)
		self.hlsLiveRecordInfo.tsName = fmt.Sprintf("%d.ts",nowTime)
		fmt.Println(self.hlsLiveRecordInfo.tsBackFileName)
		f1, err := FileCreate(self.hlsLiveRecordInfo.tsBackFileName)
		if err != nil {
			fmt.Printf("create ts file %s err the err is %s\n",self.hlsLiveRecordInfo.tsBackFileName,err.Error())
		}
		self.hlsLiveRecordInfo.muxer = ts.NewMuxer(f1)
		self.hlsLiveRecordInfo.muxer.WriteHeader()

		//self.hlsLiveRecordInfo.lasetTs = pkt.Time
		if pkt.PacketType == RtmpMsgAudio {
			self.hlsLiveRecordInfo.lastAudioTs = pkt.Time
		} else{
			self.hlsLiveRecordInfo.lastVideoTs = pkt.Time
		}
		self.hlsLiveRecordInfo.lastTs =  pkt.Time
		self.hlsLiveRecordInfo.m3u8BackFileName = fmt.Sprintf("%sindex.m3u8",self.UserCnf.RecodeHlsPath)
		self.hlsLiveRecordInfo.m3u8Box = NewM3u8Box(self.StreamId)
		self.hlsLiveRecordInfo.seqNum = uint64(nowTime)

	}


	switch pkt.PacketType {
	case RtmpMsgAudio:
		hlsAudioRecord(self,stream,pkt)
	case RtmpMsgVideo:
		hlsVedioRecord(self,stream,pkt)
	}
	return
}

func m3u8Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL.Path)
	//itmes:=strings.Split(r.URL.Path, ".flv")
	host := r.Host
	m, _ := url.ParseQuery(r.URL.RawQuery)
	if len(m["vhost"]) > 0 {
		host = m["vhost"][0]
	}
	if _, PlayOk := Gconfig.UserConf.PlayDomain[host]; PlayOk == false {
		w.WriteHeader(404)
	}

	//hashPath:=itmes[0]
	//fmt.Println(hashPath)
	name := mux.Vars(r)["name"]
	app := mux.Vars(r)["app"]
	fmt.Println(name, app)
	StreamAnchor := name + ":" + Gconfig.UserConf.PlayDomain[host].UniqueName + ":" + app
	pubSession := RtmpSessionGet(StreamAnchor)
	fmt.Println(pubSession)
}

func tsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL.Path)
	//itmes:=strings.Split(r.URL.Path, ".flv")
	host := r.Host
	m, _ := url.ParseQuery(r.URL.RawQuery)
	if len(m["vhost"]) > 0 {
		host = m["vhost"][0]
	}
	if _, PlayOk := Gconfig.UserConf.PlayDomain[host]; PlayOk == false {
		w.WriteHeader(404)
	}

	//hashPath:=itmes[0]
	//fmt.Println(hashPath)
	name := mux.Vars(r)["name"]
	app := mux.Vars(r)["app"]
	fmt.Println(name, app)
	StreamAnchor := name + ":" + Gconfig.UserConf.PlayDomain[host].UniqueName + ":" + app
	pubSession := RtmpSessionGet(StreamAnchor)
	fmt.Println(pubSession)
}