package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
)

func AVX(c ctx) {
	TEXT("round_avx", NOSPLIT, "func(v, m *[16][8]uint32)")
	vp := Mem{Base: Load(Param("v"), GP64())}
	mp := Mem{Base: Load(Param("m"), GP64())}
	t := AllocLocal(32)

	bs := [4]*block{
		newBlock().load(vp, 0),
		newBlock().load(vp, 1),
		newBlock().load(vp, 2),
		newBlock().load(vp, 3),
	}

	c.round(vp, mp, t, 0, bs)
	c.round(vp, mp, t, 1, bs)
	c.round(vp, mp, t, 2, bs)
	c.round(vp, mp, t, 3, bs)
	c.round(vp, mp, t, 4, bs)
	c.round(vp, mp, t, 5, bs)
	c.round(vp, mp, t, 6, bs)

	bs[0].store()
	bs[1].store()
	bs[2].store()
	bs[3].store()

	RET()
}

var msgSched = [7][16]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
	{2, 6, 3, 10, 7, 0, 4, 13, 1, 11, 12, 5, 9, 14, 15, 8},
	{3, 4, 10, 12, 13, 2, 7, 14, 6, 5, 9, 0, 11, 15, 8, 1},
	{10, 7, 12, 9, 14, 3, 13, 15, 4, 0, 11, 2, 5, 8, 1, 6},
	{12, 13, 9, 11, 15, 10, 14, 8, 7, 2, 5, 3, 0, 1, 6, 4},
	{9, 14, 11, 5, 8, 12, 15, 1, 13, 3, 0, 10, 2, 6, 4, 7},
	{11, 15, 5, 0, 1, 9, 8, 6, 14, 10, 2, 12, 3, 4, 7, 13},
}

func (c ctx) round(vp, mp, t Mem, r int, bs [4]*block) {
	addm := func(b *block, x [4]int) {
		for i := 0; i < 4; i++ {
			VPADDD(mp.Offset(msgSched[r][x[i]]*32), b.r[i], b.r[i])
		}
	}

	stash := func(in VecVirtual, cb func(temp VecVirtual)) {
		VMOVDQU(in, t)
		cb(in)
		VMOVDQU(t, in)
	}

	{
		addm(bs[0], [4]int{0, 2, 4, 6})
		bs[0].add(bs[1])
		bs[3].xor(bs[0])
		bs[3].rotTable(c.rot16)
		bs[2].add(bs[3])
		bs[1].xor(bs[2])
		stash(bs[0].r[0], func(temp VecVirtual) {
			bs[1].rot(12, temp)
		})

		addm(bs[0], [4]int{1, 3, 5, 7})
		bs[0].add(bs[1])
		bs[3].xor(bs[0])
		bs[3].rotTable(c.rot8)
		bs[2].add(bs[3])
		bs[1].xor(bs[2])
		stash(bs[0].r[0], func(temp VecVirtual) {
			bs[1].rot(7, temp)
		})
	}

	{
		bs[0].roll(0)
		bs[1].roll(1)
		bs[2].roll(2)
		bs[3].roll(3)
	}

	{
		addm(bs[0], [4]int{8, 10, 12, 14})
		bs[0].add(bs[1])
		bs[3].xor(bs[0])
		bs[3].rotTable(c.rot16)
		bs[2].add(bs[3])
		bs[1].xor(bs[2])
		stash(bs[0].r[0], func(temp VecVirtual) {
			bs[1].rot(12, temp)
		})

		addm(bs[0], [4]int{9, 11, 13, 15})
		bs[0].add(bs[1])
		bs[3].xor(bs[0])
		bs[3].rotTable(c.rot8)
		bs[2].add(bs[3])
		bs[1].xor(bs[2])
		stash(bs[0].r[0], func(temp VecVirtual) {
			bs[1].rot(7, temp)
		})
	}

	{
		bs[0].roll(4)
		bs[1].roll(3)
		bs[2].roll(2)
		bs[3].roll(1)
	}
}
