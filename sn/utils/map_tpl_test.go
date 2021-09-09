package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMPrefixMap(t *testing.T) {
	var data = []*MPrefixElement{
		{
			Key:   "a",
			Value: 1,
		},
		{
			Key:   "b",
			Value: 2,
		},
	}

	t.Run("Set Get", func(t *testing.T) {
		var m MPrefixMap
		m.Set(data[0].Key, data[0].Value)
		res, ok := m.Get(data[0].Key)
		assert.Equal(t, data[0].Value, res)
		assert.True(t, ok)
		res, ok = m.Get(data[1].Key)
		assert.Equal(t, nil, res)
		assert.False(t, ok)
		m.Set(data[0].Key, data[1].Value)
		res, ok = m.Get(data[0].Key)
		assert.Equal(t, data[1].Value, res)
		assert.True(t, ok)
	})

	t.Run("MustGet", func(t *testing.T) {
		var m MPrefixMap
		assert.Panics(t, func() {
			m.MustGet(data[0].Key)
		})
		m.Set(data[0].Key, data[0].Value)
		assert.NotPanics(t, func() {
			m.MustGet(data[0].Key)
		})
	})

	t.Run("Has", func(t *testing.T) {
		var m MPrefixMap
		assert.False(t, m.Has(data[0].Key))
		m.Set(data[0].Key, data[0].Value)
		assert.True(t, m.Has(data[0].Key))
	})

	t.Run("SetIfAbsent", func(t *testing.T) {
		var m MPrefixMap
		res, ok := m.SetIfAbsent(data[0].Key, data[0].Value)
		assert.Equal(t, data[0].Value, res)
		assert.False(t, ok)
		res, ok = m.SetIfAbsent(data[0].Key, data[1].Value)
		assert.Equal(t, data[0].Value, res)
		assert.True(t, ok)
		m.Set(data[0].Key, nil)
		assert.Panics(t, func() {
			m.SetIfAbsent(data[0].Key, data[0].Value)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		var m MPrefixMap
		m.Set(data[0].Key, data[0].Value)
		assert.True(t, m.Has(data[0].Key))
		m.Delete(data[0].Key)
		assert.False(t, m.Has(data[0].Key))
	})

	t.Run("Len", func(t *testing.T) {
		var m MPrefixMap
		m.Set(data[0].Key, data[0].Value)
		assert.Equal(t, 1, m.Len())
		m.Set(data[0].Key, data[1].Value)
		assert.Equal(t, 1, m.Len())
		m.Set(data[1].Key, data[1].Value)
		assert.Equal(t, 2, m.Len())
	})

	t.Run("clear", func(t *testing.T) {
		var m MPrefixMap
		m.Set(data[0].Key, data[0].Value)
		m.Clear()
		assert.Equal(t, 0, m.Len())
	})

	t.Run("Range", func(t *testing.T) {
		var m MPrefixMap
		m.Set(data[0].Key, data[0].Value)
		m.Set(data[1].Key, data[1].Value)
		var res []*MPrefixElement
		m.Range(func(k MTypeA, v MTypeB) bool {
			res = append(res, &MPrefixElement{
				Key:   k,
				Value: v,
			})
			return true
		})
		assert.Equal(t, 2, len(res))
		var res2 []*MPrefixElement
		m.Range(func(k MTypeA, v MTypeB) bool {
			res2 = append(res2, &MPrefixElement{
				Key:   k,
				Value: v,
			})
			return false
		})
		assert.Equal(t, 1, len(res2))
	})

	t.Run("Element", func(t *testing.T) {
		var m MPrefixMap
		m.Set(data[0].Key, data[0].Value)
		m.Set(data[1].Key, data[1].Value)
		assert.Equal(t, 2, len(m.Keys()))
		assert.Equal(t, 2, len(m.Values()))
		assert.Equal(t, 2, len(m.Elements()))
		assert.Equal(t, 2, len(m.Map()))
	})

	t.Run("Sort", func(t *testing.T) {
		var m MPrefixMap
		m.Set(data[0].Key, data[0].Value)
		m.Set(data[1].Key, data[1].Value)
		res := m.SortElement(func(a, b *MPrefixElement) bool {
			return a.Value.(int) > b.Value.(int)
		})
		assert.Equal(t, data[1].Key, res[0].Key)
		assert.Equal(t, data[1].Value, res[0].Value)
		assert.Equal(t, data[0].Key, res[1].Key)
		assert.Equal(t, data[0].Value, res[1].Value)
		res2 := m.SortKey(func(a, b *MPrefixElement) bool {
			return a.Value.(int) > b.Value.(int)
		})
		assert.Equal(t, data[1].Key, res2[0])
		assert.Equal(t, data[0].Key, res2[1])
		res3 := m.SortValue(func(a, b *MPrefixElement) bool {
			return a.Value.(int) > b.Value.(int)
		})
		assert.Equal(t, data[1].Value, res3[0])
		assert.Equal(t, data[0].Value, res3[1])
	})
}
