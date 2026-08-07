//go:build !amd64
// +build !amd64

package compress_sse41

import "github.com/zeebo/blake3/internal/alg/compress/compress_pure"

func Compress(chain *[32]byte, block *[64]byte, counter uint64, blen uint32, flags uint32, out *[64]byte) {
	compress_pure.Compress(chain, block, counter, blen, flags, out)
}
