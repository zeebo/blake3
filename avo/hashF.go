package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
)

func HashF(c ctx) {
	TEXT("hashF_avx", 0, `func(
		input *[8192]byte,
		length uint64,
		counter uint64,
		flags uint32,
		out *[32]uint32,
	)`)

	var (
		input   = Mem{Base: Load(Param("input"), GP64())}
		length  = Load(Param("length"), GP64()).(GPVirtual)
		counter = Load(Param("counter"), GP64()).(GPVirtual)
		flags   = Load(Param("flags"), GP32()).(GPVirtual)
		out     = Mem{Base: Load(Param("out"), GP64())}
	)

	loop := GP64()
	maskO := Mem{Base: GP64()}
	maskP := Mem{Base: GP64()}
	chunks := GP64()
	blocks := GP64()
	blen := GP64()

	alloc := NewAlloc(AllocLocal(roundSize))
	defer alloc.Free()

	flags_mem := AllocLocal(8)
	counter_mem := AllocLocal(8)
	blen_mem := AllocLocal(8)

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

	h_vecs = alloc.ValuesWith(8, c.iv)
	blen_vec = alloc.ValueFrom(c.blockLen)
	flags_vec = alloc.ValueWith(flags_mem)
	iv = alloc.ValuesWith(4, c.iv)
	ctr_low = alloc.ValueFrom(ctr_lo_mem)
	ctr_hi = alloc.ValueFrom(ctr_hi_mem)

	{
		Comment("Skip if the length is zero")
		TESTQ(length, length)
		JZ(LabelRef("return"))
	}

	{
		Comment("Compute complete chunks, blocks and blen")

		// chunks = (length / 1024) * 32
		MOVQ(length, chunks)
		SHRQ(U8(10), chunks)
		SHLQ(U8(5), chunks)

		// blocks = (length - 1) % 1024 / 64 * 64
		SUBQ(U8(1), length)
		MOVQ(length, blocks)
		ANDQ(U32(960), blocks)

		// blen = (length - 1) % 64 + 1
		MOVQ(length, blen)
		ANDQ(U8(63), blen)
		ADDQ(U8(1), blen)
	}

	{
		Comment("Load some params into the stack (avo improvment?)")
		MOVL(flags, flags_mem)
		MOVQ(counter, counter_mem)
		MOVQ(blen, blen_mem)
	}

	{
		Comment("Set up masks for block flags and stores")
		LEAQ(c.maskO, maskO.Base)
		LEAQ(maskO.Idx(chunks, 1), maskO.Base)
		LEAQ(c.maskP, maskP.Base)
		LEAQ(maskP.Idx(chunks, 1), maskP.Base)
	}

	{
		Comment("Load IV into vectors")
		h_regs = make([]int, 8)
		for i, v := range h_vecs {
			h_regs[i] = v.Reg()
			_ = v.Get()
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
		Comment("Load constants for the round")
		_ = blen_vec.Get()
		_ = flags_vec.Get()
		for _, v := range iv {
			_ = v.Get()
		}
		_ = ctr_low.Get()
		_ = ctr_hi.Get()
	}

	{
		Comment("Insert flag and length if last block in partial chunk")
		CMPQ(loop, blocks)
		JNE(LabelRef("begin_rounds"))

		// or in the chunk end flag
		tmp := alloc.ValueWith(c.chunkEnd)
		VPAND(maskO, tmp.Get(), tmp.Get())
		VPOR(tmp.Consume(), flags_vec.Get(), flags_vec.Get())

		// clear out the block len
		tmp = alloc.ValueFrom(maskO)
		VPXOR(c.all, tmp.Get(), tmp.Get())
		VPAND(tmp.Consume(), blen_vec.Get(), blen_vec.Get())

		// or in the appropriate block len
		tmp = alloc.ValueWith(blen_mem)
		VPAND(maskO, tmp.Get(), tmp.Get())
		VPOR(tmp.Consume(), blen_vec.Get(), blen_vec.Get())
	}

	Label("begin_rounds")

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
			roundF(c, alloc, vs, r, msg)
		}
	}

	{
		Comment("Finalize rounds")
		finalizeRounds(alloc, vs, h_vecs, h_regs)
	}

	{
		Comment("Save state for partial chunk if necessary")
		CMPQ(loop, blocks)
		JNE(LabelRef("loop_tail"))

		extractMask(c, alloc, h_vecs, maskO, out)
	}

	{
		Comment("If we have zero complete chunks, we're done")
		CMPQ(chunks, U8(0))
		JNE(LabelRef("loop_tail"))

		RET()
	}

	Label("loop_tail")

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
		Comment("Store prefix of full chunks into output")
		extractMask(c, alloc, h_vecs, maskP, out)
		for _, v := range h_vecs {
			v.Free()
		}
	}

	Label("return")
	RET()
}

func roundF(c ctx, alloc *Alloc, vs []*Value, r int, mp Mem) {
	round(c, alloc, vs, r, func(n int) Mem {
		return mp.Offset(n * 32)
	})
}

func extractMask(c ctx, alloc *Alloc, vs []*Value, mask, mp Mem) {
	tmp := alloc.ValueFrom(mask)
	for i, v := range vs {
		VPMASKMOVD(v.Get(), tmp.Get(), mp.Offset(32*i))
	}
	tmp.Free()
}
