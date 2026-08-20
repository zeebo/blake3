package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
	. "github.com/zeebo/blake3/_asm"
)

// A chunkGroup is the register assignment for four chunks hashed together.
type chunkGroup struct {
	rows  int // rows[0..3] at ZmmRegs[rows..rows+3]
	poolA int // message pool A
	poolB int // message pool B
	tmp   int // scratch register
	msg   int // byte offset of this group's chunks in the input
	row3  int // byte offset of this group's row 3 data in the arena
}

func pack(a, b, c, d int) U8 {
	return U8(a<<6 | b<<4 | c<<2 | d)
}

func gvF(g chunkGroup, m, rotB int) {
	r := g.rows
	rotD := 16
	if rotB == 7 {
		rotD = 8
	}
	VPADDD(ZmmRegs[m], ZmmRegs[r], ZmmRegs[r])
	VPADDD(ZmmRegs[r+1], ZmmRegs[r], ZmmRegs[r])
	VPXORD(ZmmRegs[r], ZmmRegs[r+3], ZmmRegs[r+3])
	VPRORD(U8(rotD), ZmmRegs[r+3], ZmmRegs[r+3])
	VPADDD(ZmmRegs[r+3], ZmmRegs[r+2], ZmmRegs[r+2])
	VPXORD(ZmmRegs[r+2], ZmmRegs[r+1], ZmmRegs[r+1])
	VPRORD(U8(rotB), ZmmRegs[r+1], ZmmRegs[r+1])
}

func diagF(g chunkGroup) {
	r := g.rows
	VPSHUFD(pack(2, 1, 0, 3), ZmmRegs[r], ZmmRegs[r])
	VPSHUFD(pack(1, 0, 3, 2), ZmmRegs[r+3], ZmmRegs[r+3])
	VPSHUFD(pack(0, 3, 2, 1), ZmmRegs[r+2], ZmmRegs[r+2])
}

func undiagF(g chunkGroup) {
	r := g.rows
	VPSHUFD(pack(0, 3, 2, 1), ZmmRegs[r], ZmmRegs[r])
	VPSHUFD(pack(1, 0, 3, 2), ZmmRegs[r+3], ZmmRegs[r+3])
	VPSHUFD(pack(2, 1, 0, 3), ZmmRegs[r+2], ZmmRegs[r+2])
}

func round1MsgF(g chunkGroup) {
	a, b := g.poolA, g.poolB
	VSHUFPS(pack(2, 0, 2, 0), ZmmRegs[a+1], ZmmRegs[a], ZmmRegs[b])
	VSHUFPS(pack(3, 1, 3, 1), ZmmRegs[a+1], ZmmRegs[a], ZmmRegs[b+1])
	VSHUFPS(pack(2, 0, 2, 0), ZmmRegs[a+3], ZmmRegs[a+2], ZmmRegs[b+2])
	VSHUFPS(pack(2, 1, 0, 3), ZmmRegs[b+2], ZmmRegs[b+2], ZmmRegs[b+2])
	VSHUFPS(pack(3, 1, 3, 1), ZmmRegs[a+3], ZmmRegs[a+2], ZmmRegs[b+3])
	VSHUFPS(pack(2, 1, 0, 3), ZmmRegs[b+3], ZmmRegs[b+3], ZmmRegs[b+3])
}

func permuteMsgF(g chunkGroup, p, q int) {
	t := g.tmp
	VSHUFPS(pack(3, 1, 1, 2), ZmmRegs[p+1], ZmmRegs[p], ZmmRegs[q])
	VSHUFPS(pack(0, 3, 2, 1), ZmmRegs[q], ZmmRegs[q], ZmmRegs[q])
	VSHUFPS(pack(3, 3, 2, 2), ZmmRegs[p+3], ZmmRegs[p+2], ZmmRegs[q+1])
	VPSHUFD(pack(0, 0, 3, 3), ZmmRegs[p], ZmmRegs[t])
	VPBLENDMD(ZmmRegs[t], ZmmRegs[q+1], K1, ZmmRegs[q+1])
	VPUNPCKLDQ(ZmmRegs[p+1], ZmmRegs[p+3], ZmmRegs[q+2])
	VPBLENDMD(ZmmRegs[p+2], ZmmRegs[q+2], K2, ZmmRegs[q+2])
	VSHUFPS(pack(2, 3, 1, 0), ZmmRegs[q+2], ZmmRegs[q+2], ZmmRegs[q+2])
	VPUNPCKHDQ(ZmmRegs[p+3], ZmmRegs[p+1], ZmmRegs[t])
	VPUNPCKLDQ(ZmmRegs[t], ZmmRegs[p+2], ZmmRegs[q+3])
	VSHUFPS(pack(0, 1, 3, 2), ZmmRegs[q+3], ZmmRegs[q+3], ZmmRegs[q+3])
}

