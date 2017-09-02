package rtmp

import "os"

/*
#define NGX_RTMP_RECORD_OFF                 0x01
#define NGX_RTMP_RECORD_AUDIO               0x02
#define NGX_RTMP_RECORD_VIDEO               0x04
#define NGX_RTMP_RECORD_KEYFRAMES           0x08
#define NGX_RTMP_RECORD_MANUAL              0x10*/

//may be use
const (
	RtmpRecordOff = 0x01
	RtmpRecordAudio = 0x02
	RtmpRecordVideo = 0x04
	RtmpRecordKeyFrames = 0x08
	RtmpRecordManual = 0x10
)

const(
	BasePath="/data/rtmp/"
)



//just make dirPath
func (self *Session)RtmpRecordNodeInit()(err error){
	//just create the flv path
	if self.RecordCnf == nil{
		self.RecordCnf = new(RecordInfo)
	}
	self.RecordCnf.flvVodPath = BasePath+ "/flv/"
	os.MkdirAll(self.RecordCnf.flvVodPath, 0777)
	return
}

func (self *Session)rtmpRecordNodeClose()(err error){

	return
}

func (self *Session)rtmpRecordNodeOpen()(err error){

	return
}

func (self *Session)rtmpRecordWriteHeader()(err error){
	return
}

func (self *Session)rtmpRecordWriteFrame()(err error){

	return
}