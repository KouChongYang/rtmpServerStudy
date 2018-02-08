package rtmp
import (
	"github.com/nareix/bits/pio"
	"time"
	"fmt"
	"rtmpServerStudy/av"
	"rtmpServerStudy/flv"
)
type Tag uint32

const ABST = Tag(0x61627374)
const ASRT = Tag(0x61667274)
const AFRT = Tag(0x61667274)

mts:=`<?xml version="1.0" encoding="UTF-8"?>
<manifest xmlns="http://ns.adobe.com/f4m/1.0">
        <id>index.f4m</id>
        <streamType>live</streamType>
        <deliveryType>streaming</deliveryType>
        <duration>0</duration>
        <bootstrapInfo profile="named" url="%s" id="bootstrap0"/>
        <media streamId="%shds_0" bitrate="0" url="../%s/" bootstrapInfoId="bootstrap0"></media>
</manifest>`

type ServerEntry struct{
//something add
	ServerEntry []byte
}

type QualityEntry struct {
//something add
	QualityEntry []byte
}

type byteString struct{
	data []byte
}

type SegmentRunTableEntry struct {
	FirstSegment uint32
	FragmentsPersegment uint32
}

type Asrt struct{
	Version uint8
	Flags   uint32//24
	QualityEntryCount uint8
	QualitySegmentUrl [][]byte
	SegmentRunEntryCount uint32
	SegmentRunTableEntrys []*SegmentRunTableEntry
}

func (self *Asrt) Marshal(b []byte) (n int) {
	pio.PutU32BE(b[4:], uint32(ASRT))
	n += self.marshal(b[8:])+8
	pio.PutU32BE(b[0:], uint32(n))
	return n
}

func (self *Asrt)marshal(b []byte)(n int){
	pio.PutU8(b[n:], self.Version)
	n += 1
	pio.PutU24BE(b[n:], self.Flags)
	n += 3
	pio.PutU8(b[n:],self.QualityEntryCount)
	n += 1
	for index,_:= range self.QualitySegmentUrl {
		n += copy(b[n:],self.QualitySegmentUrl[index])
	}
	pio.PutU8(b[n:],self.SegmentRunEntryCount)
	n += 1

	/*
	FirstSegment uint32
	FragmentsPersegment uint32
	*/

	for index,_:= range self.SegmentRunTableEntrys {
		PutTime32(b, self.SegmentRunTableEntrys[index].FirstSegment)
		n += 4
		PutTime32(b, self.SegmentRunTableEntrys[index].FragmentsPersegment)
		n += 4
	}

	return n
}

type FragmentRunTableEntry struct{
	//something add
	/*
	*/
	FirstFragment uint32

	/*
	*/
	FirstFragmentTimestamp uint64

	/*
	*/
	FragmentDuration uint32

	/*
	*/

	DiscontinuityIndicator uint8
}

type Afrt struct {
	Version uint8
	Flags uint32
	TimeScale uint32
	QualityEntryCount uint8
	QualitySegmentUrlModifiers [][]byte
	FragmentRunEntryCount uint32
	FragmentRunEntrys []*FragmentRunTableEntry
}

func (self *Afrt) Marshal(b []byte) (n int) {
	pio.PutU32BE(b[4:], uint32(AFRT))
	n += self.marshal(b[8:])+8
	pio.PutU32BE(b[0:], uint32(n))
	return n
}

func (self *Afrt)marshal(b []byte)(n int){
	pio.PutU8(b[n:], self.Version)
	n += 1
	pio.PutU24BE(b[n:], self.Flags)
	n += 3
	PutTime32(b[n:], self.TimeScale)
	n += 4
	pio.PutU8(b[n:],self.QualityEntryCount)
	n += 1
	for index,_:= range self.QualitySegmentUrlModifiers {
		n += copy(b[n:],self.QualitySegmentUrlModifiers[index])
	}
	pio.PutU8(b[n:],self.FragmentRunEntryCount)
	n += 1
	/*
	FirstFragment uint32
	FirstFragmentTimestamp uint64
	FragmentDuration uint32
	DiscontinuityIndicator uint8
	*/
	for index,_:= range self.FragmentRunEntrys {
		PutTime32(b, self.FragmentRunEntrys[index].FirstFragment)
		n += 4
		PutTime64(b,self.FragmentRunEntrys[index].FirstFragmentTimestamp)
		n += 8
		pio.PutU32BE(b, self.FragmentRunEntrys[index].FragmentDuration)
		n += 4
		//断流的时候设置为3 这样播放器类似m3u8的discontinue
		if self.FragmentRunEntrys[index].FragmentDuration >0 {
			pio.PutU8(b[n:], self.Version)
			n += 1
		}
	}
	return n
}

