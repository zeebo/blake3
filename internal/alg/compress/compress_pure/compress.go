package compress_pure

import (
	"encoding/binary"
	"math/bits"

	"github.com/zeebo/blake3/internal/consts"
)

func Compress(
	chain *[32]byte,
	m *[64]byte,
	counter uint64,
	blen uint32,
	flags uint32,
	out *[64]byte,
) {

	s := [16]uint32{
		binary.LittleEndian.Uint32(chain[0:]),
		binary.LittleEndian.Uint32(chain[4:]),
		binary.LittleEndian.Uint32(chain[8:]),
		binary.LittleEndian.Uint32(chain[12:]),
		binary.LittleEndian.Uint32(chain[16:]),
		binary.LittleEndian.Uint32(chain[20:]),
		binary.LittleEndian.Uint32(chain[24:]),
		binary.LittleEndian.Uint32(chain[28:]),
		consts.IV0, consts.IV1, consts.IV2, consts.IV3,
		uint32(counter), uint32(counter >> 32), blen, flags,
	}

	rcompress(&s, (*block)(m), (*output)(out))
}

// A block is the 64-byte message block of one compression.
type block [64]byte

// w returns message word i of the block:
// the m_i of the specification, read little-endian.
func (m *block) w(i int) uint32 {
	return binary.LittleEndian.Uint32(m[4*i:])
}

// An output is the 64-byte result of one compression.
type output [64]byte

// setw writes v as word i of the output, little-endian.
func (o *output) setw(i int, v uint32) {
	binary.LittleEndian.PutUint32(o[4*i:], v)
}

func g(a, b, c, d, mx, my uint32) (uint32, uint32, uint32, uint32) {
	a += b + mx
	d = bits.RotateLeft32(d^a, -16)
	c += d
	b = bits.RotateLeft32(b^c, -12)
	a += b + my
	d = bits.RotateLeft32(d^a, -8)
	c += d
	b = bits.RotateLeft32(b^c, -7)
	return a, b, c, d
}

