asm: avx2/impl.s sse41/impl.s

avx2/impl.s: avo/avx2/*.go
	( cd avo; go run ./avx2 ) > avx2/impl_amd64.s

sse41/impl.s: avo/sse41/*.go
	( cd avo; go run ./sse41 ) > sse41/impl_amd64.s
