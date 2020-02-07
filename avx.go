package blake3

type chainVector = [64]uint32

//go:noescape
func hashF_avx(input *[8192]byte, length, counter uint64, flags uint32, out *chainVector, chain *[8]uint32)

//go:noescape
func hashP_avx(left, right *chainVector, flags uint32, out *chainVector)

//go:noescape
func compress_sse41(chain *[8]uint32, block *[16]uint32, counter uint64, blen uint32, flags uint32, out *[16]uint32)
