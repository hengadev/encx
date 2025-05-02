package encx

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsZeroPepper(t *testing.T) {
	tests := []struct {
		name   string
		pepper []byte
		want   bool
	}{
		{
			name:   "all zeros",
			pepper: []byte{0, 0, 0, 0},
			want:   true,
		},
		{
			name:   "single zero byte",
			pepper: []byte{0},
			want:   true,
		},
		{
			name:   "empty slice",
			pepper: []byte{},
			want:   true,
		},
		{
			name:   "non-zero at beginning",
			pepper: []byte{1, 0, 0, 0},
			want:   false,
		},
		{
			name:   "non-zero at middle",
			pepper: []byte{0, 0, 1, 0},
			want:   false,
		},
		{
			name:   "non-zero at end",
			pepper: []byte{0, 0, 0, 1},
			want:   false,
		},
		{
			name:   "all non-zero",
			pepper: []byte{1, 2, 3, 4},
			want:   false,
		},
		{
			name:   "nil slice",
			pepper: nil,
			want:   true,
		},
		{
			name:   "large zero slice",
			pepper: make([]byte, 1024), // all zeros
			want:   true,
		},
		{
			name:   "large non-zero slice",
			pepper: bytes.Repeat([]byte{1}, 1024),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isZeroPepper(tt.pepper))
		})
	}
}
