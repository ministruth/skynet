package shared

import (
	"skynet/sn/tpl"

	"github.com/google/uuid"
)

type ChanMap struct {
	tpl.SafeMap[uuid.UUID, chan any]
}

func (m *ChanMap) Set(k uuid.UUID) chan any {
	c := make(chan any, 1)
	m.SafeMap.Set(k, c)
	return c
}

func (m *ChanMap) SetIfAbsent(k uuid.UUID) (chan any, bool) {
	c := make(chan any, 1)
	return m.SafeMap.SetIfAbsent(k, c)
}

func (m *ChanMap) Push(k uuid.UUID, v any) bool {
	c, _ := m.SetIfAbsent(k)
	if len(c) > 0 {
		return false
	}
	c <- v
	return true
}
