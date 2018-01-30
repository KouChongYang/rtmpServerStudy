package rtmp

import (
	"rtmpServerStudy/av"
	//"net/url"
	"io"
	"os"
)

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

var RecordOnPublishs []RecordOnPublish
var RecordOnPublishDones []RecordOnPublisDone
var Records []Record

//publish 初始化目录等相关工作
type RecordOnPublish func(*Session)

//断流销毁资源
type RecordOnPublisDone func(*Session)

//录制过程
type Record func(*Session,av.CodecData,*av.Packet)

func RecordPublishHandler(self *Session){
	for i,_:= range RecordOnPublishs {
		RecordOnPublishs[i](self)
	}
}

func RecordPublishDoneHandler(self *Session){
	for i,_:= range RecordOnPublishDones {
		RecordOnPublishDones[i](self)
	}
}



func RecordHandler(self *Session,stream av.CodecData,pkt *av.Packet){
	for i,_:= range Records {
		Records[i](self,stream,pkt)
	}
}

func FileCreate(uri string) (w io.WriteCloser, err error) {
	w, err = os.Create(uri)
	return
}

func init(){
	//

	RecordOnPublishs = append(RecordOnPublishs,hlsLiveRecordOnPublish)
	RecordOnPublishs = append(RecordOnPublishs,flvRecordOnPublish)

	//
	Records = append(Records,hlsLiveRecord)
	Records = append(Records,flvRecord)

	//
	RecordOnPublishDones = append(RecordOnPublishDones,hlsLiveRecordOnPublishDone)
	RecordOnPublishDones = append(RecordOnPublishDones,hlsRecordOnPublishDone)
	RecordOnPublishDones = append(RecordOnPublishDones,flvRecordOnPublishDone)
}


