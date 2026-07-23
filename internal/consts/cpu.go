//go:build !purego

package consts

import (
	"os"
	"runtime"

	"github.com/klauspost/cpuid/v2"
)

var (
	HasAVX2 = cpuid.CPU.Has(cpuid.AVX2) &&
		os.Getenv("BLAKE3_DISABLE_AVX2") == "" &&
		os.Getenv("BLAKE3_PUREGO") == ""

	HasSSE41 = cpuid.CPU.Has(cpuid.SSE4) &&
		os.Getenv("BLAKE3_DISABLE_SSE41") == "" &&
		os.Getenv("BLAKE3_PUREGO") == ""

	// NEON is part of the armv8-a baseline, so no cpuid check is needed.
	HasNEON = runtime.GOARCH == "arm64" &&
		os.Getenv("BLAKE3_DISABLE_NEON") == "" &&
		os.Getenv("BLAKE3_PUREGO") == ""
)
