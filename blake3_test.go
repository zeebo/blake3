package blake3

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/zeebo/assert"
)

func TestVectors(t *testing.T) {
	for _, tv := range vectors {
		h := newHasher()
		h.update(tv.input())
		for j := 0; j < len(tv.hash)/2; j++ {
			buf := make([]byte, j)
			h.finalize(buf)
			assert.Equal(t, tv.hash[:2*j], hex.EncodeToString(buf))
		}
	}
}

func TestHashF(t *testing.T) {
	for n := 64; n <= 8192; n++ {
		var input [8192]byte
		for i := 0; i < n; i++ {
			input[i] = byte(i+1) % 251
		}

		var out cv
		hashF_avx(&input, uint64(n), 0, 0, &out)

		for i := 0; i < 8 && i*1024 < n; i++ {
			high := 1024 * (i + 1)
			if high > n {
				high = n
			}

			chain := [8]uint32{iv0, iv1, iv2, iv3, iv4, iv5, iv6, iv7}
			chunk := newChunkState(chain, uint64(i), 0)
			chunk.update(input[1024*i : high])
			output := chunk.output()
			exp := output.compress()

			var got [8]uint32
			for j := 0; j < 8; j++ {
				got[j] = out[8*j+i]
			}

			assert.Equal(t, exp, got)
		}
	}
}

func TestHashP(t *testing.T) {
	var data [8][16]uint32
	for i := range data {
		for j := range data[i] {
			data[i][j] = uint32(i + j)
		}
	}

	var left, right, out cv
	for i, ents := range data {
		for j, val := range ents {
			if j < 8 {
				left[8*i+j] = val
			} else {
				right[8*i+(j-8)] = val
			}
		}
	}

	hashP_avx(&left, &right, 0, &out)
	var got [8][8]uint32
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			got[i][j] = out[8*i+j]
		}
	}

	for i := 0; i < 8; i++ {
		key := [8]uint32{iv0, iv1, iv2, iv3, iv4, iv5, iv6, iv7}
		var a, b [8]uint32
		copy(a[:], data[i][0:])
		copy(b[:], data[i][8:])
		exp := parentChain(a, b, key, 0)

		for j := 0; j < 8; j++ {
			assert.Equal(t, exp[j], got[j][i])
		}
	}
}

func TestMovcol(t *testing.T) {
	patterns := []uint32{
		0x11111111, 0x22222222, 0x33333333, 0x44444444,
		0x55555555, 0x66666666, 0x77777777, 0x88888888,
	}

	var buf cv
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			buf[8*i+j] = patterns[j]
		}
	}

	for icol := 0; icol < 8; icol++ {
		for ocol := 0; ocol < 8; ocol++ {
			var out cv
			movcol(&buf, uint64(icol), &out, uint64(ocol))

			var exp cv
			for i := 0; i < 8; i++ {
				exp[8*i+ocol] = patterns[icol]
			}

			assert.Equal(t, out, exp)
		}
	}
}

func TestVectorsAVX2(t *testing.T) {
	t.SkipNow()

	for _, tv := range vectors {
		var h avxHasher
		h.update(tv.input())
		for j := 0; j < len(tv.hash)/2; j++ {
			buf := make([]byte, j)
			h.finalize(buf)
			assert.Equal(t, tv.hash[:2*j], hex.EncodeToString(buf))
		}
	}
}

func BenchmarkHashF_1(b *testing.B) {
	var input [8192]byte
	var out cv

	b.SetBytes(1)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hashF_avx(&input, 1, 0, 0, &out)
	}
}

func BenchmarkHashF_8K(b *testing.B) {
	var input [8192]byte
	var out cv

	b.SetBytes(8192)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hashF_avx(&input, 8192, 0, 0, &out)
	}
}

func BenchmarkHashP(b *testing.B) {
	var left cv
	var right cv
	var out cv

	b.SetBytes(512)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hashP_avx(&left, &right, 0, &out)
	}
}

func BenchmarkMovcol(b *testing.B) {
	var input cv
	var out cv

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		movcol(&input, uint64(i)%8, &out, uint64(i+1)%8)
	}
}

func BenchmarkCompress(b *testing.B) {
	var s, m [16]uint32

	b.SetBytes(64)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rcompress(&s, &m)
	}
}

func BenchmarkBasic(b *testing.B) {
	sizes := []int64{0, 16, 32, 64, 128, 256, 512, 1024, 4 * 1024, 8 * 1024}

	for _, size := range sizes {
		size := size
		input := make([]byte, size)

		b.Run(fmt.Sprint(size), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(size)

			for i := 0; i < b.N; i++ {
				var buf [32]byte
				h := newHasher()
				h.update(input)
				h.finalize(buf[:])
			}
		})
	}
}

func BenchmarkBasic_AVX2(b *testing.B) {
	sizes := []int64{8*1024 - 1, 8 * 1024, 16 * 1024, 32 * 1024, 64 * 1024}

	for _, size := range sizes {
		size := size
		input := make([]byte, size)

		b.Run(fmt.Sprint(size), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(size)

			for i := 0; i < b.N; i++ {
				var buf [32]byte
				var h avxHasher
				h.update(input)
				h.finalize(buf[:])
			}
		})
	}
}
