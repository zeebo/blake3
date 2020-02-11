package avo

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

//
// used set
//

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

func (u used) mustAlloc() (n int) {
	n, ok := u.alloc(0)
	if !ok {
		panic("unable to alloc")
	}
	return n
}

func (u used) free(n int) {
	delete(u, n)
}

//
// alloc
//

type Alloc struct {
	m      Mem
	regs   used
	stack  used
	values map[int]*Value
	ctr    int
	spills int
	mslot  int
	phys   []VecPhysical
	span   int
}

func NewAlloc(m Mem) *Alloc {
	return &Alloc{
		m:      m,
		regs:   make(used),
		stack:  make(used),
		values: make(map[int]*Value),
		ctr:    0,
		spills: 0,
		mslot:  -1,
		phys:   ymmRegs[:],
		span:   32,
	}
}

func (a *Alloc) stats(name, when string) {
	fmt.Printf("// [%s] %s: %d/16 free (%d total + %d spills + %d slots)\n",
		name, when, 16-len(a.regs), len(a.values), a.spills, a.mslot+1)
}

func (a *Alloc) newStateLive(reg int) stateLive {
	return stateLive{Reg: reg, phys: a.phys}
}

func (a *Alloc) newStateSpilled(slot int) stateSpilled {
	return stateSpilled{Slot: slot, mem: a.m, span: a.span, aligned: true}
}

func (a *Alloc) Debug(name string) func() {
	a.stats(name, "in")
	return func() { a.stats(name, "out") }
}

func (a *Alloc) FreeReg() int {
	n, ok := a.regs.alloc(16)
	if !ok {
		return -1
	}
	a.regs.free(n)
	return n
}

func (a *Alloc) Free() {
	for id, v := range a.values {
		fmt.Println("leaked value:", id, "==", v.id, "\n", v.stack)
	}
}

func (a *Alloc) findOldestLive(except *Value) *Value {
	var oldest *Value
	for _, v := range a.values {
		if oldest == except || !v.state.Live() {
			continue
		}
		if oldest == nil || v.age < oldest.age {
			oldest = v
		}
	}
	return oldest
}

func (a *Alloc) allocSpot() valueState {
	reg, ok := a.regs.alloc(16)
	if ok {
		return a.newStateLive(reg)
	}
	slot := a.stack.mustAlloc()
	a.spills++
	if slot > a.mslot {
		a.mslot = slot
	}
	return a.newStateSpilled(slot)
}

func (a *Alloc) allocReg(except *Value) int {
	reg, ok := a.regs.alloc(16)
	if ok {
		return reg
	}
	oldest := a.findOldestLive(except)
	state := oldest.state.(stateLive)
	oldest.displaceTo(a.allocSpot())
	a.regs[state.Reg] = struct{}{}
	return state.Reg
}

func (a *Alloc) Value() *Value {
	a.ctr++

	var buf [4096]byte

	v := &Value{
		a:     a,
		id:    a.ctr,
		age:   a.ctr,
		stack: string(buf[:runtime.Stack(buf[:], false)]),
		reg:   -1,

		state: stateEmpty{},
	}
	a.values[v.id] = v

	return v
}

func (a *Alloc) Values(n int) []*Value {
	out := make([]*Value, n)
	for i := range out {
		out[i] = a.Value()
	}
	return out
}

func (a *Alloc) ValueFrom(m Mem) *Value {
	v := a.Value()
	v.state = stateLazy{Mem: m}
	return v
}

func (a *Alloc) ValuesFrom(n int, m Mem) []*Value {
	out := make([]*Value, n)
	for i := range out {
		out[i] = a.ValueFrom(m.Offset(a.span * i))
	}
	return out
}

func (a *Alloc) ValueWith(m Mem) *Value {
	v := a.Value()
	v.state = stateLazy{Mem: m, Broadcast: true}
	return v
}

func (a *Alloc) ValuesWith(n int, m Mem) []*Value {
	out := make([]*Value, n)
	for i := range out {
		out[i] = a.ValueWith(m.Offset(4 * i))
	}
	return out
}

//
// value states
//

type valueState interface {
	Op() Op
	Live() bool
	String() string

	ymmState()
}

type stateBase struct{}

func (stateBase) Op() Op         { panic("no location for this state") }
func (stateBase) Live() bool     { return false }
func (stateBase) String() string { return "Base" }

func (stateBase) ymmState() {}

type stateEmpty struct {
	stateBase
}

func (stateEmpty) String() string { return "Empty" }

type stateLive struct {
	stateBase

	Reg  int
	phys []VecPhysical
}

func (s stateLive) Op() Op         { return s.Register() }
func (s stateLive) Live() bool     { return true }
func (s stateLive) String() string { return fmt.Sprintf("Live(%d)", s.Reg) }

