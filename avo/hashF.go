package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
)

func HashF(c ctx) {
	TEXT("hashF_avx", 0, `func(
		input *[8192]byte,
		chunks uint64,
		blocks uint64,
		counter uint64,
		flags uint32,
		out *[256]byte,
	)`)

	var (
		input   = Mem{Base: Load(Param("input"), GP64())}
		chunks  = Load(Param("chunks"), GP64())
		blocks  = Load(Param("blocks"), GP64())
		counter = Load(Param("counter"), GP64()).(GPVirtual)
		flags   = Load(Param("flags"), GP32()).(GPVirtual)
		out     = Mem{Base: Load(Param("out"), GP64())}
	)

	_ = chunks

	alloc := NewAlloc(AllocLocal(32))
	defer alloc.Free()

	loop := GP64()
	block_flags := AllocLocal(8) // only need 4, but keeps 64 bit alignment
	ctr_lo_mem := AllocLocal(32)
	ctr_hi_mem := AllocLocal(32)
	msg_vecs := AllocLocal(32 * 16)

	Comment("Load IV into vectors")
	h_vecs := alloc.Values(8)
	h_regs := make([]int, 8)
	for i, v := range h_vecs {
		VPBROADCASTD(c.iv.Offset(4*i), v.Get())
		h_regs[i] = v.Reg()
	}

	{
		c := func(n int) GPVirtual {
			r := GP64()
			if n > 0 {
				LEAQ(Mem{Base: counter}.Offset(n), r)
			} else {
				MOVQ(counter, r)
			}
			return r
		}

		Comment("Build and store counter data on the stack")
		for i := 0; i < 8; i++ {
			r := c(i)
			MOVL(r.As32(), ctr_lo_mem.Offset(4*i))
			SHRQ(U8(32), r)
			MOVL(r.As32(), ctr_hi_mem.Offset(4*i))
		}
	}

	Comment("Set up block flags for first iteration")
	MOVQ(U32(0), loop)
	MOVL(flags, block_flags)
	ORL(U8(flag_chunkStart), block_flags)

	{
		Label("loop")
		CMPQ(blocks, Imm(16))
		JEQ(LabelRef("finalize"))

		Comment("Include end flags if last block")
		CMPQ(blocks, Imm(15))
		JNE(LabelRef("round_setup"))
		ORL(U8(flag_chunkEnd), block_flags)

		Label("round_setup")
		Comment("Load and transpose message vectors")
		transpose_msg_vecs_and_inc(c, alloc, loop, input, msg_vecs)

		Comment("Set up block length and flag vectors")
		block_len_vec := alloc.Value()
		VMOVDQU(c.block_len, block_len_vec.Get())
		block_flags_vec := alloc.Value()
		VPBROADCASTD(block_flags, block_flags_vec.Get())

		Comment("Set up IV vectors")
		iv := alloc.Values(4)
		for i, v := range iv {
			VPBROADCASTD(c.iv.Offset(4*i), v.Get())
		}

		Comment("Set up counter vectors")
		ctr_low := alloc.Value()
		VMOVDQU(ctr_lo_mem, ctr_low.Get())
		ctr_hi := alloc.Value()
		VMOVDQU(ctr_hi_mem, ctr_hi.Get())

		vs := []*Value{
			h_vecs[0], h_vecs[1], h_vecs[2], h_vecs[3],
			h_vecs[4], h_vecs[5], h_vecs[6], h_vecs[7],
			iv[0], iv[1], iv[2], iv[3],
			ctr_low, ctr_hi, block_len_vec, block_flags_vec,
		}

		for r := 0; r < 7; r++ {
			Commentf("Round %d", r+1)
			roundF(c, alloc, vs, r, msg_vecs)
		}

		Comment("Finalize rounds")
		for i := 0; i < 8; i++ {
			h_vecs[i] = alloc.Value()
			VPXOR(vs[i].ConsumeOp(), vs[8+i].Consume(), h_vecs[i].Get())
		}

		Comment("Fix up registers for next iteration")
		for i := 7; i >= 0; i-- {
			h_vecs[i].Become(h_regs[i])
		}

		Comment("Decrement and loop")
		DECQ(blocks)
		MOVL(flags, block_flags)
		JMP(LabelRef("loop"))
	}

	Label("finalize")

	Comment("Transpose output vectors")
	transpose_vecs(c, alloc, h_vecs)

	Comment("Store into output")
	for i, v := range h_vecs {
		VMOVDQU(v.Consume(), out.Offset(32*i))
	}

	RET()
}

func roundF(c ctx, alloc *Alloc, vs []*Value, r int, mp Mem) {
	round(c, alloc, vs, r, func(n int) Mem {
		return mp.Offset(n * 32)
	})
}
