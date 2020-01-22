package blake3

// func round_avx(v *[16][8]uint32, m *[16][8]uint32)
func hash8_avx(inputs *[8]*byte, blocks int, key *[8]uint32, counter, inc uint64, flags, flags_start, flags_end uint8, out *[256]byte, v, m *[16][8]uint32)
