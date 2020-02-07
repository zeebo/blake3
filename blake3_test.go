package blake3

import (
	"encoding/hex"
	"fmt"
	"testing"
	"unsafe"

	"github.com/zeebo/assert"
)

func TestBytesToWords(t *testing.T) {
	if !isLittleEndian {
		t.SkipNow()
	}

	var bytes [64]uint8
	for i := range bytes {
		bytes[i] = byte(i)
	}

	var words [16]uint32
	bytesToWords(&bytes, &words)

	assert.Equal(t, *(*[16]uint32)(unsafe.Pointer(&bytes[0])), words)
}

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
	for n := 0; n <= 8192; n++ {
		var input [8192]byte
		for i := 0; i < n; i++ {
			input[i] = byte(i+1) % 251
		}

		var chain [8]uint32
		var out chainVector
		hashF_avx(&input, uint64(n), 0, 0, &out, &chain)

		for i := 0; i < 8 && (i*1024 < n || (i == 0 && n == 0)); i++ {
			high := 1024 * (i + 1)
			if high <= n {
				chain := [8]uint32{iv0, iv1, iv2, iv3, iv4, iv5, iv6, iv7}
				chunk := newChunkState(chain, uint64(i), 0)
				chunk.update(input[1024*i : high])
				op := chunk.output()
				exp := op.compress()

				var got [8]uint32
				for j := 0; j < 8; j++ {
					got[j] = out[8*j+i]
				}

				assert.Equal(t, exp, got)
			} else {
				high = n

				chunk := newChunkState([8]uint32{iv0, iv1, iv2, iv3, iv4, iv5, iv6, iv7}, uint64(i), 0)
				chunk.update(input[1024*i : high])
				op := chunk.output()

				assert.Equal(t, *op.chain, chain)
			}
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

	var left, right, out chainVector
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

func TestHashP_One(t *testing.T) {
	a := [8]uint32{1, 1, 1, 1, 1, 1, 1, 1}
	b := [8]uint32{2, 2, 2, 2, 2, 2, 2, 2}

	var left, right, out chainVector
	writeChain(&a, &left, 0)
	writeChain(&b, &right, 0)
	hashP_avx(&left, &right, 0, &out)

	key := [8]uint32{iv0, iv1, iv2, iv3, iv4, iv5, iv6, iv7}
	exp := parentChain(a, b, key, 0)

	var got [8]uint32
	readChain(&out, 0, &got)
	assert.Equal(t, exp, got)
}

func TestVectorsAVX2(t *testing.T) {
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
	var chain [8]uint32
	var out chainVector

	b.SetBytes(1)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hashF_avx(&input, 1, 0, 0, &out, &chain)
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
		hashF_avx(&input, 8192, 0, 0, &out, &chain)
	}
}

func BenchmarkHashP(b *testing.B) {
	var left chainVector
	var right chainVector
	var out chainVector

	b.SetBytes(512)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hashP_avx(&left, &right, 0, &out)
	}
}

func TestCompressSSE41(t *testing.T) {
	var chain0, chain1 [8]uint32
	var block [16]uint32

	for j := range block {
		block[j] = uint32(j | j<<8 | j<<16 | j<<24)
	}

	for i := 0; i <= 64; i++ {
		var exp, got [16]uint32

		compress(&chain0, &block, 1, 2, 3, &exp)
		compress_sse41(&chain1, &block, 1, 2, 3, &got)

		assert.Equal(t, exp, got)
	}
}

func BenchmarkCompress(b *testing.B) {
	var c [8]uint32
	var m, o [16]uint32

	b.SetBytes(64)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		compress(&c, &m, 0, 0, 0, &o)
	}
}

func BenchmarkCompress_SSE41(b *testing.B) {
	var chain [8]uint32
	var block [16]uint32
	var buf [16]uint32

	b.SetBytes(64)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		compress_sse41(&chain, &block, 0, 0, 0, &buf)
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
	sizes := []int64{
		0, 16, 32, 64, 128, 256, 512, 1024, 4 * 1024,
		8*1024 - 1, 8 * 1024, 16 * 1024, 32 * 1024, 64 * 1024,
		1024 * 1024 * 10,
	}

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
