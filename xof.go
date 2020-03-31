package blake3

import (
	"errors"
	"unsafe"

	"github.com/zeebo/blake3/internal/consts"
	"github.com/zeebo/blake3/internal/utils"
)

type xof struct {
	counter uint64
	chain   [8]uint32
	block   [16]uint32
	blen    uint32
	flags   uint32
	buf     [16]uint32
	bufn    int
}

func (x *xof) Read(out []byte) (n int, err error) {
	n = len(out)

	if x.bufn > 0 {
		n := x.slowCopy(out)
		out = out[n:]
		x.bufn -= n
	}

	for len(out) >= 64 {
		x.fillBuf()

		if consts.IsLittleEndian {
			*(*[64]byte)(unsafe.Pointer(&out[0])) = *(*[64]byte)(unsafe.Pointer(&x.buf[0]))
		} else {
			utils.WordsToBytes(&x.buf, out)
		}

		out = out[64:]
		x.bufn = 0
	}

	if len(out) == 0 {
		return n, nil
	}

	x.fillBuf()
	x.bufn -= x.slowCopy(out)

	return n, nil
}

func (x *xof) Seek(offset int64, whence int) (int64, error) {
	return 0, errors.New("Seek not implemented yet")
}

func (x *xof) slowCopy(out []byte) (n int) {
	off := uint(64-x.bufn) % 64
	if consts.IsLittleEndian {
		n = copy(out, (*[64]byte)(unsafe.Pointer(&x.buf[0]))[off:])
	} else {
		var tmp [64]byte
		utils.WordsToBytes(&x.buf, tmp[:])
		n = copy(out, tmp[off:])
	}
	return n
}

func (x *xof) fillBuf() {
	compress(&x.chain, &x.block, x.counter, x.blen, x.flags, &x.buf)
	x.counter++
	x.bufn = 64
}
