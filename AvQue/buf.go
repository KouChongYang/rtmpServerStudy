package AvQue

import (
	"fmt"
	"rtmpServerStudy/av"
)

type Buf struct {
	Head, Tail BufPos
	pkts       [](*av.Packet)
	Size       int
	Count      int
}

func NewBuf(size int) *Buf {
	return &Buf{
		pkts: make([](*av.Packet), size),
	}
}

func (self *Buf) CleanUp() {

	for i := self.Head; i.LT(self.Tail); i++ {
		self.pkts[int(i)&(len(self.pkts)-1)] = nil
	}
	self.Head = 0
	self.Tail = 0
	self.Size = 0
	self.Count = 0
}

func (self *Buf) Pop() *av.Packet {
	fmt.Println("=======================================1=")
	if self.Count == 0 {
		return nil
	}
	fmt.Println("=======================================2=")

	i := int(self.Head) & (int(self.Tail) - 1)
	fmt.Println("=======================================3=:", i)
	pkt := self.pkts[i]
	self.pkts[i] = nil
	if pkt == nil {
		fmt.Println("=============================is or nil ====================")
	}
	fmt.Println("=======================================3=:", len(pkt.Data))
	self.Size -= len(pkt.Data)
	self.Head++
	self.Count--

	return pkt
}

func (self *Buf) PktsLen() int {
	return len(self.pkts)
}

func (self *Buf) Copy(dst *Buf) *Buf {

	dst.pkts = make([]*av.Packet, len(self.pkts))
	for i := self.Head; i.LT(self.Tail); i++ {
		dst.pkts[int(i)&(len(dst.pkts)-1)] = self.pkts[int(i)&(len(self.pkts)-1)]
	}
	dst.Tail = self.Tail
	dst.Count = self.Count
	dst.Size = self.Size

	return dst
}

func (self *Buf) grow() {
	newpkts := make([]*av.Packet, len(self.pkts)*2)
	for i := self.Head; i.LT(self.Tail); i++ {
		newpkts[int(i)&(len(newpkts)-1)] = self.pkts[int(i)&(len(self.pkts)-1)]
	}
	self.pkts = newpkts
}

func (self *Buf) Push(pkt *av.Packet) {
	if self.Count == len(self.pkts) {
		self.grow()
	}
	self.pkts[int(self.Tail)&(len(self.pkts)-1)] = pkt
	self.Tail++
	self.Count++
	self.Size += len(pkt.Data)
}

func (self *Buf) Get(pos BufPos) *av.Packet {
	return self.pkts[int(pos)&(len(self.pkts)-1)]
}

func (self *Buf) IsValidPos(pos BufPos) bool {
	return pos.GE(self.Head) && pos.LT(self.Tail)
}

type BufPos int

func (self BufPos) LT(pos BufPos) bool {
	return self-pos < 0
}

func (self BufPos) GE(pos BufPos) bool {
	return self-pos >= 0
}

func (self BufPos) GT(pos BufPos) bool {
	return self-pos > 0
}
