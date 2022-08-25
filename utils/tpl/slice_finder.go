package tpl

type SliceFinder[T comparable] struct {
	Data map[T]bool
}

func NewSliceFinder[T comparable](s []T) *SliceFinder[T] {
	ret := new(SliceFinder[T])
	ret.Data = make(map[T]bool)
	ret.Add(s)
	return ret
}

func (f *SliceFinder[T]) GetSlice() []T {
	ret := []T{}
	for k := range f.Data {
		ret = append(ret, k)
	}
	return ret
}

func (f *SliceFinder[T]) Add(v []T) {
	for _, e := range v {
		f.Data[e] = true
	}
}

func (f *SliceFinder[T]) Find(v T) bool {
	_, ok := f.Data[v]
	return ok
}

func (f *SliceFinder[T]) Clear() {
	f.Data = make(map[T]bool)
}

func (f *SliceFinder[T]) Len() int {
	return len(f.Data)
}
