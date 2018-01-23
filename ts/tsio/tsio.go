package tsio

import (
	"io"
	"time"
	"fmt"
	"github.com/nareix/bits/pio"
	"bufio"
	//"github.com/nareix/joy4/format/ts/tsio"
)


const (
	StreamIdH264 = 0xe0
	StreamIdAAC  = 0xc0
)

const (
	PAT_PID = 0
	PMT_PID = 0x1000
)

const TableIdPMT = 2
const TableExtPMT = 1

const TableIdPAT = 0
const TableExtPAT = 1

const MaxPESHeaderLength = 19
const MaxTSHeaderLength = 12

var ErrPESHeader = fmt.Errorf("invalid.PES.Header")
var ErrPSIHeader = fmt.Errorf("invalid.PSI.Header")
var ErrParsePMT = fmt.Errorf("invalid.PMT")
var ErrParsePAT = fmt.Errorf("invalid.PAT")

const (
	ElementaryStreamTypeH264    = 0x1B
	ElementaryStreamTypeAdtsAAC = 0x0F
)

type PATEntry struct {
	ProgramNumber uint16
	NetworkPID    uint16
	ProgramMapPID uint16
}

type PAT struct {
	Entries []PATEntry
}

func (self PAT) Len() (n int) {
	return len(self.Entries)*4
}

/*
PAT (Program association specific data)[edit]
PAT specific data repeated until end of section length
Name	Number
of bits	Description
Program num	16	Relates to the Table ID extension in the associated PMT. A value of 0 is reserved for a NIT packet identifier.
Reserved bits	3	Set to 0x07 (all bits on)
Program map PID	13	The packet identifier that contains the associated PMT

The PAT is assigned PID 0x0000 and table id of 0x00. The transport stream contains at least one or more TS packets with PID 0x0000. Some of these consecutive packets form the PAT. At the decoder side the PSI section filter listens to the incoming TS packets. After the filter identifies the PAT table they assemble the packet and decode it. A PAT has information about all
 the programs contained in the TS. The PAT contains information showing the
 association of Program Map Table PID and Program Number. The PAT should end
 with a 32-bit CRC
*/

func (self PAT) Marshal(b []byte) (n int) {
	for _, entry := range self.Entries {
		pio.PutU16BE(b[n:], entry.ProgramNumber)
		n += 2
		if entry.ProgramNumber == 0 {
			pio.PutU16BE(b[n:], entry.NetworkPID&0x1fff|7<<13)
			n += 2
		} else {
			pio.PutU16BE(b[n:], entry.ProgramMapPID&0x1fff|7<<13)
			n += 2
		}
	}
	return
}

func (self *PAT) Unmarshal(b []byte) (n int, err error) {
	for n < len(b) {
		if n+4 <= len(b) {
			var entry PATEntry
			entry.ProgramNumber = pio.U16BE(b[n:])
			n += 2
			if entry.ProgramNumber == 0 {
				entry.NetworkPID = pio.U16BE(b[n:])&0x1fff
				n += 2
			} else {
				entry.ProgramMapPID = pio.U16BE(b[n:])&0x1fff
				n += 2
			}
			self.Entries = append(self.Entries, entry)
		} else {
			break
		}
	}
	if n < len(b) {
		err = ErrParsePAT
		return
	}
	return
}

type Descriptor struct {
	Tag  uint8
	Data []byte
}

type ElementaryStreamInfo struct {
	StreamType    uint8
	ElementaryPID uint16
	Descriptors   []Descriptor
}

type PMT struct {
	PCRPID                uint16
	ProgramDescriptors    []Descriptor
	ElementaryStreamInfos []ElementaryStreamInfo
}

//pmt len get the pmt len
func (self PMT) Len() (n int) {
	// 111(3)
	// PCRPID(13)
	n += 2

	// desclen(16)
	n += 2

	for _, desc := range self.ProgramDescriptors {
		n += 2+len(desc.Data)
	}

	for _, info := range self.ElementaryStreamInfos {
		// streamType
		n += 1

		// Reserved(3)
		// Elementary PID(13)
		n += 2

		// Reserved(6)
		// ES Info length length(10)
		n += 2

		for _, desc := range info.Descriptors {
			n += 2+len(desc.Data)
		}
	}

	return
}

func (self PMT) fillDescs(b []byte, descs []Descriptor) (n int) {
	for _, desc := range descs {
		b[n] = desc.Tag
		n++
		b[n] = uint8(len(desc.Data))
		n++
		copy(b[n:], desc.Data)
		n += len(desc.Data)
	}
	return
}

