package blake3

import (
	"encoding/hex"
	"testing"
	"unsafe"

	"github.com/zeebo/assert"
)

func TestBytesToWords(t *testing.T) {
	if !isLittleEndian {
		t.SkipNow()
	}

	var bytes [64]uint8
	for i := range bytes {
		bytes[i] = byte(i)
	}

	var words [16]uint32
	bytesToWords(&bytes, &words)

	assert.Equal(t, *(*[16]uint32)(unsafe.Pointer(&bytes[0])), words)
}

func TestVectors(t *testing.T) {
	for _, tv := range vectors {
		h := hasher{key: iv}
		h.update(tv.input())
		for j := 0; j < len(tv.hash)/2; j++ {
			buf := make([]byte, j)
			h.finalize(buf)
			assert.Equal(t, tv.hash[:2*j], hex.EncodeToString(buf))
		}
	}
}

func TestVectorsKeyed(t *testing.T) {
	for _, tv := range vectors {
		h := hasher{
			flags: flag_keyed,
		}
		keyFromBytes([]byte(testVectorKey), &h.key)
		h.update(tv.input())
		for j := 0; j < len(tv.hash)/2; j++ {
			buf := make([]byte, j)
			h.finalize(buf)
			assert.Equal(t, tv.keyedHash[:2*j], hex.EncodeToString(buf))
		}
	}
}

func TestVectors_Finalize(t *testing.T) {
	var buf [32]byte
	for _, tv := range vectors {
		h := hasher{key: iv}
		for i, v := range tv.input() {
			h.update([]byte{v})
			if i%11 == 0 {
				h.finalize(buf[:])
			}
		}
		buf := make([]byte, len(tv.hash)/2)
		h.finalize(buf)
		assert.Equal(t, tv.hash, hex.EncodeToString(buf))
	}
}