func rcompress(s *[16]uint32, m *block, out *output) {
	const (
		a = 10
		b = 11
		c = 12
		d = 13
		e = 14
		f = 15
	)

	s0, s1, s2, s3 := s[0+0], s[0+1], s[0+2], s[0+3]
	s4, s5, s6, s7 := s[0+4], s[0+5], s[0+6], s[0+7]
	s8, s9, sa, sb := s[8+0], s[8+1], s[8+2], s[8+3]
	sc, sd, se, sf := s[8+4], s[8+5], s[8+6], s[8+7]

	s0, s4, s8, sc = g(s0, s4, s8, sc, m.w(0), m.w(1))
	s1, s5, s9, sd = g(s1, s5, s9, sd, m.w(2), m.w(3))
	s2, s6, sa, se = g(s2, s6, sa, se, m.w(4), m.w(5))
	s3, s7, sb, sf = g(s3, s7, sb, sf, m.w(6), m.w(7))
	s0, s5, sa, sf = g(s0, s5, sa, sf, m.w(8), m.w(9))
	s1, s6, sb, sc = g(s1, s6, sb, sc, m.w(a), m.w(b))
	s2, s7, s8, sd = g(s2, s7, s8, sd, m.w(c), m.w(d))
	s3, s4, s9, se = g(s3, s4, s9, se, m.w(e), m.w(f))

	s0, s4, s8, sc = g(s0, s4, s8, sc, m.w(2), m.w(6))
	s1, s5, s9, sd = g(s1, s5, s9, sd, m.w(3), m.w(a))
	s2, s6, sa, se = g(s2, s6, sa, se, m.w(7), m.w(0))
	s3, s7, sb, sf = g(s3, s7, sb, sf, m.w(4), m.w(d))
	s0, s5, sa, sf = g(s0, s5, sa, sf, m.w(1), m.w(b))
	s1, s6, sb, sc = g(s1, s6, sb, sc, m.w(c), m.w(5))
	s2, s7, s8, sd = g(s2, s7, s8, sd, m.w(9), m.w(e))
	s3, s4, s9, se = g(s3, s4, s9, se, m.w(f), m.w(8))

	s0, s4, s8, sc = g(s0, s4, s8, sc, m.w(3), m.w(4))
	s1, s5, s9, sd = g(s1, s5, s9, sd, m.w(a), m.w(c))
	s2, s6, sa, se = g(s2, s6, sa, se, m.w(d), m.w(2))
	s3, s7, sb, sf = g(s3, s7, sb, sf, m.w(7), m.w(e))
	s0, s5, sa, sf = g(s0, s5, sa, sf, m.w(6), m.w(5))
	s1, s6, sb, sc = g(s1, s6, sb, sc, m.w(9), m.w(0))
	s2, s7, s8, sd = g(s2, s7, s8, sd, m.w(b), m.w(f))
	s3, s4, s9, se = g(s3, s4, s9, se, m.w(8), m.w(1))

	s0, s4, s8, sc = g(s0, s4, s8, sc, m.w(a), m.w(7))
	s1, s5, s9, sd = g(s1, s5, s9, sd, m.w(c), m.w(9))
	s2, s6, sa, se = g(s2, s6, sa, se, m.w(e), m.w(3))
	s3, s7, sb, sf = g(s3, s7, sb, sf, m.w(d), m.w(f))
	s0, s5, sa, sf = g(s0, s5, sa, sf, m.w(4), m.w(0))
	s1, s6, sb, sc = g(s1, s6, sb, sc, m.w(b), m.w(2))
	s2, s7, s8, sd = g(s2, s7, s8, sd, m.w(5), m.w(8))
	s3, s4, s9, se = g(s3, s4, s9, se, m.w(1), m.w(6))

	s0, s4, s8, sc = g(s0, s4, s8, sc, m.w(c), m.w(d))
	s1, s5, s9, sd = g(s1, s5, s9, sd, m.w(9), m.w(b))
	s2, s6, sa, se = g(s2, s6, sa, se, m.w(f), m.w(a))
	s3, s7, sb, sf = g(s3, s7, sb, sf, m.w(e), m.w(8))
	s0, s5, sa, sf = g(s0, s5, sa, sf, m.w(7), m.w(2))
	s1, s6, sb, sc = g(s1, s6, sb, sc, m.w(5), m.w(3))
	s2, s7, s8, sd = g(s2, s7, s8, sd, m.w(0), m.w(1))
	s3, s4, s9, se = g(s3, s4, s9, se, m.w(6), m.w(4))

	s0, s4, s8, sc = g(s0, s4, s8, sc, m.w(9), m.w(e))
	s1, s5, s9, sd = g(s1, s5, s9, sd, m.w(b), m.w(5))
	s2, s6, sa, se = g(s2, s6, sa, se, m.w(8), m.w(c))
	s3, s7, sb, sf = g(s3, s7, sb, sf, m.w(f), m.w(1))
	s0, s5, sa, sf = g(s0, s5, sa, sf, m.w(d), m.w(3))
	s1, s6, sb, sc = g(s1, s6, sb, sc, m.w(0), m.w(a))
	s2, s7, s8, sd = g(s2, s7, s8, sd, m.w(2), m.w(6))
	s3, s4, s9, se = g(s3, s4, s9, se, m.w(4), m.w(7))

	s0, s4, s8, sc = g(s0, s4, s8, sc, m.w(b), m.w(f))
	s1, s5, s9, sd = g(s1, s5, s9, sd, m.w(5), m.w(0))
	s2, s6, sa, se = g(s2, s6, sa, se, m.w(1), m.w(9))
	s3, s7, sb, sf = g(s3, s7, sb, sf, m.w(8), m.w(6))
	s0, s5, sa, sf = g(s0, s5, sa, sf, m.w(e), m.w(a))
	s1, s6, sb, sc = g(s1, s6, sb, sc, m.w(2), m.w(c))
	s2, s7, s8, sd = g(s2, s7, s8, sd, m.w(3), m.w(4))
	s3, s4, s9, se = g(s3, s4, s9, se, m.w(7), m.w(d))

	out.setw(0, s0^s8)
	out.setw(1, s1^s9)
	out.setw(2, s2^sa)
	out.setw(3, s3^sb)
	out.setw(4, s4^sc)
	out.setw(5, s5^sd)
	out.setw(6, s6^se)
	out.setw(7, s7^sf)

	out.setw(8, s8^s[0])
	out.setw(9, s9^s[1])
	out.setw(a, sa^s[2])
	out.setw(b, sb^s[3])
	out.setw(c, sc^s[4])
	out.setw(d, sd^s[5])
	out.setw(e, se^s[6])
	out.setw(f, sf^s[7])
}