/*
PMT (Program map specific data)[edit]
PMT specific data
Name	Number
of bits	Description
Reserved bits	3	Set to 0x07 (all bits on)
PCR PID	13	The packet identifier that contains the program clock reference used to improve the random access accuracy of the stream's timing that is derived from the program timestamp. If this is unused. then it is set to 0x1FFF (all bits on).
Reserved bits	4	Set to 0x0F (all bits on)
Program info length unused bits	2	Set to 0 (all bits off)
Program info length	10	The number of bytes that follow for the program descriptors.
Program descriptors	N*8	When the program info length is non-zero, this is the program info length number of program descriptor bytes.
Elementary stream info data	N*8	The streams used in this program map.

Name	Number
of bits	Description
stream type	8	This defines the structure of the data contained within the elementary packet identifier.
Reserved bits	3	Set to 0x07 (all bits on)
Elementary PID	13	The packet identifier that contains the stream type data.
Reserved bits	4	Set to 0x0F (all bits on)
ES Info length unused bits	2	Set to 0 (all bits off)
ES Info length length	10	The number of bytes that follow for the elementary stream descriptors.
Elementary stream descriptors	N*8	When the ES info length is non-zero, this is the ES info length number of elementary stream descriptor bytes
*/

func (self PMT) Marshal(b []byte) (n int) {
	// 111(3)
	// PCRPID(13)
	pio.PutU16BE(b[n:], self.PCRPID|7<<13)
	n += 2

	hold := n
	n += 2
	pos := n
	n += self.fillDescs(b[n:], self.ProgramDescriptors)
	desclen := n-pos
	pio.PutU16BE(b[hold:], uint16(desclen)|0xf<<12)

	for _, info := range self.ElementaryStreamInfos {
		b[n] = info.StreamType
		n++
		// Reserved(3)
		// Elementary PID(13)
		pio.PutU16BE(b[n:], info.ElementaryPID|7<<13)
		n += 2

		hold := n
		n += 2
		pos := n
		n += self.fillDescs(b[n:], info.Descriptors)
		desclen := n-pos
		pio.PutU16BE(b[hold:], uint16(desclen)|0x3c<<10)
	}

	return
}

func (self PMT) parseDescs(b []byte) (descs []Descriptor, err error) {
	n := 0
	for n < len(b) {
		if n+2 <= len(b) {
			desc := Descriptor{}
			desc.Tag = b[n]
			desc.Data = make([]byte, b[n+1])
			n += 2
			if n+len(desc.Data) < len(b) {
				copy(desc.Data, b[n:])
				descs = append(descs, desc)
				n += len(desc.Data)
			} else {
				break
			}
		} else {
			break
		}
	}
	if n < len(b) {
		err = ErrParsePMT
		return
	}
	return
}

func (self *PMT) Unmarshal(b []byte) (n int, err error) {
	if len(b) < n+4 {
		err = ErrParsePMT
		return
	}

	// 111(3)
	// PCRPID(13)
	self.PCRPID = pio.U16BE(b[0:2])&0x1fff
	n += 2

	// Reserved(4)=0xf
	// Reserved(2)=0x0
	// Program info length(10)
	desclen := int(pio.U16BE(b[2:4])&0x3ff)
	n += 2

	if desclen > 0 {
		if len(b) < n+desclen {
			err = ErrParsePMT
			return
		}
		if self.ProgramDescriptors, err = self.parseDescs(b[n:n+desclen]); err != nil {
			return
		}
		n += desclen
	}

	for n < len(b) {
		if len(b) < n+5 {
			err = ErrParsePMT
			return
		}

		var info ElementaryStreamInfo
		info.StreamType = b[n]
		n++

		// Reserved(3)
		// Elementary PID(13)
		info.ElementaryPID = pio.U16BE(b[n:])&0x1fff
		n += 2

		// Reserved(6)
		// ES Info length(10)
		desclen := int(pio.U16BE(b[n:])&0x3ff)
		n += 2

		if desclen > 0 {
			if len(b) < n+desclen {
				err = ErrParsePMT
				return
			}
			if info.Descriptors, err = self.parseDescs(b[n:n+desclen]); err != nil {
				return
			}
			n += desclen
		}

		self.ElementaryStreamInfos = append(self.ElementaryStreamInfos, info)
	}

	return
}

