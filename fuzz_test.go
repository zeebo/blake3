package blake3

import (
	"math/rand"
	"testing"
)

func FuzzHash(f *testing.F) {
	f.Fuzz(func(t *testing.T, prog []byte) {
		l := 0
		for _, v := range prog {
			l += int(v)
		}
		data := make([]byte, l)
		rand.New(rand.NewSource(0)).Read(data)

		h, b := New(), data
		for _, v := range prog {
			h.Write(b[:v])
			b = b[v:]
		}
		v1 := h.Sum(nil)
		v2 := Sum256(data)
		if string(v1) != string(v2[:]) {
			t.Fatalf("v1: %v, v2: %v", v1, v2)
		}
	})
}
