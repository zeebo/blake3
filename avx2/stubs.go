package avx2

//go:noescape
func HashF(input *[8192]byte, length, counter uint64, flags uint32, out *[64]uint32, chain *[8]uint32)

//go:noescape
func HashP(left, right *[64]uint32, flags uint32, out *[64]uint32, n int)

//go:noescape
func Compress(chain *[8]uint32, block *[16]uint32, counter uint64, blen uint32, flags uint32, out *[16]uint32)