type AbstHeader struct {
	Version uint8  /// Either 0 or 1
	Flags   uint32 //24  Reserved. Set to 0
	//The version number of the bootstrap information.
	// When the Update field is set, BootstrapinfoVersion indicates
	//the version number that is being updated.
	BootstrapinfoVersion uint32
	//0
	PLURFlags uint8
	/*
	The number of time units per second. The field CurrentMediaTime
	uses this value to represent accurate time. Typically, the value is 1000, for a unit of milliseconds.
	*/
	TimeScale uint32
	/*
	The timestamp in TimeScale units of the latest available Fragment in the media presentation. This timestamp is used to request
	 the right fragment number. The CurrentMediaTime can be the total duration. For media presentations that are not live, CurrentMediaTime can be 0.
	*/
	CurrentMediaTime uint64
	/*
	 The offset of the CurrentMediaTime from the SMPTE time code, converted to milliseconds.
	  This offset is not in TimeScale units. This field is zero when not used.
	  The server uses the SMPTE time code modulo 24 hours to make the offset positive.
	*/
	SmpteTimeCodeOffset uint64

	/*
	The identifier of this presentation. The identifier is a null-terminated UTF-8 string.
	 For example, it can be a filename or pathname in a URL. See Annex C.4 URL Construction for usage.
	*/

	MovieIdentifier []byte

	/*
	The number of ServerEntryTable entries.
	 The minimum value is 0.
	*/
	ServerEntryCount uint8
	/*
	Server URLs in descending order of preference
	*/
	ServerEntrys		[][]byte

	/*
	The number of QualityEntryTable entries, which is also the number of available quality levels.
	 The minimum value is 0. Available quality levels are for,
	 for example, multi bit rate files or trick files.
	*/
	QualityEntryCount uint8
	/*
	Quality file references in order from high to low quality
	*/
	QualityEntrys           [][]byte

	/*
	Null or null-terminated UTF-8 string. This string holds Digital Rights Management metadata.
	Encrypted files use this metadata to get the necessary keys and licenses for decryption and playback.
	*/
	DrMData []byte

	/*
	Null or null-terminated UTF-8 string that holds metadata
	*/
	Meta    []byte

	/*
	The number of entries in SegmentRunTableEntries.
	The minimum value is 1. Typically, one table contains all segment runs.
	However, this count provides the flexibility to define the segment runs individually for each quality level (or trick file).
	*/
	SegmentRunTableCount uint8

	/*
	Array of SegmentRunTable elements
	*/
	SegmentRunTableEntrys []*Asrt

	/*
	The number of entries in FragmentRunTableEntries.
	 The minimum value is 1.
	*/
	FragmentRunTableCount uint8

	/*
	Array of FragmentRunTable elements
	*/
	FragmentRunTableEntrys []*Afrt
}

func (self *AbstHeader) Marshal(b []byte) (n int) {
	pio.PutU32BE(b[4:], uint32(ABST))
	n += self.marshal(b[8:])+8
	pio.PutU32BE(b[0:], uint32(n))
	return
}

func PutTime32(b []byte, t time.Time) {
	dur := t.Sub(time.Date(1904, time.January, 1, 0, 0, 0, 0, time.UTC))
	sec := uint32(dur/time.Second)
	pio.PutU32BE(b, sec)
}

func PutTime64(b []byte, t time.Time) {
	dur := t.Sub(time.Date(1904, time.January, 1, 0, 0, 0, 0, time.UTC))
	sec := uint64(dur/time.Second)
	pio.PutU64BE(b, sec)
}

