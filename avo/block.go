package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
)

type block struct {
	r [4]VecVirtual
	v bool
	m Mem
	n int
	l int
}

func newBlock() *block {
	return &block{
		r: [4]VecVirtual{YMM(), YMM(), YMM(), YMM()},
		v: false,
	}
}

func (b *block) live() {
	b.v = true
	b.l = 0
}

func (b *block) dead() {
	b.v = false
	b.l = 0
}

func (b *block) assertDead() {
	if b.v {
		panic("invalid use of live block")
	}
}

func (b *block) assertLive() {
	if !b.v {
		panic("invalid use of dead block")
	}
}

func (b *block) load(m Mem, n int) *block {
	b.assertDead()
	defer b.live()

	b.m, b.n = m, n
	for i := 0; i < 4; i++ {
		VMOVDQU(b.m.Offset(0x80*b.n+0x20*i), b.r[i])
	}

	return b
}

func (b *block) store() {
	b.assertLive()
	defer b.dead()

	for i := 0; i < 4; i++ {
		VMOVDQU(b.r[i], b.m.Offset(0x80*b.n+0x20*i))
	}
}

func (b *block) roll(n int) *block {
	b.assertLive()

	b.l += n
	return b
}

func (b *block) idx(n int) VecVirtual {
	return b.r[(n+b.l)%4]
}

func (b *block) add(c *block) {
	b.assertLive()
	c.assertLive()

	for i := 0; i < 4; i++ {
		VPADDD(c.idx(i), b.idx(i), b.idx(i))
	}
}

func (b *block) xor(c *block) {
	b.assertLive()
	c.assertLive()

	for i := 0; i < 4; i++ {
		VPXOR(c.idx(i), b.idx(i), b.idx(i))
	}
}

func (b *block) rot(n int, tmp Op) {
	b.assertLive()

	for i := 0; i < 4; i++ {
		VPSRLD(U8(n), b.idx(i), tmp)
		VPSLLD(U8(32-n), b.idx(i), b.idx(i))
		VPOR(tmp, b.idx(i), b.idx(i))
	}
}

func (b *block) rotTable(tab Mem) {
	for i := 0; i < 4; i++ {
		VPSHUFB(tab, b.idx(i), b.idx(i))
	}
}
