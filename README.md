# BLAKE3
<!-- [![GoDoc](https://godoc.org/github.com/zeebo/blake3?status.svg)](https://godoc.org/github.com/zeebo/blake3)
[![Sourcegraph](https://sourcegraph.com/github.com/zeebo/blake3/-/badge.svg)](https://sourcegraph.com/github.com/zeebo/blake3?badge)
[![Go Report Card](https://goreportcard.com/badge/github.com/zeebo/blake3)](https://goreportcard.com/report/github.com/zeebo/blake3) -->

Pure Go implementation of [BLAKE3](https://blake3.io) with AVX2 and SSE4.1 acceleration.

Special thanks to the excellent [avo](https://github.com/mmcloughlin/avo) making writing vectorized version much easier.

# Benchmarks

- All benchmarks run on my i7-6700K, with no control for noise or throttling or anything. So take these results with a bunch of salt.
- Incremental means writes of 1 kilobyte. A new hash object is created each time (worst case).
- Entire means writing the entire buffer in a single update. A new hash object is created each time (likely case).
- Reset means writing the entire buffer in a single update. Hash state is reused through a `sync.Pool` and reset (best case).

## AVX2+SSE41

### Graphs with Rust comparison

![barchart](/assets/barchart.png)

### Small

| Size   | Entire     | Entire + Reset | | Entire Rate  | Entire + Reset Rate |
|--------|------------|----------------|-|--------------|---------------------|
| 64 b   |  `205 ns`  |  `86.5 ns`     | |  `312 MB/s`  |   `740 MB/s`        |
| 256 b  |  `364 ns`  |   `250 ns`     | |  `703 MB/s`  |  `1.03 GB/s`        |
| 512 b  |  `575 ns`  |   `468 ns`     | |  `892 MB/s`  |  `1.10 GB/s`        |
| 768 b  |  `795 ns`  |   `682 ns`     | |  `967 MB/s`  |  `1.13 GB/s`        |

### Large

| Size          | Incremental | Entire      | Entire + Reset | | Incremental Rate | Entire Rate   | Entire + Reset Rate |
|---------------|-------------|-------------|----------------|-|------------------|---------------|---------------------|
| 1 kib         |  `1.02 µs`  |  `1.01 µs`  |   `891 ns`     | |  `1.00 GB/s`     |  `1.01 GB/s`  |  `1.15 GB/s`        |
| 2 kib         |  `2.11 µs`  |  `2.07 µs`  |  `1.95 µs`     | |   `968 MB/s`     |   `990 MB/s`  |  `1.05 GB/s`        |
| 4 kib         |  `2.28 µs`  |  `2.15 µs`  |  `2.05 µs`     | |  `1.80 GB/s`     |  `1.90 GB/s`  |  `2.00 GB/s`        |
| 8 kib         |  `2.64 µs`  |  `2.52 µs`  |  `2.44 µs`     | |  `3.11 GB/s`     |  `3.25 GB/s`  |  `3.36 GB/s`        |
| 16 kib        |  `4.93 µs`  |  `4.54 µs`  |  `4.48 µs`     | |  `3.33 GB/s`     |  `3.61 GB/s`  |  `3.66 GB/s`        |
| 32 kib        |  `9.41 µs`  |  `8.62 µs`  |  `8.54 µs`     | |  `3.48 GB/s`     |  `3.80 GB/s`  |  `3.84 GB/s`        |
| 64 kib        |  `18.2 µs`  |  `16.7 µs`  |  `16.6 µs`     | |  `3.59 GB/s`     |  `3.91 GB/s`  |  `3.94 GB/s`        |
| 128 kib       |  `36.3 µs`  |  `32.9 µs`  |  `33.1 µs`     | |  `3.61 GB/s`     |  `3.99 GB/s`  |  `3.96 GB/s`        |
| 256 kib       |  `72.5 µs`  |  `65.7 µs`  |  `66.0 µs`     | |  `3.62 GB/s`     |  `3.99 GB/s`  |  `3.97 GB/s`        |
| 512 kib       |   `145 µs`  |   `131 µs`  |   `132 µs`     | |  `3.60 GB/s`     |  `4.00 GB/s`  |  `3.97 GB/s`        |
| 1024 kib      |   `290 µs`  |   `262 µs`  |   `262 µs`     | |  `3.62 GB/s`     |  `4.00 GB/s`  |  `4.00 GB/s`        |

## No ASM

| Size          | Incremental | Entire      | Entire + Reset | | Incremental Rate | Entire Rate  | Entire + Reset Rate |
|---------------|-------------|-------------|----------------|-|------------------|--------------|---------------------|
| 64 b          |   `253 ns`  |   `254 ns`  |   `134 ns`     | |  `253 MB/s`      |  `252 MB/s`  |  `478 MB/s`         |
| 256 b         |   `553 ns`  |   `557 ns`  |   `441 ns`     | |  `463 MB/s`      |  `459 MB/s`  |  `580 MB/s`         |
| 512 b         |   `948 ns`  |   `953 ns`  |   `841 ns`     | |  `540 MB/s`      |  `538 MB/s`  |  `609 MB/s`         |
| 768 b         |  `1.38 µs`  |  `1.40 µs`  |  `1.35 µs`     | |  `558 MB/s`      |  `547 MB/s`  |  `570 MB/s`         |
| 1 kib         |  `1.77 µs`  |  `1.77 µs`  |  `1.70 µs`     | |  `577 MB/s`      |  `580 MB/s`  |  `602 MB/s`         |
|               |             |             |                | |                  |              |                     |
| 1024 kib      |   `880 µs`  |   `883 µs`  |   `878 µs`     | |  `596 MB/s`      |  `595 MB/s`  |  `598 MB/s`         |

- Rows elided from the no asm version as they all stabilize around the same rate.
