package blake3

//go:noescape
func hashF_avx(input *[8192]byte, length, counter uint64, flags uint32, out *[256]byte)

//go:noescape
func hashP_avx(left, right *[256]byte, flags uint32, out *[256]byte)

//go:noescape
func rotate_avx(in *[256]byte)
