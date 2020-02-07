package blake3

import (
	"encoding/binary"
	"unsafe"
)

const (
	iv0 = 0x6A09E667
	iv1 = 0xBB67AE85
	iv2 = 0x3C6EF372
	iv3 = 0xA54FF53A
	iv4 = 0x510E527F
	iv5 = 0x9B05688C
	iv6 = 0x1F83D9AB
	iv7 = 0x5BE0CD19
)

const (
	flag_chunkStart uint32 = 1 << 0
	flag_chunkEnd   uint32 = 1 << 1
	flag_parent     uint32 = 1 << 2
	flag_root       uint32 = 1 << 3
	flag_keyed      uint32 = 1 << 4
	flag_keyCtx     uint32 = 1 << 5
	flag_keyMat     uint32 = 1 << 6
)

const (
	blockLen = 64
	chunkLen = 1024
)

//
// helpers
//

var isLittleEndian = *(*uint32)(unsafe.Pointer(&[4]byte{0, 0, 0, 1})) != 1

func bytesToWords(bytes *[64]uint8, words *[16]uint32) {
	words[0] = binary.LittleEndian.Uint32(bytes[0*4:])
	words[1] = binary.LittleEndian.Uint32(bytes[1*4:])
	words[2] = binary.LittleEndian.Uint32(bytes[2*4:])
	words[3] = binary.LittleEndian.Uint32(bytes[3*4:])
	words[4] = binary.LittleEndian.Uint32(bytes[4*4:])
	words[5] = binary.LittleEndian.Uint32(bytes[5*4:])
	words[6] = binary.LittleEndian.Uint32(bytes[6*4:])
	words[7] = binary.LittleEndian.Uint32(bytes[7*4:])
	words[8] = binary.LittleEndian.Uint32(bytes[8*4:])
	words[9] = binary.LittleEndian.Uint32(bytes[9*4:])
	words[10] = binary.LittleEndian.Uint32(bytes[10*4:])
	words[11] = binary.LittleEndian.Uint32(bytes[11*4:])
	words[12] = binary.LittleEndian.Uint32(bytes[12*4:])
	words[13] = binary.LittleEndian.Uint32(bytes[13*4:])
	words[14] = binary.LittleEndian.Uint32(bytes[14*4:])
	words[15] = binary.LittleEndian.Uint32(bytes[15*4:])
}

func first8words(in [16]uint32) [8]uint32 {
	return [8]uint32{
		in[0], in[1], in[2], in[3],
		in[4], in[5], in[6], in[7],
	}
}

//
// output
//

type output struct {
	chain   *[8]uint32
	block   *[16]uint32
	counter uint64
	blen    uint32
	flags   uint32
}

func (o *output) compress() [8]uint32 {
	var buf [16]uint32
	compress(o.chain, o.block, o.counter, o.blen, o.flags, &buf)
	return first8words(buf)
}

func (o *output) rootOutput(out []byte) {
	var block [16]uint32
	var counter uint64

	for len(out) > 64 {
		compress(
			o.chain,
			o.block,
			counter,
			o.blen,
			o.flags|flag_root,
			&block,
		)

		if isLittleEndian {
			copy(out, (*[64]byte)(unsafe.Pointer(&block[0]))[:])
		} else {
			binary.LittleEndian.PutUint32(out[0*4:], block[0])
			binary.LittleEndian.PutUint32(out[1*4:], block[1])
			binary.LittleEndian.PutUint32(out[2*4:], block[2])
			binary.LittleEndian.PutUint32(out[3*4:], block[3])
			binary.LittleEndian.PutUint32(out[4*4:], block[4])
			binary.LittleEndian.PutUint32(out[5*4:], block[5])
			binary.LittleEndian.PutUint32(out[6*4:], block[6])
			binary.LittleEndian.PutUint32(out[7*4:], block[7])
			binary.LittleEndian.PutUint32(out[8*4:], block[8])
			binary.LittleEndian.PutUint32(out[9*4:], block[9])
			binary.LittleEndian.PutUint32(out[10*4:], block[10])
			binary.LittleEndian.PutUint32(out[11*4:], block[11])
			binary.LittleEndian.PutUint32(out[12*4:], block[12])
			binary.LittleEndian.PutUint32(out[13*4:], block[13])
			binary.LittleEndian.PutUint32(out[14*4:], block[14])
			binary.LittleEndian.PutUint32(out[15*4:], block[15])
		}

		counter++
		out = out[64:]
	}

	if len(out) == 0 {
		return
	}

	compress(
		o.chain,
		o.block,
		counter,
		o.blen,
		o.flags|flag_root,
		&block,
	)

	if isLittleEndian {
		copy(out, (*[64]byte)(unsafe.Pointer(&block[0]))[:])
		return
	}

	for i := 0; i < 16; i++ {
		if len(out) > 4 {
			binary.LittleEndian.PutUint32(out[0:4], block[i])
			out = out[4:]
			continue
		}

		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], block[i])
		copy(out[:], buf[:])
		return
	}
}

//
// compress <= 1024 bytes in one shot
//

