package avx2

//go:noescape
func hashF_avx(input *[8192]byte, length, counter uint64, flags uint32, out *[64]uint32, chain *[8]uint32)

func HashF(input *[8192]byte, length, counter uint64, flags uint32, out *[64]uint32, chain *[8]uint32) {
	hashF_avx(input, length, counter, flags, out, chain)
}

//go:noescape
func hashP_avx(left, right *[64]uint32, flags uint32, out *[64]uint32)

func HashP(left, right *[64]uint32, flags uint32, out *[64]uint32, n int) {
	hashP_avx(left, right, flags, out)
}

//go:noescape
func compress_sse41(chain *[8]uint32, block *[16]uint32, counter uint64, blen uint32, flags uint32, out *[16]uint32)

func Compress(chain *[8]uint32, block *[16]uint32, counter uint64, blen uint32, flags uint32, out *[16]uint32) {
	compress_sse41(chain, block, counter, blen, flags, out)
}
