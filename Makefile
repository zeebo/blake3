avx.s: avo/*.go
	( cd avo; go run *.go ) > avx2/impl.s
