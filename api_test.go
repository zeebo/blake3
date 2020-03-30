package blake3

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/zeebo/assert"
)

func TestAPI(t *testing.T) {
	key := bytes.Repeat([]byte("a"), 32)

	cases := []struct {
		name   string
		new    func() (*Hasher, error)
		data   string
		result string
		size   int
	}{
		{
			name:   "New",
			new:    func() (*Hasher, error) { return New(), nil },
			data:   "",
			result: "af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262",
			size:   32,
		},
		{
			name:   "NewSized",
			new:    func() (*Hasher, error) { return NewSized(8) },
			data:   "",
			result: "af1349b9f5f9a1a6",
			size:   8,
		},
		{
			name:   "NewKeyed",
			new:    func() (*Hasher, error) { return NewKeyed(key) },
			data:   "",
			result: "cbf50f0463d68fd443cdb0826f387a6f57ba6dc4983ba2460fe822552d15d2f4",
			size:   32,
		},
		{
			name:   "NewKeyedSized",
			new:    func() (*Hasher, error) { return NewKeyedSized(key, 8) },
			data:   "",
			result: "cbf50f0463d68fd4",
			size:   8,
		},
		{
			name:   "New+Small",
			new:    func() (*Hasher, error) { return New(), nil },
			data:   "some data",
			result: "b224a1da2bf5e72b337dc6dde457a05265a06dec8875be379e2ad2be5edb3bf2",
			size:   32,
		},
		{
			name:   "New+Large",
			new:    func() (*Hasher, error) { return New(), nil },
			data:   strings.Repeat("a", 10240),
			result: "9afd0ba102b2cc68be10ba4d383b3139b97ed36d425b82631a7a1e2424088f7e",
			size:   32,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h, err := c.new()
			if err != nil {
				t.Fatal(err)
			}

			if got := h.Size(); got != c.size {
				t.Fatal("invalid hash size:", got)
			}

			n, err := h.Write([]byte(c.data))
			if err != nil {
				t.Fatal(err)
			}
			if n != len(c.data) {
				t.Fatal("short write")
			}

			assert.Equal(t, hex.EncodeToString(h.Sum(nil)), c.result)
			for i := 0; i < 64; i++ {
				assert.Equal(t, hex.EncodeToString(h.Sum(make([]byte, i)[:0])), c.result)
			}
			assert.Equal(t, hex.EncodeToString(h.Sum(make([]byte, 1))), "00"+c.result)
		})
	}
}
