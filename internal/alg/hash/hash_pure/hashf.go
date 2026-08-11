package hash_pure

import (
	"github.com/zeebo/blake3/internal/alg/compress"
	"github.com/zeebo/blake3/internal/consts"
	"github.com/zeebo/blake3/internal/utils"
)

func HashF(input *[8192]byte, length, counter uint64, flags uint32, key *[32]byte, out *[64]uint32, chain *[8]uint32) {
	var tmp [2][64]byte

	k := *key // compressing from a stack copy is measurably faster

	for i := uint64(0); consts.ChunkLen*i < length && i < 8; i++ {
		bchain := &k
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
				*chain = utils.ChainFromBytes(bchain)
			}

			block := (*[64]byte)(input[consts.ChunkLen*i+consts.BlockLen*n:])

			compress.Compress(bchain, block, counter, consts.BlockLen, bflags, &tmp[n&1])

			bchain = (*[32]byte)(tmp[n&1][0:32])
			bflags = flags
		}

		cv := utils.ChainFromBytes(bchain)
		out[i+0] = cv[0]
		out[i+8] = cv[1]
		out[i+16] = cv[2]
		out[i+24] = cv[3]
		out[i+32] = cv[4]
		out[i+40] = cv[5]
		out[i+48] = cv[6]
		out[i+56] = cv[7]

		counter++
	}
}
