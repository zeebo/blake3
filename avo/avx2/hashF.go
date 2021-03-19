package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
	. "github.com/zeebo/blake3/avo"
)

func HashF(c Ctx) {
	TEXT("HashF", 0, `func(
		input *[8192]byte,
		length uint64,
		counter uint64,
		flags uint32,
		key *[8]uint32,
		out *[32]uint32,
		chain *[8]uint32,
	)`)

	var (
		input   = Mem{Base: Load(Param("input"), GP64())}
		length  = Load(Param("length"), GP64()).(GPVirtual)
		counter = Load(Param("counter"), GP64()).(GPVirtual)
		flags   = Load(Param("flags"), GP32()).(GPVirtual)
		key     = Mem{Base: Load(Param("key"), GP64())}
		out     = Mem{Base: Load(Param("out"), GP64())}
		chain   = Mem{Base: Load(Param("chain"), GP64())}
	)

	loop := GP64()
	chunks := GP64()
	blocks := GP64()
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
	counter_mem := AllocLocal(8)

	tmp := AllocLocal(32)
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

	h_vecs = alloc.ValuesWith(8, key)
	blen_vec = alloc.ValueFrom(c.BlockLen)
	flags_vec = alloc.ValueWith(flags_mem)
	iv = alloc.ValuesWith(4, c.IV)
	ctr_low = alloc.ValueFrom(ctr_lo_mem)
	ctr_hi = alloc.ValueFrom(ctr_hi_mem)

	{
		Comment("Skip if the length is zero")
		XORQ(chunks, chunks)
		XORQ(blocks, blocks)
		TESTQ(length, length)
		JZ(LabelRef("skip_compute"))
	}

	{
		Comment("Compute complete chunks and blocks")

		// chunks = (length - 1) / 1024
		SUBQ(U8(1), length)
		MOVQ(length, chunks)
		SHRQ(U8(10), chunks)

		// blocks = (length - 1) % 1024 / 64 * 64
		MOVQ(length, blocks)
		ANDQ(U32(960), blocks)
	}

	Label("skip_compute")

	{
		Comment("Load some params into the stack (avo improvment?)")
		MOVL(flags, flags_mem)
		MOVQ(counter, counter_mem)
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
		for _, v := range h_vecs {
			_ = v.Get()
		}
		_ = blen_vec.Get()
		_ = flags_vec.Get()
		for _, v := range iv {
			_ = v.Get()
		}
		_ = ctr_low.Get()
		_ = ctr_hi.Get()
	}

	{
		Comment("Save state for partial chunk if necessary")
		CMPQ(loop, blocks)
		JNE(LabelRef("begin_rounds"))

		for i, v := range h_vecs {
			tmp32 := GP32()
			VMOVDQU(v.Get(), tmp)
			MOVL(tmp.Idx(chunks, 4), tmp32)
			MOVL(tmp32, chain.Offset(4*i))
		}
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
		Comment("Fix up registers for next iteration")
		for i := 7; i >= 0; i-- {
			h_vecs[i].Become(h_regs[i])
		}
	}

	{
		Comment("If we have zero complete chunks, we're done")
		CMPQ(chunks, U8(0))
		JNE(LabelRef("loop_trailer"))
		CMPQ(blocks, loop)
		JEQ(LabelRef("finalize"))
	}

	Label("loop_trailer")

	{
		Comment("Increment, reset flags, and loop")
		CMPQ(loop, U32(15*64))
		JEQ(LabelRef("finalize"))

		ADDQ(Imm(64), loop)
		MOVL(flags, flags_mem)
		JMP(LabelRef("loop"))
	}

	Label("finalize")

	{
		Comment("Store result into out")
		for i, v := range h_vecs {
			VMOVDQU(v.Consume(), out.Offset(32*i))
		}
	}

	VZEROUPPER()
	RET()
}

func roundF(c Ctx, alloc *Alloc, vs []*Value, r int, mp Mem) {
	round(c, alloc, vs, r, func(n int) Mem {
		return mp.Offset(n * 32)
	})
}
