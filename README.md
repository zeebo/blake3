
## Benchmarks

Usually within ~10% of the rust version on my machine. Worse at smaller sizes, but total nanoseconds.

```
RUST 0001_block                56 ns/op    1142 MB/s
RUST 0004_block               202 ns/op    1267 MB/s
RUST 0008_block               399 ns/op    1283 MB/s
RUST 0012_block               602 ns/op    1275 MB/s

GO   0001_block                75 ns/op     856 MB/s  +34%
GO   0004_block               231 ns/op    1106 MB/s  +14%
GO   0008_block               435 ns/op    1176 MB/s  +9%
GO   0012_block               697 ns/op    1102 MB/s  +16%

///////////////////////

RUST 0001_kib                 799 ns/op    1281 MB/s
RUST 0002_kib                1703 ns/op    1202 MB/s
RUST 0004_kib                2078 ns/op    1971 MB/s
RUST 0008_kib                2111 ns/op    3880 MB/s
RUST 0016_kib                4036 ns/op    4059 MB/s
RUST 0032_kib                7991 ns/op    4100 MB/s
RUST 0064_kib               15946 ns/op    4109 MB/s
RUST 0128_kib               31823 ns/op    4118 MB/s
RUST 0256_kib               63866 ns/op    4104 MB/s
RUST 0512_kib              128632 ns/op    4075 MB/s
RUST 1024_kib              257244 ns/op    4076 MB/s

GO   0001_kib                 845 ns/op    1211 MB/s  +6%
GO   0002_kib                1863 ns/op    1099 MB/s  +9%
GO   0004_kib                2108 ns/op    1943 MB/s  +1%
GO   0008_kib                2380 ns/op    3441 MB/s  +9%
GO   0016_kib                4298 ns/op    3812 MB/s  +6%
GO   0032_kib                8228 ns/op    3983 MB/s  +3%
GO   0064_kib               16011 ns/op    4093 MB/s  ---
GO   0128_kib               31610 ns/op    4147 MB/s  -1%
GO   0256_kib               63397 ns/op    4134 MB/s  -1%
GO   0512_kib              127181 ns/op    4122 MB/s  -1%
GO   1024_kib              252222 ns/op    4157 MB/s  -2%

///////////////////////

RUST 0001_kib+512            1260 ns/op    1219 MB/s
RUST 0002_kib+512            2168 ns/op    1180 MB/s
RUST 0004_kib+512            2579 ns/op    1786 MB/s
RUST 0008_kib+512            2581 ns/op    3372 MB/s
RUST 0016_kib+512            4504 ns/op    3751 MB/s
RUST 0032_kib+512            8497 ns/op    3916 MB/s
RUST 0064_kib+512           16378 ns/op    4032 MB/s
RUST 0128_kib+512           32240 ns/op    4081 MB/s
RUST 0256_kib+512           64220 ns/op    4089 MB/s
RUST 0512_kib+512          128536 ns/op    4082 MB/s
RUST 1024_kib+512          256453 ns/op    4090 MB/s

GO   0001_kib+512            1449 ns/op    1060 MB/s  +15%
GO   0002_kib+512            2006 ns/op    1276 MB/s  -8%
GO   0004_kib+512            2161 ns/op    2133 MB/s  -19%
GO   0008_kib+512            2688 ns/op    3238 MB/s  +4%
GO   0016_kib+512            4597 ns/op    3675 MB/s  +2%
GO   0032_kib+512            8544 ns/op    3895 MB/s  +1%
GO   0064_kib+512           16370 ns/op    4035 MB/s  ---
GO   0128_kib+512           31997 ns/op    4112 MB/s  -1%
GO   0256_kib+512           63918 ns/op    4109 MB/s  -1%
GO   0512_kib+512          128344 ns/op    4089 MB/s  ---
GO   1024_kib+512          252815 ns/op    4150 MB/s  -1%
```
