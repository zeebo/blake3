package blake3

import (
	"encoding/hex"
	"testing"

	"github.com/zeebo/assert"
)

func TestAPI(t *testing.T) {
	h := NewSized(8)

	assert.Equal(t, hex.EncodeToString(h.Sum(nil)), "af1349b9f5f9a1a6")
	for i := 0; i < 16; i++ {
		assert.Equal(t, hex.EncodeToString(h.Sum(make([]byte, i)[:0])), "af1349b9f5f9a1a6")
	}
	assert.Equal(t, hex.EncodeToString(h.Sum(make([]byte, 1))), "00af1349b9f5f9a1a6")
}
