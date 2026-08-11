package hash_pure

import (
	"encoding/binary"

	"github.com/zeebo/blake3/internal/alg/compress"
)

func HashP(left, right *[64]uint32, flags uint32, key *[32]byte, out *[64]uint32, n int) {
	var tmp [64]byte

	k := *key // compressing from a stack copy is measurably faster
	var block [64]byte

	for i := 0; i < n && i < 8; i++ {
		binary.LittleEndian.PutUint32(block[0:], left[i+0])
		binary.LittleEndian.PutUint32(block[4:], left[i+8])
		binary.LittleEndian.PutUint32(block[8:], left[i+16])
		binary.LittleEndian.PutUint32(block[12:], left[i+24])
		binary.LittleEndian.PutUint32(block[16:], left[i+32])
		binary.LittleEndian.PutUint32(block[20:], left[i+40])
		binary.LittleEndian.PutUint32(block[24:], left[i+48])
		binary.LittleEndian.PutUint32(block[28:], left[i+56])
		binary.LittleEndian.PutUint32(block[32:], right[i+0])
		binary.LittleEndian.PutUint32(block[36:], right[i+8])
		binary.LittleEndian.PutUint32(block[40:], right[i+16])
		binary.LittleEndian.PutUint32(block[44:], right[i+24])
		binary.LittleEndian.PutUint32(block[48:], right[i+32])
		binary.LittleEndian.PutUint32(block[52:], right[i+40])
		binary.LittleEndian.PutUint32(block[56:], right[i+48])
		binary.LittleEndian.PutUint32(block[60:], right[i+56])

		compress.Compress(&k, &block, 0, 64, flags, &tmp)

		out[i+0] = binary.LittleEndian.Uint32(tmp[0:])
		out[i+8] = binary.LittleEndian.Uint32(tmp[4:])
		out[i+16] = binary.LittleEndian.Uint32(tmp[8:])
		out[i+24] = binary.LittleEndian.Uint32(tmp[12:])
		out[i+32] = binary.LittleEndian.Uint32(tmp[16:])
		out[i+40] = binary.LittleEndian.Uint32(tmp[20:])
		out[i+48] = binary.LittleEndian.Uint32(tmp[24:])
		out[i+56] = binary.LittleEndian.Uint32(tmp[28:])
	}
}
