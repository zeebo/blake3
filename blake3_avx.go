package blake3

import "unsafe"

type avxStackEntry struct {
	data   *[256]byte
	normed bool
}

type avxHasher struct {
	buf    [8192]byte
	len    uint64
	stack  []avxStackEntry
	chunks uint64
	flags  uint32
}

func (a *avxHasher) getOutputBuffer() (out avxStackEntry) {
	if cap(a.stack) > len(a.stack) {
		out = a.stack[:len(a.stack)+1][len(a.stack)]
	}
	if out.data == nil {
		out.data = new([256]byte)
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
		hashF_avx(input, 8192, a.chunks, a.flags, out.data)
		out.normed = false

		// update our state
		a.stack = append(a.stack, out)
		a.len = 0
		a.chunks += 8

		// greedily combine parents
		for chunks := a.chunks; chunks&15 == 0; chunks >>= 1 {
			hashP_avx(
				a.stack[len(a.stack)-1].data,
				a.stack[len(a.stack)-2].data,
				a.flags,
				a.stack[len(a.stack)-1].data,
			)
			a.stack = a.stack[:len(a.stack)-1]
		}
	}
}

var scrap [256]byte

func (a *avxHasher) finalize() {
	stack := make([][32]byte, len(a.stack))
	for i := range a.stack {
		entry := &a.stack[i]

		if !entry.normed {
			a.normalize(entry.data)
			entry.normed = true
		}

		type ptr = unsafe.Pointer
		*(*uint32)(ptr(&stack[i][0*4])) = *(*uint32)(ptr(&entry.data[0*32]))
		*(*uint32)(ptr(&stack[i][1*4])) = *(*uint32)(ptr(&entry.data[1*32]))
		*(*uint32)(ptr(&stack[i][2*4])) = *(*uint32)(ptr(&entry.data[2*32]))
		*(*uint32)(ptr(&stack[i][3*4])) = *(*uint32)(ptr(&entry.data[3*32]))
		*(*uint32)(ptr(&stack[i][4*4])) = *(*uint32)(ptr(&entry.data[4*32]))
		*(*uint32)(ptr(&stack[i][5*4])) = *(*uint32)(ptr(&entry.data[5*32]))
		*(*uint32)(ptr(&stack[i][6*4])) = *(*uint32)(ptr(&entry.data[6*32]))
		*(*uint32)(ptr(&stack[i][7*4])) = *(*uint32)(ptr(&entry.data[7*32]))
	}

	trailing := a.len / 1024
	chunks := a.chunks + trailing + 1

	var final [256]byte
	hashF_avx(&a.buf, a.len, a.chunks, a.flags, &final)

	// compress the final one? it's tricky. need at worst 3 copies i think.
	// maybe we can append in to the stack backwards?
}

func (a *avxHasher) normalize(in *[256]byte) {
	hashP_avx(in, &scrap, a.flags, in)
	hashP_avx(in, &scrap, a.flags, in)
	hashP_avx(in, &scrap, a.flags, in)
}