func ParsePSI(h []byte) (tableid uint8, tableext uint16, hdrlen int, datalen int, err error) {
	if len(h) < 8 {
		err = ErrPSIHeader
		return
	}

	// pointer(8)
	pointer := h[0]
	hdrlen++
	if pointer > 0 {
		hdrlen += int(pointer)
		if len(h) < hdrlen {
			err = ErrPSIHeader
			return
		}
	}

	if len(h) < hdrlen+12 {
		err = ErrPSIHeader
		return
	}

	// table_id(8)
	tableid = h[hdrlen]
	hdrlen++

	// section_syntax_indicator(1)=1,private_bit(1)=0,reserved(2)=3,unused(2)=0,section_length(10)
	datalen = int(pio.U16BE(h[hdrlen:]))&0x3ff - 9
	hdrlen += 2

	if datalen < 0 {
		err = ErrPSIHeader
		return
	}

	// Table ID extension(16)
	tableext = pio.U16BE(h[hdrlen:])
	hdrlen += 2

	// resverd(2)=3
	// version(5)
	// Current_next_indicator(1)
	hdrlen++

	// section_number(8)
	hdrlen++

	// last_section_number(8)
	hdrlen++

	// data

	// crc(32)

	return
}

const PSIHeaderLength = 9

/*

Name	        Number of bits Description
Pointer field	             8	  Present at the start of the TS packet payload signaled by the payload_unit_start_indicator bit in the TS header. Used to set packet alignment bytes or content before the start of tabled payload data.
Pointer filler bytes	     N*8  When the pointer field is non-zero, this is the pointer field number of alignment padding bytes set to 0xFF or the end of the previous table section spanning across TS packets (electronic program guide).

		Table header[2] [3] repeated until end of TS packet payload[1]
Name	        Number of bits Description
Table ID	                8   Table Identifier, that defines the structure of the syntax section and other contained data. As an exception, if this is the byte that immediately follow previous table section and is set to 0xFF, then it indicates that the repeat of table section end here and the rest of TS packet payload shall be stuffed with 0xFF. Consequently, the value 0xFF shall not be used for the Table Identifier.[1]
Section syntax indicator	1   A flag that indicates if the syntax section follows the section length. The PAT, PMT, and CAT all set this to 1.
Private bit	                1   The PAT, PMT, and CAT all set this to 0. Other tables set this to 1.
Reserved bits	                2   Set to 0x03 (all bits on)
Section length unused bits	2   Set to 0 (all bits off)
Section length	                10  The number of bytes that follow for the syntax section (with CRC value) and/or table data. These bytes must not exceed a value of 1021.
Syntax section/Table data	N*8 When the section length is non-zero, this is the section length number of syntax and data bytes.

				Table syntax section
Name	   Number of bits Description
Table ID extension	16   Informational only identifier. The PAT uses this for the transport stream identifier and the PMT uses this for the Program number.
Reserved bits	        2    Set to 0x03 (all bits on)
Version number	        5    Syntax version number. Incremented when data is changed and wrapped around on overflow for values greater than 32.
Current/next indicator	1    Indicates if data is current in effect or is for future use. If the bit is flagged on, then the data is to be used at the present moment.
Section number	        8    This is an index indicating which table this is in a related sequence of tables. The first table starts from 0.
Last section number	8    This indicates which table is the last table in the sequence of tables.
Table data	        N*8  Data as defined by the Table Identifier.
CRC32	                32   A checksum of the entire table excluding the pointer field, pointer filler bytes and the trailing CRC32.

Descriptor[edit]
                    Descriptor[2] [3]
Name	Number of bits Description

descriptor tag	        8	the tag defines the structure of the contained data following the descriptor length.
descriptor length	8	The number of bytes that are to follow.
Descriptor data	        N*8	Data as defined by the Descriptor Tag.
*/

func FillPSI(h []byte, tableid uint8, tableext uint16, datalen int) (n int) {
	// pointer(8)
	h[n] = 0
	n++

	// table_id(8)
	h[n] = tableid
	n++

	// section_syntax_indicator(1)=1,private_bit(1)=0,reserved(2)=3,unused(2)=0,section_length(10)
	pio.PutU16BE(h[n:], uint16(0xa<<12 | 2+3+4+datalen))
	n += 2

	// Table ID extension(16)
	pio.PutU16BE(h[n:], tableext)
	n += 2

	// resverd(2)=3,version(5)=0,Current_next_indicator(1)=1
	h[n] = 0x3<<6 | 1
	n++

	// section_number(8)
	h[n] = 0
	n++

	// last_section_number(8)
	h[n] = 0
	n++

	n += datalen

	crc := calcCRC32(0xffffffff, h[1:n])
	pio.PutU32LE(h[n:], crc)
	n += 4

	return
}

