package blake3

import (
	"bytes"
	"encoding/hex"
	"io"
	"strings"
	"testing"

	"github.com/zeebo/assert"
)

func TestAPI_Vectors(t *testing.T) {
	check := func(t *testing.T, h *Hasher, input []byte, hash string) {
		buf := make([]byte, len(hash)/2)

		n, err := h.Write(input)
		assert.NoError(t, err)
		assert.Equal(t, n, len(input))

		n, err = h.Digest().Read(buf)
		assert.NoError(t, err)
		assert.Equal(t, n, len(buf))

		assert.Equal(t, hash, hex.EncodeToString(buf))
	}

	t.Run("Basic", func(t *testing.T) {
		for _, tv := range vectors {
			h := New()
			check(t, h, tv.input(), tv.hash)
		}
	})

	t.Run("Keyed", func(t *testing.T) {
		for _, tv := range vectors {
			h, err := NewKeyed([]byte(testVectorKey))
			assert.NoError(t, err)
			check(t, h, tv.input(), tv.keyedHash)
		}
	})

	t.Run("DeriveKey", func(t *testing.T) {
		for _, tv := range vectors {
			h := NewDeriveKey(testVectorContext)
			check(t, h, tv.input(), tv.deriveKey)
		}
	})
}

func TestAPI(t *testing.T) {
	key := bytes.Repeat([]byte("a"), 32)
	context := strings.Repeat("c", 32)

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
			size:   32,
			result: "af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262",
		},
		{
			name:   "NewKeyed",
			new:    func() (*Hasher, error) { return NewKeyed(key) },
			data:   "",
			size:   32,
			result: "cbf50f0463d68fd443cdb0826f387a6f57ba6dc4983ba2460fe822552d15d2f4",
		},
		{
			name:   "NewDeriveKey",
			new:    func() (*Hasher, error) { return NewDeriveKey(context), nil },
			data:   "",
			size:   32,
			result: "c5ce1763648ca67eecc8a471f8efccf19dd16178e91d33130d3ae67eadde71cc",
		},
		{
			name:   "New+SmallInput",
			new:    func() (*Hasher, error) { return New(), nil },
			data:   "some data",
			size:   32,
			result: "b224a1da2bf5e72b337dc6dde457a05265a06dec8875be379e2ad2be5edb3bf2",
		},
		{
			name:   "New+LargeInput",
			new:    func() (*Hasher, error) { return New(), nil },
			data:   strings.Repeat("a", 10240),
			size:   32,
			result: "9afd0ba102b2cc68be10ba4d383b3139b97ed36d425b82631a7a1e2424088f7e",
		},
		{
			name: "New+LargeOutput",
			new:  func() (*Hasher, error) { return New(), nil },
			data: "",
			size: 256,
			result: "" +
				"af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262" +
				"e00f03e7b69af26b7faaf09fcd333050338ddfe085b8cc869ca98b206c08243a" +
				"26f5487789e8f660afe6c99ef9e0c52b92e7393024a80459cf91f476f9ffdbda" +
				"7001c22e159b402631f277ca96f2defdf1078282314e763699a31c5363165421" +
				"cce14d30f8a03e49ee25d2ea3cd48a568957b378a65af65fc35fb3e9e12b81ca" +
				"2d82cdee16c68908a6772f827564336933c89e6908b2f9c7d1811c0eb795cbd5" +
				"898fe6f5e8af763319ca863718a59aff3d99660ef642483e217ef0c878582728" +
				"4fea90d42225e3cdd6a179bee852fd24e7d45b38c27b9c2f9469ea8dbdb893f0",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h, err := c.new()
			assert.NoError(t, err)

			n, err := h.Write([]byte(c.data))
			assert.NoError(t, err)
			assert.Equal(t, n, len(c.data))

			t.Run("Size", func(t *testing.T) {
				assert.Equal(t, h.Size(), 32)
			})

			// check that we can sum multiple times, and that it does an append
			t.Run("Sum", func(t *testing.T) {
				assert.Equal(t, hex.EncodeToString(h.Sum(nil)), c.result[:64])
				for i := 0; i < 64; i++ {
					assert.Equal(t, hex.EncodeToString(h.Sum(make([]byte, i)[:0])), c.result[:64])
				}
				assert.Equal(t, hex.EncodeToString(h.Sum(make([]byte, 1))), "00"+c.result[:64])
			})

			// ensure that reset works by issuing the write again
			t.Run("Reset", func(t *testing.T) {
				_, _ = h.Write([]byte("some fake wrong data"))
				h.Reset()
				n, err := h.Write([]byte(c.data))
				assert.NoError(t, err)
				assert.Equal(t, n, len(c.data))
				assert.Equal(t, hex.EncodeToString(h.Sum(nil)), c.result[:64])
			})

			t.Run("Digest", func(t *testing.T) {
				t.Run("Read", func(t *testing.T) {
					// read up to i bytes of output in batches of at most size j
					for i := 0; i < len(c.result)/2; i++ {
						for j := 1; j < i; j++ {
							buf, d := make([]byte, i), h.Digest()

							for rem := buf; len(rem) > 0; {
								tmp := rem
								if len(tmp) > j {
									tmp = tmp[:j]
								}

								n, err := d.Read(tmp)
								assert.NoError(t, err)
								assert.Equal(t, n, len(tmp))

								rem = rem[n:]
							}

							assert.Equal(t, hex.EncodeToString(buf), c.result[:2*i])
						}
					}
				})

				t.Run("SeekStart", func(t *testing.T) {
					// seek to position i and read the remainder
					for i := 0; i < len(c.result)/2; i++ {
						buf, d := make([]byte, len(c.result)/2-i), h.Digest()

						n64, err := d.Seek(int64(i), io.SeekStart)
						assert.NoError(t, err)
						assert.Equal(t, n64, i)

						n, err := d.Read(buf)
						assert.NoError(t, err)
						assert.Equal(t, n, len(buf))

						assert.Equal(t, hex.EncodeToString(buf), c.result[2*i:])
					}
				})

				t.Run("SeekCurrent", func(t *testing.T) {
					buf, d := make([]byte, len(c.result)/2), h.Digest()

					// read then seek backward the amount we just read
					for i := 0; i < len(c.result)/2; i++ {
						n, err := d.Read(buf)
						assert.NoError(t, err)
						assert.Equal(t, n, len(buf))

						assert.Equal(t, hex.EncodeToString(buf[:len(c.result)/2-i]), c.result[2*i:])

						n64, err := d.Seek(-int64(n)+1, io.SeekCurrent)
						assert.NoError(t, err)
						assert.Equal(t, n64, i+1)
					}
				})
			})
		})
	}
}

