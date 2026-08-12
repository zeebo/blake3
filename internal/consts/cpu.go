//go:build !purego

package consts

import (
	"os"
	"runtime"

	"golang.org/x/sys/cpu"
)

var (
	HasAVX512 = cpu.X86.HasAVX512 && cpu.X86.HasAVX512VL &&
		os.Getenv("BLAKE3_DISABLE_AVX512") == "" &&
		os.Getenv("BLAKE3_PUREGO") == ""

	HasAVX2 = cpu.X86.HasAVX2 &&
		os.Getenv("BLAKE3_DISABLE_AVX2") == "" &&
		os.Getenv("BLAKE3_PUREGO") == ""

	HasSSE41 = cpu.X86.HasSSE41 &&
		os.Getenv("BLAKE3_DISABLE_SSE41") == "" &&
		os.Getenv("BLAKE3_PUREGO") == ""

	// NEON is part of the armv8-a baseline, so no cpuid check is needed.
	HasNEON = runtime.GOARCH == "arm64" &&
		os.Getenv("BLAKE3_DISABLE_NEON") == "" &&
		os.Getenv("BLAKE3_PUREGO") == ""
)
