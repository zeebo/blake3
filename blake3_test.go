package blake3

import (
	"encoding/hex"
	"testing"

	"github.com/zeebo/assert"
	"github.com/zeebo/blake3/internal/consts"
	"github.com/zeebo/blake3/internal/utils"
)

func TestHasher_Vectors(t *testing.T) {
	check := func(t *testing.T, h hasher, input []byte, hash string) {
		// ensure reset works
		h.update(input[:len(input)/2])
		h.reset()

		// write and finalize a bunch
		for i := range input {
			var tmp [32]byte
			h.update(input[i : i+1])
			switch i % 8193 {
			case 0, 1, 2:
				h.finalize(tmp[:])
			default:
			}
		}

		// check every output length requested
		for i := 0; i <= len(hash)/2; i++ {
			buf := make([]byte, i)
			h.finalize(buf)
			assert.Equal(t, hash[:2*i], hex.EncodeToString(buf))
		}

		// one more reset, full write, full read
		h.reset()
		h.update(input)
		buf := make([]byte, len(hash)/2)
		h.finalize(buf)
		assert.Equal(t, hash, hex.EncodeToString(buf))
	}

	t.Run("Basic", func(t *testing.T) {
		for _, tv := range vectors {
			h := hasher{key: consts.IV}
			check(t, h, tv.input(), tv.hash)
		}
	})

	t.Run("Keyed", func(t *testing.T) {
		for _, tv := range vectors {
			h := hasher{flags: consts.Flag_Keyed}
			utils.KeyFromBytes([]byte(testVectorKey), &h.key)
			check(t, h, tv.input(), tv.keyedHash)
		}
	})

	t.Run("DeriveKey", func(t *testing.T) {
		var buf [32]byte
		for _, tv := range vectors {
			h := hasher{flags: consts.Flag_DeriveKeyContext, key: consts.IV}
			h.updateString(testVectorContext)
			h.finalize(buf[:])
			h.reset()
			h.flags = consts.Flag_DeriveKeyMaterial
			utils.KeyFromBytes(buf[:], &h.key)
			check(t, h, tv.input(), tv.deriveKey)
		}
	})
}

func TestHasherAlignment(t *testing.T) {
	// On little endian architectures, we can do unaligned accesses of
	// uint32 values during the hashing. This test is designed to cause
	// those unaligned accesses to occur.

	var buf [32]byte

	x := make([]byte, 8194)
	for i := range x {
		x[i] = byte(i) % 251
	}

	h := hasher{key: consts.IV}
	h.update(x[1:])
	h.finalize(buf[:])

	assert.Equal(t,
		"981d32ed7aad9e408c5c36f6346c915ba11c2bd8b3e7d44902a11d7a141abdd9",
		hex.EncodeToString(buf[:]))
}
