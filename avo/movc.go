package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
)

func Movc(c ctx) {
	TEXT("movc_avx", 0, `func(
		input *[256]byte,
		icol uint64,
		out *[256]byte,
		ocol uint64,
	)`)

	var (
		input = Mem{Base: Load(Param("input"), GP64())}
		icol  = Load(Param("icol"), GP64()).(GPVirtual)
		out   = Mem{Base: Load(Param("out"), GP64())}
		ocol  = Load(Param("ocol"), GP64()).(GPVirtual)
	)

	addr := GP64()

	alloc := NewAlloc(Mem{})
	defer alloc.Free()

	SHLQ(U8(5), ocol)
	LEAQ(c.maskO, addr)

	omask := alloc.ValueFrom(Mem{Base: addr}.Idx(ocol, 1))
	defer omask.Free()

	LEAQ(input.Idx(icol, 4), input.Base)

	tmp := alloc.Value()
	defer tmp.Free()

	for i := 0; i < 8; i++ {
		VPBROADCASTD(input.Offset(32*i), tmp.Get())
		VPMASKMOVD(tmp.Get(), omask.Get(), out.Offset(32*i))
	}

	RET()
}
