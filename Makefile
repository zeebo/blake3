avx.s: avo/*.go
	( cd avo; go run *.go ) > avx.s
