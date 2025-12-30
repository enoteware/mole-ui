//go:build darwin

package main

import (
	"syscall"
	"time"
)

func getAtim(stat *syscall.Stat_t) time.Time {
	return time.Unix(stat.Atimespec.Sec, stat.Atimespec.Nsec)
}

func getBlocks(stat *syscall.Stat_t) int64 {
	return stat.Blocks
}