func TimeToPCR(tm time.Duration) (pcr uint64) {
	// base(33)+resverd(6)+ext(9)
	ts := uint64(tm*PCR_HZ/time.Second)
	base := ts / 300
	ext := ts % 300
	pcr = base<<15 | 0x3f<<9 | ext
	return
}

func PCRToTime(pcr uint64) (tm time.Duration) {
	base := pcr >> 15
	ext := pcr & 0x1ff
	ts := base*300 + ext
	tm = time.Duration(ts)*time.Second/time.Duration(PCR_HZ)
	return
}

func TimeToTs(tm time.Duration) (v uint64) {
	ts := uint64(tm*PTS_HZ/time.Second)
	// 0010	PTS 32..30 1	PTS 29..15 1 PTS 14..00 1
	v = ((ts>>30)&0x7)<<33 | ((ts>>15)&0x7fff)<<17 | (ts&0x7fff)<<1 | 0x100010001
	return
}

func TsToTime(v uint64) (tm time.Duration) {
	// 0010	PTS 32..30 1	PTS 29..15 1 PTS 14..00 1
	ts := (((v>>33)&0x7)<<30) | (((v>>17)&0x7fff) << 15) | ((v>>1)&0x7fff)
	tm = time.Duration(ts)*time.Second/time.Duration(PTS_HZ)
	return
}

const (
	PTS_HZ = 90000
	PCR_HZ = 27000000
)

func ParsePESHeader(h []byte) (hdrlen int, streamid uint8, datalen int, pts, dts time.Duration, err error) {
	if h[0] != 0 || h[1] != 0 || h[2] != 1 {
		err = ErrPESHeader
		return
	}
	streamid = h[3]

	flags := h[7]
	hdrlen = int(h[8])+9

	datalen = int(pio.U16BE(h[4:6]))
	if datalen > 0 {
		datalen -= int(h[8])+3
	}

	const PTS = 1 << 7
	const DTS = 1 << 6

	if flags&PTS != 0 {
		if len(h) < 14 {
			err = ErrPESHeader
			return
		}
		pts = TsToTime(pio.U40BE(h[9:14]))
		if flags&DTS != 0 {
			if len(h) < 19 {
				err = ErrPESHeader
				return
			}
			dts = TsToTime(pio.U40BE(h[14:19]))
		}
	}

	return
}

/*
PES packet header

Name	                        Size	                                Description
Packet start code prefix	3 bytes	0x000001
Stream id	                1 byte	                                Examples: Audio streams (0xC0-0xDF),
                                                                        Video streams (0xE0-0xEF) [4][5][6][7]
Note: The above 4 bytes is called the 32 bit start code.

PES Packet length	        2 bytes	                                Specifies the number of bytes remaining in the packet after this field. Can be zero. If the PES packet length is set to zero, the PES packet can be of any length. A value of zero for the PES packet length can be used only when the PES packet payload is a video elementary stream.[8]
Optional PES header	     variable length (length >= 3)	        not present in case of Padding stream & Private stream 2 (navigation data)
Stuffing bytes	                variable length
Data		                See elementary stream. In the case of private streams the first byte of the payload is the sub-stream number.

Optional PES header
Name	                   Number of Bits	     Description
Marker bits	                2	             10 binary or 0x2 hex
Scrambling control	        2	             00 implies not scrambled
Priority	                1
Data alignment indicator	1	             1 indicates that the PES packet header is immediately followed by the video start code or audio syncword
Copyright	                1	             1 implies copyrighted
Original or Copy	        1	             1 implies original
PTS DTS indicator	        2	             11 = both present, 01 is forbidden, 10 = only PTS, 00 = no PTS or DTS
ESCR flag	                1
ES rate flag	                1
DSM trick mode flag	        1
Additional copy info flag	1
CRC flag	                1
extension flag	                1
PES header length	        8                    gives the length of the remainder of the PES header in bytes
Optional fields	           variable length	     presence is determined by flag bits above
Stuffing Bytes	           variable length	    0xff

While above flags indicate that values are appended into variable length optional fields, they are not just
simply written out. For example, PTS (and DTS) is expanded from 33 bits to 5 bytes (40 bits). If only PTS is present, this is done by catenating 0010b,
most significant 3 bits from PTS, 1, following next 15 bits, 1, rest 15 bits and 1.
 If both PTS and DTS are present, first 4 bits are 0011 and first 4 bits for DTS are 0001. Other appended bytes have similar but different encoding.
*/

