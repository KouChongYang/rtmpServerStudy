package AvQue

import (
	//"sync"
	"container/list"
)

//just for
type CursorList struct {
	//sync.SpinLock
	data *list.List
}

func NewPublist() *CursorList {
	q := new(CursorList)
	q.data = list.New()
	return q
}

func (q *CursorList) PushFront(v interface{}) {
	//q.Lock()
	q.data.PushFront(v)
	//q.Unlock()
}

func (q *CursorList) GetList() *list.List {
	return q.data
}

func (q *CursorList) PushBack(v interface{}) {
	//q.Lock()
	q.data.PushBack(v)
	//q.Unlock()
}

func (q *CursorList) Len() int {
	//q.Lock()
	//defer q.Unlock()
	return q.data.Len()
}

func (q *CursorList) Pop() interface{} {

	//q.Lock()
	//defer q.Unlock()
	iter := q.data.Back()
	if iter == nil {
		return nil
	}
	return q.data.Remove(iter)
}
