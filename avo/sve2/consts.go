package main

var iv = [8]uint32{
	0x6A09E667, 0xBB67AE85, 0x3C6EF372, 0xA54FF53A,
	0x510E527F, 0x9B05688C, 0x1F83D9AB, 0x5BE0CD19,
}

const (
	flagChunkStart = 1 << 0
	flagChunkEnd   = 1 << 1
)

var perm = [16]int{2, 6, 3, 10, 7, 0, 4, 13, 1, 11, 12, 5, 9, 14, 15, 8}

var msgSchedule = func() [7][16]int {
	var s [7][16]int
	for i := range s[0] {
		s[0][i] = i
	}
	for r := 1; r < len(s); r++ {
		for i := range s[r] {
			s[r][i] = s[r-1][perm[i]]
		}
	}
	return s
}()

var rot8Shuf = [4]uint32{0x00030201, 0x04070605, 0x080B0A09, 0x0C0F0E0D}

func emitData(a *asm) {
	for r := 0; r < 4; r++ {
		for l := 0; l < 4; l++ {
			a.line("DATA iv_rows<>+%d(SB)/4, $0x%08x", 16*r+4*l, iv[r])
		}
	}
	a.line("GLOBL iv_rows<>(SB), RODATA|NOPTR, $64")
	a.line("")
	for i, w := range rot8Shuf {
		a.line("DATA rot8_shuf<>+%d(SB)/4, $0x%08x", 4*i, w)
	}
	a.line("GLOBL rot8_shuf<>(SB), RODATA|NOPTR, $16")
}
