package main

import (
	"fmt"
	"io"
)

type asm struct {
	w   io.Writer
	err error
}

func (a *asm) writef(format string, args ...interface{}) {
	if a.err == nil {
		_, a.err = fmt.Fprintf(a.w, format, args...)
	}
}

func (a *asm) line(format string, args ...interface{}) {
	a.writef(format+"\n", args...)
}

func (a *asm) op(format string, args ...interface{}) {
	a.writef("\t"+format+"\n", args...)
}

// XAR Zdn.S, Zdn.S, Zm.S, #rot; Go's arm64 assembler has no SVE mnemonics.
// https://developer.arm.com/documentation/ddi0602/latest/SVE-Instructions/XAR--Bitwise-exclusive-OR-and-rotate-right-by-immediate-
func (a *asm) xar(dn, m, rot int) {
	const esize = 32
	if dn < 0 || dn > 31 || m < 0 || m > 31 || rot < 1 || rot > esize {
		panic(fmt.Sprintf("XAR Z%d.S, Z%d.S, Z%d.S, $%d out of range", dn, dn, m, rot))
	}
	// let rot : integer = (2 * esize) - UInt(tsize::imm3);
	rotEnc := 2*esize - rot
	tsize, imm3 := rotEnc>>3, rotEnc&0b111
	// let tsize : bits(4) = tszh::tszl;
	tszh, tszl := tsize>>2, tsize&0b11
	//          31-24         23-22      21      20-19      18-16        15-10       9-5   4-0
	word := 0b00000100<<24 | tszh<<22 | 1<<21 | tszl<<19 | imm3<<16 | 0b001101<<10 | m<<5 | dn
	a.op("WORD $0x%08x // XAR Z%d.S, Z%d.S, Z%d.S, $%d", word, dn, dn, m, rot)
}

func (a *asm) label(name string) {
	a.writef("\n%s:\n", name)
}

func (a *asm) comment(format string, args ...interface{}) {
	a.writef("\t// "+format+"\n", args...)
}
