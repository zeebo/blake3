package blake3

import "unsafe"

type cv = [64]uint32

//go:noescape
func hashF_avx(input *[8192]byte, length, counter uint64, flags uint32, out *cv)

//go:noescape
func hashP_avx(left, right *cv, flags uint32, out *cv)

func movcol(input *cv, icol uint64, out *cv, ocol uint64) {
	type u = uintptr
	type p = unsafe.Pointer
	type a = *uint32

	i := p(u(p(input)) + u(icol*4))
	o := p(u(p(out)) + u(ocol*4))

	*a(p(u(o) + 0*32)) = *a(p(u(i) + 0*32))
	*a(p(u(o) + 1*32)) = *a(p(u(i) + 1*32))
	*a(p(u(o) + 2*32)) = *a(p(u(i) + 2*32))
	*a(p(u(o) + 3*32)) = *a(p(u(i) + 3*32))
	*a(p(u(o) + 4*32)) = *a(p(u(i) + 4*32))
	*a(p(u(o) + 5*32)) = *a(p(u(i) + 5*32))
	*a(p(u(o) + 6*32)) = *a(p(u(i) + 6*32))
	*a(p(u(o) + 7*32)) = *a(p(u(i) + 7*32))
}
