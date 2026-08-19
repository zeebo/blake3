package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
)

func main() {
	ivMem := GLOBL("iv", RODATA|NOPTR)
	for n, v := range []U32{
		0x6A09E667, 0xBB67AE85, 0x3C6EF372, 0xA54FF53A,
		0x510E527F, 0x9B05688C, 0x1F83D9AB, 0x5BE0CD19,
	} {
		DATA(4*n, v)
	}

	TEXT("Compress", NOSPLIT, `func(
		chain *[8]uint32,
		block *[16]uint32,
		counter uint64,
		blen uint32,
		flags uint32,
		out *[16]uint32,
	)`)

	var (
		chain   = Mem{Base: Load(Param("chain"), GP64())}
		block   = Mem{Base: Load(Param("block"), GP64())}
		counter = Load(Param("counter"), GP64()).(GPVirtual)
		blen    = Load(Param("blen"), GP32()).(GPVirtual)
		flags   = Load(Param("flags"), GP32()).(GPVirtual)
		out     = Mem{Base: Load(Param("out"), GP64())}
	)

	rows := []VecVirtual{XMM(), XMM(), XMM(), XMM()}

	VMOVDQU(chain.Offset(0*16), rows[0])
	VMOVDQU(chain.Offset(1*16), rows[1])
	VMOVDQU(ivMem, rows[2])

	VMOVD(counter.As32(), rows[3])
	SHRQ(U8(32), counter)
	VPINSRD(U8(1), counter.As32(), rows[3], rows[3])
	VPINSRD(U8(2), blen, rows[3], rows[3])
	VPINSRD(U8(3), flags, rows[3], rows[3])

	ms := []VecVirtual{XMM(), XMM(), XMM(), XMM()}

	VMOVDQU(block.Offset(0*16), ms[0])
	VMOVDQU(block.Offset(1*16), ms[1])
	VMOVDQU(block.Offset(2*16), ms[2])
	VMOVDQU(block.Offset(3*16), ms[3])

	{
		Comment("round 1")

		t0 := XMM()
		VSHUFPS(pack(2, 0, 2, 0), ms[1], ms[0], t0) // 6 4 2 0
		g(rows, t0, 16, 12)                         // 6 4 2 0

		t1 := XMM()
		VSHUFPS(pack(3, 1, 3, 1), ms[1], ms[0], t1) // 7 5 3 1
		g(rows, t1, 8, 7)                           // 7 5 3 1

		diagonalize(rows)

		t2 := XMM()
		VSHUFPS(pack(2, 0, 2, 0), ms[3], ms[2], t2) // e c a 8
		VSHUFPS(pack(2, 1, 0, 3), t2, t2, t2)       // c a 8 e
		g(rows, t2, 16, 12)                         // c a 8 e

		t3 := XMM()
		VSHUFPS(pack(3, 1, 3, 1), ms[3], ms[2], t3) // f d b 9
		VSHUFPS(pack(2, 1, 0, 3), t3, t3, t3)       // d b 9 f
		g(rows, t3, 8, 7)                           // d b 9 f

		undiagonalize(rows)

		ms[0] = t0
		ms[1] = t1
		ms[2] = t2
		ms[3] = t3
	}

	for i := 1; i < 7; i++ {
		tt := XMM()

		Commentf("round %d", i+1)

		t0 := XMM()
		VSHUFPS(pack(3, 1, 1, 2), ms[1], ms[0], t0)
		VSHUFPS(pack(0, 3, 2, 1), t0, t0, t0)
		g(rows, t0, 16, 12)

		t1 := XMM()
		VSHUFPS(pack(3, 3, 2, 2), ms[3], ms[2], t1)
		VPSHUFD(pack(0, 0, 3, 3), ms[0], tt)
		VPBLENDW(U8(0b00110011), tt, t1, t1)
		g(rows, t1, 8, 7)

		diagonalize(rows)

		t2 := XMM()
		VPUNPCKLDQ(ms[1], ms[3], t2)
		VPBLENDW(U8(0b11000000), ms[2], t2, t2)
		VSHUFPS(pack(2, 3, 1, 0), t2, t2, t2)
		g(rows, t2, 16, 12)

		t3 := XMM()
		VPUNPCKHDQ(ms[3], ms[1], tt)
		VPUNPCKLDQ(tt, ms[2], t3)
		VSHUFPS(pack(0, 1, 3, 2), t3, t3, t3)
		g(rows, t3, 8, 7)

		undiagonalize(rows)

		ms[0] = t0
		ms[1] = t1
		ms[2] = t2
		ms[3] = t3
	}

	Comment("finalize")

	VPXOR(rows[2], rows[0], rows[0])
	VPXOR(rows[3], rows[1], rows[1])

	VPXOR(chain.Offset(0*16), rows[2], rows[2])
	VPXOR(chain.Offset(1*16), rows[3], rows[3])

	VMOVDQU(rows[0], out.Offset(0*16))
	VMOVDQU(rows[1], out.Offset(1*16))
	VMOVDQU(rows[2], out.Offset(2*16))
	VMOVDQU(rows[3], out.Offset(3*16))

	RET()

	Generate()
}

func g(rows []VecVirtual, m VecVirtual, d, b int) {
	VPADDD(m, rows[0], rows[0])
	VPADDD(rows[1], rows[0], rows[0])
	VPXOR(rows[0], rows[3], rows[3])
	VPRORD(U8(d), rows[3], rows[3])
	VPADDD(rows[3], rows[2], rows[2])
	VPXOR(rows[2], rows[1], rows[1])
	VPRORD(U8(b), rows[1], rows[1])
}

func pack(a, b, c, d int) U8 {
	return U8(a<<6 | b<<4 | c<<2 | d)
}

func diagonalize(rows []VecVirtual) {
	VPSHUFD(pack(2, 1, 0, 3), rows[0], rows[0])
	VPSHUFD(pack(1, 0, 3, 2), rows[3], rows[3])
	VPSHUFD(pack(0, 3, 2, 1), rows[2], rows[2])
}

func undiagonalize(rows []VecVirtual) {
	VPSHUFD(pack(0, 3, 2, 1), rows[0], rows[0])
	VPSHUFD(pack(1, 0, 3, 2), rows[3], rows[3])
	VPSHUFD(pack(2, 1, 0, 3), rows[2], rows[2])
}
