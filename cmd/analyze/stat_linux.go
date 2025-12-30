//go:build !darwin

package main

import (
	"syscall"
	"time"
)

func getAtim(stat *syscall.Stat_t) time.Time {
	return time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
}

func getBlocks(stat *syscall.Stat_t) int64 {
	return int64(stat.Blocks)
}