func compressAll(in, out []byte) {
	chain := [8]uint32{iv0, iv1, iv2, iv3, iv4, iv5, iv6, iv7}
	var compressed [16]uint32
	var block [16]uint32
	var blockPtr *[16]uint32
	flags := flag_chunkStart

	for len(in) > 64 {
		buf := (*[64]byte)(unsafe.Pointer(&in[0]))

		if !isLittleEndian {
			blockPtr = &block
			bytesToWords(buf, blockPtr)
		} else {
			blockPtr = (*[16]uint32)(unsafe.Pointer(buf))
		}

		compress(&chain, blockPtr, 0, blockLen, flags, &compressed)
		copy(chain[:], compressed[:8])
		in = in[64:]
		flags = 0
	}

	var fblock [16]uint32
	copy((*[64]byte)(unsafe.Pointer(&fblock[0]))[:], in)

	op := output{
		chain:   &chain,
		block:   &fblock,
		blen:    uint32(len(in)),
		counter: 0,
		flags:   flags | flag_chunkEnd,
	}

	op.rootOutput(out)
}

//
// chunk state
//

type chunkState struct {
	chain   [8]uint32
	counter uint64
	block   [blockLen]uint8
	flags   uint32
	blen    uint
	blocks  uint
}

func newChunkState(chain [8]uint32, counter uint64, flags uint32) chunkState {
	return chunkState{
		chain:   chain,
		counter: counter,
		flags:   flags,
	}
}

func (c *chunkState) len() uint {
	return 64*c.blocks + c.blen
}

func (c *chunkState) startFlag() uint32 {
	if c.blocks == 0 {
		return flag_chunkStart
	}
	return 0
}

func (c *chunkState) update(input []byte) {
	var blockBuf *[blockLen]byte
	var block [16]uint32
	var blockPtr *[16]uint32

	for len(input) > 0 {
		if c.blen == 0 && len(input) > blockLen {
			blockBuf = (*[blockLen]byte)(unsafe.Pointer(&input[0]))
			input = input[blockLen:]
		} else if c.blen < blockLen {
			n := uint(copy(c.block[c.blen:], input))
			c.blen += n
			input = input[n:]
			continue
		} else {
			blockBuf = &c.block
		}

		if !isLittleEndian {
			blockPtr = &block
			bytesToWords(blockBuf, blockPtr)
		} else {
			blockPtr = (*[16]uint32)(unsafe.Pointer(blockBuf))
		}

		var buf [16]uint32
		compress(
			&c.chain,
			blockPtr,
			c.counter,
			blockLen,
			c.flags|c.startFlag(),
			&buf,
		)
		copy(c.chain[:], buf[:8])

		c.blocks++
		c.blen = 0
	}
}

func (c *chunkState) output() output {
	if isLittleEndian {
		return c.outputLE()
	}
	return c.outputBE()
}

func (c *chunkState) outputLE() output {
	return output{
		chain:   &c.chain,
		block:   (*[16]uint32)(unsafe.Pointer(&c.block[0])),
		blen:    uint32(c.blen),
		counter: c.counter,
		flags:   c.flags | c.startFlag() | flag_chunkEnd,
	}
}

func (c *chunkState) outputBE() output {
	var block [16]uint32
	bytesToWords(&c.block, &block)

	return output{
		chain:   &c.chain,
		block:   &block,
		blen:    uint32(c.blen),
		counter: c.counter,
		flags:   c.flags | c.startFlag() | flag_chunkEnd,
	}
}

func parentOutput(
	left [8]uint32,
	right [8]uint32,
	key [8]uint32,
	flags uint32,
) output {

	block := [16]uint32{
		left[0], left[1], left[2], left[3],
		left[4], left[5], left[6], left[7],
		right[0], right[1], right[2], right[3],
		right[4], right[5], right[6], right[7],
	}

	return output{
		chain:   &key,
		block:   &block,
		counter: 0,
		blen:    blockLen,
		flags:   flags | flag_parent,
	}
}

func parentChain(
	left [8]uint32,
	right [8]uint32,
	key [8]uint32,
	flags uint32,
) [8]uint32 {

	out := parentOutput(left, right, key, flags)
	return out.compress()
}

//
// hash interface
//

type hasher struct {
	chunk chunkState
	key   [8]uint32
	slen  uint
	flags uint32
	stack [54][8]uint32
}

func newHasher() hasher {
	key := [8]uint32{iv0, iv1, iv2, iv3, iv4, iv5, iv6, iv7}
	return hasher{
		chunk: newChunkState(key, 0, 0),
		key:   key,
	}
}

// TODO: keyed and derive key

func (h *hasher) pushStack(chain [8]uint32) {
	h.stack[h.slen] = chain
	h.slen++
}

func (h *hasher) popStack() [8]uint32 {
	h.slen--
	return h.stack[h.slen]
}

func (h *hasher) addChain(chain [8]uint32, total uint64) {
	for total&1 == 0 {
		chain = parentChain(h.popStack(), chain, h.key, h.flags)
		total >>= 1
	}
	h.pushStack(chain)
}

func (h *hasher) update(input []byte) {
	for len(input) > 0 {
		if h.chunk.len() < chunkLen {
			data := input
			if want := chunkLen - h.chunk.len(); uint(len(data)) > want {
				data = data[:want]
			}

			h.chunk.update(data)
			input = input[len(data):]
			continue
		}

		var op output
		if isLittleEndian {
			op = h.chunk.outputLE()
		} else {
			op = h.chunk.outputBE()
		}

		chain := op.compress()
		total := h.chunk.counter + 1
		h.addChain(chain, total)
		h.chunk = newChunkState(h.key, total, h.flags)
	}
}

func (h *hasher) finalize(out []byte) {
	var op output
	if isLittleEndian {
		op = h.chunk.outputLE()
	} else {
		op = h.chunk.outputBE()
	}

	parents := h.slen
	for parents > 0 {
		parents--

		op = parentOutput(
			h.stack[parents],
			op.compress(),
			h.key,
			h.flags,
		)
	}

	op.rootOutput(out)
}
