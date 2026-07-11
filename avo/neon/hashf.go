package main

import "fmt"

// HashF scratch layout, R10-based and 16-byte aligned:
//
//	  0..255  transposed message vectors m[0..15]
//	256..271  counter-low lane vector
//	272..287  counter-high lane vector
//	288..415  chain-extraction stash of V0-V7
const (
	scratchSize = 432 // 416 bytes of scratch + 16 bytes of alignment slack
	msgOff      = 0
	ctrLoOff    = 256
	ctrHiOff    = 272
	stashOff    = 288
)

// Chunks map to vector lanes four at a time; each group runs a uniform block
// loop, and lanes past the input compute garbage that the caller ignores.
func emitHashF(a *asm) {
	a.line("")
	a.line("// func HashF(input *[8192]byte, length uint64, counter uint64, flags uint32, key *[8]uint32, out *[64]uint32, chain *[8]uint32)")
	a.line("TEXT ·HashF(SB), NOSPLIT, $%d-56", scratchSize)
	a.op("MOVD input+0(FP), R0")
	a.op("MOVD length+8(FP), R1")
	a.op("MOVD counter+16(FP), R2")
	a.op("MOVWU flags+24(FP), R3")
	a.op("MOVD key+32(FP), R4")
	a.op("MOVD out+40(FP), R5")
	a.op("MOVD chain+48(FP), R6")

	a.comment("16-byte aligned scratch base")
	a.op("MOVD $scratch-%d(SP), R10", scratchSize)
	a.op("ADD $15, R10, R10")
	a.op("AND $-16, R10, R10")

	a.comment("R7 = index of the last chunk, R8 = byte offset of its last block")
	a.op("MOVD ZR, R7")
	a.op("MOVD ZR, R8")
	a.op("CBZ R1, lenzero")
	a.op("SUB $1, R1, R7")
	a.op("LSR $10, R7, R7")
	a.op("SUB $1, R1, R8")
	a.op("AND $960, R8, R8")
	a.label("lenzero")

	a.op("ORR $%d, R3, R12", flagChunkStart)
	a.op("MOVD $iv_rows<>(SB), R15")
	a.op("MOVD $rot8_shuf<>(SB), R11")
	a.op("VLD1 (R11), [V31.B16]")
	a.op("MOVD $64, R16")
	a.op("AND $3, R7, R17")

	for g := range 2 {
		emitFGroup(a, g)
	}

	a.label("done")
	a.op("RET")
}

