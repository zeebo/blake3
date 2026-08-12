package main

var gStates = [8][4]int{
	{0, 4, 8, 12}, {1, 5, 9, 13}, {2, 6, 10, 14}, {3, 7, 11, 15},
	{0, 5, 10, 15}, {1, 6, 11, 12}, {2, 7, 8, 13}, {3, 4, 9, 14},
}

type loadMsg func(a *asm, idx, vreg int)

func emitG(a *asm, va, vb, vc, vd, mx, my int, load loadMsg) {
	load(a, mx, 16)
	a.op("VADD V%d.S4, V%d.S4, V%d.S4", vb, va, va)
	a.op("VADD V16.S4, V%d.S4, V%d.S4", va, va)
	a.xar(vd, va, 16)
	a.op("VADD V%d.S4, V%d.S4, V%d.S4", vd, vc, vc)
	a.xar(vb, vc, 12)
	load(a, my, 17)
	a.op("VADD V%d.S4, V%d.S4, V%d.S4", vb, va, va)
	a.op("VADD V17.S4, V%d.S4, V%d.S4", va, va)
	a.xar(vd, va, 8)
	a.op("VADD V%d.S4, V%d.S4, V%d.S4", vd, vc, vc)
	a.xar(vb, vc, 7)
}

func emitRounds(a *asm, load loadMsg) {
	for r := range msgSchedule {
		a.comment("round %d", r+1)
		s := msgSchedule[r]
		for gi, st := range gStates {
			emitG(a, st[0], st[1], st[2], st[3], s[2*gi], s[2*gi+1], load)
		}
	}
}

func emitKeyBroadcast(a *asm, keyReg string) {
	a.op("VLD1 (%s), [V28.S4, V29.S4]", keyReg)
	for i := 0; i < 8; i++ {
		a.op("VDUP V%d.S[%d], V%d.S4", 28+i/4, i%4, i)
	}
}

func emitFeedForward(a *asm) {
	for i := 0; i < 8; i++ {
		a.op("VEOR V%d.B16, V%d.B16, V%d.B16", 8+i, i, i)
	}
}
