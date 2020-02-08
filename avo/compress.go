package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
)

func Compress(c ctx) {
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

	MOVUPS(chain.Offset(0*16), rows[0])
	MOVUPS(chain.Offset(1*16), rows[1])
	MOVUPS(c.iv, rows[2])

	PINSRD(U8(0), counter.As32(), rows[3])
	SHRQ(U8(32), counter)
	PINSRD(U8(1), counter.As32(), rows[3])
	PINSRD(U8(2), blen, rows[3])
	PINSRD(U8(3), flags, rows[3])

	ms := []VecVirtual{XMM(), XMM(), XMM(), XMM()}

	MOVUPS(block.Offset(0*16), ms[0])
	MOVUPS(block.Offset(1*16), ms[1])
	MOVUPS(block.Offset(2*16), ms[2])
	MOVUPS(block.Offset(3*16), ms[3])

	rot16, rot8 := XMM(), XMM()
	MOVUPS(c.rot16, rot16)
	MOVUPS(c.rot8, rot8)

	{
		Comment("round 1")

		t0 := XMM()
		MOVAPS(ms[0], t0)                   // 3 2 1 0
		SHUFPS(pack(2, 0, 2, 0), ms[1], t0) // 6 4 2 0
		g(rows, t0, rot16, 12)              // 6 4 2 0

		t1 := XMM()
		MOVAPS(ms[0], t1)                   // 3 2 1 0
		SHUFPS(pack(3, 1, 3, 1), ms[1], t1) // 7 5 3 1
		g(rows, t1, rot8, 7)                // 7 5 3 1

		diagonalize(rows)

		t2 := XMM()
		MOVAPS(ms[2], t2)                   // b a 9 8
		SHUFPS(pack(2, 0, 2, 0), ms[3], t2) // e c a 8
		SHUFPS(pack(2, 1, 0, 3), t2, t2)    // c a 8 e
		g(rows, t2, rot16, 12)              // c a 8 e

		t3 := XMM()
		MOVAPS(ms[2], t3)                   // b a 9 8
		SHUFPS(pack(3, 1, 3, 1), ms[3], t3) // f d b 9
		SHUFPS(pack(2, 1, 0, 3), t3, t3)    // d b 9 f
		g(rows, t3, rot8, 7)                // d b 9 f

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
		MOVAPS(ms[0], t0)
		SHUFPS(pack(3, 1, 1, 2), ms[1], t0)
		SHUFPS(pack(0, 3, 2, 1), t0, t0)
		g(rows, t0, rot16, 12)

		t1 := XMM()
		MOVAPS(ms[2], t1)
		SHUFPS(pack(3, 3, 2, 2), ms[3], t1)
		PSHUFD(pack(0, 0, 3, 3), ms[0], tt)
		PBLENDW(U8(0b00110011), tt, t1)
		g(rows, t1, rot8, 7)

		diagonalize(rows)

		t2 := XMM()
		MOVAPS(ms[3], t2)
		VPUNPCKLDQ(ms[1], t2, t2)
		PBLENDW(U8(0b11000000), ms[2], t2)
		SHUFPS(pack(2, 3, 1, 0), t2, t2)
		g(rows, t2, rot16, 12)

		t3 := XMM()
		MOVAPS(ms[1], tt)
		VPUNPCKHDQ(ms[3], tt, tt)
		MOVAPS(ms[2], t3)
		VPUNPCKLDQ(tt, t3, t3)
		SHUFPS(pack(0, 1, 3, 2), t3, t3)
		g(rows, t3, rot8, 7)

		undiagonalize(rows)

		ms[0] = t0
		ms[1] = t1
		ms[2] = t2
		ms[3] = t3
	}

	Comment("finalize")

	PXOR(rows[2], rows[0])
	PXOR(rows[3], rows[1])

	tmp := XMM()
	MOVUPS(chain.Offset(0*16), tmp)
	PXOR(tmp, rows[2])
	MOVUPS(chain.Offset(1*16), tmp)
	PXOR(tmp, rows[3])

	MOVUPS(rows[0], out.Offset(0*16))
	MOVUPS(rows[1], out.Offset(1*16))
	MOVUPS(rows[2], out.Offset(2*16))
	MOVUPS(rows[3], out.Offset(3*16))

	RET()
}

func g(rows []VecVirtual, m VecVirtual, tab VecVirtual, n int) {
	PADDD(m, rows[0])
	PADDD(rows[1], rows[0])
	PXOR(rows[0], rows[3])
	PSHUFB(tab, rows[3])
	PADDD(rows[3], rows[2])
	PXOR(rows[2], rows[1])

	tmp := XMM()
	MOVAPS(rows[1], tmp)
	PSRLL(U8(n), rows[1])
	PSLLL(U8(32-n), tmp)
	POR(tmp, rows[1])
}

func pack(a, b, c, d int) U8 {
	return U8(a<<6 | b<<4 | c<<2 | d)
}

func diagonalize(rows []VecVirtual) {
	PSHUFD(pack(2, 1, 0, 3), rows[0], rows[0])
	PSHUFD(pack(1, 0, 3, 2), rows[3], rows[3])
	PSHUFD(pack(0, 3, 2, 1), rows[2], rows[2])
}

func undiagonalize(rows []VecVirtual) {
	PSHUFD(pack(0, 3, 2, 1), rows[0], rows[0])
	PSHUFD(pack(1, 0, 3, 2), rows[3], rows[3])
	PSHUFD(pack(2, 1, 0, 3), rows[2], rows[2])
}
