package blake3

import (
	"math/bits"
	"unsafe"

	"github.com/zeebo/blake3/internal/alg"
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
	a.updateString(unsafe.String(unsafe.SliceData(buf), len(buf)))
}

func (a *hasher) updateString(buf string) {
	var input *[8192]byte

	for len(buf) > 0 {
		if a.len == 0 && len(buf) > 8192 {
			input = (*[8192]byte)(unsafe.Slice(unsafe.StringData(buf), len(buf)))
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
	alg.HashF(input, 8192, a.chunks, a.flags, &a.key, &out, &chain)
	a.stack.pushN(0, &out, 8, a.flags, &a.key)
}

func (a *hasher) finalize(p []byte) {
	var d Digest
	a.finalizeDigest(&d)
	_, _ = d.Read(p)
}

func (a *hasher) finalizeDigest(d *Digest) {
	if a.chunks == 0 && a.len <= consts.ChunkLen {
		compressAll(d, a.buf[:a.len], a.flags, a.key)
		return
	}

	d.chain = a.key
	d.flags = a.flags | consts.Flag_ChunkEnd

	if a.len > 64 {
		var buf chainVector
		alg.HashF(&a.buf, a.len, a.chunks, a.flags, &a.key, &buf, &d.chain)

		if a.len > consts.ChunkLen {
			complete := (a.len - 1) / consts.ChunkLen
			a.stack.pushN(0, &buf, int(complete), a.flags, &a.key)
			a.chunks += complete
			a.len = uint64(copy(a.buf[:], a.buf[complete*consts.ChunkLen:a.len]))
		}
	}

	if a.len <= 64 {
		d.flags |= consts.Flag_ChunkStart
	}

	d.counter = a.chunks
	d.blen = uint32(a.len) % 64

	base := a.len / 64 * 64
	if a.len > 0 && d.blen == 0 {
		d.blen = 64
		base -= 64
	}

	if consts.OptimizeLittleEndian {
		copy((*[64]byte)(unsafe.Pointer(&d.block[0]))[:], a.buf[base:a.len])
	} else {
		var tmp [64]byte
		copy(tmp[:], a.buf[base:a.len])
		utils.BytesToWords(&tmp, &d.block)
	}

	for a.stack.bufn > 0 {
		a.stack.flush(a.flags, &a.key)
	}

	var tmp [16]uint32
	for occ := a.stack.occ; occ != 0; occ &= occ - 1 {
		col := uint(bits.TrailingZeros64(occ)) % 64

		alg.Compress(&d.chain, &d.block, d.counter, d.blen, d.flags, &tmp)

		*(*[8]uint32)(d.block[0:8]) = a.stack.stack[col]
		*(*[8]uint32)(d.block[8:16]) = *(*[8]uint32)(tmp[0:8])

		if occ == a.stack.occ {
			d.chain = a.key
			d.counter = 0
			d.blen = consts.BlockLen
			d.flags = a.flags | consts.Flag_Parent
		}
	}

	d.flags |= consts.Flag_Root
}

//
// chain value stack
//

// A chainVector holds eight transposed chains:
// word w of chain c is at index c + w*8.
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
	alg.HashP(&a.buf[0], &a.buf[1], flags|consts.Flag_Parent, key, &out, a.bufn)

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
	// bounds check hint to compiler
	icol &= 7
	ocol &= 7

	out[ocol+0*8] = in[icol+0*8]
	out[ocol+1*8] = in[icol+1*8]
	out[ocol+2*8] = in[icol+2*8]
	out[ocol+3*8] = in[icol+3*8]
	out[ocol+4*8] = in[icol+4*8]
	out[ocol+5*8] = in[icol+5*8]
	out[ocol+6*8] = in[icol+6*8]
	out[ocol+7*8] = in[icol+7*8]
}

func readChain(in *chainVector, col int, out *[8]uint32) {
	// bounds check hint to compiler
	col &= 7

	out[0] = in[col+0*8]
	out[1] = in[col+1*8]
	out[2] = in[col+2*8]
	out[3] = in[col+3*8]
	out[4] = in[col+4*8]
	out[5] = in[col+5*8]
	out[6] = in[col+6*8]
	out[7] = in[col+7*8]
}

func writeChain(in *[8]uint32, out *chainVector, col int) {
	// bounds check hint to compiler
	col &= 7

	out[col+0*8] = in[0]
	out[col+1*8] = in[1]
	out[col+2*8] = in[2]
	out[col+3*8] = in[3]
	out[col+4*8] = in[4]
	out[col+5*8] = in[5]
	out[col+6*8] = in[6]
	out[col+7*8] = in[7]
}

//
// compress <= chunkLen bytes in one shot
//

func compressAll(d *Digest, in []byte, flags uint32, key [8]uint32) {
	var compressed [16]uint32

	d.chain = key
	d.flags = flags | consts.Flag_ChunkStart

	for len(in) > 64 {
		buf := (*[64]byte)(in)

		var block *[16]uint32
		if consts.OptimizeLittleEndian {
			block = (*[16]uint32)(unsafe.Pointer(buf))
		} else {
			block = &d.block
			utils.BytesToWords(buf, block)
		}

		alg.Compress(&d.chain, block, 0, consts.BlockLen, d.flags, &compressed)

		d.chain = *(*[8]uint32)(compressed[0:8])
		d.flags &^= consts.Flag_ChunkStart

		in = in[64:]
	}

	if consts.OptimizeLittleEndian {
		copy((*[64]byte)(unsafe.Pointer(&d.block[0]))[:], in)
	} else {
		var tmp [64]byte
		copy(tmp[:], in)
		utils.BytesToWords(&tmp, &d.block)
	}

	d.blen = uint32(len(in))
	d.flags |= consts.Flag_ChunkEnd | consts.Flag_Root
}
