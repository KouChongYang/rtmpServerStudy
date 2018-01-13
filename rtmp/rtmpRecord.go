package rtmp

import "rtmpServerStudy/av"

const(
	BasePath="/data/rtmp/"
)

//只是不想用反射，导致我写了这么多的代码，
// 不知道为什么这么反感反射，是c语音开发习惯了吗

type RecordMuxerInfo struct {
	RecordVodPath string
	RecordFormatName string
	stage int
	WriterMuxer av.Muxer
}

var RecordOnPublish []RecordOnPublish
var RecordOnPublishDone []RecordOnPublisDone
var Record []Record

//publish 初始化目录等相关工作
type RecordOnPublish func(*Session)

//断流销毁资源
type RecordOnPublisDone func(*Session)

//录制过程
type Record func(*Session,av.CodecData,*av.Packet)

func RecordPublishHandler(self *Session){
	for i,_:= range RecordOnPublish {
		RecordOnPublish[i](self)
	}
}

func RecordPublishDoneHandler(self *Session){
	for i,_:= range RecordOnPublishDone {
		RecordOnPublishDone[i](self)
	}
}

func RecordHandler(self *Session,stream av.CodecData,pkt *av.Packet){
	for i,_:= range Record {
		Record[i](self,stream,pkt)
	}
}

func init(){
	//
	RecordOnPublish = append(RecordOnPublish,hlsRecordOnPublish)
	RecordOnPublish = append(RecordOnPublish,hlsLiveRecordOnPublish)
	RecordOnPublish = append(RecordOnPublish,flvRecordOnPublish)

	//
	Record = append(Record,hlsRecord)
	Record = append(Record,hlsLiveRecord)
	Record = append(Record,flvRecord)

	//
	RecordOnPublishDone = append(RecordOnPublishDone,hlsLiveRecordOnPublishDone)
	RecordOnPublishDone = append(RecordOnPublishDone,hlsRecordOnPublishDone)
	RecordOnPublishDone = append(RecordOnPublishDone,flvRecordOnPublishDone)
}