func (self *AbstHeader)marshal(b[]byte)(n int){
	pio.PutU8(b[n:], self.Version)
	n += 1
	pio.PutU24BE(b[n:], self.Flags)
	n += 3
	PutTime32(b[n:], self.BootstrapinfoVersion)
	n += 4
	pio.PutU8(b[n:], self.PLURFlags)
	n += 1
	PutTime32(b[n:], self.TimeScale)
	n += 4
	PutTime64(b[n:],self.CurrentMediaTime)
	n += 8
	PutTime64(b[n:],self.SmpteTimeCodeOffset)
	n += 8
	n += copy(b[n:],self.MovieIdentifier)
	pio.PutU8(b[n:],self.ServerEntryCount)
	n += 1

	for index,_:= range self.ServerEntrys {
		n += copy(b[n:],self.ServerEntrys[index])
	}
	pio.PutU8(b[n:],self.QualityEntryCount)
	n += 1

	for index,_:= range self.QualityEntrys {
		n += copy(b[n:],self.QualityEntrys[index])
	}

	n += copy(b[n:],self.DrMData)
	n += copy(b[n:],self.Meta)

	pio.PutU8(b[n:],self.SegmentRunTableCount)
	n += 1

	for index,_:= range self.SegmentRunTableEntrys {
		n += self.SegmentRunTableEntrys[index].Marshal(b[n:])
	}
	pio.PutU8(b[n:],self.FragmentRunTableCount)
	n += 1
	for index,_:= range self.FragmentRunTableEntrys {
		n += self.FragmentRunTableEntrys[index].Marshal(b[n:])
	}

	return n
}

type  hdsLiveRecordInfo struct {
	//
	muxer            *flv.Muxer
	lastAudioTs      time.Duration
	lastVideoTs      time.Duration
	lastTs           time.Duration
	hdsBackFileName   string
	hdsName 	  string
	m3u8BackFileName  string
	//是否该切片
	force            bool
	duration 	 float32
	aframeNum      	uint64

	audioPts 	uint64
	audioBaseTime   uint64
	m3u8Box *m3u8Box
	seqNum uint64
}

func hdsLiveRecord(self *Session,stream av.CodecData,pkt *av.Packet){
	//init
	if self.hdsLiveRecordInfo.muxer == nil {
		nowTime:=time.Now().UnixNano()/1000000
		self.hdsLiveRecordInfo.hdsBackFileName = fmt.Sprintf("%s%d.tsbak",self.UserCnf.RecodeHlsPath,nowTime)
		self.hdsLiveRecordInfo.hdsName = fmt.Sprintf("%d.ts",nowTime)
		fmt.Println(self.hlsLiveRecordInfo.tsBackFileName)
		f1, err := FileCreate(self.hlsLiveRecordInfo.tsBackFileName)
		if err != nil {
			fmt.Printf("create ts file %s err the err is %s\n",self.hlsLiveRecordInfo.tsBackFileName,err.Error())
		}
		self.hlsLiveRecordInfo.muxer = flv.NewMuxer(f1)
		//self.hlsLiveRecordInfo.lasetTs = pkt.Time
		if pkt.PacketType == RtmpMsgAudio {
			self.hlsLiveRecordInfo.lastAudioTs = pkt.Time
		} else{
			self.hlsLiveRecordInfo.lastVideoTs = pkt.Time
		}
		self.hlsLiveRecordInfo.lastTs =  pkt.Time
		self.hlsLiveRecordInfo.m3u8BackFileName = fmt.Sprintf("%sindex.m3u8",self.UserCnf.RecodeHlsPath)
		self.hlsLiveRecordInfo.m3u8Box = NewM3u8Box(self.StreamId)
	}


	switch pkt.PacketType {
	case RtmpMsgAudio:
		hlsAudioRecord(self,stream,pkt)
	case RtmpMsgVideo:
		hlsVedioRecord(self,stream,pkt)
	}
	return
}


/*
<?xml version="1.0" encoding="UTF-8"?>
<manifest xmlns="http://ns.adobe.com/f4m/1.0">
        <id>index.f4m</id>
        <streamType>live</streamType>
        <deliveryType>streaming</deliveryType>
        <duration>0</duration>
        <bootstrapInfo profile="named" url="%s.abst" id="bootstrap0"/>
        <media streamId="cctv15_mdhds_0" bitrate="0" url="../%s/" bootstrapInfoId="bootstrap0"></media>
</manifest>
*/

