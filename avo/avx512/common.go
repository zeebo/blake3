package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
	. "github.com/zeebo/blake3/avo"
)

var msgSched = [7][16]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
	{2, 6, 3, 10, 7, 0, 4, 13, 1, 11, 12, 5, 9, 14, 15, 8},
	{3, 4, 10, 12, 13, 2, 7, 14, 6, 5, 9, 0, 11, 15, 8, 1},
	{10, 7, 12, 9, 14, 3, 13, 15, 4, 0, 11, 2, 5, 8, 1, 6},
	{12, 13, 9, 11, 15, 10, 14, 8, 7, 2, 5, 3, 0, 1, 6, 4},
	{9, 14, 11, 5, 8, 12, 15, 1, 13, 3, 0, 10, 2, 6, 4, 7},
	{11, 15, 5, 0, 1, 9, 8, 6, 14, 10, 2, 12, 3, 4, 7, 13},
}

const roundSize = 32

const (
	flag_chunkStart = 1 << 0
	flag_chunkEnd   = 1 << 1
	flag_parent     = 1 << 2
)

func transpose(c Ctx, alloc *Alloc, vs []*Value) {
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

	VINSERTI128(Imm(1), LL4567.Get().(VecPhysical).AsX(), LL0123.Get(), vs[0].Get())
	VPERM2I128(Imm(49), LL4567.Consume(), LL0123.Consume(), vs[4].Get())
	VINSERTI128(Imm(1), HL4567.Get().(VecPhysical).AsX(), HL0123.Get(), vs[1].Get())
	VPERM2I128(Imm(49), HL4567.Consume(), HL0123.Consume(), vs[5].Get())
	VINSERTI128(Imm(1), LH4567.Get().(VecPhysical).AsX(), LH0123.Get(), vs[2].Get())
	VPERM2I128(Imm(49), LH4567.Consume(), LH0123.Consume(), vs[6].Get())
	VINSERTI128(Imm(1), HH4567.Get().(VecPhysical).AsX(), HH0123.Get(), vs[3].Get())
	VPERM2I128(Imm(49), HH4567.Consume(), HH0123.Consume(), vs[7].Get())
}

func transposeMsg(c Ctx, alloc *Alloc, block GPVirtual, input, msg Mem) {
	for j := 0; j < 2; j++ {
		vs := alloc.Values(8)
		for i, v := range vs {
			VMOVDQU(input.Offset(1024*i+32*j).Idx(block, 1), v.Get())
		}
		transpose(c, alloc, vs)
		for i, v := range vs {
			VMOVDQU(v.Consume(), msg.Offset(32*i+256*j))
		}
	}
}

func transposeMsgN(c Ctx, alloc *Alloc, block GPVirtual, input, msg Mem, j int) {
	vs := alloc.Values(8)
	for i, v := range vs {
		VMOVDQU(input.Offset(1024*i+32*j).Idx(block, 1), v.Get())
	}
	transpose(c, alloc, vs)
	for i, v := range vs {
		VMOVDQU(v.Consume(), msg.Offset(32*i+256*j))
	}
}

func loadCounter(c Ctx, alloc *Alloc, mem, lo_mem, hi_mem Mem) {
	ctr0, ctr1 := alloc.Value(), alloc.Value()
	VPBROADCASTQ(mem, ctr0.Get())
	VPADDQ(c.Counter, ctr0.Get(), ctr0.Get())
	VPBROADCASTQ(mem, ctr1.Get())
	VPADDQ(c.Counter.Offset(32), ctr1.Get(), ctr1.Get())

	L, H := alloc.Value(), alloc.Value()
	VPUNPCKLDQ(ctr1.GetOp(), ctr0.Get(), L.Get())
	VPUNPCKHDQ(ctr1.ConsumeOp(), ctr0.Consume(), H.Get())

	LLH, HLH := alloc.Value(), alloc.Value()
	VPUNPCKLDQ(H.GetOp(), L.Get(), LLH.Get())
	VPUNPCKHDQ(H.ConsumeOp(), L.Consume(), HLH.Get())

	ctrl, ctrh := alloc.Value(), alloc.Value()
	VPERMQ(U8(0b11_01_10_00), LLH.ConsumeOp(), ctrl.Get())
	VPERMQ(U8(0b11_01_10_00), HLH.ConsumeOp(), ctrh.Get())

	VMOVDQU(ctrl.Consume(), lo_mem)
	VMOVDQU(ctrh.Consume(), hi_mem)
}

