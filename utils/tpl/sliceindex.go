package tpl

import (
	"golang.org/x/exp/constraints"
)

type SliceIndex[K constraints.Integer, V any] struct {
	data []V
}

func NewSliceIndex[K constraints.Integer, V any](maxSize int) *SliceIndex[K, V] {
	ret := new(SliceIndex[K, V])
	ret.data = make([]V, maxSize)
	return ret
}

func (s *SliceIndex[K, V]) Set(k K, v V) {
	s.data[k] = v
}

func (s *SliceIndex[K, V]) Get(k K) V {
	return s.data[k]
}
