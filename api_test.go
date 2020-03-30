package blake3

import (
	"encoding/hex"
	"testing"

	"github.com/zeebo/assert"
)

func TestNew(t *testing.T) {
	h := New()

	assert.Equal(t, hex.EncodeToString(h.Sum(nil)), "af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262")
	for i := 0; i < 64; i++ {
		assert.Equal(t, hex.EncodeToString(h.Sum(make([]byte, i)[:0])), "af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262")
	}
	assert.Equal(t, hex.EncodeToString(h.Sum(make([]byte, 1))), "00af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262")
}

func TestNewSized(t *testing.T) {
	h, err := NewSized(8)
	if err != nil {
		t.Fatalf("NewSized: %v", err)
	}

	assert.Equal(t, hex.EncodeToString(h.Sum(nil)), "af1349b9f5f9a1a6")
	for i := 0; i < 16; i++ {
		assert.Equal(t, hex.EncodeToString(h.Sum(make([]byte, i)[:0])), "af1349b9f5f9a1a6")
	}
	assert.Equal(t, hex.EncodeToString(h.Sum(make([]byte, 1))), "00af1349b9f5f9a1a6")
}

func TestNewKeyed(t *testing.T) {
	h, err := NewKeyed([]byte(`aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`))
	if err != nil {
		t.Fatalf("NewKeyed: %v", err)
	}

	assert.Equal(t, hex.EncodeToString(h.Sum(nil)), "cbf50f0463d68fd443cdb0826f387a6f57ba6dc4983ba2460fe822552d15d2f4")
}

func TestNewKeyedSized(t *testing.T) {
	h, err := NewKeyedSized([]byte(`aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`), 8)
	if err != nil {
		t.Fatalf("NewSized: %v", err)
	}

	assert.Equal(t, hex.EncodeToString(h.Sum(nil)), "cbf50f0463d68fd4")
	for i := 0; i < 16; i++ {
		assert.Equal(t, hex.EncodeToString(h.Sum(make([]byte, i)[:0])), "cbf50f0463d68fd4")
	}
	assert.Equal(t, hex.EncodeToString(h.Sum(make([]byte, 1))), "00cbf50f0463d68fd4")
}
