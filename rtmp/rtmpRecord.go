package rtmp

import "rtmpServerStudy/av"

const(
	BasePath="/data/rtmp/"
)

type RecordMuxerInfo struct {
	RecordVodPath string
	RecordFormatName string
	WriterMuxer av.Muxer
}

type RecordFormatMap map[string](*RecordMuxerInfo)
