package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
)

// MCA is used to put kernels into llvm-mca
func MCA(c ctx) {
	TEXT("MCA", NOSPLIT, `func(input *[8192]byte, buf *[512]byte)`)

	var (
		input = Mem{Base: Load(Param("input"), GP64())}
		buf   = Mem{Base: Load(Param("buf"), GP64())}
	)

	alloc := NewAlloc(Mem{})
	defer alloc.Free()

	loop := GP64()
	XORQ(loop, loop)

	transposeMsg(c, alloc, loop, input, buf)

	RET()
}
