package blake3

import (
	"encoding/hex"
	"testing"

	"github.com/zeebo/assert"
	"github.com/zeebo/blake3/internal/consts"
	"github.com/zeebo/blake3/internal/utils"
)

func TestVectors_Hash(t *testing.T) {
	for _, tv := range vectors {
		h := hasher{
			key: consts.IV,
		}

		var buf [32]byte
		for i, v := range tv.input() {
			h.update([]byte{v})
			if i%11 == 0 {
				h.finalize(buf[:])
			}
		}

		for j := 0; j < len(tv.hash)/2; j++ {
			buf := make([]byte, j)
			h.finalize(buf)
			assert.Equal(t, tv.hash[:2*j], hex.EncodeToString(buf))
		}
	}
}

func TestVectors_KeyedHash(t *testing.T) {
	for _, tv := range vectors {
		h := hasher{flags: consts.Flag_Keyed}
		utils.KeyFromBytes([]byte(testVectorKey), &h.key)

		var buf [32]byte
		for i, v := range tv.input() {
			h.update([]byte{v})
			if i%11 == 0 {
				h.finalize(buf[:])
			}
		}

		for j := 0; j < len(tv.hash)/2; j++ {
			buf := make([]byte, j)
			h.finalize(buf)
			assert.Equal(t, tv.keyedHash[:2*j], hex.EncodeToString(buf))
		}
	}
}
