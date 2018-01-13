package rtmp

import "rtmpServerStudy/av"

//截图，只生成关键帧
func flvRecordOnPublish(*Session){
	//是否录制flv点播
	//初始化相关目录
	//初始化muxer
	return
}

func flvRecord(self *Session,stream av.CodecData,pkt *av.Packet){

	//判断pkt 是否是idr帧
	//打开文件
	//写入文件
	//关闭文件
	return
}

func flvRecordOnPublishDone(self *Session){
	//是否开启
	//释放空间
}
