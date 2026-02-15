//go:build darwin

package drive

import (
	"fmt"
	"os"
	"syscall"
)

// DarwinDriveHandle implements DriveHandle for macOS using raw device I/O
type DarwinDriveHandle struct {
	file   *os.File
	path   string
	isOpen bool
}

// OpenDrive opens a raw disk device for reading/writing
func OpenDrive(path string) (DriveHandle, error) {
	// Open with O_RDWR for read/write, and O_EXCL to prevent concurrent access
	file, err := os.OpenFile(path, os.O_RDWR|syscall.O_EXCL, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open drive %s: %w", path, err)
	}

	return &DarwinDriveHandle{
		file:   file,
		path:   path,
		isOpen: true,
	}, nil
}

// Read reads data from the specified offset
func (d *DarwinDriveHandle) Read(offset int64, buf []byte) (int, error) {
	if !d.isOpen {
		return 0, fmt.Errorf("drive handle is closed")
	}

	// Seek to offset
	_, err := d.file.Seek(offset, 0)
	if err != nil {
		return 0, fmt.Errorf("seek failed: %w", err)
	}

	// Read data
	return d.file.Read(buf)
}

// Write writes data to the specified offset
func (d *DarwinDriveHandle) Write(offset int64, data []byte) error {
	if !d.isOpen {
		return fmt.Errorf("drive handle is closed")
	}

	// Seek to offset
	_, err := d.file.Seek(offset, 0)
	if err != nil {
		return fmt.Errorf("seek failed: %w", err)
	}

	// Write data
	_, err = d.file.Write(data)
	if err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	// Ensure data is flushed to disk (critical for interruption safety)
	err = d.file.Sync()
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	return nil
}

// Close closes the drive handle
func (d *DarwinDriveHandle) Close() error {
	if !d.isOpen {
		return nil
	}

	err := d.file.Close()
	d.isOpen = false
	return err
}

// openDarwinDrive opens a macOS disk device
func openDarwinDrive(path string) (DriveHandle, error) {
	return OpenDrive(path)
}
