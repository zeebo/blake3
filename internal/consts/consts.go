package consts

// IV is IV0 through IV7 serialized little-endian.
var IV = [32]byte{
	IV0 & 0xff, IV0 >> 8 & 0xff, IV0 >> 16 & 0xff, IV0 >> 24,
	IV1 & 0xff, IV1 >> 8 & 0xff, IV1 >> 16 & 0xff, IV1 >> 24,
	IV2 & 0xff, IV2 >> 8 & 0xff, IV2 >> 16 & 0xff, IV2 >> 24,
	IV3 & 0xff, IV3 >> 8 & 0xff, IV3 >> 16 & 0xff, IV3 >> 24,
	IV4 & 0xff, IV4 >> 8 & 0xff, IV4 >> 16 & 0xff, IV4 >> 24,
	IV5 & 0xff, IV5 >> 8 & 0xff, IV5 >> 16 & 0xff, IV5 >> 24,
	IV6 & 0xff, IV6 >> 8 & 0xff, IV6 >> 16 & 0xff, IV6 >> 24,
	IV7 & 0xff, IV7 >> 8 & 0xff, IV7 >> 16 & 0xff, IV7 >> 24,
}

const (
	IV0 = 0x6A09E667
	IV1 = 0xBB67AE85
	IV2 = 0x3C6EF372
	IV3 = 0xA54FF53A
	IV4 = 0x510E527F
	IV5 = 0x9B05688C
	IV6 = 0x1F83D9AB
	IV7 = 0x5BE0CD19
)

const (
	Flag_ChunkStart        uint32 = 1 << 0
	Flag_ChunkEnd          uint32 = 1 << 1
	Flag_Parent            uint32 = 1 << 2
	Flag_Root              uint32 = 1 << 3
	Flag_Keyed             uint32 = 1 << 4
	Flag_DeriveKeyContext  uint32 = 1 << 5
	Flag_DeriveKeyMaterial uint32 = 1 << 6
)

const (
	BlockLen = 64
	ChunkLen = 1024
)
