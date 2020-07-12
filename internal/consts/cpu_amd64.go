package consts

import (
	"golang.org/x/sys/cpu"
)

const IsLittleEndian = true

var (
	HasAVX2 = cpu.X86.HasAVX2

	// Note: some instructions don't seem available in the go assembler or avo. Until this
	// has been fixed, we also require AVX when we require SSE41
	HasSSE41 = cpu.X86.HasSSE41 && cpu.X86.HasAVX
)