func writeTs(src []byte, i int, fb uint, ts uint64) {
	val := uint32(0)
	if ts > 0x1ffffffff {
		ts -= 0x1ffffffff
	}
	val = uint32(fb<<4) | ((uint32(ts>>30) & 0x07) << 1) | 1
	src[i] = byte(val)
	i++

	val = ((uint32(ts>>15) & 0x7fff) << 1) | 1
	src[i] = byte(val >> 8)
	i++
	src[i] = byte(val)
	i++

	val = (uint32(ts&0x7fff) << 1) | 1
	src[i] = byte(val >> 8)
	i++
	src[i] = byte(val)
}


func FillAPESHeader(h []byte, streamid uint8, datalen int, pts, dts uint64) (n int) {
	h[0] = 0
	h[1] = 0
	h[2] = 1
	h[3] = streamid

	const PTS = 1 << 7
	const DTS = 1 << 6

	var flags uint
	if pts != 0 {
		flags |= PTS
		if dts != 0 {
			flags |= DTS
		}
	}

	if flags&PTS != 0 {
		n += 5
	}
	if flags&DTS != 0 {
		n += 5
	}

	// packet_length(16) if zero then variable length
	// Specifies the number of bytes remaining in the packet after this field. Can be zero.
	// If the PES packet length is set to zero, the PES packet can be of any length.
	// A value of zero for the PES packet length can be used only when the PES packet payload is a **video** elementary stream.
	var pktlen uint16
	if datalen >= 0 {
		pktlen = uint16(datalen + n + 3)
	}
	pio.PutU16BE(h[4:6], pktlen)

	h[6] = 2<<6|1 // resverd(6,2)=2,original_or_copy(0,1)=1
	h[7] = byte(flags)
	h[8] = uint8(n)

	// pts(40)?
	// dts(40)?
	//If only PTS is present, this is done by catenating 0010b
	//
	if flags&PTS != 0 {
		if flags&DTS != 0 {
			//first 4 bits are 0011 and first 4 bits for DTS are 0001
			//writeTs(src []byte, i int, fb int, ts int64)
			writeTs(h[9:14], 0 , flags>>6, pts)
			writeTs(h[14:19], 0 , 1, dts)
			//pio.PutU40BE(h[9:14], (pts)|3<<36)
			//pio.PutU40BE(h[14:19], (dts)|1<<36)
		} else {
			////If only PTS is present, this is done by catenating 0010b
			//pio.PutU40BE(h[9:14], (pts)|2<<36)
			writeTs(h[9:14], 0 , flags>>6, pts)
		}
	}

	n += 9
	return
}

func FillPESHeader(h []byte, streamid uint8, datalen int, pts, dts time.Duration) (n int) {
	h[0] = 0
	h[1] = 0
	h[2] = 1
	h[3] = streamid

	const PTS = 1 << 7
	const DTS = 1 << 6

	var flags uint
	if pts != 0 {
		flags |= PTS
		if dts != 0 {
			flags |= DTS
		}
	}

	if flags&PTS != 0 {
		n += 5
	}
	if flags&DTS != 0 {
		n += 5
	}

	// packet_length(16) if zero then variable length
	// Specifies the number of bytes remaining in the packet after this field. Can be zero.
	// If the PES packet length is set to zero, the PES packet can be of any length.
	// A value of zero for the PES packet length can be used only when the PES packet payload is a **video** elementary stream.
	var pktlen uint16
	if datalen >= 0 {
		pktlen = uint16(datalen + n + 3)
	}
	pio.PutU16BE(h[4:6], pktlen)

	h[6] = 2<<6|1 // resverd(6,2)=2,original_or_copy(0,1)=1
	h[7] = byte(flags)
	h[8] = uint8(n)

	// pts(40)?
	// dts(40)?
	//If only PTS is present, this is done by catenating 0010b
	//
	if flags&PTS != 0 {
		if flags&DTS != 0 {
			pts1 := uint64(pts*PTS_HZ/time.Second)
			dts1 := uint64(dts*PTS_HZ/time.Second)
			//first 4 bits are 0011 and first 4 bits for DTS are 0001
			writeTs(h[9:14], 0 , flags>>6, (pts1))
			writeTs(h[14:19], 0 , 1, (dts1))
			//pio.PutU40BE(h[9:14], tsio.TimeToTs(pts)|3<<36)
			//pio.PutU40BE(h[14:19], tsio.TimeToTs(dts)|1<<36)
		} else {
			////If only PTS is present, this is done by catenating 0010b
			//writeTs(h[9:14], 0 , 1, tsio.TimeToTs(dts))
			pts1 := uint64(pts*PTS_HZ/time.Second)
			writeTs(h[9:14], 0 , flags>>6, (pts1))
			//pio.PutU40BE(h[9:14], tsio.TimeToTs(pts)|2<<36)
		}
	}

	n += 9
	return
}

