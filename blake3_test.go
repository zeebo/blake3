package blake3

import (
	"encoding/hex"
	"testing"

	"github.com/zeebo/assert"
	"github.com/zeebo/blake3/internal/consts"
	"github.com/zeebo/blake3/internal/utils"
)

func testHasher(t *testing.T, h hasher, input []byte, hash string) {
	// ensure reset works
	h.update(input[:len(input)/2])
	h.reset()

	// write and finalize a bunch
	var buf [32]byte
	for i := range input {
		h.update(input[i : i+1])
		if i%8193 == 0 {
			h.finalize(buf[:])
		}
	}

	// check every output length requested
	for i := 0; i <= len(hash)/2; i++ {
		buf := make([]byte, i)
		h.finalize(buf)
		assert.Equal(t, hash[:2*i], hex.EncodeToString(buf))
	}
}

func TestVectors_Hash(t *testing.T) {
	for _, tv := range vectors {
		h := hasher{key: consts.IV}
		testHasher(t, h, tv.input(), tv.hash)
	}
}

func TestVectors_KeyedHash(t *testing.T) {
	for _, tv := range vectors {
		h := hasher{flags: consts.Flag_Keyed}
		utils.KeyFromBytes([]byte(testVectorKey), &h.key)
		testHasher(t, h, tv.input(), tv.keyedHash)
	}
}
