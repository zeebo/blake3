# BLAKE3
[![GoDoc](https://godoc.org/github.com/zeebo/blake3?status.svg)](https://godoc.org/github.com/zeebo/blake3)
[![Sourcegraph](https://sourcegraph.com/github.com/zeebo/blake3/-/badge.svg)](https://sourcegraph.com/github.com/zeebo/blake3?badge)
[![Go Report Card](https://goreportcard.com/badge/github.com/zeebo/blake3)](https://goreportcard.com/report/github.com/zeebo/blake3)

Pure Go implementation of [BLAKE3](https://blake3.io) with AVX2 and SSE4.1 acceleration.

Special thanks to the excellent [avo](https://github.com/mmcloughlin/avo) making writing vectorized version much easier.

# Benchmarks

- All benchmarks run on my i7-6700K, with no control for noise or throttling or anything.
- Incremental means writes of 1 kilobyte. A new hash object is created each time (worst case).
- Entire means writing the entire buffer in a single update. A new hash object is created each time (likely case).
- Reset means writing the entire buffer in a single update. Hash state is reused through a `sync.Pool` and reset (best case).

## Rust Comparison

![barchart](/assets/barchart.png)

- An attempt was made to get Go as close as possible to Rust for the benchmark. It probably failed.
- Only single-threaded performance was tested, and this Go version is only single-threaded.
- The Rust version does best when handed large buffers (8 kib or more). Be sure to hand it large buffers.
- There is no Reset method on the Rust version.

## AVX2 + SSE4.1

### Small

| Size   | Full Buffer |  Reset     | | Full Buffer Rate | Reset Rate   |
|--------|-------------|------------|-|------------------|--------------|
| 64 b   |  `205ns`    |  `86.5ns`  | |  `312MB/s`       |   `740MB/s`  |
| 256 b  |  `364ns`    |   `250ns`  | |  `703MB/s`       |  `1.03GB/s`  |
| 512 b  |  `575ns`    |   `468ns`  | |  `892MB/s`       |  `1.10GB/s`  |
| 768 b  |  `795ns`    |   `682ns`  | |  `967MB/s`       |  `1.13GB/s`  |

- Very small writes are mostly dominated by initialization of the hash state, so if you care about having the best performance for small inputs, be sure to reuse hash state as much as possible. If you don't care, you probably don't care about ~100ns either.

### Large

| Size     | Incremental | Full Buffer | Reset      | | Incremental Rate | Full Buffer Rate | Reset Rate   |
|----------|-------------|-------------|------------|-|------------------|------------------|--------------|
| 1 kib    |  `1.02µs`   |  `1.01µs`   |   `891ns`  | |  `1.00GB/s`      |  `1.01GB/s`      |  `1.15GB/s`  |
| 2 kib    |  `2.11µs`   |  `2.07µs`   |  `1.95µs`  | |   `968MB/s`      |   `990MB/s`      |  `1.05GB/s`  |
| 4 kib    |  `2.28µs`   |  `2.15µs`   |  `2.05µs`  | |  `1.80GB/s`      |  `1.90GB/s`      |  `2.00GB/s`  |
| 8 kib    |  `2.64µs`   |  `2.52µs`   |  `2.44µs`  | |  `3.11GB/s`      |  `3.25GB/s`      |  `3.36GB/s`  |
| 16 kib   |  `4.93µs`   |  `4.54µs`   |  `4.48µs`  | |  `3.33GB/s`      |  `3.61GB/s`      |  `3.66GB/s`  |
| 32 kib   |  `9.41µs`   |  `8.62µs`   |  `8.54µs`  | |  `3.48GB/s`      |  `3.80GB/s`      |  `3.84GB/s`  |
| 64 kib   |  `18.2µs`   |  `16.7µs`   |  `16.6µs`  | |  `3.59GB/s`      |  `3.91GB/s`      |  `3.94GB/s`  |
| 128 kib  |  `36.3µs`   |  `32.9µs`   |  `33.1µs`  | |  `3.61GB/s`      |  `3.99GB/s`      |  `3.96GB/s`  |
| 256 kib  |  `72.5µs`   |  `65.7µs`   |  `66.0µs`  | |  `3.62GB/s`      |  `3.99GB/s`      |  `3.97GB/s`  |
| 512 kib  |   `145µs`   |   `131µs`   |   `132µs`  | |  `3.60GB/s`      |  `4.00GB/s`      |  `3.97GB/s`  |
| 1024 kib |   `290µs`   |   `262µs`   |   `262µs`  | |  `3.62GB/s`      |  `4.00GB/s`      |  `4.00GB/s`  |

- Benchmarks of 1.5kib, 2.5kib, etc. have slightly slower rates, so have been omitted.

## No ASM

| Size     | Incremental | Full Buffer | Reset      | | Incremental Rate | Full Buffer Rate | Reset Rate  |
|----------|-------------|-------------|------------|-|------------------|------------------|-------------|
| 64 b     |   `253ns`   |   `254ns`   |   `134ns`  | |  `253MB/s`       |  `252MB/s`       |  `478MB/s`  |
| 256 b    |   `553ns`   |   `557ns`   |   `441ns`  | |  `463MB/s`       |  `459MB/s`       |  `580MB/s`  |
| 512 b    |   `948ns`   |   `953ns`   |   `841ns`  | |  `540MB/s`       |  `538MB/s`       |  `609MB/s`  |
| 768 b    |  `1.38µs`   |  `1.40µs`   |  `1.35µs`  | |  `558MB/s`       |  `547MB/s`       |  `570MB/s`  |
| 1 kib    |  `1.77µs`   |  `1.77µs`   |  `1.70µs`  | |  `577MB/s`       |  `580MB/s`       |  `602MB/s`  |
|          |             |             |            | |                  |                  |             |
| 1024 kib |   `880µs`   |   `883µs`   |   `878µs`  | |  `596MB/s`       |  `595MB/s`       |  `598MB/s`  |

- Rows elided from the no asm version as they all stabilize around the same rate.
