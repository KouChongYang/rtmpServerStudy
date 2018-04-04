package rtmp

import (
	"rtmpServerStudy/av"
	"time"
	"rtmpServerStudy/flv"
	"fmt"
	"os"
	"rtmpServerStudy/flv/flvio"
	"strconv"
)

type flvReordInfo struct  {
	muxer           	 *flv.Muxer
	lastTs           	 time.Duration
	flvBackFileName   	 string
	flvName 		 string
	RecidePicFragment        float64
}

//截图，只生成关键帧
func flvRecordOnPublish(self *Session){

	if self.UserCnf.RecodeFlv != 1{
		return
	}

	if len(self.UserCnf.RecodeFlvPath) < 0{
		self.UserCnf.RecodeFlvPath = BasePath + "/pic/"
	}

	if self.UserCnf.RecodeFlvPath[len(self.UserCnf.RecodeFlvPath)-1] !='/'{
		self.UserCnf.RecodeFlvPath = self.UserCnf.RecodeFlvPath + "/"
	}

	// /data/pic/test/app/
	self.UserCnf.RecodeFlvPath = fmt.Sprintf("%s%s/%s/%s/",self.UserCnf.RecodeFlvPath,self.uniqueName,self.App,self.StreamId)
	err:=os.MkdirAll(self.UserCnf.RecodeFlvPath,0666)
	if err != nil{
		fmt.Printf("%s\n",err.Error())
		return
	}
	RecidePicFragment:=(5.0)
	if len(self.UserCnf.RecidePicFragment)>0{
		time_len := len(self.UserCnf.RecidePicFragment)
		if self.UserCnf.RecidePicFragment[time_len-1] == 's' {
			time_len--
			RecidePicFragment, _ = strconv.ParseFloat(self.UserCnf.HlsFragment[:time_len],64)
		}
	}
	self.flvReordInfo.RecidePicFragment = (RecidePicFragment)
	//是否录制flv点播
	//初始化相关目录
	return
}

func flvVedioRecord(self *Session,stream av.CodecData,pkt *av.Packet){
	//no body
	if len(pkt.Data[pkt.DataPos:])<=0{
		return
	}
	//关键帧判断是否需求切割
	if pkt.IsKeyFrame {
		hlsLiveUpdateFragment(self ,stream,pkt,1,1)
	}
	//将vedio 写入文件
	self.hlsLiveRecordInfo.muxer.WritePacket(pkt,stream)
	return
}

func flvRecord(self *Session,stream av.CodecData,pkt *av.Packet){

	if self.UserCnf.RecodeFlv != 1{
		return
	}
	if self.vCodec == nil {
		return
	}
	//pic 只要有视频idr帧即可
	if (pkt.PacketType != RtmpMsgVideo) || (false == pkt.IsKeyFrame) {
		return
	}
	//判断pkt 是否是idr帧
	//打开文件
	//写入文件
	//关闭文件
	if self.flvReordInfo.muxer == nil {

		nowTime:=time.Now().UnixNano()/1000000
		self.flvReordInfo.flvBackFileName = fmt.Sprintf("%s%d.flvbak",self.UserCnf.RecodeHlsPath,nowTime)
		self.flvReordInfo.flvName = fmt.Sprintf("%d.flv",nowTime)
		fmt.Println(self.flvReordInfo.flvBackFileName)
		f1, err := FileCreate(self.flvReordInfo.flvBackFileName)
		if err != nil {
			fmt.Printf("create flv file %s err the err is %s\n",self.flvReordInfo.flvBackFileName,err.Error())
		}
		defer f1.Close()
		self.flvReordInfo.muxer = flv.NewMuxer(f1)
		var streams []av.CodecData
		streams = append(streams, *self.vCodec)
		self.flvReordInfo.muxer.WriteHeader(streams,self.metaData)
		tag,ts := PacketToTag(pkt)
		self.flvReordInfo.lastTs = pkt.Time
		if err = flvio.WriteTag(self.flvReordInfo.muxer.GetMuxerWrite(), tag, ts,self.flvReordInfo.muxer.B); err != nil {
			return
		}
		self.flvReordInfo.muxer.GetMuxerWrite().Flush()

	}else if (float64(flvio.TimeToTs(pkt.Time - self.flvReordInfo.lastTs))/(1000.0) >self.flvReordInfo.RecidePicFragment ){

		nowTime:=time.Now().UnixNano()/1000000
		self.flvReordInfo.flvBackFileName = fmt.Sprintf("%s%d.flvbak",self.UserCnf.RecodeHlsPath,nowTime)
		self.flvReordInfo.flvName = fmt.Sprintf("%d.flv",nowTime)
		fmt.Println(self.flvReordInfo.flvBackFileName)
		f1, err := FileCreate(self.flvReordInfo.flvBackFileName)
		if err != nil {
			fmt.Printf("create flv file %s err the err is %s\n",self.flvReordInfo.flvBackFileName,err.Error())
		}
		defer f1.Close()
		self.flvReordInfo.muxer.ResetMuxer(f1)
		var streams []av.CodecData
		streams = append(streams, *self.vCodec)
		self.flvReordInfo.muxer.WriteHeader(streams,self.metaData)
		tag,ts := PacketToTag(pkt)
		self.flvReordInfo.lastTs = pkt.Time
		if err = flvio.WriteTag(self.flvReordInfo.muxer.GetMuxerWrite(), tag, ts,self.flvReordInfo.muxer.B); err != nil {
			return
		}
		self.flvReordInfo.muxer.GetMuxerWrite().Flush()
	}

	return
}

func flvRecordOnPublishDone(self *Session){
	//是否开启
	//释放空间
	if self.UserCnf.RecodeFlv != 1{
		return
	}
}
