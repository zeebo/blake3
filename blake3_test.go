package blake3

import (
	"crypto/sha256"
	"encoding/binary"
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

func TestHashF_8K(t *testing.T) {
	var input [8192]byte
	for i := 0; i < 8; i++ {
		for j := 0; j < 1024; j++ {
			input[1024*i+j] = byte(32*i + j)
		}
	}

	out := [256]byte{}
	hashF_avx(&input, 8192, 0, 0, &out)

	for i := 0; i < 8; i++ {
		buf := make([]byte, 1024)
		for j := range buf {
			buf[j] = byte(32*i + j)
		}

		chain := [8]uint32{iv0, iv1, iv2, iv3, iv4, iv5, iv6, iv7}
		chunk := newChunkState(chain, uint64(i), 0)
		chunk.update(buf)
		output := chunk.output()
		exp := output.compress()

		var got [8]uint32
		for j := range got {
			got[j] = binary.LittleEndian.Uint32(out[32*j+4*i:])
		}

		assert.Equal(t, exp, got)
	}

	sum := sha256.Sum256(out[:])
	assert.Equal(t, hex.EncodeToString(sum[:]),
		"c0589b33091c650f868859d99e7618a745d2bbd3f81a2b9493880e9dbabcf948")
}

func TestHashF(t *testing.T) {
	for n := 1; n < 8192; n++ {
		var input [8192]byte

	fill:
		for i := 0; i < 8; i++ {
			for j := 0; j < 1024; j++ {
				if 1024*i+j >= n {
					break fill
				}
				input[1024*i+j] = byte(32*i + j)
			}
		}

		var out [256]byte
		hashF_avx(&input, uint64(n), 0, 0, &out)

		for i := 0; i < 8 && (i*1024 < n || (i == 0 && n == 0)); i++ {
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
