package ui

import (
	"github.com/filemax/filemax/pkg/termui"
)

// RetroProgress wraps the shared MatrixBox for encrypto
type RetroProgress struct {
	box *termui.MatrixBox
}

// NewRetroProgress creates a new retro-style progress tracker
func NewRetroProgress(totalBytes int64) *RetroProgress {
	return &RetroProgress{
		box: termui.NewMatrixBox("ENCRYPTO v1.0"),
	}
}

// Start begins the progress display
func (rp *RetroProgress) Start(operation string) {
	rp.box.Start(operation)
}

// Update updates the progress
func (rp *RetroProgress) Update(bytesProcessed int64) {
	rp.box.UpdateProcessed(bytesProcessed)
	rp.box.UpdateBytes(bytesProcessed, 0)
}

// Stop ends the progress display
func (rp *RetroProgress) Stop(completed bool, message string) {
	rp.box.Stop(completed, message)
}
