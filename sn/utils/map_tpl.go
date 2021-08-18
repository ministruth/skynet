package utils

import (
	"sort"
	"sync"

	"github.com/cheekybits/genny/generic"
)

type MPrefix generic.Type
type MTypeA generic.Type
type MTypeB generic.Type

type MPrefixElement struct {
	Key   MTypeA
	Value MTypeB
}

type MPrefixSorterFunc func(a *MPrefixElement, b *MPrefixElement) bool

type MPrefixMap struct {
	Data   sync.Map
	length int
}

func (m *MPrefixMap) Clear() {
	m.length = 0
	m.Data = sync.Map{}
}

func (m *MPrefixMap) Len() int {
	return m.length
}

func (m *MPrefixMap) Get(k MTypeA) (MTypeB, bool) {
	ret, ok := m.Data.Load(k)
	if ret == nil {
		return nil, false
	}
	return ret.(MTypeB), ok
}

func (m *MPrefixMap) MustGet(k MTypeA) MTypeB {
	ret, ok := m.Get(k)
	if !ok {
		panic("key not found")
	}
	return ret
}

func (m *MPrefixMap) Set(k MTypeA, v MTypeB) {
	if !m.Has(k) {
		m.length++
	}
	m.Data.Store(k, v)
}

func (m *MPrefixMap) SetIfAbsent(k MTypeA, v MTypeB) (MTypeB, bool) {
	ret, ok := m.Data.LoadOrStore(k, v)
	if !ok {
		m.length++
	}
	if ret == nil {
		panic("key not found")
	}
	return ret.(MTypeB), ok
}

func (m *MPrefixMap) Delete(k MTypeA) {
	if m.Has(k) {
		m.length--
	}
	m.Data.Delete(k)
}

func (m *MPrefixMap) Has(k MTypeA) bool {
	_, ok := m.Data.Load(k)
	return ok
}

func (m *MPrefixMap) Range(f func(k MTypeA, v MTypeB) bool) {
	m.Data.Range(func(key, value interface{}) bool {
		return f(key.(MTypeA), value.(MTypeB))
	})
}

func (m *MPrefixMap) Keys() []MTypeA {
	var ret []MTypeA
	m.Range(func(key MTypeA, value MTypeB) bool {
		ret = append(ret, key)
		return true
	})
	return ret
}

func (m *MPrefixMap) Values() []MTypeB {
	var ret []MTypeB
	m.Range(func(key MTypeA, value MTypeB) bool {
		ret = append(ret, value)
		return true
	})
	return ret
}

func (m *MPrefixMap) Elements() []*MPrefixElement {
	var ret []*MPrefixElement
	m.Range(func(key MTypeA, value MTypeB) bool {
		ret = append(ret, &MPrefixElement{
			Key:   key,
			Value: value,
		})
		return true
	})
	return ret
}

func (m *MPrefixMap) Map() map[MTypeA]MTypeB {
	ret := make(map[MTypeA]MTypeB)
	m.Range(func(key MTypeA, value MTypeB) bool {
		ret[key] = value
		return true
	})
	return ret
}

func (m *MPrefixMap) SortElement(f MPrefixSorterFunc) []*MPrefixElement {
	ret := m.Elements()
	sort.SliceStable(ret, func(i, j int) bool {
		return f(ret[i], ret[j])
	})
	return ret
}

func (m *MPrefixMap) SortKey(f MPrefixSorterFunc) []MTypeA {
	res := m.SortElement(f)
	var ret []MTypeA
	for _, v := range res {
		ret = append(ret, v.Key)
	}
	return ret
}

func (m *MPrefixMap) SortValue(f MPrefixSorterFunc) []MTypeB {
	res := m.SortElement(f)
	var ret []MTypeB
	for _, v := range res {
		ret = append(ret, v.Value)
	}
	return ret
}
