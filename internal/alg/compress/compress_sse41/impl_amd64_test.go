package compress_sse41_test

import (
	"testing"

	"github.com/zeebo/assert"
	"github.com/zeebo/blake3/internal/alg/compress/compress_pure"
	"github.com/zeebo/blake3/internal/alg/compress/compress_sse41"
	"github.com/zeebo/blake3/internal/consts"
	"github.com/zeebo/pcg"
)

func TestCompress(t *testing.T) {
	if !consts.HasSSE41 {
		t.SkipNow()
	}

	var chain [8]uint32
	var block [16]uint32

	for i := 0; i < 1e5; i++ {
		var o1, o2 [16]uint32

		counter, blen, flags := pcg.Uint64(), pcg.Uint32(), pcg.Uint32()
		for i := range &chain {
			chain[i] = pcg.Uint32()
		}
		for i := range &block {
			block[i] = pcg.Uint32()
		}

		compress_sse41.Compress(&chain, &block, counter, blen, flags, &o1)
		compress_pure.Compress(&chain, &block, counter, blen, flags, &o2)

		assert.Equal(t, o1, o2)
	}
}
