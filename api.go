package blake3

type Hasher struct {
	h    hasher
	size int
}

func New() *Hasher { return &Hasher{size: 32} }

func NewSized(size int) *Hasher { return &Hasher{size: size} }

func (h *Hasher) Write(p []byte) (int, error) {
	h.h.update(p)
	return len(p), nil
}

func (h *Hasher) Reset() {
	h.h.reset()
}

func (h *Hasher) Size() int {
	return h.size
}

func (h *Hasher) BlockSize() int {
	// TODO: is there a downside to picking this large size?
	return 8192
}

func (h *Hasher) Sum(b []byte) []byte {
	if top := uint(len(b)) + uint(h.size); uint(top) <= uint(cap(b)) {
		h.h.finalize(b[len(b):top])
		return b[:top]
	}

	tmp := make([]byte, h.size)
	h.h.finalize(tmp)
	return append(b, tmp...)
}
