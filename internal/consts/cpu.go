package consts

import "golang.org/x/sys/cpu"

var (
	HasAVX2  = cpu.X86.HasAVX2
	HasSSE41 = cpu.X86.HasSSE41
)
