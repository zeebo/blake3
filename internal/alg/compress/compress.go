package compress

import (
	"github.com/zeebo/blake3/internal/alg/compress/compress_pure"
	"github.com/zeebo/blake3/internal/alg/compress/compress_sse41"
	"github.com/zeebo/blake3/internal/consts"
)

// Compress requires that out does not alias chain.
func Compress(chain *[32]byte, block *[64]byte, counter uint64, blen uint32, flags uint32, out *[64]byte) {
	if consts.HasSSE41 {
		compress_sse41.Compress(chain, block, counter, blen, flags, out)
	} else {
		compress_pure.Compress(chain, block, counter, blen, flags, out)
	}
}
