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

	alloc := NewAlloc(AllocLocal(32))
	defer alloc.Free()

	flags_mem := AllocLocal(8)

	var (
		h_vecs    []*Value
		vs        []*Value
		iv        []*Value
		ctr_low   *Value
		ctr_hi    *Value
		blen_vec  *Value
		flags_vec *Value
	)

	{
		Comment("Set up flags value")
		MOVL(flags, flags_mem)
		ORL(U8(flag_parent), flags_mem)
	}

	{
		Comment("Load IV into vectors")
		h_vecs = alloc.ValuesWith(8, c.iv)
	}

	{
		Comment("Set up constant vectors")
		iv = alloc.ValuesWith(4, c.iv)
		ctr_low = alloc.ValueFrom(c.zero)
		ctr_hi = alloc.ValueFrom(c.zero)
		blen_vec = alloc.ValueFrom(c.blockLen)
		flags_vec = alloc.ValueWith(flags_mem)
	}

	{
		Comment("Perform the rounds")

		vs = []*Value{
			h_vecs[0], h_vecs[1], h_vecs[2], h_vecs[3],
			h_vecs[4], h_vecs[5], h_vecs[6], h_vecs[7],
			iv[0], iv[1], iv[2], iv[3],
			ctr_low, ctr_hi, blen_vec, flags_vec,
		}

		for r := 0; r < 7; r++ {
			Commentf("Round %d", r+1)
			roundP(c, alloc, vs, r, left, right)
		}
	}

	{
		Comment("Finalize")
		for i := 0; i < 8; i++ {
			h_vecs[i] = alloc.Value()
			VPXOR(vs[i].ConsumeOp(), vs[8+i].Consume(), h_vecs[i].Get())
		}
	}

	{
		Comment("Store result into out")
		for i, v := range h_vecs {
			VMOVDQU(v.Consume(), out.Offset(32*i))
		}
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
