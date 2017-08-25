package AvQue

import ()
import (
	"rtmpServerStudy/av"
	"crypto/des"
	"golang.org/x/crypto/openpgp/packet"
)

type Object [](*av.Packet)
type AvRingbuffer struct {
	write uint32
	read  uint32
	array Object
	mask  uint32
	size  uint32
	maskBase uint32
	flag  uint8
}

func RingBufferCreate(n uint32) *AvRingbuffer {

	var sz uint32
	var rb *AvRingbuffer

	if n < 1 || n > 30 {
		return nil
	}

	rb = new(AvRingbuffer)
	if rb == nil {
		return nil
	}
	rb.maskBase = n
	sz = (1 << n) + 1
	rb.mask = (1 << n) - 1
	rb.size = sz
	rb.flag = 1
	rb.array = make(Object, sz)
	if rb.array == nil {
		return nil
	}
	return rb
}

func (rb *AvRingbuffer) RingBufferSize() uint32 {

	size := (rb.write - rb.read) & rb.mask
	return (size)
}

func (rb *AvRingbuffer) RingBufferIsEmpty() int {

	if rb.write == rb.read {
		return 1
	}
	return 0
}

func (rb *AvRingbuffer) RingBufferIsFull() int {

	if ((rb.write + 1) & rb.mask) == rb.read {
		return 1
	}
	return 0
}

func (rb *AvRingbuffer) RingBufferGet() (ptr *av.Packet) {

	if rb.write == rb.read {
		return nil
	}
	ptr = rb.array[rb.read]
	//here is must
	rb.array[rb.read] = nil
	rb.read = (rb.read + 1) & rb.mask
	return ptr
}

func (rb *AvRingbuffer) RingBufferPut(ptr *av.Packet) int {

	if ((rb.write + 1) & rb.mask) == rb.read {
		return -1
	}
	rb.array[rb.write] = ptr
	rb.write = (rb.write + 1) & rb.mask
	return 0
}

func (rb *AvRingbuffer)GopCopy() *AvRingbuffer{
	dsc := RingBufferCreate(rb.maskBase)
	dsc = rb.RingBufferCopy(dsc)
	return dsc
}

func (rb *AvRingbuffer)RingBufferCopy(dsc *AvRingbuffer) *AvRingbuffer {

	write := rb.write
	read := rb.read
	mask :=rb.mask

	for {
		if write == read{
			return dsc
		}
		dsc.RingBufferPut(rb.array[read])
		read = (read +1) & mask
	}
	return dsc
}

func (rb *AvRingbuffer)RingBufferABSPut(ptr *av.Packet)(dsc *AvRingbuffer){
	n:=rb.RingBufferPut(ptr)
	if n != 0{
		dsc = RingBufferCreate(rb.maskBase + 1)
		if dsc == nil{
			return nil
		}
		dsc = rb.RingBufferCopy(dsc)
	}
	dsc.RingBufferPut(ptr)
	return dsc
}

func (rb *AvRingbuffer)RingBufferCleanOldGop(){

	rb.array[rb.read] = nil
	rb.read = (rb.read +1) & rb.mask
	for {
		if rb.read == rb.write{
			return
		}
		pkt:= rb.array[rb.read]
		if pkt.IsKeyFrame{
			break
		}else{
			rb.array[rb.read] = nil
			rb.read = (rb.read +1) & rb.mask
		}
	}
}

func (rb *AvRingbuffer)RingBufferCleanGop(){
	for {
		if rb.RingBufferGet() == nil {
			break
		}
	}
}

