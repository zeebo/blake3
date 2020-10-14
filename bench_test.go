package blake3

import (
	"fmt"
	"sync"
	"testing"

	"github.com/zeebo/blake3/internal/alg"
	"github.com/zeebo/blake3/internal/consts"
)

func BenchmarkBLAKE3(b *testing.B) {
	out := make([]byte, 32)
	buf := make([]byte, 1024*1024+512)
	pool := sync.Pool{
		New: func() interface{} { return new(hasher) },
	}

	runIncr := func(b *testing.B, size int) {
		buf := buf[:size]

		b.ReportAllocs()
		b.SetBytes(int64(len(buf)))
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			h := new(hasher)
			t := buf
			for len(t) >= 1024 {
				h.update(t[:1024])
				t = t[1024:]
			}
			if len(t) > 0 {
				h.update(t)
			}
			h.finalize(out)
		}
	}

	runEntire := func(b *testing.B, size int) {
		buf := buf[:size]

		b.ReportAllocs()
		b.SetBytes(int64(len(buf)))
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			h := new(hasher)
			h.update(buf)
			h.finalize(out)
		}
	}

	runReset := func(b *testing.B, size int) {
		buf := buf[:size]

		b.ReportAllocs()
		b.SetBytes(int64(len(buf)))
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			h := pool.Get().(*hasher)
			h.reset()
			h.update(buf)
			h.finalize(out)
			pool.Put(h)
		}
	}

	for _, kind := range []struct {
		name string
		run  func(b *testing.B, size int)
	}{
		{"Incremental", runIncr},
		{"Entire", runEntire},
		{"Reset", runReset},
	} {
		b.Run(kind.name, func(b *testing.B) {
			run := kind.run

			for _, n := range []int{
				1, 4, 8, 12, 16,
			} {
				b.Run(fmt.Sprintf("%04d_block", n), func(b *testing.B) { run(b, n*64) })
			}
			for _, n := range []int{
				1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024,
			} {
				b.Run(fmt.Sprintf("%04d_kib", n), func(b *testing.B) { run(b, n*1024) })
			}
			for _, n := range []int{
				1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024,
			} {
				b.Run(fmt.Sprintf("%04d_kib+512", n), func(b *testing.B) { run(b, n*1024+512) })
			}
		})
	}
}

func BenchmarkHashF_1(b *testing.B) {
	var input [8192]byte
	var chain [8]uint32
	var out chainVector

	b.SetBytes(1)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		alg.HashF(&input, 1, 0, 0, &consts.IV, &out, &chain)
	}
}

func BenchmarkHashF_1536(b *testing.B) {
	var input [8192]byte
	var chain [8]uint32
	var out chainVector

	b.SetBytes(1536)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		alg.HashF(&input, 1536, 0, 0, &consts.IV, &out, &chain)
	}
}

func BenchmarkHashF_8K(b *testing.B) {
	var input [8192]byte
	var chain [8]uint32
	var out chainVector

	b.SetBytes(8192)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		alg.HashF(&input, 8192, 0, 0, &consts.IV, &out, &chain)
	}
}

func BenchmarkHashP(b *testing.B) {
	var left chainVector
	var right chainVector
	var out chainVector

	for n := 1; n <= 8; n++ {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.SetBytes(int64(64 * n))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				alg.HashP(&left, &right, 0, &consts.IV, &out, n)
			}
		})
	}
}

func BenchmarkCompress(b *testing.B) {
	var c [8]uint32
	var m, o [16]uint32

	b.SetBytes(64)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		alg.Compress(&c, &m, 0, 0, 0, &o)
	}
}
