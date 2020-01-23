package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
)

var _ GPVirtual = nil

func Hash8(c ctx) {
	TEXT("hash8_avx", 0, "func(inputs *[8]*byte, blocks uint64, key *[8]uint32, counter, inc uint64, flags, flags_start, flags_end uint8, out *[256]byte)")

	var (
		inputs      = Mem{Base: Load(Param("inputs"), GP64())}
		blocks      = Load(Param("blocks"), GP64())
		key         = Mem{Base: Load(Param("key"), GP64())}
		counter     = Load(Param("counter"), GP64()).(GPVirtual)
		inc         = Load(Param("inc"), GP64()).(GPVirtual)
		flags       = Load(Param("flags"), GP32()).(GPVirtual)
		flags_start = Load(Param("flags_start"), GP32()).(GPVirtual)
		flags_end   = Load(Param("flags_end"), GP32()).(GPVirtual)
		out         = Mem{Base: Load(Param("out"), GP64())}
	)

	alloc := NewAlloc(AllocLocal(32))
	defer alloc.Free()

	block_flags := AllocLocal(8) // only need 4, but keeps 64 bit alignment
	ctr_lo_mem := AllocLocal(32)
	ctr_hi_mem := AllocLocal(32)
	msg_vecs := AllocLocal(32 * 16)

	Comment("Load key into vectors")
	h_vecs := alloc.Values(8)
	h_regs := make([]int, 8)
	for i, v := range h_vecs {
		VPBROADCASTD(key.Offset(4*i), v.Get())
		h_regs[i] = v.Reg()
	}

	{
		// TODO: probably a better way to get these values on the stack
		c := func(n int) GPVirtual {
			r := GP64()
			if n > 0 {
				MOVQ(inc, r)
				ANDQ(U8(n), r)
				ADDQ(counter, r)
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
	MOVL(flags, block_flags)
	ORL(flags_start, block_flags)

	{
		Label("loop")
		CMPQ(blocks, Imm(0))
		JEQ(LabelRef("finalize"))

		Comment("Include end flags if last block")
		CMPQ(blocks, Imm(1))
		JNE(LabelRef("round_setup"))
		ORL(flags_end, block_flags)

		Label("round_setup")
		Comment("Load and transpose message vectors")
		transpose_msg_vecs_and_inc(c, alloc, inputs, msg_vecs)

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
			round(c, alloc, msg_vecs, r, vs)
		}

		Comment("Finalize rounds")
		for i := 0; i < 8; i++ {
			h_vecs[i] = alloc.Value()
			VPXOR(vs[i].ConsumeOp(), vs[8+i].Consume(), h_vecs[i].Get())
		}

		Comment("Fix up registers for next iteration")
		for i := 7; i >= 0; i-- {
			if h_vecs[i].Reg() != h_regs[i] {
				Commentf("YMM%-2d => YMM%-2d", h_vecs[i].Reg(), h_regs[i])
				h_vecs[i].Become(h_regs[i])
			}
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

func transpose_vecs(c ctx, alloc *Alloc, vs []*Value) {
	L01, H01, L23, H23 := alloc.Value(), alloc.Value(), alloc.Value(), alloc.Value()
	L45, H45, L67, H67 := alloc.Value(), alloc.Value(), alloc.Value(), alloc.Value()

	VPUNPCKLDQ(vs[1].GetOp(), vs[0].Get(), L01.Get())
	VPUNPCKHDQ(vs[1].ConsumeOp(), vs[0].Consume(), H01.Get())
	VPUNPCKLDQ(vs[3].GetOp(), vs[2].Get(), L23.Get())
	VPUNPCKHDQ(vs[3].ConsumeOp(), vs[2].Consume(), H23.Get())
	VPUNPCKLDQ(vs[5].GetOp(), vs[4].Get(), L45.Get())
	VPUNPCKHDQ(vs[5].ConsumeOp(), vs[4].Consume(), H45.Get())
	VPUNPCKLDQ(vs[7].GetOp(), vs[6].Get(), L67.Get())
	VPUNPCKHDQ(vs[7].ConsumeOp(), vs[6].Consume(), H67.Get())

	LL0123, HL0123, LH0123, HH0123 := alloc.Value(), alloc.Value(), alloc.Value(), alloc.Value()
	LL4567, HL4567, LH4567, HH4567 := alloc.Value(), alloc.Value(), alloc.Value(), alloc.Value()

	VPUNPCKLQDQ(L23.GetOp(), L01.Get(), LL0123.Get())
	VPUNPCKHQDQ(L23.ConsumeOp(), L01.Consume(), HL0123.Get())
	VPUNPCKLQDQ(H23.GetOp(), H01.Get(), LH0123.Get())
	VPUNPCKHQDQ(H23.ConsumeOp(), H01.Consume(), HH0123.Get())
	VPUNPCKLQDQ(L67.GetOp(), L45.Get(), LL4567.Get())
	VPUNPCKHQDQ(L67.ConsumeOp(), L45.Consume(), HL4567.Get())
	VPUNPCKLQDQ(H67.GetOp(), H45.Get(), LH4567.Get())
	VPUNPCKHQDQ(H67.ConsumeOp(), H45.Consume(), HH4567.Get())

	vs[0], vs[1], vs[2], vs[3] = alloc.Value(), alloc.Value(), alloc.Value(), alloc.Value()
	vs[4], vs[5], vs[6], vs[7] = alloc.Value(), alloc.Value(), alloc.Value(), alloc.Value()

	VINSERTI128(Imm(1), LL4567.GetOp(AsX), LL0123.Get(), vs[0].Get())
	VPERM2I128(Imm(49), LL4567.Consume(), LL0123.Consume(), vs[4].Get())
	VINSERTI128(Imm(1), HL4567.GetOp(AsX), HL0123.Get(), vs[1].Get())
	VPERM2I128(Imm(49), HL4567.Consume(), HL0123.Consume(), vs[5].Get())
	VINSERTI128(Imm(1), LH4567.GetOp(AsX), LH0123.Get(), vs[2].Get())
	VPERM2I128(Imm(49), LH4567.Consume(), LH0123.Consume(), vs[6].Get())
	VINSERTI128(Imm(1), HH4567.GetOp(AsX), HH0123.Get(), vs[3].Get())
	VPERM2I128(Imm(49), HH4567.Consume(), HH0123.Consume(), vs[7].Get())
}

func transpose_msg_vecs_and_inc(c ctx, alloc *Alloc, inputs, msg_vecs Mem) {
	vs := alloc.Values(8)
	for i, v := range vs {
		addr := GP64()
		MOVQ(inputs.Offset(8*i), addr)
		VMOVDQU(Mem{Base: addr}, v.Get())
	}
	transpose_vecs(c, alloc, vs)
	for i, v := range vs {
		VMOVDQU(v.Get(), msg_vecs.Offset(32*i))
	}

	for i, v := range vs {
		addr := GP64()
		MOVQ(inputs.Offset(8*i), addr)
		VMOVDQU(Mem{Base: addr}.Offset(32), v.Get())
	}
	transpose_vecs(c, alloc, vs)
	for i, v := range vs {
		VMOVDQU(v.Consume(), msg_vecs.Offset(256+32*i))
	}

	for i := range vs {
		ADDQ(U8(64), inputs.Offset(8*i))
	}
}

func addm(alloc *Alloc, mp Mem, a *Value) *Value {
	o := alloc.Value()
	VPADDD(mp, a.Consume(), o.Get())
	return o
}

func addms(alloc *Alloc, mps []Mem, as []*Value) {
	for i, a := range as {
		as[i] = addm(alloc, mps[i], a)
	}
}

func add(alloc *Alloc, a, b *Value) *Value {
	o := alloc.Value()

	switch {
	case a.Spilled() && !b.Spilled():
		VPADDD(a.GetOp(), b.Consume(), o.Get())
	case b.Spilled() && !a.Spilled():
		VPADDD(b.ConsumeOp(), a.Get(), o.Get())
	default: // TODO: spill inefficiency
		VPADDD(a.GetOp(), b.Consume(), o.Get())
	}

	return o
}

func adds(alloc *Alloc, as, bs []*Value) {
	for i, b := range bs {
		bs[i] = add(alloc, as[i], b)
	}
}

func xor(alloc *Alloc, a, b *Value) *Value {
	o := alloc.Value()

	switch {
	case a.Spilled() && !b.Spilled():
		VPXOR(a.GetOp(), b.Consume(), o.Get())
	case b.Spilled() && !a.Spilled():
		VPXOR(b.ConsumeOp(), a.Get(), o.Get())
	default: // TODO: spill inefficiency
		VPXOR(a.GetOp(), b.Consume(), o.Get())
	}

	return o
}

func xors(alloc *Alloc, as, bs []*Value) {
	for i, b := range bs {
		bs[i] = xor(alloc, as[i], b)
	}
}

func rotN(alloc *Alloc, n int, a *Value) *Value {
	tmp, o := alloc.Value(), alloc.Value()
	VPSRLD(U8(n), a.Get(), tmp.Get())
	VPSLLD(U8(32-n), a.Get(), a.Get())
	VPOR(tmp.ConsumeOp(), a.Consume(), o.Get())
	return o
}

func rotNs(alloc *Alloc, n int, as []*Value) {
	for i, a := range as {
		as[i] = rotN(alloc, n, a)
	}
}

func rotT(alloc *Alloc, tab Mem, a *Value) *Value {
	o := alloc.Value()
	VPSHUFB(tab, a.Consume(), o.Get())
	return o
}

func rotTs(alloc *Alloc, tab Mem, as []*Value) {
	for i, a := range as {
		as[i] = rotT(alloc, tab, a)
	}
}

func round(c ctx, alloc *Alloc, mp Mem, r int, vs []*Value) {
	m := func(n int) Mem { return mp.Offset(msgSched[r][n] * 32) }
	ms := func(ns ...int) (o []Mem) {
		for _, n := range ns {
			o = append(o, m(n))
		}
		return o
	}

	addms(alloc, ms(0, 2, 4, 6), vs[0:4])
	adds(alloc, vs[4:8], vs[0:4])
	xors(alloc, vs[0:4], vs[12:16])
	rotTs(alloc, c.rot16, vs[12:16])
	adds(alloc, vs[12:16], vs[8:12])
	xors(alloc, vs[8:12], vs[4:8])
	rotNs(alloc, 12, vs[4:8])
	addms(alloc, ms(1, 3, 5, 7), vs[0:4])
	adds(alloc, vs[4:8], vs[0:4])
	xors(alloc, vs[0:4], vs[12:16])
	rotTs(alloc, c.rot8, vs[12:16])
	adds(alloc, vs[12:16], vs[8:12])
	xors(alloc, vs[8:12], vs[4:8])
	rotNs(alloc, 7, vs[4:8])

	// roll the blocks
	vs[4], vs[5], vs[6], vs[7] = vs[5], vs[6], vs[7], vs[4]
	vs[8], vs[9], vs[10], vs[11] = vs[10], vs[11], vs[8], vs[9]
	vs[12], vs[13], vs[14], vs[15] = vs[15], vs[12], vs[13], vs[14]

	addms(alloc, ms(8, 10, 12, 14), vs[0:4])
	adds(alloc, vs[4:8], vs[0:4])
	xors(alloc, vs[0:4], vs[12:16])
	rotTs(alloc, c.rot16, vs[12:16])
	adds(alloc, vs[12:16], vs[8:12])
	xors(alloc, vs[8:12], vs[4:8])
	rotNs(alloc, 12, vs[4:8])
	addms(alloc, ms(9, 11, 13, 15), vs[0:4])
	adds(alloc, vs[4:8], vs[0:4])
	xors(alloc, vs[0:4], vs[12:16])
	rotTs(alloc, c.rot8, vs[12:16])
	adds(alloc, vs[12:16], vs[8:12])
	xors(alloc, vs[8:12], vs[4:8])
	rotNs(alloc, 7, vs[4:8])

	// roll the blocks
	vs[4], vs[5], vs[6], vs[7] = vs[7], vs[4], vs[5], vs[6]
	vs[8], vs[9], vs[10], vs[11] = vs[10], vs[11], vs[8], vs[9]
	vs[12], vs[13], vs[14], vs[15] = vs[13], vs[14], vs[15], vs[12]
}
