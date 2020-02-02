package blake3

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"runtime"
	"testing"

	"github.com/zeebo/assert"
)

//go:noescape
func round_avx(x *byte)

var foo = round_avx

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

		var out [256]byte
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
			for j := range got {
				got[j] = binary.LittleEndian.Uint32(out[32*j+4*i:])
			}

			t.Log(n, i)
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

	var left, right, out [256]byte
	for i, ents := range data {
		for j, val := range ents {
			if j < 8 {
				binary.LittleEndian.PutUint32(left[32*i+4*j:], val)
			} else {
				binary.LittleEndian.PutUint32(right[32*i+4*(j-8):], val)
			}
		}
	}

	hashP_avx(&left, &right, 0, &out)
	var got [8][8]uint32
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			got[i][j] = binary.LittleEndian.Uint32(out[32*i+4*j:])
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

	sum := sha256.Sum256(out[:])
	assert.Equal(t, hex.EncodeToString(sum[:]),
		"4b162634638c59e9058342fc5daa95c0036ada22e606dc0020f7a5ee1ad08c57")
}

func TestMovc(t *testing.T) {
	patterns := []uint32{
		0x11111111, 0x22222222, 0x33333333, 0x44444444,
		0x55555555, 0x66666666, 0x77777777, 0x88888888,
	}

	var buf [256]byte
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			binary.LittleEndian.PutUint32(buf[32*i+4*j:], patterns[j])
		}
	}

	for icol := 0; icol < 8; icol++ {
		for ocol := 0; ocol < 8; ocol++ {
			var out [256]byte
			movc_avx(&buf, uint64(icol), &out, uint64(ocol))

			var exp [256]byte
			for i := 0; i < 8; i++ {
				binary.LittleEndian.PutUint32(exp[32*i+4*ocol:], patterns[icol])
			}

			assert.Equal(t, out, exp)
			runtime.KeepAlive(foo)
		}
	}
}

func BenchmarkHashF_1(b *testing.B) {
	var input [8192]byte
	var out [256]byte

	b.SetBytes(1)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hashF_avx(&input, 1, 0, 0, &out)
	}
}

func BenchmarkHashF_8K(b *testing.B) {
	var input [8192]byte
	var out [256]byte

	b.SetBytes(8192)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hashF_avx(&input, 8192, 0, 0, &out)
	}
}

func BenchmarkHashP(b *testing.B) {
	var left [256]byte
	var right [256]byte
	var out [256]byte

	b.SetBytes(512)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hashP_avx(&left, &right, 0, &out)
	}
}

func BenchmarkMovc(b *testing.B) {
	var input [256]byte
	var out [256]byte

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		movc_avx(&input, uint64(i)%8, &out, uint64(i+1)%8)
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
				var h avxHasher
				h.update(input)
				h.finalize()
			}
		})
	}
}

func flatten(x *[8][32]byte) *[256]byte {
	var o [256]byte
	for i := 0; i < 8; i++ {
		copy(o[32*i:], x[i][:])
	}
	return &o
}
