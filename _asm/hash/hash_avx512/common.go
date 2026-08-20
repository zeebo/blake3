package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/zeebo/blake3/_asm"
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
		ms   []Mem
		rotD int
		rotB int
	}{
		{ms(0, 2, 4, 6), 16, 12},
		{ms(1, 3, 5, 7), 8, 7},
		{ms(8, 10, 12, 14), 16, 12},
		{ms(9, 11, 13, 15), 8, 7},
	}

	for i, p := range partials {
		addms(alloc, p.ms, vs[0:4])

		for j := 0; j < 4; j++ {
			vs[0+j] = add(alloc, vs[4+j], vs[0+j])
			vs[12+j] = xor(alloc, vs[0+j], vs[12+j])
			vs[12+j] = rotN(alloc, p.rotD, vs[12+j])
		}

		for j := 0; j < 4; j++ {
			vs[8+j] = add(alloc, vs[12+j], vs[8+j])
			vs[4+j] = xor(alloc, vs[8+j], vs[4+j])
		}

		rotNs(alloc, p.rotB, vs[4:8])

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
	o := alloc.Value()
	VPRORD(U8(n), a.ConsumeOp(), o.Get())
	return o
}

func rotNs(alloc *Alloc, n int, as []*Value) {
	for i, a := range as {
		as[i] = rotN(alloc, n, a)
	}
}
