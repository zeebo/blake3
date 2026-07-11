package main

import (
	"fmt"
	"io"
)

type asm struct {
	w io.Writer
}

func (a *asm) line(format string, args ...any) {
	fmt.Fprintf(a.w, format+"\n", args...)
}

func (a *asm) op(format string, args ...any) {
	fmt.Fprintf(a.w, "\t"+format+"\n", args...)
}

func (a *asm) label(name string) {
	fmt.Fprintf(a.w, "\n%s:\n", name)
}

func (a *asm) comment(format string, args ...any) {
	fmt.Fprintf(a.w, "\t// "+format+"\n", args...)
}
