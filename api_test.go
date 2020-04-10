package blake3

import (
	"bytes"
	"encoding/hex"
	"io"
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
			size:   32,
			result: "af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262",
		},
		{
			name:   "NewSized",
			new:    func() (*Hasher, error) { return NewSized(8) },
			data:   "",
			size:   8,
			result: "af1349b9f5f9a1a6",
		},
		{
			name:   "NewKeyed",
			new:    func() (*Hasher, error) { return NewKeyed(key) },
			data:   "",
			size:   32,
			result: "cbf50f0463d68fd443cdb0826f387a6f57ba6dc4983ba2460fe822552d15d2f4",
		},
		{
			name:   "NewKeyedSized",
			new:    func() (*Hasher, error) { return NewKeyedSized(key, 8) },
			data:   "",
			size:   8,
			result: "cbf50f0463d68fd4",
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
			name: "NewSized+LargeOutput",
			new:  func() (*Hasher, error) { return NewSized(256) },
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
			if err != nil {
				t.Fatal(err)
			}

			if n, err := h.Write([]byte(c.data)); err != nil {
				t.Fatal(err)
			} else if n != len(c.data) {
				t.Fatal("short write")
			}

			t.Run("Size", func(t *testing.T) {
				if got := h.Size(); got != c.size {
					t.Fatal("invalid hash size:", got)
				}
			})

			// check that we can sum mutliple times, and that it does an append
			t.Run("Sum", func(t *testing.T) {
				assert.Equal(t, hex.EncodeToString(h.Sum(nil)), c.result)
				for i := 0; i < 64; i++ {
					assert.Equal(t, hex.EncodeToString(h.Sum(make([]byte, i)[:0])), c.result)
				}
				assert.Equal(t, hex.EncodeToString(h.Sum(make([]byte, 1))), "00"+c.result)
			})

			// ensure that reset works by issuing the write again
			t.Run("Reset", func(t *testing.T) {
				h.Reset()
				if n, err := h.Write([]byte(c.data)); err != nil {
					t.Fatal(err)
				} else if n != len(c.data) {
					t.Fatal("short write")
				}
				assert.Equal(t, hex.EncodeToString(h.Sum(nil)), c.result)
			})

			t.Run("XOF", func(t *testing.T) {
				t.Run("Read", func(t *testing.T) {
					// read up to i bytes of output in batches of at most size j
					for i := 0; i < len(c.result)/2; i++ {
						for j := 1; j < i; j++ {
							buf, r := make([]byte, i), h.XOF()

							for rem := buf; len(rem) > 0; {
								tmp := rem
								if len(tmp) > j {
									tmp = tmp[:j]
								}

								n, err := r.Read(tmp)
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
						buf, r := make([]byte, len(c.result)/2-i), h.XOF()

						n64, err := r.Seek(int64(i), io.SeekStart)
						assert.NoError(t, err)
						assert.Equal(t, n64, i)

						n, err := r.Read(buf)
						assert.NoError(t, err)
						assert.Equal(t, n, len(buf))

						assert.Equal(t, hex.EncodeToString(buf), c.result[2*i:])
					}
				})

				t.Run("SeekCurrent", func(t *testing.T) {
					buf, r := make([]byte, len(c.result)/2), h.XOF()

					// read then seek backward the amount we just read
					for i := 0; i < len(c.result)/2; i++ {
						n, err := r.Read(buf)
						assert.NoError(t, err)
						assert.Equal(t, n, len(buf))

						assert.Equal(t, hex.EncodeToString(buf[:len(c.result)/2-i]), c.result[2*i:])

						n64, err := r.Seek(-int64(n)+1, io.SeekCurrent)
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

	_, err = NewSized(-1)
	assert.Error(t, err)

	_, err = NewKeyed(make([]byte, 31))
	assert.Error(t, err)

	_, err = NewKeyedSized(make([]byte, 32), -1)
	assert.Error(t, err)

	_, err = NewKeyedSized(make([]byte, 31), 8)
	assert.Error(t, err)

	xof := New().XOF()

	_, err = xof.Seek(-1, io.SeekStart)
	assert.Error(t, err)

	_, err = xof.Seek(-1, io.SeekCurrent)
	assert.Error(t, err)

	_, err = xof.Seek(0, io.SeekEnd)
	assert.Error(t, err)

	_, err = xof.Seek(0, 9999)
	assert.Error(t, err)
}
