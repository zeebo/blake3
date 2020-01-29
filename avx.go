package blake3

//go:noescape
func hash8_avx(
	input *[8192]byte,
	counter uint64,
	flags uint32,
	out *[256]byte,
)

//go:noescape
func hashF_avx(
	input *[8192]byte,
	blocks uint64,
	chunks uint64,
	counter uint64,
	flags uint32,
	out *[256]byte,
)

//go:noescape
func hashP_avx(
	left *[256]byte,
	right *[256]byte,
	flags uint32,
	out *[256]byte,
)