func finalizeRounds(alloc *Alloc, vs, h_vecs []*Value, h_regs []int) {
	finalized := [8]bool{}

finalize:
	for j := 0; j < 8; j++ {
		free := alloc.FreeReg()
		for i, reg := range h_regs {
			if reg == free && !finalized[i] {
				h_vecs[i] = xorb(alloc, vs[i], vs[8+i])
				finalized[i] = true
				continue finalize
			}
		}

		for i, f := range finalized[:] {
			if !f {
				h_vecs[i] = xorb(alloc, vs[i], vs[8+i])
				finalized[i] = true
				continue finalize
			}
		}
	}
}

func round(c Ctx, alloc *Alloc, vs []*Value, r int, m func(n int) Mem) {
	ms := func(ns ...int) (o []Mem) {
		for _, n := range ns {
			o = append(o, m(msgSched[r][n]))
		}
		return o
	}

	partials := []struct {
		ms  []Mem
		tab Mem
		rot int
	}{
		{ms(0, 2, 4, 6), c.Rot16, 12},
		{ms(1, 3, 5, 7), c.Rot8, 7},
		{ms(8, 10, 12, 14), c.Rot16, 12},
		{ms(9, 11, 13, 15), c.Rot8, 7},
	}

	for i, p := range partials {
		addms(alloc, p.ms, vs[0:4])

		tab := alloc.ValueFrom(p.tab)
		for j := 0; j < 4; j++ {
			vs[0+j] = add(alloc, vs[4+j], vs[0+j])
			vs[12+j] = xor(alloc, vs[0+j], vs[12+j])
			vs[12+j] = rotTv(alloc, tab, vs[12+j])
		}
		tab.Free()

		for j := 0; j < 4; j++ {
			vs[8+j] = add(alloc, vs[12+j], vs[8+j])
			vs[4+j] = xor(alloc, vs[8+j], vs[4+j])
		}

		rotNs(alloc, p.rot, vs[4:8])

		// roll the blocks
		if i == 1 {
			vs[4], vs[5], vs[6], vs[7] = vs[5], vs[6], vs[7], vs[4]
			vs[8], vs[9], vs[10], vs[11] = vs[10], vs[11], vs[8], vs[9]
			vs[12], vs[13], vs[14], vs[15] = vs[15], vs[12], vs[13], vs[14]
		} else if i == 3 {
			vs[4], vs[5], vs[6], vs[7] = vs[7], vs[4], vs[5], vs[6]
			vs[8], vs[9], vs[10], vs[11] = vs[10], vs[11], vs[8], vs[9]
			vs[12], vs[13], vs[14], vs[15] = vs[13], vs[14], vs[15], vs[12]
		}
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
	VPADDD(a.Get(), b.Consume(), o.Get())
	return o
}

func xor(alloc *Alloc, a, b *Value) *Value {
	o := alloc.Value()
	VPXOR(a.Get(), b.Consume(), o.Get())
	return o
}

func xorb(alloc *Alloc, a, b *Value) *Value {
	o := alloc.Value()
	switch {
	case a.HasReg():
		VPXOR(b.ConsumeOp(), a.Consume(), o.Get())
	case b.HasReg():
		VPXOR(a.ConsumeOp(), b.Consume(), o.Get())
	default:
		VPXOR(a.ConsumeOp(), b.Consume(), o.Get())
	}
	return o
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

func rotTv(alloc *Alloc, tab, a *Value) *Value {
	o := alloc.Value()
	VPSHUFB(tab.GetOp(), a.Consume(), o.Get())
	return o
}
