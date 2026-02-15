package ui

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// RetroProgress provides an 80s-style progress display
type RetroProgress struct {
	totalBytes     int64
	currentBytes   int64
	startTime      time.Time
	lastUpdateTime time.Time
	isRunning      bool
	interrupted    bool
	updateInterval time.Duration

	// Retro styling
	greenColor string
	resetColor string
	boxWidth   int
}

// NewRetroProgress creates a new retro-style progress tracker
func NewRetroProgress(totalBytes int64) *RetroProgress {
	return &RetroProgress{
		totalBytes:     totalBytes,
		startTime:      time.Now(),
		lastUpdateTime: time.Now(),
		updateInterval: 100 * time.Millisecond,
		greenColor:     "\033[32m",
		resetColor:     "\033[0m",
		boxWidth:       50,
	}
}

// Start begins the progress display
func (rp *RetroProgress) Start(operation string) {
	rp.isRunning = true
	rp.startTime = time.Now()
	rp.lastUpdateTime = time.Now()

	// Clear screen and hide cursor
	fmt.Print("\033[2J\033[H")
	fmt.Print("\033[?25l")

	// Setup Ctrl+C handler
	rp.setupInterruptHandler()

	// Draw initial box
	rp.drawBox(operation)
	rp.drawProgress()
}

// Update updates the progress
func (rp *RetroProgress) Update(bytesProcessed int64) {
	rp.currentBytes = bytesProcessed

	// Only update display if enough time has passed
	if time.Since(rp.lastUpdateTime) < rp.updateInterval {
		return
	}

	rp.lastUpdateTime = time.Now()
	rp.drawProgress()
}

// Stop ends the progress display
func (rp *RetroProgress) Stop(completed bool, message string) {
	rp.isRunning = false

	// Show final state
	rp.drawProgress()

	// Move to bottom of box
	fmt.Printf("\033[11;0H")

	if completed {
		fmt.Printf("%s║  STATUS: ✓ %-28s%s║%s\n",
			rp.greenColor, message, rp.greenColor, rp.resetColor)
	} else if rp.interrupted {
		fmt.Printf("%s║  STATUS: ✗ %-28s%s║%s\n",
			rp.greenColor, "ABORTED", rp.greenColor, rp.resetColor)
	}

	// Draw bottom border
	fmt.Printf("%s╚══════════════════════════════════════════╝%s\n",
		rp.greenColor, rp.resetColor)

	// Show cursor
	fmt.Print("\033[?25h")
}

// setupInterruptHandler sets up Ctrl+C handling
func (rp *RetroProgress) setupInterruptHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		rp.interrupted = true
		rp.Stop(false, "")
		fmt.Println("\n\n⚠️  Encryption aborted by user")
		os.Exit(1)
	}()
}

// drawBox draws the retro box frame
func (rp *RetroProgress) drawBox(operation string) {
	fmt.Printf("%s", rp.greenColor)
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Printf("║  ENCRYPTO v1.0 - %-23s║\n", operation)
	fmt.Println("║                                          ║")
	fmt.Println("║                                          ║")
	fmt.Println("║                                          ║")
	fmt.Println("║                                          ║")
	fmt.Println("║                                          ║")
	fmt.Println("║                                          ║")
	fmt.Println("║                                          ║")
	fmt.Println("║                                          ║")
	fmt.Println("║                                          ║")
	fmt.Printf("%s", rp.resetColor)
}

// drawProgress draws the current progress
func (rp *RetroProgress) drawProgress() {
	if rp.totalBytes == 0 {
		return
	}

	percent := float64(rp.currentBytes) * 100.0 / float64(rp.totalBytes)
	filled := int(percent / 2) // 50 chars wide = 2% per char
	if filled > 50 {
		filled = 50
	}

	// Calculate speed
	elapsed := time.Since(rp.startTime).Seconds()
	var speed float64
	if elapsed > 0 {
		speed = float64(rp.currentBytes) / elapsed / 1024 / 1024 // MB/s
	}

	// Calculate ETA
	var eta time.Duration
	if speed > 0 {
		remaining := rp.totalBytes - rp.currentBytes
		etaSeconds := float64(remaining) / (speed * 1024 * 1024)
		eta = time.Duration(etaSeconds) * time.Second
	}

	// Build progress bar
	bar := ""
	for i := 0; i < 50; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	// Position cursor and draw
	fmt.Printf("%s", rp.greenColor)

	// Line 3: Progress bar
	fmt.Printf("\033[3;3H[%s] %5.1f%%", bar, percent)

	// Line 5: Bytes
	fmt.Printf("\033[5;3HBYTES:   %7.1f / %7.1f GB",
		float64(rp.currentBytes)/1024/1024/1024,
		float64(rp.totalBytes)/1024/1024/1024)

	// Line 6: Speed
	fmt.Printf("\033[6;3HSPEED:  %6.1f MB/s", speed)

	// Line 8: ETA
	etaStr := rp.formatDuration(eta)
	fmt.Printf("\033[8;3HETA:    %s", etaStr)

	fmt.Printf("%s", rp.resetColor)
}

// formatDuration formats a duration as MM:SS
func (rp *RetroProgress) formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
