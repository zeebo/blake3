def load(path):
    out = {}
    for line in open(path):
        if not line.startswith('BenchmarkBLAKE3'):
            continue
        name, _, time, _, rate, _, _, _, _, _ = line.strip().split()
        _, kind, size = name.split('/')
        out[(kind, size)] = (int(float(time)), int(float(rate)))
    return out

def scale(ns):
    if ns < 1000:
        return float(ns), 'ns'
    if ns < 1000000:
        return ns / 1000, 'µs'
    return ns / 1000000, 'ms'

def print_short_row(bytes, size):
    fb = bench[('Entire', size)]
    reset = bench[('Reset', size)]
    fmt = '| {:6} | {:-3} ns      | {:-3} ns     | | {:-3} MB/s         | {:-3} MB/s     |'
    print(fmt.format(bytes, fb[0], reset[0], fb[1], reset[1]))

def print_row(bench, row, size):
    inc = bench[('Incremental', size)]
    fb = bench[('Entire', size)]
    reset = bench[('Reset', size)]

    incr, incs = scale(inc[0])
    fbr, fbs = scale(fb[0])
    resetr, resets = scale(reset[0])

    fmt = '| {:8} | {:-5.4} {}    | {:-5.4} {}    | {:-5.4} {}   | | {:-4} MB/s        | {:-4} MB/s        | {:-4} MB/s    |'
    print(fmt.format(row, incr, incs, fbr, fbs, resetr, resets, inc[1], fb[1], reset[1]))

bench = load('bench.txt')
bench_pure = load('bench-pure.txt')

print("### Small")
print()
print('| Size   | Full Buffer |  Reset     | | Full Buffer Rate | Reset Rate   |')
print('|--------|-------------|------------|-|------------------|--------------|')
print_short_row('64 b', '0001_block')
print_short_row('256 b', '0004_block')
print_short_row('512 b', '0008_block')
print_short_row('768 b', '0012_block')
print()
print("### Large")
print()
print('| Size     | Incremental | Full Buffer | Reset      | | Incremental Rate | Full Buffer Rate | Reset Rate   |')
print('|----------|-------------|-------------|------------|-|------------------|------------------|--------------|')
print_row(bench, '1 kib', '0001_kib')
print_row(bench, '2 kib', '0002_kib')
print_row(bench, '4 kib', '0004_kib')
print_row(bench, '8 kib', '0008_kib')
print_row(bench, '16 kib', '0016_kib')
print_row(bench, '32 kib', '0032_kib')
print_row(bench, '64 kib', '0064_kib')
print_row(bench, '128 kib', '0128_kib')
print_row(bench, '256 kib', '0256_kib')
print_row(bench, '512 kib', '0512_kib')
print_row(bench, '1024 kib', '1024_kib')
print()
print("### No ASM")
print()
print('| Size     | Incremental | Full Buffer | Reset      | | Incremental Rate | Full Buffer Rate | Reset Rate   |')
print('|----------|-------------|-------------|------------|-|------------------|------------------|--------------|')
print_row(bench_pure, '64 b',    '0001_block')
print_row(bench_pure, '256 b',    '0004_block')
print_row(bench_pure, '512 b',    '0008_block')
print_row(bench_pure, '768 b',    '0012_block')
print_row(bench_pure, '1 kib',    '0016_block')
print('|          |             |             |            | |                  |                  |              |')
print_row(bench_pure, '1 mib',    '1024_kib')
