package main

// The transposed layout makes message vector m[j] for four consecutive
// parents a contiguous quad in left (j < 8) or right (j >= 8), so message
// loads are single FMOVQs and no transpose or stack is needed.
func emitHashP(a *asm) {
	a.line("")
	a.line("// func HashP(left *[64]uint32, right *[64]uint32, flags uint32, key *[8]uint32, out *[64]uint32, n int)")
	a.line("TEXT ·HashP(SB), NOSPLIT, $0-48")
	a.op("MOVD left+0(FP), R0")
	a.op("MOVD right+8(FP), R1")
	a.op("MOVWU flags+16(FP), R2")
	a.op("MOVD key+24(FP), R3")
	a.op("MOVD out+32(FP), R4")
	a.op("MOVD n+40(FP), R5")
	a.op("MOVD $iv_rows<>(SB), R15")
	a.op("MOVD $64, R16")

	for g := 0; g < 2; g++ {
		a.line("")
		a.comment("---- parents %d-%d ----", 4*g, 4*g+3)
		if g == 1 {
			a.op("CMP $4, R5")
			a.op("BLE done")
		}
		emitKeyBroadcast(a, "R3")
		a.op("VLD1 (R15), [V8.S4, V9.S4, V10.S4, V11.S4]")
		a.op("VEOR V12.B16, V12.B16, V12.B16")
		a.op("VEOR V13.B16, V13.B16, V13.B16")
		a.op("VDUP R16, V14.S4")
		a.op("VDUP R2, V15.S4")

		emitRounds(a, func(a *asm, idx, vreg int) {
			if idx < 8 {
				a.op("FMOVQ %d(R0), F%d", 32*idx+16*g, vreg)
			} else {
				a.op("FMOVQ %d(R1), F%d", 32*(idx-8)+16*g, vreg)
			}
		})

		emitFeedForward(a)
		for i := 0; i < 8; i++ {
			a.op("FMOVQ F%d, %d(R4)", i, 32*i+16*g)
		}
	}

	a.label("done")
	a.op("RET")
}
