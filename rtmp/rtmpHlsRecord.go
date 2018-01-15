package rtmp

import (
	"rtmpServerStudy/av"
	"os"
	"fmt"
	"time"
	"rtmpServerStudy/ts"
	"rtmpServerStudy/AvQue"
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
	self.UserCnf.RecodeHlsPath = fmt.Sprintf("%s%s/%s/",self.UserCnf.RecodeHlsPath,self.uniqueName,self.App)
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
	audioCachedPkts  [](*ts.AudioPacket)
	tsBackFileName   string
	m3u8BackFileName string
	//是否该切片
	force            bool
	duration 	 float32
}

func hlsLiveRecordOpenFragment(self *Session,stream av.CodecData,pkt *av.Packet){

}

func hlsLiveRecordCloseFragment(self *Session,stream av.CodecData,pkt *av.Packet){

}

func hlsLiveUpdateFragment(self *Session,stream av.CodecData,pkt *av.Packet,flush_rate int){
	self.hlsLiveRecordInfo.duration = pkt.Time - self.hlsLiveRecordInfo.lastTs
	if self.hlsLiveRecordInfo.duration > 50000{

	}else{
		return
	}
	/*
	b = ctx->aframe;
	if (ctx->opened && b && b->last > b->pos &&
		ctx->aframe_pts + (uint64_t) hacf->max_audio_delay * 90 / flush_rate
	< ts)
	{
	ngx_rtmp_hls_flush_audio(s);
	}*/
	if len(self.hlsLiveRecordInfo.audioCachedPkts)>0 &&
		(self.hlsLiveRecordInfo.audioCachedPkts[len(self.hlsLiveRecordInfo.audioCachedPkts)-1].Time * 90 + 300/flush_rate) <pkt.Time{
		self.hlsLiveRecordInfo.muxer.WriteAudioPacket(self.hlsLiveRecordInfo.audioCachedPkts,self.aCodec)
		self.hlsLiveRecordInfo.audioCachedPkts = make([](*ts.AudioPacket),0,10)
	}
	return
}

func hlsVedioRecord(self *Session,stream av.CodecData,pkt *av.Packet){
	//no body
	if len(pkt.Data[pkt.DataPos:])<=0{
		return
	}
	/*
	b = ctx->aframe;
	    boundary = frame.key && (codec_ctx->aac_header == NULL || !ctx->opened ||
				     (b && b->last > b->pos));
	*/
	if pkt.IsKeyFrame {
		hlsLiveUpdateFragment(self ,stream,pkt,1)
	}
	self.hlsLiveRecordInfo.muxer.WriteVedioPacket(pkt,stream)
	return
}

func hlsAudioRecord(self *Session,stream av.CodecData,pkt *av.Packet){

	//no body
	if len(pkt.Data[pkt.DataPos:])<=0{
		return
	}
/*
b = ctx->aframe;
    boundary = frame.key && (codec_ctx->aac_header == NULL || !ctx->opened ||
                             (b && b->last > b->pos));
*/

	if pkt.IsKeyFrame && (self.aCodec == nil){

		return
	}
	return
}

func hlsLiveRecord(self *Session,stream av.CodecData,pkt *av.Packet){
	if self.hlsLiveRecordInfo.muxer == nil {
		self.hlsLiveRecordInfo.audioCachedPkts = make([]*av.Packet,0,1024)
		self.hlsLiveRecordInfo.tsBackFileName = fmt.Sprintf("%s%d.tsbak",self.UserCnf.RecodeHlsPath,time.Now().UnixNano()/1000000)
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
	}

	switch pkt.PacketType {
	case RtmpMsgAudio:
		hlsAudioRecord(self,stream,pkt)
	case RtmpMsgVideo:
		hlsVedioRecord(self,stream,pkt)
	}

	if pkt.IsKeyFrame && (pkt.Time - self.hlsLiveRecordInfo.lastTs) > time.Duration(5000){
		return
	}
}