func TestAPI_Errors(t *testing.T) {
	var err error

	_, err = NewKeyed(make([]byte, 31))
	assert.Error(t, err)

	d := New().Digest()

	_, err = d.Seek(-1, io.SeekStart)
	assert.Error(t, err)

	_, err = d.Seek(-1, io.SeekCurrent)
	assert.Error(t, err)

	_, err = d.Seek(0, io.SeekEnd)
	assert.Error(t, err)

	_, err = d.Seek(0, 9999)
	assert.Error(t, err)
}

func TestSum256(t *testing.T) {
	h := New()
	x := make([]byte, 1<<16)

	for i := range x {
		x[i] = byte(i) % 251
		if i%32 != 0 {
			continue
		}

		h.Reset()
		_, _ = h.Write(x[:i])

		var exp [32]byte
		_, _ = h.Digest().Read(exp[:])
		got := Sum256(x[:i])

		assert.Equal(t, hex.EncodeToString(got[:]), hex.EncodeToString(exp[:]))
	}
}

func TestSum512(t *testing.T) {
	h := New()
	x := make([]byte, 1<<16)

	for i := range x {
		x[i] = byte(i) % 251
		if i%32 != 0 {
			continue
		}

		h.Reset()
		_, _ = h.Write(x[:i])

		var exp [64]byte
		_, _ = h.Digest().Read(exp[:])
		got := Sum512(x[:i])

		assert.Equal(t, hex.EncodeToString(got[:]), hex.EncodeToString(exp[:]))
	}
}

func TestClone(t *testing.T) {
	sum := func(h *Hasher) string { return hex.EncodeToString(h.Sum(nil)) }

	h1 := New()
	h1.WriteString("1")

	h0 := h1.Clone()
	assert.Equal(t, sum(h1), sum(h0))

	h2 := h1.Clone()
	assert.Equal(t, sum(h1), sum(h2))

	h2.WriteString("2")
	assert.Equal(t, sum(h1), sum(h0))

	h1.WriteString("2")
	assert.Equal(t, sum(h1), sum(h2))
}
