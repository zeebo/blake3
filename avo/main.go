package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
)

const roundSize = 32

var msgSched = [7][16]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
	{2, 6, 3, 10, 7, 0, 4, 13, 1, 11, 12, 5, 9, 14, 15, 8},
	{3, 4, 10, 12, 13, 2, 7, 14, 6, 5, 9, 0, 11, 15, 8, 1},
	{10, 7, 12, 9, 14, 3, 13, 15, 4, 0, 11, 2, 5, 8, 1, 6},
	{12, 13, 9, 11, 15, 10, 14, 8, 7, 2, 5, 3, 0, 1, 6, 4},
	{9, 14, 11, 5, 8, 12, 15, 1, 13, 3, 0, 10, 2, 6, 4, 7},
	{11, 15, 5, 0, 1, 9, 8, 6, 14, 10, 2, 12, 3, 4, 7, 13},
}

type ctx struct {
	rot16    Mem
	rot8     Mem
	iv       Mem
	blockLen Mem
	zero     Mem
	counter  Mem
	maskO    Mem
	maskP    Mem
	all      Mem
	chunkEnd Mem
}

func main() {
	var c ctx

	c.iv = GLOBL("iv", RODATA|NOPTR)
	for n, v := range []U32{
		0x6A09E667, 0xBB67AE85, 0x3C6EF372, 0xA54FF53A,
		0x510E527F, 0x9B05688C, 0x1F83D9AB, 0x5BE0CD19,
	} {
		DATA(4*n, v)
	}

	c.rot16 = GLOBL("rot16_shuf", RODATA|NOPTR)
	for n, v := range []U8{
		0x02, 0x03, 0x00, 0x01, 0x06, 0x07, 0x04, 0x05,
		0x0A, 0x0B, 0x08, 0x09, 0x0E, 0x0F, 0x0C, 0x0D,
		0x12, 0x13, 0x10, 0x11, 0x16, 0x17, 0x14, 0x15,
		0x1A, 0x1B, 0x18, 0x19, 0x1E, 0x1F, 0x1C, 0x1D,
	} {
		DATA(n, v)
	}

	c.rot8 = GLOBL("rot8_shuf", RODATA|NOPTR)
	for n, v := range []U8{
		0x01, 0x02, 0x03, 0x00, 0x05, 0x06, 0x07, 0x04,
		0x09, 0x0A, 0x0B, 0x08, 0x0D, 0x0E, 0x0F, 0x0C,
		0x11, 0x12, 0x13, 0x10, 0x15, 0x16, 0x17, 0x14,
		0x19, 0x1A, 0x1B, 0x18, 0x1D, 0x1E, 0x1F, 0x1C,
	} {
		DATA(n, v)
	}

	c.blockLen = GLOBL("block_len", RODATA|NOPTR)
	for i := 0; i < 8; i++ {
		DATA(4*i, U32(64))
	}

	c.zero = GLOBL("zero", RODATA|NOPTR)
	for i := 0; i < 8; i++ {
		DATA(4*i, U32(0))
	}

	c.counter = GLOBL("counter", RODATA|NOPTR)
	for i := 0; i < 8; i++ {
		DATA(8*i, U64(i))
	}

	c.maskO = GLOBL("maskO", RODATA|NOPTR)
	for i := 0; i < 9; i++ {
		for j := 0; j < 8; j++ {
			if i == j {
				DATA(32*i+4*j, ^U32(0))
			} else {
				DATA(32*i+4*j, U32(0))
			}
		}
	}

	c.maskP = GLOBL("maskP", RODATA|NOPTR)
	for i := 0; i < 9; i++ {
		for j := 0; j < 8; j++ {
			if i > j {
				DATA(32*i+4*j, ^U32(0))
			} else {
				DATA(32*i+4*j, U32(0))
			}
		}
	}

	c.all = GLOBL("all", RODATA|NOPTR)
	for i := 0; i < 8; i++ {
		DATA(4*i, ^U32(0))
	}

	c.chunkEnd = GLOBL("chunk_end", RODATA|NOPTR)
	DATA(0, U32(flag_chunkEnd))

	HashF(c)
	HashP(c)

	Generate()
}
