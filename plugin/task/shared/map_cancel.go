package shared

import (
	"skynet/sn/utils"
)

type CancelMap struct {
	utils.IntMap
}

func (m *CancelMap) Get(k int) (func() error, bool) {
	v, ok := m.IntMap.Get(k)
	if !ok {
		return nil, false
	}
	return v.(func() error), true
}

func (m *CancelMap) Set(k int, v func() error) {
	m.IntMap.Set(k, v)
}
