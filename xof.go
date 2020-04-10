package blake3

import (
	"fmt"
	"io"
	"unsafe"

	"github.com/zeebo/blake3/internal/consts"
	"github.com/zeebo/blake3/internal/utils"
)

// XOF captures the state of a Hasher allowing reading and seeking through
// the output stream.
type XOF struct {
	counter uint64
	chain   [8]uint32
	block   [16]uint32
	blen    uint32
	flags   uint32
	buf     [16]uint32
	bufn    int
}

// Read reads data frm the hasher into out. It always fills the entire buffer and
// never errors.
func (x *XOF) Read(out []byte) (n int, err error) {
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

// Seek sets the position to the provided location. Only SeekStart and
// SeekCurrent are allowed.
func (x *XOF) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
	case io.SeekEnd:
		return 0, fmt.Errorf("seek from end not supported")
	case io.SeekCurrent:
		offset += int64(consts.BlockLen*x.counter) - int64(x.bufn)
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}
	if offset < 0 {
		return 0, fmt.Errorf("seek before start")
	}
	x.setPosition(uint64(offset))
	return offset, nil
}

func (x *XOF) setPosition(pos uint64) {
	x.counter = pos / consts.BlockLen
	x.fillBuf()
	x.bufn -= int(pos % consts.BlockLen)
}

func (x *XOF) slowCopy(out []byte) (n int) {
	off := uint(consts.BlockLen-x.bufn) % consts.BlockLen
	if consts.IsLittleEndian {
		n = copy(out, (*[consts.BlockLen]byte)(unsafe.Pointer(&x.buf[0]))[off:])
	} else {
		var tmp [consts.BlockLen]byte
		utils.WordsToBytes(&x.buf, tmp[:])
		n = copy(out, tmp[off:])
	}
	return n
}

func (x *XOF) fillBuf() {
	compress(&x.chain, &x.block, x.counter, x.blen, x.flags, &x.buf)
	x.counter++
	x.bufn = consts.BlockLen
}
