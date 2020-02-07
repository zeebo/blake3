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
test bench_incremental_0001_block      ... bench:          61 ns/iter (+/- 0) = 1049 MB/s
test bench_incremental_0001_kib        ... bench:         800 ns/iter (+/- 3) = 1280 MB/s
test bench_incremental_0002_kib        ... bench:       1,727 ns/iter (+/- 13) = 1185 MB/s
test bench_incremental_0004_kib        ... bench:       2,097 ns/iter (+/- 36) = 1953 MB/s
test bench_incremental_0008_kib        ... bench:       2,124 ns/iter (+/- 59) = 3856 MB/s
test bench_incremental_0016_kib        ... bench:       4,049 ns/iter (+/- 26) = 4046 MB/s
test bench_incremental_0032_kib        ... bench:       8,018 ns/iter (+/- 162) = 4086 MB/s
test bench_incremental_0064_kib        ... bench:      15,917 ns/iter (+/- 347) = 4117 MB/s
test bench_incremental_0128_kib        ... bench:      31,690 ns/iter (+/- 650) = 4136 MB/s
test bench_incremental_0256_kib        ... bench:      64,031 ns/iter (+/- 893) = 4094 MB/s
test bench_incremental_0512_kib        ... bench:     128,437 ns/iter (+/- 1,659) = 4082 MB/s
test bench_incremental_1024_kib        ... bench:     257,244 ns/iter (+/- 3,465) = 4076 MB/s
*/

/*
BenchmarkIncremental/0001_block-8         	 3948670	        76.2 ns/op	 839.62 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0004_block-8         	 1288316	       233 ns/op	1100.06 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0008_block-8         	  685488	       446 ns/op	1146.87 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0012_block-8         	  460201	       643 ns/op	1194.08 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0016_block-8         	  351516	       848 ns/op	1207.93 MB/s	       0 B/op	       0 allocs/op

BenchmarkIncremental/0001_kib-8           	  354973	       861 ns/op	1189.33 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0002_kib-8           	  162572	      1843 ns/op	1111.03 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0004_kib-8           	  141768	      2111 ns/op	1940.30 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0008_kib-8           	  123852	      2420 ns/op	3384.74 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0016_kib-8           	   68227	      4378 ns/op	3742.01 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0032_kib-8           	   36218	      8302 ns/op	3947.22 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0064_kib-8           	   18076	     16107 ns/op	4068.86 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0128_kib-8           	    9330	     32541 ns/op	4027.96 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0256_kib-8           	    4430	     63583 ns/op	4122.87 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0512_kib-8           	    2307	    129846 ns/op	4037.77 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/1024_kib-8           	    1160	    254917 ns/op	4113.40 MB/s	       0 B/op	       0 allocs/op

BenchmarkIncremental/0001_kib+512-8       	  160516	      1848 ns/op	 831.25 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0002_kib+512-8       	  147438	      2036 ns/op	1257.40 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0004_kib+512-8       	  137811	      2175 ns/op	2118.32 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0008_kib+512-8       	   96183	      3117 ns/op	2792.06 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0016_kib+512-8       	   59749	      5052 ns/op	3344.39 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0032_kib+512-8       	   32965	      9041 ns/op	3681.15 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0064_kib+512-8       	   17696	     17069 ns/op	3869.57 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0128_kib+512-8       	    8005	     32855 ns/op	4005.05 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0256_kib+512-8       	    4662	     64571 ns/op	4067.69 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/0512_kib+512-8       	    2238	    127850 ns/op	4104.80 MB/s	       0 B/op	       0 allocs/op
BenchmarkIncremental/1024_kib+512-8       	    1173	    256238 ns/op	4094.19 MB/s	       0 B/op	       0 allocs/op
*/
