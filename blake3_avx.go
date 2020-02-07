package blake3

import (
	"math/bits"
	"unsafe"
)

type avxHasher struct {
	len    uint64
	chunks uint64
	flags  uint32
	stack  avxStack
	buf    [8192]byte
}

func (a *avxHasher) reset() {
	a.len = 0
	a.chunks = 0
	a.stack.occ = 0
	a.stack.lvls = [8]uint8{}
	a.stack.bufn = 0
}

func (a *avxHasher) update(buf []byte) {
	var input *[8192]byte
	var chain [8]uint32
	var out chainVector

	for len(buf) > 0 {
		if a.len == 0 && len(buf) > 8192 {
			input = (*[8192]byte)(unsafe.Pointer(&buf[0]))
			buf = buf[8192:]
		} else if a.len < 8192 {
			n := copy(a.buf[a.len:], buf)
			a.len += uint64(n)
			buf = buf[n:]
			continue
		} else {
			input = &a.buf
		}

		hashF_avx(input, 8192, a.chunks, a.flags, &out, &chain)
		a.stack.pushN(0, &out, 8)
		a.len = 0
		a.chunks += 8
	}
}

func (a *avxHasher) finalize(out []byte) {
	if a.chunks == 0 && a.len <= 1024 {
		compressAll(a.buf[:a.len], out)
		return
	}

	o := output{
		chain:   &[8]uint32{iv0, iv1, iv2, iv3, iv4, iv5, iv6, iv7},
		counter: a.chunks,
		blen:    uint32(a.len) % 64,
		flags:   a.flags | flag_chunkEnd,
	}

	if a.len > 64 {
		var buf chainVector
		hashF_avx(&a.buf, a.len, a.chunks, a.flags, &buf, o.chain)

		if a.len > 1024 {
			complete := (a.len - 1) / 1024
			a.stack.pushN(0, &buf, int(complete))
			a.chunks += complete
			o.counter += complete
			a.len = uint64(copy(a.buf[:], a.buf[complete*1024:a.len]))
		}
	} else {
		o.flags |= flag_chunkStart
	}

	base := a.len / 64 * 64
	if a.len > 0 && o.blen == 0 {
		o.blen = 64
		base -= 64
	}

	var tmp [64]byte
	copy(tmp[:], a.buf[base:a.len])
	if isLittleEndian {
		o.block = (*[16]uint32)(unsafe.Pointer(&tmp[0]))
	} else {
		var block [16]uint32
		bytesToWords(&tmp, &block)
		o.block = &block
	}

	for a.stack.bufn > 0 {
		a.stack.flush()
	}

	key := [8]uint32{iv0, iv1, iv2, iv3, iv4, iv5, iv6, iv7}
	for occ := a.stack.occ; occ != 0; occ &= occ - 1 {
		col := bits.TrailingZeros64(occ)
		r := o.compress()
		copy(o.block[:], a.stack.stack[col&63][:])
		copy(o.block[8:], r[:])
		o.chain = &key
		o.counter = 0
		o.blen = blockLen
		o.flags = flag_parent
	}

	o.rootOutput(out)
}

type avxStack struct {
	occ   uint64   // which levels in stack are occupied
	lvls  [8]uint8 // what level the buf input was in
	bufn  int      // how many pairs are loaded into buf
	buf   [2]chainVector
	stack [64][8]uint32
}

func (a *avxStack) pushN(l uint8, cv *chainVector, n int) {
	for i := 0; i < n; i++ {
		a.pushL(l, cv, i)
		if a.bufn == 8 {
			a.flush()
		}
	}
}

func (a *avxStack) pushL(l uint8, cv *chainVector, n int) {
	bit := uint64(1) << (l & 63)
	if a.occ&bit == 0 {
		readChain(cv, n, &a.stack[l&63])
		a.occ ^= bit
		return
	}

	a.lvls[a.bufn&7] = l
	writeChain(&a.stack[l&63], &a.buf[0], a.bufn)
	copyChain(cv, n, &a.buf[1], a.bufn)
	a.bufn++
	a.occ ^= bit
}

func (a *avxStack) flush() {
	var out chainVector
	hashP_avx(&a.buf[0], &a.buf[1], 0, &out)

	bufn, lvls := a.bufn, a.lvls
	a.bufn, a.lvls = 0, [8]uint8{}

	for i := 0; i < bufn; i++ {
		a.pushL(lvls[i]+1, &out, i)
	}

	if a.bufn == 8 {
		a.flush()
	}
}

func copyChain(in *chainVector, icol int, out *chainVector, ocol int) {
	type u = uintptr
	type p = unsafe.Pointer
	type a = *uint32

	i := p(u(p(in)) + u(icol*4))
	o := p(u(p(out)) + u(ocol*4))

	*a(p(u(o) + 0*32)) = *a(p(u(i) + 0*32))
	*a(p(u(o) + 1*32)) = *a(p(u(i) + 1*32))
	*a(p(u(o) + 2*32)) = *a(p(u(i) + 2*32))
	*a(p(u(o) + 3*32)) = *a(p(u(i) + 3*32))
	*a(p(u(o) + 4*32)) = *a(p(u(i) + 4*32))
	*a(p(u(o) + 5*32)) = *a(p(u(i) + 5*32))
	*a(p(u(o) + 6*32)) = *a(p(u(i) + 6*32))
	*a(p(u(o) + 7*32)) = *a(p(u(i) + 7*32))
}

func readChain(in *chainVector, col int, out *[8]uint32) {
	type u = uintptr
	type p = unsafe.Pointer
	type a = *uint32

	i := p(u(p(in)) + u(col*4))

	out[0] = *a(p(u(i) + 0*32))
	out[1] = *a(p(u(i) + 1*32))
	out[2] = *a(p(u(i) + 2*32))
	out[3] = *a(p(u(i) + 3*32))
	out[4] = *a(p(u(i) + 4*32))
	out[5] = *a(p(u(i) + 5*32))
	out[6] = *a(p(u(i) + 6*32))
	out[7] = *a(p(u(i) + 7*32))
}

func writeChain(in *[8]uint32, out *chainVector, col int) {
	type u = uintptr
	type p = unsafe.Pointer
	type a = *uint32

	o := p(u(p(out)) + u(col*4))

	*a(p(u(o) + 0*32)) = in[0]
	*a(p(u(o) + 1*32)) = in[1]
	*a(p(u(o) + 2*32)) = in[2]
	*a(p(u(o) + 3*32)) = in[3]
	*a(p(u(o) + 4*32)) = in[4]
	*a(p(u(o) + 5*32)) = in[5]
	*a(p(u(o) + 6*32)) = in[6]
	*a(p(u(o) + 7*32)) = in[7]
}
