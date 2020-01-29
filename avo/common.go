package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
)

const (
	flag_chunkStart = 1 << 0
	flag_chunkEnd   = 1 << 1
	flag_parent     = 1 << 2
	flag_root       = 1 << 3
	flag_keyed      = 1 << 4
	flag_keyCtx     = 1 << 5
	flag_keyMat     = 1 << 6
)

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

func transpose_msg_vecs_and_inc(c ctx, alloc *Alloc, block GPVirtual, input, msg_vecs Mem) {
	vs := alloc.Values(8)
	for i, v := range vs {
		VMOVDQU(input.Offset(1024*i).Idx(block, 1), v.Get())
	}
	transpose_vecs(c, alloc, vs)
	for i, v := range vs {
		VMOVDQU(v.Get(), msg_vecs.Offset(32*i))
	}

	for i, v := range vs {
		VMOVDQU(input.Offset(1024*i+32).Idx(block, 1), v.Get())
	}
	transpose_vecs(c, alloc, vs)
	for i, v := range vs {
		VMOVDQU(v.Consume(), msg_vecs.Offset(32*i+256))
	}
}

func round(c ctx, alloc *Alloc, vs []*Value, r int, m func(n int) Mem) {
	ms := func(ns ...int) (o []Mem) {
		for _, n := range ns {
			o = append(o, m(msgSched[r][n]))
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
