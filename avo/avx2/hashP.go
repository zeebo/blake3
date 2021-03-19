package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
	. "github.com/zeebo/blake3/avo"
)

func HashP(c Ctx) {
	TEXT("HashP", NOSPLIT, `func(
		left *[32]uint32,
		right *[32]uint32,
		flags uint8,
		key *[8]uint32,
		out *[32]uint32,
		n int,
	)`)

	var (
		left  = Mem{Base: Load(Param("left"), GP64())}
		right = Mem{Base: Load(Param("right"), GP64())}
		flags = Load(Param("flags"), GP32()).(GPVirtual)
		key   = Mem{Base: Load(Param("key"), GP64())}
		out   = Mem{Base: Load(Param("out"), GP64())}
	)

	stash := GP64()

	{
		Comment("Allocate local space and align it")
		local := AllocLocal(roundSize + 32)
		LEAQ(local.Offset(31), stash)
		// TODO: avo improvement
		tmp := GP64()
		MOVQ(U64(31), tmp)
		NOTQ(tmp)
		ANDQ(tmp, stash)
	}

	alloc := NewAlloc(Mem{Base: stash})
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
	}

	h_vecs = alloc.ValuesWith(8, key)
	iv = alloc.ValuesWith(4, c.IV)
	ctr_low = alloc.ValueFrom(c.Zero)
	ctr_hi = alloc.ValueFrom(c.Zero)
	blen_vec = alloc.ValueFrom(c.BlockLen)
	flags_vec = alloc.ValueWith(flags_mem)

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
		finalizeRounds(alloc, vs, h_vecs, nil)
	}

	{
		Comment("Store result into out")
		for i, v := range h_vecs {
			VMOVDQU(v.Consume(), out.Offset(32*i))
		}
	}

	VZEROUPPER()
	RET()
}

func roundP(c Ctx, alloc *Alloc, vs []*Value, r int, left, right Mem) {
	round(c, alloc, vs, r, func(n int) Mem {
		if n < 8 {
			return left.Offset(n * 32)
		} else {
			return right.Offset((n - 8) * 32)
		}
	})
}
