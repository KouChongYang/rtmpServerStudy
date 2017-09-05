package rtmp

import "rtmpServerStudy/av"

const(
	BasePath="/data/rtmp/"
)

type RecordMuxerInfo struct {
	RecordVodPath string
	RecordFormatName string
	stage int
	WriterMuxer av.Muxer
}


