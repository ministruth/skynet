package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlicePagination(t *testing.T) {
	type args struct {
		s    []int
		page int
		size int
	}
	tests := []struct {
		name string
		args args
		want []int
	}{
		{
			name: "Slice pagination 1",
			args: args{
				s:    []int{1, 2, 3, 4},
				page: 0,
				size: 2,
			},
			want: []int{1, 2},
		},
		{
			name: "Slice pagination 2",
			args: args{
				s:    []int{1, 2, 3, 4},
				page: 1,
				size: 2,
			},
			want: []int{1, 2},
		},
		{
			name: "Slice pagination 3",
			args: args{
				s:    []int{1, 2, 3, 4},
				page: 1,
				size: 10,
			},
			want: []int{1, 2, 3, 4},
		},
		{
			name: "Slice pagination 4",
			args: args{
				s:    []int{1, 2, 3, 4},
				page: 2,
				size: 10,
			},
			want: []int{},
		},
		{
			name: "Slice pagination 5",
			args: args{
				s:    []int{1, 2, 3, 4},
				page: 2,
				size: 2,
			},
			want: []int{3, 4},
		},
		{
			name: "Slice pagination 6",
			args: args{
				s:    []int{1, 2, 3, 4},
				page: 3,
				size: 2,
			},
			want: []int{},
		},
		{
			name: "Slice pagination 7",
			args: args{
				s:    []int{1, 2, 3, 4},
				page: 2,
				size: 3,
			},
			want: []int{4},
		},
		{
			name: "Slice pagination 8",
			args: args{
				s:    []int{1, 2, 3, 4},
				page: 2,
				size: 4,
			},
			want: []int{},
		},
		{
			name: "Slice pagination 9",
			args: args{
				s:    []int{1, 2, 3, 4},
				page: 10,
				size: 10,
			},
			want: []int{},
		},
		{
			name: "Slice pagination 10",
			args: args{
				s:    []int{1, 2, 3, 4},
				page: 2,
				size: -1,
			},
			want: []int{2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SlicePagination(tt.args.s, tt.args.page, tt.args.size)
			assert.Equal(t, tt.want, got)
		})
	}
}
