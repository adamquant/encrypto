package ui

import (
	"fmt"
	"time"
)

type RetroProgress struct {
	totalBytes int64
	operation  string
	startTime  time.Time
}

func NewRetroProgress(totalBytes int64) *RetroProgress {
	return &RetroProgress{totalBytes: totalBytes}
}

func (rp *RetroProgress) Start(operation string) {
	rp.operation = operation
	rp.startTime = time.Now()
	fmt.Printf("\r[%s] Starting...", rp.operation)
}

func (rp *RetroProgress) Update(bytesProcessed int64) {
	percent := float64(bytesProcessed) / float64(rp.totalBytes) * 100
	elapsed := time.Since(rp.startTime).Seconds()
	rate := float64(bytesProcessed) / elapsed / 1024 / 1024
	fmt.Printf("\r[%s] %.1f%% (%.1f MB/s)", rp.operation, percent, rate)
}

func (rp *RetroProgress) Stop(completed bool, message string) {
	if completed {
		fmt.Printf("\r[%s] 100%% - Done!\n", rp.operation)
	} else {
		fmt.Printf("\r[%s] Failed: %s\n", rp.operation, message)
	}
}
