package blake3

import (
	"math/bits"
	"unsafe"

	"github.com/zeebo/blake3/internal/consts"
	"github.com/zeebo/blake3/internal/utils"
)

//
// hasher contains state for a blake3 hash
//

type hasher struct {
	len    uint64
	chunks uint64
	flags  uint32
	key    [8]uint32
	stack  cvstack
	buf    [8192]byte
}

func (a *hasher) reset() {
	a.len = 0
	a.chunks = 0
	a.stack.occ = 0
	a.stack.lvls = [8]uint8{}
	a.stack.bufn = 0
}

func (a *hasher) update(buf []byte) {
	var input *[8192]byte

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

		a.consume(input)
		a.len = 0
		a.chunks += 8
	}
}

func (a *hasher) consume(input *[8192]byte) {
	var out chainVector
	var chain [8]uint32
	hashF(input, 8192, a.chunks, a.flags, &a.key, &out, &chain)
	a.stack.pushN(0, &out, 8, a.flags, &a.key)
}

func (a *hasher) finalize(out []byte) {
	var x XOF
	a.finalizeOutput(&x)
	_, _ = x.Read(out)
}

func (a *hasher) finalizeOutput(x *XOF) {
	if a.chunks == 0 && a.len <= consts.ChunkLen {
		compressAll(x, a.buf[:a.len], a.flags, a.key)
		return
	}

	x.chain = a.key
	x.flags = a.flags | consts.Flag_ChunkEnd

	if a.len > 64 {
		var buf chainVector
		if a.len <= 2*consts.ChunkLen {
			hashFSmall(&a.buf, a.len, a.chunks, a.flags, &a.key, &buf, &x.chain)
		} else {
			hashF(&a.buf, a.len, a.chunks, a.flags, &a.key, &buf, &x.chain)
		}

		if a.len > consts.ChunkLen {
			complete := (a.len - 1) / consts.ChunkLen
			a.stack.pushN(0, &buf, int(complete), a.flags, &a.key)
			a.chunks += complete
			a.len = uint64(copy(a.buf[:], a.buf[complete*consts.ChunkLen:a.len]))
		}
	}

	if a.len <= 64 {
		x.flags |= consts.Flag_ChunkStart
	}

	x.counter = a.chunks
	x.blen = uint32(a.len) % 64

	base := a.len / 64 * 64
	if a.len > 0 && x.blen == 0 {
		x.blen = 64
		base -= 64
	}

	if consts.IsLittleEndian {
		copy((*[64]byte)(unsafe.Pointer(&x.block[0]))[:], a.buf[base:a.len])
	} else {
		var tmp [64]byte
		copy(tmp[:], a.buf[base:a.len])
		utils.BytesToWords(&tmp, &x.block)
	}

	for a.stack.bufn > 0 {
		a.stack.flush(a.flags, &a.key)
	}

	var tmp [16]uint32
	for occ := a.stack.occ; occ != 0; occ &= occ - 1 {
		col := uint(bits.TrailingZeros64(occ)) % 64

		compress(&x.chain, &x.block, x.counter, x.blen, x.flags, &tmp)

		*(*[8]uint32)(unsafe.Pointer(&x.block[0])) = a.stack.stack[col]
		*(*[8]uint32)(unsafe.Pointer(&x.block[8])) = *(*[8]uint32)(unsafe.Pointer(&tmp[0]))

		if occ == a.stack.occ {
			x.chain = a.key
			x.counter = 0
			x.blen = consts.BlockLen
			x.flags = a.flags | consts.Flag_Parent
		}
	}

	x.flags |= consts.Flag_Root
}

//
// chain value stack
//

type chainVector = [64]uint32

type cvstack struct {
	occ   uint64   // which levels in stack are occupied
	lvls  [8]uint8 // what level the buf input was in
	bufn  int      // how many pairs are loaded into buf
	buf   [2]chainVector
	stack [64][8]uint32
}

func (a *cvstack) pushN(l uint8, cv *chainVector, n int, flags uint32, key *[8]uint32) {
	for i := 0; i < n; i++ {
		a.pushL(l, cv, i)
		for a.bufn == 8 {
			a.flush(flags, key)
		}
	}
}

func (a *cvstack) pushL(l uint8, cv *chainVector, n int) {
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

func (a *cvstack) flush(flags uint32, key *[8]uint32) {
	var out chainVector
	if a.bufn < 2 {
		hashPSmall(&a.buf[0], &a.buf[1], flags|consts.Flag_Parent, key, &out, a.bufn)
	} else {
		hashP(&a.buf[0], &a.buf[1], flags|consts.Flag_Parent, key, &out, a.bufn)
	}

	bufn, lvls := a.bufn, a.lvls
	a.bufn, a.lvls = 0, [8]uint8{}

	for i := 0; i < bufn; i++ {
		a.pushL(lvls[i]+1, &out, i)
	}
}

//
// helpers to deal with reading/writing transposed values
//

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

//
// compress <= chunkLen bytes in one shot
//

func compressAll(x *XOF, in []byte, flags uint32, key [8]uint32) {
	var compressed [16]uint32

	x.chain = key
	x.flags = flags | consts.Flag_ChunkStart

	for len(in) > 64 {
		buf := (*[64]byte)(unsafe.Pointer(&in[0]))

		var block *[16]uint32
		if consts.IsLittleEndian {
			block = (*[16]uint32)(unsafe.Pointer(buf))
		} else {
			block = &x.block
			utils.BytesToWords(buf, block)
		}

		compress(&x.chain, block, 0, consts.BlockLen, x.flags, &compressed)

		x.chain = *(*[8]uint32)(unsafe.Pointer(&compressed[0]))
		x.flags &^= consts.Flag_ChunkStart

		in = in[64:]
	}

	if consts.IsLittleEndian {
		copy((*[64]byte)(unsafe.Pointer(&x.block[0]))[:], in)
	} else {
		var tmp [64]byte
		copy(tmp[:], in)
		utils.BytesToWords(&tmp, &x.block)
	}

	x.blen = uint32(len(in))
	x.flags |= consts.Flag_ChunkEnd | consts.Flag_Root
}