// HashF hashes up to 8 chunks in parallel. Every vector holds one state row of
// four chunks, one chunk per 128-bit lane, so a chunk's four G columns share a
// lane and the rounds never shuffle across one.
func HashF(c Ctx) {
	TEXT("HashF", 0, `func(
		input *[8192]byte,
		length uint64,
		counter uint64,
		flags uint32,
		key *[8]uint32,
		out *[64]uint32,
		chain *[8]uint32,
	)`)

	var (
		input   = Mem{Base: Load(Param("input"), GP64())}
		length  = Load(Param("length"), GP64()).(GPVirtual)
		counter = Load(Param("counter"), GP64()).(GPVirtual)
		flags   = Load(Param("flags"), GP32()).(GPVirtual)
		key     = Mem{Base: Load(Param("key"), GP64())}
		out     = Mem{Base: Load(Param("out"), GP64())}
		chain   = Mem{Base: Load(Param("chain"), GP64())}
	)

	chunks := GP64()
	blocks := GP64()
	stash := GP64()

	// All of the locals share one 64-byte aligned arena, so no vector slot straddles a cache line.
	const (
		arenaStart = 0
		arenaMid   = arenaStart + 128
		arenaEnd   = arenaMid + 128
		arenaChain = arenaEnd + 128
		arenaSize  = arenaChain + 256
	)

	{
		Comment("Allocate local space and align it")
		local := AllocLocal(arenaSize + 64)
		LEAQ(local.Offset(63), stash)
		ANDQ(I32(^63), stash)
	}

	arena := Mem{Base: stash}

	{
		Comment("Compute complete chunks and blocks")
		XORQ(chunks, chunks)
		XORQ(blocks, blocks)
		TESTQ(length, length)
		JZ(LabelRef("skip_compute"))

		// chunks = (length - 1) / 1024, blocks = (length - 1) % 1024 / 64 * 64
		SUBQ(U8(1), length)
		MOVQ(length, chunks)
		SHRQ(U8(10), chunks)
		MOVQ(length, blocks)
		ANDQ(U32(960), blocks)
	}

	Label("skip_compute")

	{
		Comment("Build the counters and block length of every chunk")
		ctr := GP64()
		for i := 0; i < 8; i++ {
			LEAQ(Mem{Base: counter, Disp: i}, ctr)
			MOVL(ctr.As32(), arena.Offset(arenaStart+16*i))
			SHRQ(U8(32), ctr)
			MOVL(ctr.As32(), arena.Offset(arenaStart+16*i+4))
			MOVL(U32(64), arena.Offset(arenaStart+16*i+8))
		}
		VMOVDQU32(arena.Offset(arenaStart), ZmmRegs[0])
		VMOVDQU32(arena.Offset(arenaStart+64), ZmmRegs[1])
		VMOVDQU32(ZmmRegs[0], arena.Offset(arenaMid))
		VMOVDQU32(ZmmRegs[1], arena.Offset(arenaMid+64))
		VMOVDQU32(ZmmRegs[0], arena.Offset(arenaEnd))
		VMOVDQU32(ZmmRegs[1], arena.Offset(arenaEnd+64))

		Comment("Build the first, middle, and last block flags")
		bflags := GP32()
		MOVL(flags, bflags)
		ORL(U8(flag_chunkStart), bflags)
		for i := 0; i < 8; i++ {
			MOVL(bflags, arena.Offset(arenaStart+16*i+12))
		}
		for i := 0; i < 8; i++ {
			MOVL(flags, arena.Offset(arenaMid+16*i+12))
		}
		MOVL(flags, bflags)
		ORL(U8(flag_chunkEnd), bflags)
		for i := 0; i < 8; i++ {
			MOVL(bflags, arena.Offset(arenaEnd+16*i+12))
		}
	}

	{
		Comment("Set up the blend masks for the message permutation")
		mask := GP32()
		MOVL(U32(0x5555), mask)
		KMOVW(mask, K1)
		MOVL(U32(0x8888), mask)
		KMOVW(mask, K2)
	}

	{
		Comment("Four chunks or fewer need only the narrow kernel")
		CMPQ(chunks, U8(4))
		JB(LabelRef("narrow"))
	}

	g1 := chunkGroup{rows: 0, poolA: 4, poolB: 8, tmp: 12, msg: 0, row3: 0}
	g2 := chunkGroup{rows: 13, poolA: 17, poolB: 21, tmp: 25, msg: 4096, row3: 64}

	emit := func(name string, groups []chunkGroup) {
		loop := GP64()
		row3_ptr := GP64()

		{
			Comment("Load the key into the chaining value rows of every chunk")
			for _, g := range groups {
				VBROADCASTI32X4(key.Offset(0), ZmmRegs[g.rows])
				VBROADCASTI32X4(key.Offset(16), ZmmRegs[g.rows+1])
			}
			XORQ(loop, loop)
			LEAQ(arena.Offset(arenaStart), row3_ptr)
		}

		Label(name + "_loop")

		{
			Comment("Include end flags if last block")
			CMPQ(loop, U32(15*64))
			JNE(LabelRef(name + "_flags_done"))
			LEAQ(arena.Offset(arenaEnd), row3_ptr)
		}

		Label(name + "_flags_done")

		{
			Comment("Load and group the message words of every chunk")
			for _, g := range groups {
				for k := 0; k < 4; k++ {
					z := g.poolA + k
					VMOVDQU32(input.Idx(loop, 1).Offset(g.msg+16*k), ZmmRegs[z].AsX())
					for l := 1; l < 4; l++ {
						VINSERTI32X4(U8(l), input.Idx(loop, 1).Offset(g.msg+1024*l+16*k), ZmmRegs[z], ZmmRegs[z])
					}
				}
			}
		}

		{
			Comment("Build rows 2 and 3 from the IV, counters, and flags")
			for _, g := range groups {
				VBROADCASTI32X4(c.IV.Offset(0), ZmmRegs[g.rows+2])
				VMOVDQU32(Mem{Base: row3_ptr}.Offset(g.row3), ZmmRegs[g.rows+3])
			}
		}

		{
			Comment("Save the chaining value before the partial chunk boundary")
			CMPQ(loop, blocks)
			JNE(LabelRef(name + "_chain_done"))

			for i, g := range groups {
				VMOVDQU32(ZmmRegs[g.rows], arena.Offset(arenaChain+128*i))
				VMOVDQU32(ZmmRegs[g.rows+1], arena.Offset(arenaChain+128*i+64))
			}
			lane_off := GP64()
			half := GP64()
			tmp32 := GP32()
			MOVQ(chunks, lane_off)
			ANDQ(U8(3), lane_off)
			SHLQ(U8(4), lane_off)
			MOVQ(chunks, half)
			ANDQ(U8(4), half)
			SHLQ(U8(5), half)
			ADDQ(half, lane_off)
			for i := 0; i < 4; i++ {
				MOVL(arena.Offset(arenaChain+4*i).Idx(lane_off, 1), tmp32)
				MOVL(tmp32, chain.Offset(4*i))
				MOVL(arena.Offset(arenaChain+64+4*i).Idx(lane_off, 1), tmp32)
				MOVL(tmp32, chain.Offset(16+4*i))
			}
		}

		Label(name + "_chain_done")

		{
			Comment("Round 1")
			for _, g := range groups {
				round1MsgF(g)
				gvF(g, g.poolB, 12)
				gvF(g, g.poolB+1, 7)
				diagF(g)
				gvF(g, g.poolB+2, 12)
				gvF(g, g.poolB+3, 7)
				undiagF(g)
			}
		}

		for r := 2; r <= 7; r++ {
			Commentf("Round %d", r)
			for _, g := range groups {
				p, q := g.poolA, g.poolB
				if r%2 == 0 {
					p, q = g.poolB, g.poolA
				}
				permuteMsgF(g, p, q)
				gvF(g, q, 12)
				gvF(g, q+1, 7)
				diagF(g)
				gvF(g, q+2, 12)
				gvF(g, q+3, 7)
				undiagF(g)
			}
		}

		{
			Comment("Compute the chaining values for the next block")
			for _, g := range groups {
				VPXORD(ZmmRegs[g.rows+2], ZmmRegs[g.rows], ZmmRegs[g.rows])
				VPXORD(ZmmRegs[g.rows+3], ZmmRegs[g.rows+1], ZmmRegs[g.rows+1])
			}
		}

		{
			Comment("If we have zero complete chunks, we're done")
			CMPQ(chunks, U8(0))
			JNE(LabelRef(name + "_trailer"))
			CMPQ(blocks, loop)
			JEQ(LabelRef(name + "_finalize"))
		}

		Label(name + "_trailer")

		{
			Comment("Increment, use the middle-block flags, and loop")
			CMPQ(loop, U32(15*64))
			JEQ(LabelRef(name + "_finalize"))
			ADDQ(Imm(64), loop)
			LEAQ(arena.Offset(arenaMid), row3_ptr)
			JMP(LabelRef(name + "_loop"))
		}

		Label(name + "_finalize")

		{
			// VMOVDQU is VEX encoded and cannot reach Y16 and above, so the
			// transpose lands in the first group's message pool, free by now.
			Comment("Transpose the chaining values into the word-major out layout")
			first, second := groups[0], groups[len(groups)-1]
			for w := 0; w < 4; w++ {
				VMOVDQU32(c.Transpose.Offset(64*w), ZmmRegs[first.tmp])
				VMOVDQA32(ZmmRegs[first.rows], ZmmRegs[first.poolA])
				VPERMT2D(ZmmRegs[second.rows], ZmmRegs[first.tmp], ZmmRegs[first.poolA])
				VMOVDQU(ZmmRegs[first.poolA].AsY(), out.Offset(32*w))
				VMOVDQA32(ZmmRegs[first.rows+1], ZmmRegs[first.poolA+1])
				VPERMT2D(ZmmRegs[second.rows+1], ZmmRegs[first.tmp], ZmmRegs[first.poolA+1])
				VMOVDQU(ZmmRegs[first.poolA+1].AsY(), out.Offset(32*(4+w)))
			}
		}

		VZEROUPPER()
		RET()
	}

	// The dispatch above falls through into the wide kernel.
	emit("wide", []chunkGroup{g1, g2})

	Label("narrow")
	emit("narrow", []chunkGroup{g1})
}
