package blake3

import (
	"github.com/zeebo/blake3/avx2"
	"github.com/zeebo/blake3/internal/consts"
	"github.com/zeebo/blake3/ref"
	"github.com/zeebo/blake3/sse41"
)

func hashF(input *[8192]byte, length, counter uint64, flags uint32, key *[8]uint32, out *[64]uint32, chain *[8]uint32) {
	if consts.HasAVX2 {
		avx2.HashF(input, length, counter, flags, key, out, chain)
	} else {
		ref.HashF(input, length, counter, flags, key, out, chain)
	}
}

func hashP(left, right *[64]uint32, flags uint32, key *[8]uint32, out *[64]uint32, n int) {
	if consts.HasAVX2 {
		avx2.HashP(left, right, flags, key, out, n)
	} else {
		ref.HashP(left, right, flags, key, out, n)
	}
}

func compress(chain *[8]uint32, block *[16]uint32, counter uint64, blen uint32, flags uint32, out *[16]uint32) {
	if consts.HasSSE41 {
		sse41.Compress(chain, block, counter, blen, flags, out)
	} else {
		ref.Compress(chain, block, counter, blen, flags, out)
	}
}
