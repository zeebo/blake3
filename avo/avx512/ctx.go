package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
)

type Ctx struct {
	IV       Mem
	BlockLen Mem
	Zero     Mem
	Counter  Mem
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

	c.Counter = GLOBL("counter", RODATA|NOPTR)
	for i := 0; i < 8; i++ {
		DATA(8*i, U64(i))
	}

	return c
}
