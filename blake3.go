package blake3

import (
	"encoding/binary"
	"math/bits"
	"unsafe"
)

func keyFromBytes(key []byte, out *[8]uint32) {
	out[0] = binary.LittleEndian.Uint32(key[0:])
	out[1] = binary.LittleEndian.Uint32(key[4:])
	out[2] = binary.LittleEndian.Uint32(key[8:])
	out[3] = binary.LittleEndian.Uint32(key[12:])
	out[4] = binary.LittleEndian.Uint32(key[16:])
	out[5] = binary.LittleEndian.Uint32(key[20:])
	out[6] = binary.LittleEndian.Uint32(key[24:])
	out[7] = binary.LittleEndian.Uint32(key[28:])
}

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
	if a.chunks == 0 && a.len <= chunkLen {
		compressAll(a.buf[:a.len], a.flags, &a.key, out)
		return
	}

	tmpChain := a.key
	chain := &tmpChain
	flags := a.flags | flag_chunkEnd

	if a.len > 64 {
		var buf chainVector
		if a.len <= 2*chunkLen {
			hashFSmall(&a.buf, a.len, a.chunks, a.flags, &a.key, &buf, chain)
		} else {
			hashF(&a.buf, a.len, a.chunks, a.flags, &a.key, &buf, chain)
		}

		if a.len > chunkLen {
			complete := (a.len - 1) / chunkLen
			a.stack.pushN(0, &buf, int(complete), a.flags, &a.key)
			a.chunks += complete
			a.len = uint64(copy(a.buf[:], a.buf[complete*chunkLen:a.len]))
		}
	}

	if a.len <= 64 {
		flags |= flag_chunkStart
	}

	var blockPtr *[16]uint32
	counter := a.chunks
	blen := uint32(a.len) % 64

	base := a.len / 64 * 64
	if a.len > 0 && blen == 0 {
		blen = 64
		base -= 64
	}

	{
		var tmp [64]byte
		copy(tmp[:], a.buf[base:a.len])

		if isLittleEndian {
			blockPtr = (*[16]uint32)(unsafe.Pointer(&tmp[0]))
		} else {
			var block [16]uint32
			bytesToWords(&tmp, &block)
			blockPtr = &block
		}
	}

	for a.stack.bufn > 0 {
		a.stack.flush(a.flags, &a.key)
	}

	var tmp [16]uint32
	for occ := a.stack.occ; occ != 0; occ &= occ - 1 {
		col := bits.TrailingZeros64(occ)

		compress(chain, blockPtr, counter, blen, flags, &tmp)

		*(*[8]uint32)(unsafe.Pointer(&blockPtr[0])) = a.stack.stack[col&63]
		*(*[8]uint32)(unsafe.Pointer(&blockPtr[8])) = *(*[8]uint32)(unsafe.Pointer(&tmp[0]))

		chain = &a.key
		counter = 0
		blen = blockLen
		flags = a.flags | flag_parent
	}

	writeOutput(out, chain, blockPtr, blen, flags|flag_root)
}

//
// chain value stack
//

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
		hashPSmall(&a.buf[0], &a.buf[1], flags|flag_parent, key, &out, a.bufn)
	} else {
		hashP(&a.buf[0], &a.buf[1], flags|flag_parent, key, &out, a.bufn)
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

func compressAll(in []byte, flags uint32, key *[8]uint32, out []byte) {
	chain := *key

	flags |= flag_chunkStart

	for len(in) > 64 {
		buf := (*[64]byte)(unsafe.Pointer(&in[0]))

		var blockPtr *[16]uint32
		if !isLittleEndian {
			var block [16]uint32
			blockPtr = &block
			bytesToWords(buf, blockPtr)
		} else {
			blockPtr = (*[16]uint32)(unsafe.Pointer(buf))
		}

		var compressed [16]uint32
		compress(&chain, blockPtr, 0, blockLen, flags, &compressed)

		chain = *(*[8]uint32)(unsafe.Pointer(&compressed[0]))
		in = in[64:]
		flags &^= flag_chunkStart
	}

	var fblock [16]uint32
	blockPtr := &fblock

	if isLittleEndian {
		copy((*[64]byte)(unsafe.Pointer(&fblock[0]))[:], in)
	} else {
		in := in
		for i := 0; i < 16; i++ {
			if len(in) > 4 {
				fblock[i] = binary.LittleEndian.Uint32(in[0:4])
				in = in[4:]
				continue
			}
			var tmp [4]byte
			copy(tmp[:], in)
			fblock[i] = binary.LittleEndian.Uint32(tmp[0:4])
			break
		}
	}

	writeOutput(out, &chain, blockPtr, uint32(len(in)), flags|flag_chunkEnd|flag_root)
}

//
// writes for some given root state
//

func writeOutput(out []byte, chain *[8]uint32, block *[16]uint32, blen uint32, flags uint32) {
	var counter uint64
	var buf [16]uint32

	for len(out) >= 64 {
		compress(chain, block, counter, blen, flags, &buf)

		if isLittleEndian {
			*(*[64]byte)(unsafe.Pointer(&out[0])) = *(*[64]byte)(unsafe.Pointer(&buf[0]))
		} else {
			binary.LittleEndian.PutUint32(out[0*4:1*4], buf[0])
			binary.LittleEndian.PutUint32(out[1*4:2*4], buf[1])
			binary.LittleEndian.PutUint32(out[2*4:3*4], buf[2])
			binary.LittleEndian.PutUint32(out[3*4:4*4], buf[3])
			binary.LittleEndian.PutUint32(out[4*4:5*4], buf[4])
			binary.LittleEndian.PutUint32(out[5*4:6*4], buf[5])
			binary.LittleEndian.PutUint32(out[6*4:7*4], buf[6])
			binary.LittleEndian.PutUint32(out[7*4:8*4], buf[7])
			binary.LittleEndian.PutUint32(out[8*4:9*4], buf[8])
			binary.LittleEndian.PutUint32(out[9*4:10*4], buf[9])
			binary.LittleEndian.PutUint32(out[10*4:11*4], buf[10])
			binary.LittleEndian.PutUint32(out[11*4:12*4], buf[11])
			binary.LittleEndian.PutUint32(out[12*4:13*4], buf[12])
			binary.LittleEndian.PutUint32(out[13*4:14*4], buf[13])
			binary.LittleEndian.PutUint32(out[14*4:15*4], buf[14])
			binary.LittleEndian.PutUint32(out[15*4:16*4], buf[15])
		}

		counter++
		out = out[64:]
	}

	if len(out) == 0 {
		return
	}

	compress(chain, block, counter, blen, flags, &buf)

	if isLittleEndian {
		copy(out, (*[64]byte)(unsafe.Pointer(&buf[0]))[:])
		return
	}

	for i := 0; i < 16; i++ {
		if len(out) > 4 {
			binary.LittleEndian.PutUint32(out[0:4], buf[i])
			out = out[4:]
			continue
		}

		var tmp [4]byte
		binary.LittleEndian.PutUint32(tmp[:], buf[i])
		copy(out[:], tmp[:])
		return
	}
}
