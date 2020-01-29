package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
)

func Hash8(c ctx) {
	TEXT("hash8_avx", 0, `func(
		input *[8192]byte,
		counter uint64,
		flags uint32,
		out *[256]byte,
	)`)

	var (
		input   = Mem{Base: Load(Param("input"), GP64())}
		counter = Load(Param("counter"), GP64()).(GPVirtual)
		flags   = Load(Param("flags"), GP32()).(GPVirtual)
		out     = Mem{Base: Load(Param("out"), GP64())}
	)

	loop := GP64()

	alloc := NewAlloc(AllocLocal(32))
	defer alloc.Free()

	flags_mem := AllocLocal(8)
	counter_mem := AllocLocal(8)

	ctr_lo_mem := AllocLocal(32)
	ctr_hi_mem := AllocLocal(32)
	msg := AllocLocal(32 * 16)

	var (
		h_vecs    []*Value
		h_regs    []int
		vs        []*Value
		iv        []*Value
		ctr_low   *Value
		ctr_hi    *Value
		blen_vec  *Value
		flags_vec *Value
	)

	{
		Comment("Load some params into the stack (avo improvment?)")
		MOVL(flags, flags_mem)
		MOVQ(counter, counter_mem)
	}

	{
		Comment("Load IV into vectors")
		h_vecs = alloc.ValuesWith(8, c.iv)
		h_regs = make([]int, 8)
		for i, v := range h_vecs {
			h_regs[i] = v.Reg()
		}
	}

	{
		Comment("Build and store counter data on the stack")
		loadCounter(c, alloc, counter_mem, ctr_lo_mem, ctr_hi_mem)
	}

	{
		Comment("Set up block flags and variables for iteration")
		XORQ(loop, loop)
		ORL(U8(flag_chunkStart), flags_mem)
	}

	Label("loop")

	{
		CMPQ(loop, U32(16*64))
		JEQ(LabelRef("finalize"))
	}

	{
		Comment("Include end flags if last block")
		CMPQ(loop, U32(15*64))
		JNE(LabelRef("round_setup"))
		ORL(U8(flag_chunkEnd), flags_mem)
	}

	Label("round_setup")

	{
		Comment("Load and transpose message vectors")
		transposeMsg(c, alloc, loop, input, msg)
	}

	{
		Comment("Set up block length and flag vectors")
		blen_vec = alloc.ValueFrom(c.blockLen)
		flags_vec = alloc.ValueWith(flags_mem)
	}

	{
		Comment("Set up IV vectors")
		iv = alloc.ValuesWith(4, c.iv)
	}

	{
		Comment("Set up counter vectors")
		ctr_low = alloc.ValueFrom(ctr_lo_mem)
		ctr_hi = alloc.ValueFrom(ctr_hi_mem)
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
			round8(c, alloc, vs, r, msg)
		}
	}

	{
		Comment("Finalize rounds")
		for i := 0; i < 8; i++ {
			h_vecs[i] = alloc.Value()
			VPXOR(vs[i].ConsumeOp(), vs[8+i].Consume(), h_vecs[i].Get())
		}
	}

	{
		Comment("Fix up registers for next iteration")
		for i := 7; i >= 0; i-- {
			h_vecs[i].Become(h_regs[i])
		}
	}

	{
		Comment("Increment, reset flags, and loop")
		ADDQ(Imm(64), loop)
		MOVL(flags, flags_mem)
		JMP(LabelRef("loop"))
	}

	Label("finalize")

	{
		Comment("Store into output")
		for i, v := range h_vecs {
			VMOVDQU(v.Consume(), out.Offset(32*i))
		}
	}

	RET()
}

func round8(c ctx, alloc *Alloc, vs []*Value, r int, mp Mem) {
	round(c, alloc, vs, r, func(n int) Mem {
		return mp.Offset(n * 32)
	})
}
