package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
)

func Hash8(c ctx) {
	TEXT("hash8_avx", NOSPLIT, "func(inputs *[8]*byte, blocks int, key *[8]uint32, counter, inc uint64, flags, flags_start, flags_end uint8, out *[256]byte, v, m *[16][8]uint32)")

	var (
		// inputs      = Mem{Base: Load(Param("inputs"), GP64())}
		// blocks      = Load(Param("blocks"), GP64())
		// key         = Mem{Base: Load(Param("key"), GP64())}
		// counter     = Load(Param("counter"), GP64())
		// inc         = Load(Param("inc"), GP64())
		// flags       = Load(Param("flags"), GP32())
		// flags_start = Load(Param("flags_start"), GP32())
		// flags_end   = Load(Param("flags_end"), GP32())
		// out = Mem{Base: Load(Param("out"), GP64())}

		vp = Mem{Base: Load(Param("v"), GP64())}
		mp = Mem{Base: Load(Param("m"), GP64())}
	)

	alloc := NewAlloc(AllocLocal(32))

	// v0, v1, v2, v3 := alloc.Value(), alloc.Value(), alloc.Value(), alloc.Value()
	// v4, v5, v6, v7 := alloc.Value(), alloc.Value(), alloc.Value(), alloc.Value()
	// _, _, _, _ = v0.Get(), v1.Get(), v2.Get(), v3.Get()
	// _, _, _, _ = v4.Get(), v5.Get(), v6.Get(), v7.Get()

	// v0, v1, v2, v3, v4, v5, v6, v7 = transpose(c, alloc, v0, v1, v2, v3, v4, v5, v6, v7)

	vs := [16]*Value{
		alloc.Value(), alloc.Value(), alloc.Value(), alloc.Value(),
		alloc.Value(), alloc.Value(), alloc.Value(), alloc.Value(),
		alloc.Value(), alloc.Value(), alloc.Value(), alloc.Value(),
		alloc.Value(), alloc.Value(), alloc.Value(), alloc.Value(),
	}

	for i, v := range vs {
		VMOVDQU(vp.Offset(32*i), v.Get())
	}

	for i := 0; i < 7; i++ {
		vs = round(c, alloc, mp, i, vs)
	}

	// { // just to use all the registers for allocation purposes
	// 	tmp := GP64()
	// 	MOVQ(inputs.Base, tmp)
	// 	MOVQ(blocks, tmp)
	// 	MOVQ(key.Base, tmp)
	// 	MOVQ(counter, tmp)
	// 	MOVQ(inc, tmp)
	// 	MOVL(flags, tmp.As32())
	// 	MOVL(flags_start, tmp.As32())
	// 	MOVL(flags_end, tmp.As32())
	// 	MOVQ(out.Base, tmp)
	// }

	Label("finalize")

	for i, v := range vs {
		VMOVDQU(v.Consume(), vp.Offset(32*i))
	}

	RET()
}

func transpose(c ctx, alloc *Alloc,
	v0, v1, v2, v3, v4, v5, v6, v7 *Value) (
	_, _, _, _, _, _, _, _ *Value) {

	L01, H01, L23, H23 := alloc.Value(), alloc.Value(), alloc.Value(), alloc.Value()
	L45, H45, L67, H67 := alloc.Value(), alloc.Value(), alloc.Value(), alloc.Value()

	VPUNPCKLDQ(v0.GetOp(), v1.Get(), L01.Get())
	VPUNPCKHDQ(v0.ConsumeOp(), v1.Consume(), H01.Get())
	VPUNPCKLDQ(v2.GetOp(), v3.Get(), L23.Get())
	VPUNPCKHDQ(v2.ConsumeOp(), v3.Consume(), H23.Get())
	VPUNPCKLDQ(v4.GetOp(), v5.Get(), L45.Get())
	VPUNPCKHDQ(v4.ConsumeOp(), v5.Consume(), H45.Get())
	VPUNPCKLDQ(v6.GetOp(), v7.Get(), L67.Get())
	VPUNPCKHDQ(v6.ConsumeOp(), v7.Consume(), H67.Get())

	LL0123, HL0123, LH0123, HH0123 := alloc.Value(), alloc.Value(), alloc.Value(), alloc.Value()
	LL4567, HL4567, LH4567, HH4567 := alloc.Value(), alloc.Value(), alloc.Value(), alloc.Value()

	VPUNPCKLQDQ(L01.GetOp(), L23.Get(), LL0123.Get())
	VPUNPCKHQDQ(L01.ConsumeOp(), L23.Consume(), HL0123.Get())
	VPUNPCKLQDQ(H01.GetOp(), H23.Get(), LH0123.Get())
	VPUNPCKHQDQ(H01.ConsumeOp(), H23.Consume(), HH0123.Get())
	VPUNPCKLQDQ(L45.GetOp(), L67.Get(), LL4567.Get())
	VPUNPCKHQDQ(L45.ConsumeOp(), L67.Consume(), HL4567.Get())
	VPUNPCKLQDQ(H45.GetOp(), H67.Get(), LH4567.Get())
	VPUNPCKHQDQ(H45.ConsumeOp(), H67.Consume(), HH4567.Get())

	v0, v1, v2, v3 = alloc.Value(), alloc.Value(), alloc.Value(), alloc.Value()
	v4, v5, v6, v7 = alloc.Value(), alloc.Value(), alloc.Value(), alloc.Value()

	VINSERTI128(Imm(1), LL4567.GetOp(AsX), LL0123.Get(), v0.Get())
	VPERM2I128(Imm(49), LL4567.Consume(), LL0123.Consume(), v4.Get())
	VINSERTI128(Imm(1), HL4567.GetOp(AsX), HL0123.Get(), v1.Get())
	VPERM2I128(Imm(49), HL4567.Consume(), HL0123.Consume(), v5.Get())
	VINSERTI128(Imm(1), LH4567.GetOp(AsX), LH0123.Get(), v2.Get())
	VPERM2I128(Imm(49), LH4567.Consume(), LH0123.Consume(), v6.Get())
	VINSERTI128(Imm(1), HH4567.GetOp(AsX), HH0123.Get(), v3.Get())
	VPERM2I128(Imm(49), HH4567.Consume(), HH0123.Consume(), v7.Get())

	return v0, v1, v2, v3, v4, v5, v6, v7
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

func round(c ctx, alloc *Alloc, mp Mem, r int, vs [16]*Value) (out [16]*Value) {
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

	return vs
}
