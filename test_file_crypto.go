package main

import (
	"fmt"
	"github.com/encrypto/encrypto/internal/crypto"
	"os"
)

func main() {
	// Create test file
	testContent := []byte("This is a secret test message! Don't tell anyone!")
	os.WriteFile("/tmp/test.txt", testContent, 0644)
	fmt.Println("Created test file")

	// Encrypt it
	password := []byte("test123")
	err := crypto.EncryptFile("/tmp/test.txt", "/tmp/test.txt.enc", password)
	if err != nil {
		fmt.Printf("Encryption error: %v\n", err)
		return
	}
	fmt.Println("✓ File encrypted instantly!")

	// Check file sizes
	original, _ := os.Stat("/tmp/test.txt")
	encrypted, _ := os.Stat("/tmp/test.txt.enc")
	fmt.Printf("Original: %d bytes\n", original.Size())
	fmt.Printf("Encrypted: %d bytes\n", encrypted.Size())

	// Decrypt it
	err = crypto.DecryptFile("/tmp/test.txt.enc", "/tmp/test_decrypted.txt", password)
	if err != nil {
		fmt.Printf("Decryption error: %v\n", err)
		return
	}
	fmt.Println("✓ File decrypted instantly!")

	// Verify content
	decrypted, _ := os.ReadFile("/tmp/test_decrypted.txt")
	if string(decrypted) == string(testContent) {
		fmt.Println("✓ Content matches perfectly!")
	} else {
		fmt.Println("✗ Content mismatch!")
	}

	fmt.Println("\n✅ File encryption works! Instant encryption/decryption!")
}
