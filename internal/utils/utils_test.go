package utils

import (
	"testing"
	"unsafe"

	"github.com/zeebo/assert"
	"github.com/zeebo/blake3/internal/consts"
)

func TestBytesToWords(t *testing.T) {
	if !consts.OptimizeLittleEndian {
		t.SkipNow()
	}

	var bytes [64]uint8
	for i := range bytes {
		bytes[i] = byte(i)
	}

	var words [16]uint32
	BytesToWords(&bytes, &words)

	assert.Equal(t, *(*[16]uint32)(unsafe.Pointer(&bytes[0])), words)
}