func (s stateLive) Register() VecPhysical { return s.phys[s.Reg] }

type stateSpilled struct {
	stateBase

	mem     Mem
	Slot    int
	span    int
	aligned bool
}

func (s stateSpilled) Op() Op         { return s.GetMem() }
func (s stateSpilled) String() string { return fmt.Sprintf("Spilled(%d)", s.Slot) }

func (s stateSpilled) GetMem() Mem { return s.mem.Offset(s.span * s.Slot) }

type stateLazy struct {
	stateBase

	Mem       Mem
	Broadcast bool
}

func (s stateLazy) String() string { return fmt.Sprintf("Lazy(%s, %t)", s.Mem.Asm(), s.Broadcast) }

//
// value
//

type Value struct {
	a     *Alloc
	id    int
	age   int
	stack string
	reg   int // currently allocated register (sometimes dup'd in state)

	state valueState
}

func (v *Value) Reg() int {
	if v.reg < 0 {
		v.reg = v.a.allocReg(v)
	}
	return v.reg
}

func (v *Value) setState(state valueState) {
	v.freeSpot()
	v.state = state
	v.useSpot()
}

func (v *Value) Become(reg int) {
	// if we already are/will be it: done.
	if v.reg == reg {
		return
	}

	// if it's free: displace to it.
	if _, ok := v.a.regs[reg]; !ok {
		v.a.regs[reg] = struct{}{}
		v.displaceTo(v.a.newStateLive(reg))
		return
	}

	// someone else owns it. displace them and then displace ourselves.
	for _, cand := range v.a.values {
		if cand.reg != reg {
			continue
		}
		state := cand.state
		cand.displaceTo(cand.a.allocSpot())
		v.displaceTo(state)
		return
	}
}

func (v *Value) displaceTo(dest valueState) {
	if state, ok := dest.(stateSpilled); ok && state.aligned {
		VMOVDQA(v.Get(), dest.Op())
	} else {
		VMOVDQU(v.Get(), dest.Op())
	}
	v.setState(dest)
}

func (v *Value) freeSpot() {
	switch state := v.state.(type) {
	case stateLive:
		v.a.regs.free(state.Reg)
		v.reg = -1
	case stateSpilled:
		v.a.stack.free(state.Slot)
	}
}

func (v *Value) useSpot() {
	switch state := v.state.(type) {
	case stateLive:
		v.a.regs[state.Reg] = struct{}{}
		v.reg = state.Reg
	case stateSpilled:
		v.a.stack[state.Slot] = struct{}{}
	}
}

func (v *Value) Free() {
	v.setState(nil)
	delete(v.a.values, v.id)
}

func (v *Value) Consume() VecPhysical {
	reg := v.Get()
	v.Free()
	return reg
}

func (v *Value) ConsumeOp() Op {
	op := v.GetOp()
	v.Free()
	return op
}

func (v *Value) HasReg() bool {
	return v.reg >= 0
}

func (v *Value) allocReg() int {
	if v.reg >= 0 {
		return v.reg
	}
	return v.a.allocReg(v)
}

func (v *Value) Touch() {
	v.a.ctr++
	v.age = v.a.ctr
}

func (v *Value) GetOp() Op {
	v.Touch()

	switch state := v.state.(type) {
	case stateLive:
	case stateSpilled:
		return state.GetMem()
	case stateLazy:
		if !state.Broadcast {
			return state.Mem
		}
		reg := v.allocReg()
		VPBROADCASTD(state.Mem, ymmRegs[reg])
		v.setState(v.a.newStateLive(reg))
	case stateEmpty:
		reg := v.allocReg()
		v.setState(v.a.newStateLive(reg))
	}

	return v.state.(stateLive).Register()
}

func (v *Value) Get() VecPhysical {
	v.Touch()

	switch state := v.state.(type) {
	case stateLive:
	case stateSpilled:
		reg := v.allocReg()
		if state.aligned {
			VMOVDQA(state.GetMem(), v.a.phys[reg])
		} else {
			VMOVDQU(state.GetMem(), v.a.phys[reg])
		}
		v.setState(v.a.newStateLive(reg))
	case stateLazy:
		reg := v.allocReg()
		v.setState(v.a.newStateLive(reg))
		if !state.Broadcast {
			VMOVDQU(state.Mem, v.state.(stateLive).Register())
		} else {
			VPBROADCASTD(state.Mem, v.state.(stateLive).Register())
		}
	case stateEmpty:
		reg := v.allocReg()
		v.setState(v.a.newStateLive(reg))
	}

	return v.state.(stateLive).Register()
}

func (v *Value) String() string {
	return fmt.Sprintf("Value(reg:%-2d state:%s)", v.reg, v.state)
}
