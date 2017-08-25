package AvQue


import (
"io"
"sync"
"rtmpServerStudy/av"
)

//        time
// ----------------->
//
// V-A-V-V-A-V-V-A-V-V
// |                 |
// 0        5        10
// head             tail
// oldest          latest
//

const (
	AudioAfterLastVideoCnt = 311
)

// One publisher and multiple subscribers thread-safe packet buffer queue.
type AvQueue struct {
	buf                      *Buf
	CachedVideoCount       uint32
	audioAfterLastVideoCnt uint32
	head, tail               int
	lock                     *sync.RWMutex
	cond                     *sync.Cond
	curgopcount, maxgopcount int
	streams                  []av.CodecData
	videoidx                 int
	closed                   bool
}

func NewQueue(size int,gopsize int) *AvQueue {
	q := &AvQueue{}
	q.buf = NewBuf(size)
	q.maxgopcount = gopsize
	q.CachedVideoCount = 0
	q.lock = &sync.RWMutex{}
	q.cond = sync.NewCond(q.lock.RLocker())
	q.videoidx = -1
	return q
}

func (self *AvQueue)Copy(dst *AvQueue)(err error){

	dst.closed = self.closed
	dst.maxgopcount = self.maxgopcount
	dst.videoidx = self.videoidx
	dst.curgopcount = self.curgopcount
	dst.streams = self.streams

	buf := self.buf.Copy(dst.buf)

	dst.buf = buf
	return
}

func (self *AvQueue) SetMaxGopCount(n int) {
	self.lock.Lock()
	self.maxgopcount = n
	self.lock.Unlock()
	return
}

func (self *AvQueue) WriteTrailer() error {
	return nil
}

// After Close() called, all QueueCursor's ReadPacket will return io.EOF.
func (self *AvQueue) Close() (err error) {
	self.lock.Lock()

	self.closed = true
	self.cond.Signal()

	self.lock.Unlock()
	return
}

// Put packet into buffer, old packets will be discared.
func (self *AvQueue) WritePacket(pkt *av.Packet) (err error) {

	self.lock.Lock()

	self.buf.Push(pkt)
	if  pkt.GopIsKeyFrame {
		self.curgopcount++
	}

	//释放挤压的未发出的gop
	for self.curgopcount >= self.maxgopcount && self.buf.Count > 1 {
		pkt := self.buf.Pop()
		if  pkt.GopIsKeyFrame {
			self.curgopcount--
		}
		if self.curgopcount < self.maxgopcount {
			break
		}
	}

	self.cond.Signal()

	self.lock.Unlock()
	return
}

type QueueCursor struct {
	que    *AvQueue
	pos    BufPos
	gotpos bool
	init   func(buf *Buf, videoidx int) BufPos
}



func (self *AvQueue) newCursor() *QueueCursor {
	return &QueueCursor{
		que: self,
	}
}

// Create cursor position at latest packet.
func (self *AvQueue) Latest() *QueueCursor {
	cursor := self.newCursor()
	cursor.init = func(buf *Buf, videoidx int) BufPos {
		return buf.Tail
	}
	return cursor
}

// Create cursor position at oldest buffered packet.
func (self *AvQueue) Oldest() *QueueCursor {
	cursor := self.newCursor()
	cursor.init = func(buf *Buf, videoidx int) BufPos {
		return buf.Head
	}
	return cursor
}



// ReadPacket will not consume packets in Queue, it's just a cursor.
func (self *QueueCursor) ReadPacket() (pkt *av.Packet, err error) {
	self.que.cond.L.Lock()
	buf := self.que.buf
	if !self.gotpos {
		self.pos = self.init(buf, self.que.videoidx)
		self.gotpos = true
	}
	for {
		if self.pos.LT(buf.Head) {
			self.pos = buf.Head
		} else if self.pos.GT(buf.Tail) {
			self.pos = buf.Tail
		}
		if buf.IsValidPos(self.pos) {
			pkt = buf.Get(self.pos)
			self.pos++
			break
		}
		if self.que.closed {
			err = io.EOF
			break
		}
		self.que.cond.Wait()
	}
	self.que.cond.L.Unlock()
	return
}


