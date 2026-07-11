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

func (a *asm) label(name string) {
	a.writef("\n%s:\n", name)
}

func (a *asm) comment(format string, args ...interface{}) {
	a.writef("\t// "+format+"\n", args...)
}
