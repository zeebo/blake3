package consts

import "golang.org/x/sys/cpu"

const OptimizeLittleEndian = !cpu.IsBigEndian
