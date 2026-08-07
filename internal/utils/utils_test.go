package utils

import (
	"testing"

	"github.com/zeebo/assert"
)

func TestChainFromBytes(t *testing.T) {
	var chain [32]byte
	for i := range chain {
		chain[i] = byte(i)
	}

	words := ChainFromBytes(&chain)
	for i, w := range words {
		b := 4 * uint32(i)
		assert.Equal(t, b|(b+1)<<8|(b+2)<<16|(b+3)<<24, w)
	}
}

func TestChainToBytes(t *testing.T) {
	var chain [8]uint32
	for i := range chain {
		b := 4 * uint32(i)
		chain[i] = b | (b+1)<<8 | (b+2)<<16 | (b+3)<<24
	}

	var dst [32]byte
	ChainToBytes(&chain, &dst)
	for i, v := range dst {
		assert.Equal(t, byte(i), v)
	}
}
