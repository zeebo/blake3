package blake3

import (
	"unsafe"
)

type avxHasher struct {
	buf    [8192]byte
	len    uint64
	stack  []*cv
	chunks uint64
	flags  uint32
}

func (a *avxHasher) getOutputBuffer() (out *cv) {
	if cap(a.stack) > len(a.stack) {
		out = a.stack[:len(a.stack)+1][len(a.stack)]
	}
	if out == nil {
		out = new(cv)
	}
	return out
}

func (a *avxHasher) update(buf []byte) {
	var input *[8192]byte

	for len(buf) > 0 {
		if a.len == 0 && len(buf) >= 8*1024 {
			// consume 8k directly with no memcpy if possible
			a.len = 8 * 1024
			input = (*[8192]byte)(unsafe.Pointer(&buf[0]))
			buf = buf[8*1024:]

		} else {
			// otherwise, copy into the buffer
			n := copy(a.buf[a.len:], buf)
			a.len += uint64(n)
			buf = buf[n:]
			input = &a.buf
		}

		if a.len != 8192 {
			continue
		}

		// allocate or reuse an output buffer
		out := a.getOutputBuffer()
		hashF_avx(input, 8192, a.chunks, a.flags, out)

		// update our state
		a.stack = append(a.stack, out)
		a.len = 0
		a.chunks += 8

		// greedily combine parents
		for chunks := a.chunks; chunks&15 == 0; chunks >>= 1 {
			top0 := a.stack[len(a.stack)-1]
			top1 := a.stack[len(a.stack)-2]
			hashP_avx(top0, top1, a.flags, top1)
			a.stack = a.stack[:len(a.stack)-1]
		}
	}
}

var scrap cv

func (a *avxHasher) finalize(out []byte) {

}
