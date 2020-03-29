package blake3

import (
	"encoding/binary"
	"unsafe"
)

type chainVector = [64]uint32

var isLittleEndian = *(*uint32)(unsafe.Pointer(&[4]byte{0, 0, 0, 1})) != 1

var iv = [8]uint32{iv0, iv1, iv2, iv3, iv4, iv5, iv6, iv7}

const (
	iv0 = 0x6A09E667
	iv1 = 0xBB67AE85
	iv2 = 0x3C6EF372
	iv3 = 0xA54FF53A
	iv4 = 0x510E527F
	iv5 = 0x9B05688C
	iv6 = 0x1F83D9AB
	iv7 = 0x5BE0CD19
)

const (
	flag_chunkStart uint32 = 1 << 0
	flag_chunkEnd   uint32 = 1 << 1
	flag_parent     uint32 = 1 << 2
	flag_root       uint32 = 1 << 3
	flag_keyed      uint32 = 1 << 4
	// flag_keyCtx     uint32 = 1 << 5
	// flag_keyMat     uint32 = 1 << 6
)

const (
	blockLen = 64
	chunkLen = 1024
)

func bytesToWords(bytes *[64]uint8, words *[16]uint32) {
	words[0] = binary.LittleEndian.Uint32(bytes[0*4:])
	words[1] = binary.LittleEndian.Uint32(bytes[1*4:])
	words[2] = binary.LittleEndian.Uint32(bytes[2*4:])
	words[3] = binary.LittleEndian.Uint32(bytes[3*4:])
	words[4] = binary.LittleEndian.Uint32(bytes[4*4:])
	words[5] = binary.LittleEndian.Uint32(bytes[5*4:])
	words[6] = binary.LittleEndian.Uint32(bytes[6*4:])
	words[7] = binary.LittleEndian.Uint32(bytes[7*4:])
	words[8] = binary.LittleEndian.Uint32(bytes[8*4:])
	words[9] = binary.LittleEndian.Uint32(bytes[9*4:])
	words[10] = binary.LittleEndian.Uint32(bytes[10*4:])
	words[11] = binary.LittleEndian.Uint32(bytes[11*4:])
	words[12] = binary.LittleEndian.Uint32(bytes[12*4:])
	words[13] = binary.LittleEndian.Uint32(bytes[13*4:])
	words[14] = binary.LittleEndian.Uint32(bytes[14*4:])
	words[15] = binary.LittleEndian.Uint32(bytes[15*4:])
}
