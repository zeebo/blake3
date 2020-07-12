// +build !amd64

package consts

import "unsafe"

// TODO: maybe this would be better if it was a const. then the compiler could
// do dead code elimination.
var IsLittleEndian = *(*uint32)(unsafe.Pointer(&[4]byte{0, 0, 0, 1})) != 1

const (
	HasAVX2  = false
	HasSSE41 = false
)
