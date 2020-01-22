package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
)

var ymmRegs = [...]VecPhysical{
	Y0, Y1, Y2, Y3,
	Y4, Y5, Y6, Y7,
	Y8, Y9, Y10, Y11,
	Y12, Y13, Y14, Y15,
}

type used map[int]struct{}

func (u used) alloc(max int) (n int, ok bool) {
	for max == 0 || n < max {
		if _, ok := u[n]; !ok {
			u[n] = struct{}{}
			return n, true
		}
		n++
	}
	return 0, false
}

func (u used) free(n int) {
	delete(u, n)
}

type Alloc struct {
	m      Mem
	regs   used
	slots  used
	values map[int]*Value
	ctr    int
}

func NewAlloc(m Mem) *Alloc {
	return &Alloc{
		m:      m,
		regs:   make(used),
		slots:  make(used),
		values: make(map[int]*Value),
		ctr:    0,
	}
}

type ymmState int

const (
	ymmState_empty ymmState = iota
	ymmState_live
	ymmState_spilled
)

type Value struct {
	a     *Alloc
	id    int
	age   int
	state ymmState
	regn  int
	slotn int
}

func (a *Alloc) spillOldest(except *Value) int {
	var oldest *Value
	for _, val := range a.values {
		if val.state != ymmState_live || oldest == except {
			continue
		}
		if oldest == nil || val.age < oldest.age {
			oldest = val
		}
	}

	n := oldest.regn
	oldest.slotn, _ = a.slots.alloc(0)
	oldest.regn = -1
	oldest.state = ymmState_spilled

	VMOVDQU(ymmRegs[n], a.m.Offset(oldest.slotn*32))

	return n
}

func (a *Alloc) Value() *Value {
	a.ctr++

	v := &Value{
		a:     a,
		id:    a.ctr,
		age:   a.ctr,
		state: ymmState_empty,
	}
	a.values[v.id] = v

	return v
}

func (a *Alloc) StackValue(m Mem) *Value {
	a.ctr++

	v := &Value{
		a:     a,
		id:    a.ctr,
		age:   a.ctr,
		state: ymmState_empty,
	}
	a.values[v.id] = v

	return v
}

func (v *Value) Free() {
	if v.state == ymmState_spilled {
		v.a.slots.free(v.slotn)
	}
	if v.state == ymmState_live {
		v.a.regs.free(v.regn)
	}
	delete(v.a.values, v.id)
}

type XF func(VecPhysical) VecPhysical

func AsX(p VecPhysical) VecPhysical { return p.AsX().(VecPhysical) }

func (v *Value) Spilled() bool {
	return v.state == ymmState_spilled
}

func (v *Value) Consume() VecPhysical {
	reg := v.Get()
	v.Free()
	return reg
}

func (v *Value) ConsumeOp(xfs ...XF) Op {
	op := v.GetOp(xfs...)
	v.Free()
	return op
}

func (v *Value) GetOp(xfs ...XF) Op {
	v.a.ctr++
	v.age = v.a.ctr

	if v.state == ymmState_spilled {
		return v.a.m.Offset(v.slotn * 32)
	} else if v.state == ymmState_empty {
		n, ok := v.a.regs.alloc(16)
		if !ok {
			n = v.a.spillOldest(v)
		}
		v.regn = n
		v.slotn = -1
		v.state = ymmState_live
	}

	vp := ymmRegs[v.regn]
	for _, xf := range xfs {
		vp = xf(vp)
	}
	return vp
}

func (v *Value) Get() VecPhysical {
	v.a.ctr++
	v.age = v.a.ctr

	if v.state != ymmState_live {
		n, ok := v.a.regs.alloc(16)
		if !ok {
			n = v.a.spillOldest(v)
		}
		v.regn = n
		if v.state == ymmState_spilled {
			VMOVDQU(v.a.m.Offset(v.slotn*32), ymmRegs[v.regn])
			v.a.slots.free(v.slotn)
		}
		v.slotn = -1
		v.state = ymmState_live
	}

	return ymmRegs[v.regn]
}
