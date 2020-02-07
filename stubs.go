package blake3

import "github.com/zeebo/blake3/avx2"

func hashF(input *[8192]byte, length, counter uint64, flags uint32, out *chainVector, chain *[8]uint32) {
	avx2.HashF(input, length, counter, flags, out, chain)
}

func hashP(left, right *chainVector, flags uint32, out *chainVector, n int) {
	avx2.HashP(left, right, flags, out, n)
}

func compress(chain *[8]uint32, block *[16]uint32, counter uint64, blen uint32, flags uint32, out *[16]uint32) {
	avx2.Compress(chain, block, counter, blen, flags, out)
}
