package rtmp

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"sync"
)


var (
	ErrNoKey = errors.New("No key for cache")
)

type TSItem struct {
	Name     string
	SeqNum   uint64
	Duration float32
}

func NewTSItem(name string, duration float32 ,seqNum uint64) *TSItem {
	var item TSItem
	item.Name = name
	item.SeqNum = seqNum
	item.Duration = duration
	return &item
}

type m3u8Box struct {
	id   string
	num  int
	lock sync.RWMutex
	ll   *list.List
}

func NewM3u8Box(id string) *m3u8Box {
	return &m3u8Box{
		id:  id,
		ll:  list.New(),
		num: 3,
	}
}

func (tcCacheItem *m3u8Box) ID() string {
	return tcCacheItem.id
}


func (self *m3u8Box) GenM3U8PlayList() ([]byte, error) {
	var seq uint64
	var getSeq bool
	var maxDuration float32
	m3u8body := bytes.NewBuffer(nil)
	for e := self.ll.Front(); e != nil; e = e.Next() {
		key := e.Value.(*TSItem)
		if key.Duration > maxDuration {
			maxDuration = key.Duration
		}
		if !getSeq {
			getSeq = true
			seq = key.SeqNum
		}
		fmt.Fprintf(m3u8body, "#EXTINF:%.3f,\n%s\n", float64(key.Duration), key.Name)

	}

	w := bytes.NewBuffer(nil)
	fmt.Fprintf(w,
		"#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:%d\n#EXT-X-MEDIA-SEQUENCE:%d\n\n",
		int32(maxDuration)+1, seq)
	w.Write(m3u8body.Bytes())
	return w.Bytes(), nil
}

func (self *m3u8Box) SetItem(item *TSItem) {
	if self.ll.Len() == self.num {
		e := self.ll.Front()
		self.ll.Remove(e)
	}
	self.ll.PushBack(item)
}
