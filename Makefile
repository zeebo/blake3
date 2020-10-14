asm: alg/hash/hash_avx2/impl_amd64.s alg/compress/compress_sse41/impl_amd64.s

alg/hash/hash_avx2/impl_amd64.s: avo/avx2/*.go
	( cd avo; go run ./avx2 ) > alg/hash/hash_avx2/impl_amd64.s

alg/compress/compress_sse41/impl_amd64.s: avo/sse41/*.go
	( cd avo; go run ./sse41 ) > alg/compress/compress_sse41/impl_amd64.s
