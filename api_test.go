package blake3

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/zeebo/assert"
)

func TestAPI(t *testing.T) {
	key := bytes.Repeat([]byte("a"), 32)

	cases := []struct {
		name   string
		new    func() (*Hasher, error)
		result string
	}{
		{
			name:   "New",
			new:    func() (*Hasher, error) { return New(), nil },
			result: "af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262",
		},
		{
			name:   "NewSized",
			new:    func() (*Hasher, error) { return NewSized(8) },
			result: "af1349b9f5f9a1a6",
		},
		{
			name:   "NewKeyed",
			new:    func() (*Hasher, error) { return NewKeyed(key) },
			result: "cbf50f0463d68fd443cdb0826f387a6f57ba6dc4983ba2460fe822552d15d2f4",
		},
		{
			name:   "NewKeyedSized",
			new:    func() (*Hasher, error) { return NewKeyedSized(key, 8) },
			result: "cbf50f0463d68fd4",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h, err := c.new()
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, hex.EncodeToString(h.Sum(nil)), c.result)
			for i := 0; i < 64; i++ {
				assert.Equal(t, hex.EncodeToString(h.Sum(make([]byte, i)[:0])), c.result)
			}
			assert.Equal(t, hex.EncodeToString(h.Sum(make([]byte, 1))), "00"+c.result)
		})
	}
}
