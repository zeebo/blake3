ASM := \
	internal/alg/hash/hash_avx512/hash_amd64.s \
	internal/alg/hash/hash_avx2/hash_amd64.s \
	internal/alg/compress/compress_sse41/compress_amd64.s \
	internal/alg/hash/hash_sve2/hash_arm64.s \
	internal/alg/hash/hash_neon/hash_arm64.s

asm: $(ASM)

internal/alg/%/hash_amd64.s: _asm/%/*.go
	( cd _asm; go run ./$* ) > $@

internal/alg/%/hash_arm64.s: _asm/%/*.go
	( cd _asm; go run ./$* ) > $@

internal/alg/%/compress_amd64.s: _asm/%/*.go
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
