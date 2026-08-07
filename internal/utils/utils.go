package utils

import (
	"encoding/binary"
)

// ChainFromBytes reads the eight little-endian words of a chaining value.
func ChainFromBytes(chain *[32]byte) [8]uint32 {
	return [8]uint32{
		binary.LittleEndian.Uint32(chain[0:]),
		binary.LittleEndian.Uint32(chain[4:]),
		binary.LittleEndian.Uint32(chain[8:]),
		binary.LittleEndian.Uint32(chain[12:]),
		binary.LittleEndian.Uint32(chain[16:]),
		binary.LittleEndian.Uint32(chain[20:]),
		binary.LittleEndian.Uint32(chain[24:]),
		binary.LittleEndian.Uint32(chain[28:]),
	}
}

// ChainToBytes writes the eight chaining-value words as little-endian bytes.
func ChainToBytes(chain *[8]uint32, dst *[32]byte) {
	binary.LittleEndian.PutUint32(dst[0:], chain[0])
	binary.LittleEndian.PutUint32(dst[4:], chain[1])
	binary.LittleEndian.PutUint32(dst[8:], chain[2])
	binary.LittleEndian.PutUint32(dst[12:], chain[3])
	binary.LittleEndian.PutUint32(dst[16:], chain[4])
	binary.LittleEndian.PutUint32(dst[20:], chain[5])
	binary.LittleEndian.PutUint32(dst[24:], chain[6])
	binary.LittleEndian.PutUint32(dst[28:], chain[7])
}