func emitFGroup(a *asm, g int) {
	sfx := fmt.Sprintf("_g%d", g)

	a.line("")
	a.comment("---- chunks %d-%d ----", 4*g, 4*g+3)
	if g == 1 {
		a.op("CMP $4096, R1")
		a.op("BLS done")
	}

	a.comment("counter lane vectors, split into low and high words")
	for i := range 4 {
		a.op("ADD $%d, R2, R25", 4*g+i)
		a.op("VMOV R25, V12.S[%d]", i)
		a.op("LSR $32, R25, R26")
		a.op("VMOV R26, V13.S[%d]", i)
	}
	a.op("FMOVQ F12, %d(R10)", ctrLoOff)
	a.op("FMOVQ F13, %d(R10)", ctrHiOff)

	emitKeyBroadcast(a, "R4")

	a.comment("lane input pointers")
	if g == 0 {
		a.op("MOVD R0, R19")
	} else {
		a.op("ADD $4096, R0, R19")
	}
	a.op("ADD $1024, R19, R20")
	a.op("ADD $1024, R20, R21")
	a.op("ADD $1024, R21, R22")

	a.comment("R23 = chain extraction offset, R24 = last block offset")
	if g == 0 {
		a.op("MOVD $-1, R23")
		a.op("CMP $4, R7")
		a.op("BGE trigdone" + sfx)
		a.op("MOVD R8, R23")
		a.label("trigdone" + sfx)
		a.op("MOVD $960, R24")
		a.op("CBNZ R7, lastdone" + sfx)
		a.op("MOVD R8, R24")
		a.label("lastdone" + sfx)
	} else {
		a.comment("group 1 only runs when 4 <= last chunk <= 7, so it always extracts")
		a.op("MOVD R8, R23")
		a.op("MOVD $960, R24")
		a.op("CMP $4, R7")
		a.op("BNE lastdone" + sfx)
		a.op("MOVD R8, R24")
		a.label("lastdone" + sfx)
	}

	a.op("MOVD ZR, R9")
	a.label("blockloop" + sfx)

	a.comment("extract the CV entering the block that holds the final byte")
	a.op("CMP R23, R9")
	a.op("BNE noextract" + sfx)
	for i := range 8 {
		a.op("FMOVQ F%d, %d(R10)", i, stashOff+16*i)
	}
	a.op("ADD R17<<2, R10, R25")
	for i := range 8 {
		a.op("MOVWU %d(R25), R26", stashOff+16*i)
		a.op("MOVW R26, %d(R6)", 4*i)
	}
	a.label("noextract" + sfx)

	a.comment("R11 = flags for this block")
	a.op("MOVD R3, R11")
	a.op("CBNZ R9, notfirst" + sfx)
	a.op("MOVD R12, R11")
	a.label("notfirst" + sfx)
	a.op("CMP $960, R9")
	a.op("BNE notlast" + sfx)
	a.op("ORR $%d, R11, R11", flagChunkEnd)
	a.label("notlast" + sfx)

	a.comment("load a 64-byte block per lane and transpose into m[0..15]")
	a.op("VLD1.P 64(R19), [V19.S4, V20.S4, V21.S4, V22.S4]")
	a.op("VLD1.P 64(R20), [V23.S4, V24.S4, V25.S4, V26.S4]")
	a.op("VLD1.P 64(R21), [V27.S4, V28.S4, V29.S4, V30.S4]")
	a.op("VLD1.P 64(R22), [V8.S4, V9.S4, V10.S4, V11.S4]")
	for w := range 4 {
		l0, l1, l2, l3 := 19+w, 23+w, 27+w, 8+w
		a.op("VZIP1 V%d.S4, V%d.S4, V12.S4", l1, l0)
		a.op("VZIP2 V%d.S4, V%d.S4, V13.S4", l1, l0)
		a.op("VZIP1 V%d.S4, V%d.S4, V14.S4", l3, l2)
		a.op("VZIP2 V%d.S4, V%d.S4, V15.S4", l3, l2)
		a.op("VZIP1 V14.D2, V12.D2, V16.D2")
		a.op("VZIP2 V14.D2, V12.D2, V17.D2")
		a.op("VZIP1 V15.D2, V13.D2, V18.D2")
		a.op("VZIP2 V15.D2, V13.D2, V12.D2")
		a.op("FMOVQ F16, %d(R10)", msgOff+16*(4*w+0))
		a.op("FMOVQ F17, %d(R10)", msgOff+16*(4*w+1))
		a.op("FMOVQ F18, %d(R10)", msgOff+16*(4*w+2))
		a.op("FMOVQ F12, %d(R10)", msgOff+16*(4*w+3))
	}

	a.comment("state rows 8-15: IV, counter, block length, flags")
	a.op("VLD1 (R15), [V8.S4, V9.S4, V10.S4, V11.S4]")
	a.op("FMOVQ %d(R10), F12", ctrLoOff)
	a.op("FMOVQ %d(R10), F13", ctrHiOff)
	a.op("VDUP R16, V14.S4")
	a.op("VDUP R11, V15.S4")

	emitRounds(a, func(a *asm, idx, vreg int) {
		a.op("FMOVQ %d(R10), F%d", msgOff+16*idx, vreg)
	})

	a.comment("chain feed-forward")
	emitFeedForward(a)

	a.op("CMP R24, R9")
	a.op("BEQ groupdone" + sfx)
	a.op("ADD $64, R9")
	a.op("B blockloop" + sfx)

	a.label("groupdone" + sfx)
	a.comment("store transposed output rows")
	for i := range 8 {
		a.op("FMOVQ F%d, %d(R5)", i, 32*i+16*g)
	}
}
