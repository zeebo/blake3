package blake3

import (
	"fmt"
	"io"
	"unsafe"

	"github.com/zeebo/blake3/internal/consts"
	"github.com/zeebo/blake3/internal/utils"
)

// Output captures the state of a Hasher allowing reading and seeking through
// the output stream.
type Output struct {
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
func (out *Output) Read(p []byte) (n int, err error) {
	n = len(p)

	if out.bufn > 0 {
		n := out.slowCopy(p)
		p = p[n:]
		out.bufn -= n
	}

	for len(p) >= 64 {
		out.fillBuf()

		if consts.IsLittleEndian {
			*(*[64]byte)(unsafe.Pointer(&p[0])) = *(*[64]byte)(unsafe.Pointer(&out.buf[0]))
		} else {
			utils.WordsToBytes(&out.buf, p)
		}

		p = p[64:]
		out.bufn = 0
	}

	if len(p) == 0 {
		return n, nil
	}

	out.fillBuf()
	out.bufn -= out.slowCopy(p)

	return n, nil
}

// Seek sets the position to the provided location. Only SeekStart and
// SeekCurrent are allowed.
func (out *Output) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
	case io.SeekEnd:
		return 0, fmt.Errorf("seek from end not supported")
	case io.SeekCurrent:
		offset += int64(consts.BlockLen*out.counter) - int64(out.bufn)
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}
	if offset < 0 {
		return 0, fmt.Errorf("seek before start")
	}
	out.setPosition(uint64(offset))
	return offset, nil
}

func (out *Output) setPosition(pos uint64) {
	out.counter = pos / consts.BlockLen
	out.fillBuf()
	out.bufn -= int(pos % consts.BlockLen)
}

func (out *Output) slowCopy(p []byte) (n int) {
	off := uint(consts.BlockLen-out.bufn) % consts.BlockLen
	if consts.IsLittleEndian {
		n = copy(p, (*[consts.BlockLen]byte)(unsafe.Pointer(&out.buf[0]))[off:])
	} else {
		var tmp [consts.BlockLen]byte
		utils.WordsToBytes(&out.buf, tmp[:])
		n = copy(p, tmp[off:])
	}
	return n
}

func (out *Output) fillBuf() {
	compress(&out.chain, &out.block, out.counter, out.blen, out.flags, &out.buf)
	out.counter++
	out.bufn = consts.BlockLen
}
