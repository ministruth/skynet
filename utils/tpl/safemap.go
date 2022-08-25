package tpl

import (
	"sort"
	"sync"
)

type SafeMapElement[K comparable, V any] struct {
	Key   K
	Value V
}

type SafeMapSorterFunc[K comparable, V any] func(a *SafeMapElement[K, V], b *SafeMapElement[K, V]) bool

type SafeMap[K comparable, V any] struct {
	Data sync.Map
}

func (m *SafeMap[K, V]) Clear() {
	m.Data = sync.Map{}
}

func (m *SafeMap[K, V]) Len() int {
	return len(m.Keys())
}

func (m *SafeMap[K, V]) Get(k K) (V, bool) {
	ret, ok := m.Data.Load(k)
	if ret == nil {
		var tmp V
		return tmp, ok
	}
	return ret.(V), ok
}

func (m *SafeMap[K, V]) MustGet(k K) V {
	ret, ok := m.Get(k)
	if !ok {
		panic("key not found")
	}
	return ret
}

func (m *SafeMap[K, V]) Set(k K, v V) {
	m.Data.Store(k, v)
}

func (m *SafeMap[K, V]) SetIfAbsent(k K, v V) (V, bool) {
	ret, ok := m.Data.LoadOrStore(k, v)
	if ret == nil {
		var tmp V
		return tmp, ok
	}
	return ret.(V), ok
}

func (m *SafeMap[K, V]) Delete(k K) {
	m.Data.Delete(k)
}

func (m *SafeMap[K, V]) Has(k K) bool {
	_, ok := m.Data.Load(k)
	return ok
}

func (m *SafeMap[K, V]) Range(f func(k K, v V) bool) {
	m.Data.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}

func (m *SafeMap[K, V]) Keys() []K {
	var ret []K
	m.Range(func(k K, v V) bool {
		ret = append(ret, k)
		return true
	})
	return ret
}

func (m *SafeMap[K, V]) Values() []V {
	var ret []V
	m.Range(func(k K, v V) bool {
		ret = append(ret, v)
		return true
	})
	return ret
}

func (m *SafeMap[K, V]) Elements() []*SafeMapElement[K, V] {
	var ret []*SafeMapElement[K, V]
	m.Range(func(k K, v V) bool {
		ret = append(ret, &SafeMapElement[K, V]{
			Key:   k,
			Value: v,
		})
		return true
	})
	return ret
}

func (m *SafeMap[K, V]) Map() map[K]V {
	ret := make(map[K]V)
	m.Range(func(k K, v V) bool {
		ret[k] = v
		return true
	})
	return ret
}

func (m *SafeMap[K, V]) SortElement(f SafeMapSorterFunc[K, V]) []*SafeMapElement[K, V] {
	ret := m.Elements()
	sort.SliceStable(ret, func(i, j int) bool {
		return f(ret[i], ret[j])
	})
	return ret
}

func (m *SafeMap[K, V]) SortKey(f SafeMapSorterFunc[K, V]) []K {
	res := m.SortElement(f)
	var ret []K
	for _, v := range res {
		ret = append(ret, v.Key)
	}
	return ret
}

func (m *SafeMap[K, V]) SortValue(f SafeMapSorterFunc[K, V]) []V {
	res := m.SortElement(f)
	var ret []V
	for _, v := range res {
		ret = append(ret, v.Value)
	}
	return ret
}
