package blake3

import (
	"errors"
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
			key: iv,
		},
	}
}

// NewSized returns a new Hasher with the given output size.
func NewSized(size int) *Hasher {
	if size < 0 {
		panic("must specify non-negative size")
	}
	return &Hasher{
		size: size,
		h: hasher{
			key: iv,
		},
	}
}

func NewKeyed(key []byte) (*Hasher, error) {
	if len(key) != 32 {
		return nil, errors.New("invalid key size")
	}
	h := &Hasher{
		size: 32,
		h: hasher{
			flags: flag_keyed,
		},
	}
	keyFromBytes(key, &h.h.key)
	return h, nil
}

// TODO NewKeyedSized

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
