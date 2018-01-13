package rtmp

import (
	"rtmpServerStudy/av"
	"os"
	"fmt"
	"time"
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


	self.UserCnf.RecodeHlsPath = fmt.Sprint("%s%d")
	err:=os.MkdirAll(self.UserCnf.RecodeHlsPath,0666)
	if err != nil{
		fmt.Println("%s",err.Error())
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

func hlsLiveRecord(self *Session,stream av.CodecData,pkt *av.Packet){

}
