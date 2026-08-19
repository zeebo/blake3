ASM := \
	internal/alg/hash/hash_avx512/impl_amd64.s \
	internal/alg/hash/hash_avx2/impl_amd64.s \
	internal/alg/compress/compress_avx512/impl_amd64.s \
	internal/alg/compress/compress_sse41/impl_amd64.s \
	internal/alg/hash/hash_sve2/impl_arm64.s \
	internal/alg/hash/hash_neon/impl_arm64.s

asm: $(ASM)

internal/alg/%/impl_amd64.s: _asm/%/*.go
	( cd _asm; go run ./$* ) > $@

internal/alg/%/impl_arm64.s: _asm/%/*.go
	( cd _asm; go run ./$* ) > $@

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: clean
clean:
	rm -f $(ASM)

.PHONY: test
test:
	go test -race -bench=. -benchtime=1x

.PHONY: vet
vet:
	go tool dist list         \
	| sed -e 's#/# #g'        \
	| while read goos goarch; \
	do                        \
		echo $$goos $$goarch; \
		GOOS=$$goos GOARCH=$$goarch CGO_ENABLED=1 GO386=softfloat go vet              ./...; \
		GOOS=$$goos GOARCH=$$goarch CGO_ENABLED=1 GO386=softfloat go vet -tags=purego ./...; \
	done
