// +build !amd64

package sse41

import "github.com/zeebo/blake3/ref"

func Compress(chain *[8]uint32, block *[16]uint32, counter uint64, blen uint32, flags uint32, out *[16]uint32) {
	ref.Compress(chain, block, counter, blen, flags, out)
}
