package shared

import (
	"skynet/sn/utils"

	"github.com/google/uuid"
)

type ChanMap struct {
	utils.UUIDMap
}

func (m *ChanMap) Get(k uuid.UUID) (chan interface{}, bool) {
	v, ok := m.UUIDMap.Get(k)
	if !ok {
		return nil, false
	}
	return v.(chan interface{}), true
}

func (m *ChanMap) Set(k uuid.UUID) chan interface{} {
	c := make(chan interface{}, 1)
	m.UUIDMap.Set(k, c)
	return c
}

func (m *ChanMap) SetIfAbsent(k uuid.UUID) (chan interface{}, bool) {
	c := make(chan interface{}, 1)
	ret, ok := m.UUIDMap.SetIfAbsent(k, c)
	return ret.(chan interface{}), ok
}

func (m *ChanMap) Push(k uuid.UUID, v interface{}) bool {
	c, _ := m.SetIfAbsent(k)
	if len(c) > 0 {
		return false
	}
	c <- v
	return true
}
