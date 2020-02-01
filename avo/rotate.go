package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
)

func Rotate(c ctx) {
	TEXT("rotate_avx", 0, `func(in *[256]byte)`)

	in := Mem{Base: Load(Param("in"), GP64())}

	for i := 0; i < 8; i++ {
		reg := YMM()
		VPSHUFD(U8(0b10_01_00_11), in.Offset(32*i), reg)
		VMOVDQU(reg, in.Offset(32*i))
	}

	RET()
}
