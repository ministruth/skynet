package tpl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSliceFinder(t *testing.T) {
	t.Run("Add finder", func(t *testing.T) {
		f := NewSliceFinder([]int{
			1, 1, 2, 2, 3, 3,
		})
		assert.Equal(t, 3, len(f.Data))
		assert.Equal(t, true, f.Data[1])
		assert.Equal(t, true, f.Data[2])
		assert.Equal(t, true, f.Data[3])
		assert.Equal(t, false, f.Data[4])
		f.Add([]int{4})
		assert.Equal(t, true, f.Data[4])
	})

	t.Run("Find finder", func(t *testing.T) {
		f := NewSliceFinder([]int{
			1, 1, 2, 2, 3, 3,
		})
		assert.Equal(t, true, f.Find(1))
		assert.Equal(t, true, f.Find(2))
		assert.Equal(t, true, f.Find(3))
		assert.Equal(t, false, f.Find(4))
	})

	t.Run("Clear finder", func(t *testing.T) {
		f := NewSliceFinder([]int{
			1, 1, 2, 2, 3, 3,
		})
		assert.Equal(t, 3, len(f.Data))
		f.Clear()
		assert.Equal(t, 0, len(f.Data))
	})

	t.Run("Len finder", func(t *testing.T) {
		f := NewSliceFinder([]int{
			1, 1, 2, 2, 3, 3,
		})
		assert.Equal(t, 3, f.Len())
	})

	t.Run("GetSlice finder", func(t *testing.T) {
		f := NewSliceFinder([]int{
			1, 1, 2, 2, 3, 3,
		})
		assert.ElementsMatch(t, []int{1, 2, 3}, f.GetSlice())
	})
}
