package main

import (
	"fmt"
	"runtime"

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
	spills int
	mslot  int
}

func NewAlloc(m Mem) *Alloc {
	return &Alloc{
		m:      m,
		regs:   make(used),
		slots:  make(used),
		values: make(map[int]*Value),
		ctr:    0,
		spills: 0,
		mslot:  -1,
	}
}

func (a *Alloc) stats(name, when string) {
	fmt.Printf("// [%s] %s: %d/16 free (%d total + %d spills + %d slots)\n",
		name, when, 16-len(a.regs), len(a.values), a.spills, a.mslot+1)
}

func (a *Alloc) Debug(name string) func() {
	a.stats(name, "in")
	return func() { a.stats(name, "out") }
}

func (a *Alloc) Free() {
	for id, v := range a.values {
		fmt.Println("leaked value:", id, "==", v.id, "\n", v.stack)
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
	stack string
	free  bool
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
	a.spill(oldest)
	return n
}

func (a *Alloc) spill(v *Value) {
	n := v.regn
	v.slotn, _ = a.slots.alloc(0)
	v.regn = -1
	v.state = ymmState_spilled

	a.spills++
	VMOVDQU(ymmRegs[n], a.m.Offset(v.slotn*32))
	if v.slotn > a.mslot {
		a.mslot = v.slotn
	}
}

func (a *Alloc) Value() *Value {
	a.ctr++

	var buf [4096]byte

	v := &Value{
		a:     a,
		id:    a.ctr,
		age:   a.ctr,
		state: ymmState_empty,
		stack: string(buf[:runtime.Stack(buf[:], false)]),
	}
	a.values[v.id] = v

	return v
}

func (v *Value) Reg() int {
	_ = v.Get()
	return v.regn
}

func (v *Value) Become(regn int) {
	if v.state == ymmState_live && v.regn == regn {
		return
	}

	for _, v := range v.a.values {
		if v.regn == regn {
			n, ok := v.a.regs.alloc(16)
			if !ok {
				v.a.spill(v)
			} else {
				delete(v.a.regs, v.regn)
				v.a.regs[n] = struct{}{}
				VMOVDQU(ymmRegs[v.regn], ymmRegs[n])
				v.regn = n
			}
			break
		}
	}

	if v.state == ymmState_live {
		delete(v.a.regs, v.regn)
	}
	v.a.regs[regn] = struct{}{}
	VMOVDQU(v.GetOp(), ymmRegs[regn])
	v.regn = regn
	v.state = ymmState_live
}

func (a *Alloc) Values(n int) []*Value {
	out := make([]*Value, n)
	for i := range out {
		out[i] = a.Value()
	}
	return out
}

func (v *Value) Free() {
	if v.state == ymmState_spilled {
		v.a.slots.free(v.slotn)
	}
	if v.state == ymmState_live {
		v.a.regs.free(v.regn)
	}
	delete(v.a.values, v.id)
	v.free = true
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
