package blake3_test

import (
	"bytes"
	"fmt"
	"io"

	"github.com/zeebo/blake3"
)

func ExampleNew() {
	h := blake3.New()

	h.Write([]byte("some data"))

	fmt.Printf("%x\n", h.Sum(nil))
	//output:
	// b224a1da2bf5e72b337dc6dde457a05265a06dec8875be379e2ad2be5edb3bf2
}

func ExampleNewKeyed() {
	h1, err := blake3.NewKeyed(bytes.Repeat([]byte("1"), 32))
	if err != nil {
		panic(err)
	}

	h2, err := blake3.NewKeyed(bytes.Repeat([]byte("2"), 32))
	if err != nil {
		panic(err)
	}

	h1.Write([]byte("some data"))
	h2.Write([]byte("some data"))

	fmt.Printf("%x\n", h1.Sum(nil))
	fmt.Printf("%x\n", h2.Sum(nil))
	//output:
	// 107c6f88638356d73cdb80f4d56ffe50abcbd9664a80c8ab2b83b1f946ebaba1
	// b4be81075bef5a2448158ee5eeddaed897fe44a564c2cb088facbe7824a25073
}

func ExampleDeriveKey() {
	out := make([]byte, 32)

	// See the documentation for good practices on what the context should be.
	blake3.DeriveKey(
		"my-application v0.1.1 session tokens v1",  // context
		[]byte("some material to derive key from"), // material
		out,
	)

	fmt.Printf("%x\n", out)
	//output:
	// 98a3333af735f89eb301b56eaf6a77713aa03cdb0057e5b04352a63ea9204add
}

func ExampleNewDeriveKey() {
	// See the documentation for good practices on what the context should be.
	h := blake3.NewDeriveKey("my-application v0.1.1 session tokens v1")

	h.Write([]byte("some material to derive key from"))

	fmt.Printf("%x\n", h.Sum(nil))
	//output:
	// 98a3333af735f89eb301b56eaf6a77713aa03cdb0057e5b04352a63ea9204add
}

func ExampleHasher_Reset() {
	h := blake3.New()

	h.Write([]byte("some data"))
	fmt.Printf("%x\n", h.Sum(nil))

	h.Reset()

	h.Write([]byte("some data"))
	fmt.Printf("%x\n", h.Sum(nil))
	//output:
	// b224a1da2bf5e72b337dc6dde457a05265a06dec8875be379e2ad2be5edb3bf2
	// b224a1da2bf5e72b337dc6dde457a05265a06dec8875be379e2ad2be5edb3bf2
}

func ExampleHasher_Digest() {
	h := blake3.New()
	h.Write([]byte("some data"))
	d := h.Digest()

	out := make([]byte, 64)
	d.Read(out)

	fmt.Printf("%x\n", out[0:32])
	fmt.Printf("%x\n", out[32:64])
	//output:
	// b224a1da2bf5e72b337dc6dde457a05265a06dec8875be379e2ad2be5edb3bf2
	// 1b55688951738e3a7155d6398eb56c6bc35d5bca5f139d98eb7409be51d1be32
}

func ExampleHasher_Clone() {
	h1 := blake3.New()
	h1.WriteString("some")

	h2 := h1.Clone()
	fmt.Println("before:")
	fmt.Printf("h1: %x\n", h1.Sum(nil))
	fmt.Printf("h2: %x\n\n", h2.Sum(nil))

	h2.WriteString(" data")

	fmt.Println("h2 modified:")
	fmt.Printf("h1: %x\n", h1.Sum(nil))
	fmt.Printf("h2: %x\n\n", h2.Sum(nil))

	h1.WriteString(" data")

	fmt.Println("h1 converged:")
	fmt.Printf("h1: %x\n", h1.Sum(nil))
	fmt.Printf("h2: %x\n", h2.Sum(nil))

	//output:
	// before:
	// h1: 2f610cf2e7e0dc09384cbaa75b2ae5d9704ac9a5ac7f28684342856e2867c707
	// h2: 2f610cf2e7e0dc09384cbaa75b2ae5d9704ac9a5ac7f28684342856e2867c707
	//
	// h2 modified:
	// h1: 2f610cf2e7e0dc09384cbaa75b2ae5d9704ac9a5ac7f28684342856e2867c707
	// h2: b224a1da2bf5e72b337dc6dde457a05265a06dec8875be379e2ad2be5edb3bf2
	//
	// h1 converged:
	// h1: b224a1da2bf5e72b337dc6dde457a05265a06dec8875be379e2ad2be5edb3bf2
	// h2: b224a1da2bf5e72b337dc6dde457a05265a06dec8875be379e2ad2be5edb3bf2
}

func ExampleDigest_Seek() {
	h := blake3.New()
	h.Write([]byte("some data"))
	d := h.Digest()

	out := make([]byte, 32)
	d.Seek(32, io.SeekStart)
	d.Read(out)

	fmt.Printf("%x\n", out)
	//output:
	// 1b55688951738e3a7155d6398eb56c6bc35d5bca5f139d98eb7409be51d1be32
}

func ExampleSum256() {
	digest := blake3.Sum256([]byte("some data"))

	fmt.Printf("%x\n", digest[:])
	//output:
	// b224a1da2bf5e72b337dc6dde457a05265a06dec8875be379e2ad2be5edb3bf2
}

func ExampleSum512() {
	digest := blake3.Sum512([]byte("some data"))

	fmt.Printf("%x\n", digest[0:32])
	fmt.Printf("%x\n", digest[32:64])
	//output:
	// b224a1da2bf5e72b337dc6dde457a05265a06dec8875be379e2ad2be5edb3bf2
	// 1b55688951738e3a7155d6398eb56c6bc35d5bca5f139d98eb7409be51d1be32
}
