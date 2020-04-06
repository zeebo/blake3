package blake3

import (
	"errors"
	"io"

	"github.com/zeebo/blake3/internal/consts"
	"github.com/zeebo/blake3/internal/utils"
)

// Hasher is a hash.Hash for BLAKE3.
type Hasher struct {
	size int
	h    hasher
}

// New returns a new Hasher with the default output size (32 bytes).
func New() *Hasher {
	return &Hasher{
		size: 32,
		h: hasher{
			key: consts.IV,
		},
	}
}

// NewSized returns a new Hasher with the given output size.
func NewSized(size int) (*Hasher, error) {
	if size < 0 {
		return nil, errors.New("invalid output size")
	}

	return &Hasher{
		size: size,
		h: hasher{
			key: consts.IV,
		},
	}, nil
}

// NewKeyed returns a new Hasher that uses the 32 byte input key and default output size (32 bytes).
func NewKeyed(key []byte) (*Hasher, error) {
	if len(key) != 32 {
		return nil, errors.New("invalid key size")
	}

	h := &Hasher{
		size: 32,
		h: hasher{
			flags: consts.Flag_Keyed,
		},
	}
	utils.KeyFromBytes(key, &h.h.key)

	return h, nil
}

// NewKeyedSized returns a new Hasher that uses the 32 byte input key and given output size.
func NewKeyedSized(key []byte, size int) (*Hasher, error) {
	if size < 0 {
		return nil, errors.New("invalid output size")
	}
	if len(key) != 32 {
		return nil, errors.New("invalid key size")
	}

	h := &Hasher{
		size: size,
		h: hasher{
			flags: consts.Flag_Keyed,
		},
	}
	utils.KeyFromBytes(key, &h.h.key)

	return h, nil
}

// DeriveKey derives a key based on reusable key material of any
// length, in the given context. The key will be stored in out, using
// all of its current length.
//
// Context strings must be hardcoded
// constants, and the recommended format is "[application] [commit
// timestamp] [purpose]", e.g., "example.com 2019-12-25 16:18:03
// session tokens v1".
func DeriveKey(context string, material []byte, out []byte) {
	h := NewDeriveKey(context)
	_, _ = h.Write(material)
	_, _ = io.ReadFull(h.XOF(), out)
}

// NewDeriveKey returns  from key material written to Hasher. See
// DeriveKey.
func NewDeriveKey(context string) *Hasher {
	// hash the context string and use that instead of IV
	c := hasher{
		key:   consts.IV,
		flags: consts.Flag_DeriveKeyContext,
	}
	c.update([]byte(context))
	b := make([]byte, 32)
	c.finalize(b)

	h := &Hasher{
		size: 32,
		h: hasher{
			key:   consts.IV,
			flags: consts.Flag_DeriveKeyMaterial,
		},
	}
	utils.KeyFromBytes(b, &h.h.key)
	return h
}

// Write implements part of the hash.Hash interface. It never returns an error.
func (h *Hasher) Write(p []byte) (int, error) {
	h.h.update(p)
	return len(p), nil
}

// Reset implements part of the hash.Hash interface. It causes the Hasher to
// act as if it was newly created.
func (h *Hasher) Reset() {
	h.h.reset()
}

// Size implements part of the hash.Hash interface. It returns the number of
// bytes the hash will output.
func (h *Hasher) Size() int {
	return h.size
}

// BlockSize implements part of the hash.Hash interface. It returns the most
// natural size to write to the Hasher.
func (h *Hasher) BlockSize() int {
	// TODO: is there a downside to picking this large size?
	return 8192
}

// Sum implements part of the hash.Hash interface. It appends the digest of
// the Hasher to the provided buffer and returns it.
func (h *Hasher) Sum(b []byte) []byte {
	if top := len(b) + h.size; top <= cap(b) && top >= len(b) {
		h.h.finalize(b[len(b):top])
		return b[:top]
	}

	tmp := make([]byte, h.size)
	h.h.finalize(tmp)
	return append(b, tmp...)
}

// XOF returns an io.Reader containing 2^64 bytes of lazily generated hash output.
func (h *Hasher) XOF() *XOF {
	var x XOF
	h.h.finalizeOutput(&x)
	return &x
}
