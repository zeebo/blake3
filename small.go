package blake3

import (
	"unsafe"

	"github.com/zeebo/blake3/internal/consts"
	"github.com/zeebo/blake3/internal/utils"
)

func hashFSmall(input *[8192]byte, length, counter uint64, flags uint32, key *[8]uint32, out *[64]uint32, chain *[8]uint32) {
	var tmp [16]uint32

	for i := uint64(0); consts.ChunkLen*i < length && i < 8; i++ {
		bchain := *key
		bflags := flags | consts.Flag_ChunkStart
		start := consts.ChunkLen * i

		for n := uint64(0); n < 16; n++ {
			if n == 15 {
				bflags |= consts.Flag_ChunkEnd
			}
			if start+64*n >= length {
				break
			}
			if start+64+64*n >= length {
				*chain = bchain
			}

			var blockPtr *[16]uint32
			if consts.IsLittleEndian {
				blockPtr = (*[16]uint32)(unsafe.Pointer(&input[consts.ChunkLen*i+consts.BlockLen*n]))
			} else {
				var block [16]uint32
				utils.BytesToWords((*[64]uint8)(unsafe.Pointer(&input[consts.ChunkLen*i+consts.BlockLen*n])), &block)
				blockPtr = &block
			}

			compress(&bchain, blockPtr, counter, consts.BlockLen, bflags, &tmp)

			bchain = *(*[8]uint32)(unsafe.Pointer(&tmp[0]))
			bflags = flags
		}

		out[i+0] = bchain[0]
		out[i+8] = bchain[1]
		out[i+16] = bchain[2]
		out[i+24] = bchain[3]
		out[i+32] = bchain[4]
		out[i+40] = bchain[5]
		out[i+48] = bchain[6]
		out[i+56] = bchain[7]

		counter++
	}
}

func hashPSmall(left, right *chainVector, flags uint32, key *[8]uint32, out *chainVector, n int) {
	var tmp [16]uint32
	var block [16]uint32

	for i := 0; i < n && i < 8; i++ {
		block[0] = left[i+0]
		block[1] = left[i+8]
		block[2] = left[i+16]
		block[3] = left[i+24]
		block[4] = left[i+32]
		block[5] = left[i+40]
		block[6] = left[i+48]
		block[7] = left[i+56]
		block[8] = right[i+0]
		block[9] = right[i+8]
		block[10] = right[i+16]
		block[11] = right[i+24]
		block[12] = right[i+32]
		block[13] = right[i+40]
		block[14] = right[i+48]
		block[15] = right[i+56]

		compress(key, &block, 0, 64, flags, &tmp)

		out[i+0] = tmp[0]
		out[i+8] = tmp[1]
		out[i+16] = tmp[2]
		out[i+24] = tmp[3]
		out[i+32] = tmp[4]
		out[i+40] = tmp[5]
		out[i+48] = tmp[6]
		out[i+56] = tmp[7]
	}
}
