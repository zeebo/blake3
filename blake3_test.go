package blake3

import (
	"encoding/hex"
	"io"
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

func TestVectors_DeriveKey(t *testing.T) {
	for _, tv := range vectors {
		// DeriveKey is implemented quite differently from the other
		// modes, it's basically a two-stage hash where the context is
		// hashed into an IV for the "real" hash. At this point, we
		// should have faith in the internal workings of hasher, so
		// test key derivation through the API.
		derived := make([]byte, hex.DecodedLen(len(tv.deriveKey)))
		DeriveKey(testVectorContext, tv.input(), derived)
		assert.Equal(t, hex.EncodeToString(derived), tv.deriveKey)
	}
}

func TestVectors_NewDeriveKey(t *testing.T) {
	for _, tv := range vectors {
		h := NewDeriveKey(testVectorContext)
		h.Write(tv.input())
		derived := make([]byte, hex.DecodedLen(len(tv.deriveKey)))
		xof := h.XOF()
		n, err := io.ReadFull(xof, derived)
		if g, e := n, len(derived); g != e {
			t.Errorf("wrong read length: %v != %v", g, e)
		}
		if err != nil {
			t.Errorf("error reading from hash: %v", err)
		}
		assert.Equal(t, hex.EncodeToString(derived), tv.deriveKey)
	}
}
