package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
)

func HashP(c ctx) {
	TEXT("hashP_avx", 0, `func(
		left *[256]byte,
		right *[256]byte,
		flags uint8,
		out *[256]byte,
	)`)

	var (
		left  = Mem{Base: Load(Param("left"), GP64())}
		right = Mem{Base: Load(Param("right"), GP64())}
		flags = Load(Param("flags"), GP32()).(GPVirtual)
		out   = Mem{Base: Load(Param("out"), GP64())}
	)

	block_flags := AllocLocal(8)
	alloc := NewAlloc(AllocLocal(32))
	defer alloc.Free()

	Comment("Set up flags value")
	MOVL(flags, block_flags)
	ORL(U8(flag_parent), block_flags)

	Comment("Load IV into vectors")
	h_vecs := alloc.Values(8)
	for i, v := range h_vecs {
		VPBROADCASTD(c.iv.Offset(4*i), v.Get())
	}

	Comment("Set up constant vectors")
	iv := alloc.Values(4)
	ctr_low := alloc.Value()
	ctr_hi := alloc.Value()
	block_len_vec := alloc.Value()
	block_flags_vec := alloc.Value()

	for i, v := range iv {
		VPBROADCASTD(c.iv.Offset(4*i), v.Get())
	}
	VMOVDQU(c.zero, ctr_low.Get())
	VMOVDQU(c.zero, ctr_hi.Get())
	VMOVDQU(c.block_len, block_len_vec.Get())
	VPBROADCASTD(block_flags, block_flags_vec.Get())

	vs := []*Value{
		h_vecs[0], h_vecs[1], h_vecs[2], h_vecs[3],
		h_vecs[4], h_vecs[5], h_vecs[6], h_vecs[7],
		iv[0], iv[1], iv[2], iv[3],
		ctr_low, ctr_hi, block_len_vec, block_flags_vec,
	}

	for r := 0; r < 7; r++ {
		Commentf("Round %d", r+1)
		roundP(c, alloc, vs, r, left, right)
	}

	Comment("Finalize")
	for i := 0; i < 8; i++ {
		h_vecs[i] = alloc.Value()
		VPXOR(vs[i].ConsumeOp(), vs[8+i].Consume(), h_vecs[i].Get())
	}

	Comment("Store result into out")
	for i, v := range h_vecs {
		VMOVDQU(v.Consume(), out.Offset(32*i))
	}

	RET()
}

func roundP(c ctx, alloc *Alloc, vs []*Value, r int, left, right Mem) {
	round(c, alloc, vs, r, func(n int) Mem {
		if n < 8 {
			return left.Offset(n * 32)
		} else {
			return right.Offset((n - 8) * 32)
		}
	})
}
