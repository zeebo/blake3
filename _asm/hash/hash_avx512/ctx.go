package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
)

type Ctx struct {
	IV        Mem
	BlockLen  Mem
	Zero      Mem
	Transpose Mem
}

func NewCtx() (c Ctx) {
	c.IV = GLOBL("iv", RODATA|NOPTR)
	for n, v := range []U32{
		0x6A09E667, 0xBB67AE85, 0x3C6EF372, 0xA54FF53A,
		0x510E527F, 0x9B05688C, 0x1F83D9AB, 0x5BE0CD19,
	} {
		DATA(4*n, v)
	}

	c.BlockLen = GLOBL("block_len", RODATA|NOPTR)
	for i := 0; i < 8; i++ {
		DATA(4*i, U32(64))
	}

	c.Zero = GLOBL("zero", RODATA|NOPTR)
	for i := 0; i < 8; i++ {
		DATA(4*i, U32(0))
	}

	// One VPERMT2D index per word in a row, gathering that word from all eight chunks.
	c.Transpose = GLOBL("transpose_idx", RODATA|NOPTR)
	for w := 0; w < 4; w++ {
		for i := 0; i < 4; i++ {
			DATA(64*w+4*i, U32(uint32(w+4*i)))
		}
		for i := 0; i < 4; i++ {
			DATA(64*w+16+4*i, U32(uint32(16+w+4*i)))
		}
		for i := 8; i < 16; i++ {
			DATA(64*w+4*i, U32(0))
		}
	}

	return c
}