type TSWriter struct {
	w   io.Writer
	ContinuityCounter uint
	tshdr []byte
}

func NewTSWriter(pid uint16) *TSWriter {
	w := &TSWriter{}
	w.tshdr = make([]byte, 188)
	w.tshdr[0] = 0x47
	pio.PutU16BE(w.tshdr[1:3], pid&0x1fff)
	for i := 6; i < 188; i++ {
		w.tshdr[i] = 0xff
	}
	return w
}

func  writePcr(b []byte, i byte, pcr int64) error {
	b[i] = byte(pcr >> 25)
	i++
	b[i] = byte((pcr >> 17) & 0xff)
	i++
	b[i] = byte((pcr >> 9) & 0xff)
	i++
	b[i] = byte((pcr >> 1) & 0xff)
	i++
	b[i] = byte(((pcr & 0x1) << 7) | 0x7e)
	i++
	b[i] = 0x00

	return nil
}

//write frame to ts file
func (self *TSWriter) WritePackets(w *bufio.Writer, datav [][]byte, pcr time.Duration, sync bool, paddata bool) (err error) {
	datavlen := pio.VecLen(datav)
	writev := make([][]byte, len(datav))
	writepos := 0

	for writepos < datavlen {
		//payload_unit_start_indicator  1: first ,0: not fist
		self.tshdr[1] = self.tshdr[1]&0x1f
		self.tshdr[3] = byte(self.ContinuityCounter)&0xf|0x30
		self.tshdr[5] = 0 // flags
		hdrlen := 6
		self.ContinuityCounter++

		//just first page write
		if writepos == 0 {
			//payload_unit_start_indicator  1: first ,0: not fist
			self.tshdr[1] = 0x40|self.tshdr[1] // Payload Unit Start Indicator
			if pcr != 0 {
				hdrlen += 6
				//set pcr flag 0x10
				self.tshdr[5] = 0x10|self.tshdr[5] // PCR flag (Discontinuity indicator 0x80)
				//pio.PutU48BE(self.tshdr[6:12], (pcr))
				pcr1 := uint64(pcr*PTS_HZ/time.Second)
				writePcr(self.tshdr[6:12],0,pcr1)
			}
			if sync {
				self.tshdr[5] = 0x40|self.tshdr[5] // Random Access indicator
			}
		}

		padtail := 0
		end := writepos + 188 - hdrlen
		if end > datavlen {
			if paddata {
				padtail = end - datavlen
			} else {
				hdrlen += end - datavlen
			}
			end = datavlen
		}
		n := pio.VecSliceTo(datav, writev, writepos, end)

		self.tshdr[4] = byte(hdrlen)-5 // length
		if _, err = w.Write(self.tshdr[:hdrlen]); err != nil {
			return
		}
		for i := 0; i < n; i++ {
			if _, err = w.Write(writev[i]); err != nil {
				return
			}
		}
		if padtail > 0 {
			if _, err = w.Write(self.tshdr[188-padtail:188]); err != nil {
				return
			}
		}

		writepos = end
	}

	return
}

//parse ts header
func ParseTSHeader(tshdr []byte) (pid uint16, start bool, iskeyframe bool, hdrlen int, err error) {
	// https://en.wikipedia.org/wiki/MPEG_transport_stream
	if tshdr[0] != 0x47 {
		err = fmt.Errorf("tshdr sync invalid")
		return
	}
	pid = uint16((tshdr[1]&0x1f))<<8|uint16(tshdr[2])
	start = tshdr[1]&0x40 != 0
	hdrlen += 4
	if tshdr[3]&0x20 != 0 {
		hdrlen += int(tshdr[4])+1
		iskeyframe = tshdr[5]&0x40 != 0
	}
	return
}


