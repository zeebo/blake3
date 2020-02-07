package blake3

import (
	"fmt"
	"testing"
)

func BenchmarkIncremental(b *testing.B) {
	run := func(b *testing.B, size int) {
		h := new(avxHasher)
		out := make([]byte, 32)
		buf := make([]byte, size)
		b.ReportAllocs()
		b.SetBytes(int64(len(buf)))
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			h.update(buf)
			h.finalize(out)
			h.reset()
		}
	}

	for _, n := range []int{
		1, 4, 8, 12, 16,
	} {
		b.Run(fmt.Sprintf("%04d_block", n), func(b *testing.B) { run(b, n*64) })
	}

	for _, n := range []int{
		1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024,
	} {
		b.Run(fmt.Sprintf("%04d_kib", n), func(b *testing.B) { run(b, n*1024) })
		b.Run(fmt.Sprintf("%04d_kib+512", n), func(b *testing.B) { run(b, n*1024+512) })
	}
}

/*
BenchmarkIncremental/0001_block-8         76.2 ns/op      839.62 MB/s
test bench_atonce_0001_block                56 ns/iter   1142 MB/s

BenchmarkIncremental/0004_block-8          233 ns/op     1100.06 MB/s
test bench_atonce_0004_block               202 ns/iter   1267 MB/s

BenchmarkIncremental/0008_block-8          446 ns/op     1146.87 MB/s
test bench_atonce_0008_block               399 ns/iter   1283 MB/s

BenchmarkIncremental/0012_block-8          643 ns/op     1194.08 MB/s
test bench_atonce_0012_block               602 ns/iter   1275 MB/s

BenchmarkIncremental/0016_block-8          848 ns/op     1207.93 MB/s
test bench_atonce_0015_block               749 ns/iter   1281 MB/s

///////////////////////

BenchmarkIncremental/0001_kib-8            861 ns/op      1189.33 MB/s
test bench_atonce_0001_kib                 799 ns/iter    1281 MB/s

BenchmarkIncremental/0002_kib-8           1843 ns/op      1111.03 MB/s
test bench_atonce_0002_kib                1703 ns/iter    1202 MB/s

BenchmarkIncremental/0004_kib-8           2111 ns/op      1940.30 MB/s
test bench_atonce_0004_kib                2078 ns/iter    1971 MB/s

BenchmarkIncremental/0008_kib-8           2420 ns/op      3384.74 MB/s
test bench_atonce_0008_kib                2111 ns/iter    3880 MB/s

BenchmarkIncremental/0016_kib-8           4378 ns/op      3742.01 MB/s
test bench_atonce_0016_kib                4036 ns/iter    4059 MB/s

BenchmarkIncremental/0032_kib-8           8302 ns/op      3947.22 MB/s
test bench_atonce_0032_kib                7991 ns/iter    4100 MB/s

BenchmarkIncremental/0064_kib-8          16107 ns/op      4068.86 MB/s
test bench_atonce_0064_kib               15946 ns/iter    4109 MB/s

BenchmarkIncremental/0128_kib-8          32541 ns/op      4027.96 MB/s
test bench_atonce_0128_kib               31823 ns/iter    4118 MB/s

BenchmarkIncremental/0256_kib-8          63583 ns/op      4122.87 MB/s
test bench_atonce_0256_kib               63866 ns/iter    4104 MB/s

BenchmarkIncremental/0512_kib-8         129846 ns/op      4037.77 MB/s
test bench_atonce_0512_kib              128632 ns/iter    4075 MB/s

BenchmarkIncremental/1024_kib-8         254917 ns/op      4113.40 MB/s
test bench_atonce_1024_kib              257244 ns/iter    4076 MB/s

///////////////////////

BenchmarkIncremental/0001_kib+512-8       1848 ns/op      831.25 MB/s
test bench_atonce_0001_kib                1260 ns/iter   1219 MB/s

BenchmarkIncremental/0002_kib+512-8       2036 ns/op    1257.40 MB/s
test bench_atonce_0002_kib                2168 ns/iter  1180 MB/s

BenchmarkIncremental/0004_kib+512-8       2175 ns/op    2118.32 MB/s
test bench_atonce_0004_kib                2579 ns/iter  1786 MB/s

BenchmarkIncremental/0008_kib+512-8       3117 ns/op    2792.06 MB/s
test bench_atonce_0008_kib                2581 ns/iter  3372 MB/s

BenchmarkIncremental/0016_kib+512-8       5052 ns/op    3344.39 MB/s
test bench_atonce_0016_kib                4504 ns/iter  3751 MB/s

BenchmarkIncremental/0032_kib+512-8       9041 ns/op    3681.15 MB/s
test bench_atonce_0032_kib                8497 ns/iter  3916 MB/s

BenchmarkIncremental/0064_kib+512-8      17069 ns/op    3869.57 MB/s
test bench_atonce_0064_kib               16378 ns/iter  4032 MB/s

BenchmarkIncremental/0128_kib+512-8      32855 ns/op    4005.05 MB/s
test bench_atonce_0128_kib               32240 ns/iter  4081 MB/s

BenchmarkIncremental/0256_kib+512-8      64571 ns/op    4067.69 MB/s
test bench_atonce_0256_kib               64220 ns/iter  4089 MB/s

BenchmarkIncremental/0512_kib+512-8     127850 ns/op    4104.80 MB/s
test bench_atonce_0512_kib              128536 ns/iter  4082 MB/s

BenchmarkIncremental/1024_kib+512-8     256238 ns/op    4094.19 MB/s
test bench_atonce_1024_kib              256453 ns/iter  4090 MB/s
*/
