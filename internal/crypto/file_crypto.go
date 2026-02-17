package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// FileHeader contains metadata for encrypted files
type FileHeader struct {
	Magic   [8]byte
	Version uint32
	Salt    [16]byte
	Nonce   [12]byte
}

const FileMagic = "ENCFile1"

// EncryptFile encrypts a single file using AES-256-GCM
func EncryptFile(inputPath, outputPath string, password []byte) error {
	// Generate salt and nonce
	salt, err := GenerateRandomBytes(16)
	if err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Derive key
	key, err := DeriveKey(string(password), salt)
	if err != nil {
		return fmt.Errorf("failed to derive key: %w", err)
	}

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	// Open input file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Write header
	header := make([]byte, 40)
	copy(header[0:8], FileMagic)
	header[8] = 1 // version
	copy(header[9:25], salt)
	copy(header[25:37], nonce)

	if _, err := outputFile.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Read and encrypt file in chunks.
	// Each chunk gets a unique nonce: first 4 bytes are the random prefix from
	// the header nonce, last 8 bytes are the chunk counter. This guarantees
	// nonce uniqueness across all chunks while keeping the base nonce in the
	// header so decryption can reconstruct the same sequence.
	chunkNonce := make([]byte, 12)
	copy(chunkNonce, nonce[:4])
	buf := make([]byte, 64*1024) // 64KB chunks
	for chunkNum := uint64(0); ; chunkNum++ {
		binary.BigEndian.PutUint64(chunkNonce[4:], chunkNum)

		n, err := inputFile.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read input: %w", err)
		}
		if n == 0 {
			break
		}

		// Encrypt chunk
		ciphertext := aesgcm.Seal(nil, chunkNonce, buf[:n], nil)

		// Write encrypted chunk
		if _, err := outputFile.Write(ciphertext); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	}

	return nil
}

// DecryptFile decrypts a single file
func DecryptFile(inputPath, outputPath string, password []byte) error {
	// Open input file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	// Read header
	header := make([]byte, 40)
	if _, err := io.ReadFull(inputFile, header); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	// Verify magic
	if string(header[0:8]) != FileMagic {
		return fmt.Errorf("invalid file magic")
	}

	// Extract salt and nonce
	salt := header[9:25]
	nonce := header[25:37]

	// Derive key
	key, err := DeriveKey(string(password), salt)
	if err != nil {
		return fmt.Errorf("failed to derive key: %w", err)
	}

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Read and decrypt file using the same counter-based nonce as encryption.
	chunkNonce := make([]byte, 12)
	copy(chunkNonce, nonce[:4])
	for chunkNum := uint64(0); ; chunkNum++ {
		binary.BigEndian.PutUint64(chunkNonce[4:], chunkNum)

		// Each encrypted chunk is plaintext + 16-byte GCM tag
		encryptedChunk := make([]byte, 64*1024+16)
		n, err := inputFile.Read(encryptedChunk)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read input: %w", err)
		}
		if n == 0 {
			break
		}

		// Decrypt chunk
		plaintext, err := aesgcm.Open(nil, chunkNonce, encryptedChunk[:n], nil)
		if err != nil {
			return fmt.Errorf("failed to decrypt: %w (wrong password?)", err)
		}

		// Write decrypted chunk
		if _, err := outputFile.Write(plaintext); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	}

	return nil
}
