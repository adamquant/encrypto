//go:build ignore

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/encrypto/encrypto/internal/crypto"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run verify_password.go <salt_hex> <hashed_key_hex>")
		os.Exit(1)
	}

	saltHex := os.Args[1]
	storedHashHex := os.Args[2]

	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		fmt.Printf("Invalid salt: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Testing passwords against stored hash:\n")
	fmt.Printf("Stored hash: %s\n\n", storedHashHex)

	// Test common passwords
	testPasswords := []string{"123", "1234", "password", "", "admin", "test"}

	for _, pwd := range testPasswords {
		key, err := crypto.DeriveKey(pwd, salt)
		if err != nil {
			fmt.Printf("Error deriving key for '%s': %v\n", pwd, err)
			continue
		}

		// Hash the derived key (same as in header.go verifyKey)
		hash := sha256.Sum256(key)
		hashHex := hex.EncodeToString(hash[:])

		match := "NO"
		if hashHex == storedHashHex {
			match = "YES ✓"
		}

		fmt.Printf("Password '%s' -> %s [%s]\n", pwd, hashHex, match)
	}

	fmt.Println("\nIf none match, the password might have been read incorrectly during encryption.")
}
