// +build !amd64

package avx2

import "github.com/zeebo/blake3/ref"

func HashF(input *[8192]byte, length, counter uint64, flags uint32, key *[8]uint32, out *[64]uint32, chain *[8]uint32) {
	ref.HashF(input, length, counter, flags, key, out, chain)
}

func HashP(left, right *[64]uint32, flags uint32, key *[8]uint32, out *[64]uint32, n int) {
	ref.HashP(left, right, flags, key, out, n)
}
