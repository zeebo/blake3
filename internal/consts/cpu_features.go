//go:build !js
// +build !js

package consts

import (
	"golang.org/x/sys/cpu"
)

var (
	CPUHasAVX2  = cpu.X86.HasAVX2
	CPUHasSSE41 = cpu.X86.HasSSE41
)
