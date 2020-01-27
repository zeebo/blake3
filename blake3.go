package blake3

import (
	"encoding/binary"
	"math/bits"
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
// core primitives
//

const (
	r0 = 0xfedcba9876543210
	r1 = 0x8fe95cb1d407a362
	r2 = 0x18fb0956e72dca43
	r3 = 0x61852b04fd3e9c7a
	r4 = 0x461035278eafb9dc
	r5 = 0x7462a03d1fc85be9
	r6 = 0xd743c2ae689105fb
)

func round(state, m *[16]uint32, r uint64) {
	{
		const a, b, c, d = 0, 4, 8, 12
		state[a] = state[a] + state[b] + m[r%16]
		r >>= 4
		state[d] = bits.RotateLeft32(state[d]^state[a], -16)
		state[c] = state[c] + state[d]
		state[b] = bits.RotateLeft32(state[b]^state[c], -12)
		state[a] = state[a] + state[b] + m[r%16]
		r >>= 4
		state[d] = bits.RotateLeft32(state[d]^state[a], -8)
		state[c] = state[c] + state[d]
		state[b] = bits.RotateLeft32(state[b]^state[c], -7)
	}
	{
		const a, b, c, d = 1, 5, 9, 13
		state[a] = state[a] + state[b] + m[r%16]
		r >>= 4
		state[d] = bits.RotateLeft32(state[d]^state[a], -16)
		state[c] = state[c] + state[d]
		state[b] = bits.RotateLeft32(state[b]^state[c], -12)
		state[a] = state[a] + state[b] + m[r%16]
		r >>= 4
		state[d] = bits.RotateLeft32(state[d]^state[a], -8)
		state[c] = state[c] + state[d]
		state[b] = bits.RotateLeft32(state[b]^state[c], -7)
	}
	{
		const a, b, c, d = 2, 6, 10, 14
		state[a] = state[a] + state[b] + m[r%16]
		r >>= 4
		state[d] = bits.RotateLeft32(state[d]^state[a], -16)
		state[c] = state[c] + state[d]
		state[b] = bits.RotateLeft32(state[b]^state[c], -12)
		state[a] = state[a] + state[b] + m[r%16]
		r >>= 4
		state[d] = bits.RotateLeft32(state[d]^state[a], -8)
		state[c] = state[c] + state[d]
		state[b] = bits.RotateLeft32(state[b]^state[c], -7)
	}
	{
		const a, b, c, d = 3, 7, 11, 15
		state[a] = state[a] + state[b] + m[r%16]
		r >>= 4
		state[d] = bits.RotateLeft32(state[d]^state[a], -16)
		state[c] = state[c] + state[d]
		state[b] = bits.RotateLeft32(state[b]^state[c], -12)
		state[a] = state[a] + state[b] + m[r%16]
		r >>= 4
		state[d] = bits.RotateLeft32(state[d]^state[a], -8)
		state[c] = state[c] + state[d]
		state[b] = bits.RotateLeft32(state[b]^state[c], -7)
	}

	{
		const a, b, c, d = 0, 5, 10, 15
		state[a] = state[a] + state[b] + m[r%16]
		r >>= 4
		state[d] = bits.RotateLeft32(state[d]^state[a], -16)
		state[c] = state[c] + state[d]
		state[b] = bits.RotateLeft32(state[b]^state[c], -12)
		state[a] = state[a] + state[b] + m[r%16]
		r >>= 4
		state[d] = bits.RotateLeft32(state[d]^state[a], -8)
		state[c] = state[c] + state[d]
		state[b] = bits.RotateLeft32(state[b]^state[c], -7)
	}
	{
		const a, b, c, d = 1, 6, 11, 12
		state[a] = state[a] + state[b] + m[r%16]
		r >>= 4
		state[d] = bits.RotateLeft32(state[d]^state[a], -16)
		state[c] = state[c] + state[d]
		state[b] = bits.RotateLeft32(state[b]^state[c], -12)
		state[a] = state[a] + state[b] + m[r%16]
		r >>= 4
		state[d] = bits.RotateLeft32(state[d]^state[a], -8)
		state[c] = state[c] + state[d]
		state[b] = bits.RotateLeft32(state[b]^state[c], -7)
	}
	{
		const a, b, c, d = 2, 7, 8, 13
		state[a] = state[a] + state[b] + m[r%16]
		r >>= 4
		state[d] = bits.RotateLeft32(state[d]^state[a], -16)
		state[c] = state[c] + state[d]
		state[b] = bits.RotateLeft32(state[b]^state[c], -12)
		state[a] = state[a] + state[b] + m[r%16]
		r >>= 4
		state[d] = bits.RotateLeft32(state[d]^state[a], -8)
		state[c] = state[c] + state[d]
		state[b] = bits.RotateLeft32(state[b]^state[c], -7)
	}
	{
		const a, b, c, d = 3, 4, 9, 14
		state[a] = state[a] + state[b] + m[r%16]
		r >>= 4
		state[d] = bits.RotateLeft32(state[d]^state[a], -16)
		state[c] = state[c] + state[d]
		state[b] = bits.RotateLeft32(state[b]^state[c], -12)
		state[a] = state[a] + state[b] + m[r%16]
		r >>= 4
		state[d] = bits.RotateLeft32(state[d]^state[a], -8)
		state[c] = state[c] + state[d]
		state[b] = bits.RotateLeft32(state[b]^state[c], -7)
	}
}

func compress(
	chain *[8]uint32,
	block *[16]uint32,
	counter uint64,
	blen uint32,
	flags uint32,
) [16]uint32 {

	state := [16]uint32{
		chain[0], chain[1], chain[2], chain[3],
		chain[4], chain[5], chain[6], chain[7],
		iv0, iv1, iv2, iv3,
		uint32(counter), uint32(counter >> 32), blen, flags,
	}

	{
		round(&state, block, r0)
		round(&state, block, r1)
		round(&state, block, r2)
		round(&state, block, r3)
		round(&state, block, r4)
		round(&state, block, r5)
		round(&state, block, r6)
	}

	{
		state[0] ^= state[8]
		state[8] ^= chain[0]
		state[1] ^= state[9]
		state[9] ^= chain[1]
		state[2] ^= state[10]
		state[10] ^= chain[2]
		state[3] ^= state[11]
		state[11] ^= chain[3]
		state[4] ^= state[12]
		state[12] ^= chain[4]
		state[5] ^= state[13]
		state[13] ^= chain[5]
		state[6] ^= state[14]
		state[14] ^= chain[6]
		state[7] ^= state[15]
		state[15] ^= chain[7]
	}

	return state
}

//
// output
//

type output struct {
	chain   [8]uint32
	block   [16]uint32
	counter uint64
	blen    uint32
	flags   uint32
}

func (o *output) compress() [8]uint32 {
	return first8words(compress(
		&o.chain,
		&o.block,
		o.counter,
		o.blen,
		o.flags,
	))
}

func (o *output) rootOutput(out []byte) {
	var counter uint64
	for len(out) > 64 {
		block := compress(
			&o.chain,
			&o.block,
			counter,
			o.blen,
			o.flags|flag_root,
		)

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

		counter++
		out = out[64:]
	}

	if len(out) == 0 {
		return
	}

	block := compress(
		&o.chain,
		&o.block,
		counter,
		o.blen,
		o.flags|flag_root,
	)

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

func newChunkState(key [8]uint32, counter uint64, flags uint32) chunkState {
	return chunkState{
		chain:   key,
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
	for len(input) > 0 {
		if c.blen < blockLen {
			n := uint(copy(c.block[c.blen:], input))
			c.blen += n
			input = input[n:]

			continue
		}

		var block [16]uint32
		bytesToWords(&c.block, &block)

		c.chain = first8words(compress(
			&c.chain,
			&block,
			c.counter,
			blockLen,
			c.flags|c.startFlag(),
		))
		c.blocks++
		c.block = [blockLen]uint8{}
		c.blen = 0
	}
}

func (c *chunkState) output() output {
	var block [16]uint32
	bytesToWords(&c.block, &block)

	return output{
		chain:   c.chain,
		block:   block,
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
		chain:   key,
		block:   block,
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

		output := h.chunk.output()
		chain := output.compress()
		total := h.chunk.counter + 1
		h.addChain(chain, total)
		h.chunk = newChunkState(h.key, total, h.flags)
	}
}

func (h *hasher) finalize(out []byte) {
	output := h.chunk.output()

	parents := h.slen
	for parents > 0 {
		parents--

		output = parentOutput(
			h.stack[parents],
			output.compress(),
			h.key,
			h.flags,
		)
	}

	output.rootOutput(out)
}
