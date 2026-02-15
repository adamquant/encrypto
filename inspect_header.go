//go:build ignore

package main

import (
	"encoding/hex"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run inspect_header.go <device>")
		os.Exit(1)
	}

	device := os.Args[1]

	// Open device
	file, err := os.Open(device)
	if err != nil {
		fmt.Printf("Failed to open %s: %v\n", device, err)
		os.Exit(1)
	}
	defer file.Close()

	// Read first 512 bytes (header)
	header := make([]byte, 512)
	_, err = file.Read(header)
	if err != nil {
		fmt.Printf("Failed to read header: %v\n", err)
		os.Exit(1)
	}

	// Check magic
	magic := string(header[0:8])
	fmt.Printf("Magic: %s\n", magic)

	if magic != "ENCrypto" {
		fmt.Println("Not an encrypto header!")
		return
	}

	// Parse version
	version := header[8] | (header[9] << 8) | (header[10] << 16) | (header[11] << 24)
	fmt.Printf("Version: %d\n", version)

	// Parse salt
	salt := header[12:28]
	fmt.Printf("Salt: %s\n", hex.EncodeToString(salt))

	// Parse hashed key
	hashedKey := header[28:60]
	fmt.Printf("Hashed Key: %s\n", hex.EncodeToString(hashedKey))

	// Parse flags
	flags := header[88] | (header[89] << 8) | (header[90] << 16) | (header[91] << 24)
	fmt.Printf("Flags: %d (hidden=%v)\n", flags, flags&1 != 0)

	// Check if there's a hidden volume
	if flags&1 != 0 {
		hiddenSalt := header[100:116]
		fmt.Printf("Hidden Salt: %s\n", hex.EncodeToString(hiddenSalt))
		hashedHiddenKey := header[116:148]
		fmt.Printf("Hashed Hidden Key: %s\n", hex.EncodeToString(hashedHiddenKey))
	}

	fmt.Println("\nHeader inspection complete.")
	fmt.Println("The password was hashed with Argon2id and cannot be recovered.")
	fmt.Println("But we can verify if the header structure is valid.")
}
