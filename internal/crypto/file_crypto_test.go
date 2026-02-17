package crypto

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecryptFileRoundTrip(t *testing.T) {
	dir := t.TempDir()

	original := []byte("secret message for testing encryption and decryption")
	inputPath := filepath.Join(dir, "input.txt")
	encPath := filepath.Join(dir, "input.enc")
	decPath := filepath.Join(dir, "output.txt")

	if err := os.WriteFile(inputPath, original, 0644); err != nil {
		t.Fatal(err)
	}

	password := []byte("testpassword123")

	if err := EncryptFile(inputPath, encPath, password); err != nil {
		t.Fatalf("EncryptFile: %v", err)
	}

	if _, err := os.Stat(encPath); err != nil {
		t.Fatal("encrypted file was not created")
	}

	if err := DecryptFile(encPath, decPath, password); err != nil {
		t.Fatalf("DecryptFile: %v", err)
	}

	decrypted, err := os.ReadFile(decPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(decrypted) != string(original) {
		t.Errorf("decrypted = %q, want %q", decrypted, original)
	}
}

func TestEncryptFileProducesLargerOutput(t *testing.T) {
	dir := t.TempDir()

	inputPath := filepath.Join(dir, "input.txt")
	encPath := filepath.Join(dir, "input.enc")
	content := []byte("some data to encrypt")

	if err := os.WriteFile(inputPath, content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := EncryptFile(inputPath, encPath, []byte("pass")); err != nil {
		t.Fatal(err)
	}

	origInfo, _ := os.Stat(inputPath)
	encInfo, _ := os.Stat(encPath)

	if encInfo.Size() <= origInfo.Size() {
		t.Error("encrypted file should be larger than original due to header and GCM overhead")
	}
}

func TestDecryptFileWrongPassword(t *testing.T) {
	dir := t.TempDir()

	inputPath := filepath.Join(dir, "input.txt")
	encPath := filepath.Join(dir, "input.enc")
	decPath := filepath.Join(dir, "output.txt")

	if err := os.WriteFile(inputPath, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := EncryptFile(inputPath, encPath, []byte("correct")); err != nil {
		t.Fatal(err)
	}

	if err := DecryptFile(encPath, decPath, []byte("wrong")); err == nil {
		t.Error("expected error when decrypting with wrong password")
	}
}

func TestDecryptFileInvalidMagic(t *testing.T) {
	dir := t.TempDir()

	fakePath := filepath.Join(dir, "fake.enc")
	data := make([]byte, 100)
	copy(data, "NOTVALID")
	if err := os.WriteFile(fakePath, data, 0644); err != nil {
		t.Fatal(err)
	}

	decPath := filepath.Join(dir, "output.txt")
	if err := DecryptFile(fakePath, decPath, []byte("password")); err == nil {
		t.Error("expected error for invalid file magic")
	}
}

func TestEncryptFileMissingInput(t *testing.T) {
	dir := t.TempDir()
	err := EncryptFile("/nonexistent/path.txt", filepath.Join(dir, "out.enc"), []byte("pass"))
	if err == nil {
		t.Error("expected error for missing input file")
	}
}

func TestDecryptFileMissingInput(t *testing.T) {
	dir := t.TempDir()
	err := DecryptFile("/nonexistent/path.enc", filepath.Join(dir, "out.txt"), []byte("pass"))
	if err == nil {
		t.Error("expected error for missing input file")
	}
}

func TestEncryptDecryptLargeFile(t *testing.T) {
	dir := t.TempDir()

	large := make([]byte, 256*1024)
	for i := range large {
		large[i] = byte(i % 251)
	}

	inputPath := filepath.Join(dir, "large.bin")
	encPath := filepath.Join(dir, "large.enc")
	decPath := filepath.Join(dir, "large_dec.bin")

	if err := os.WriteFile(inputPath, large, 0644); err != nil {
		t.Fatal(err)
	}

	password := []byte("largefilepassword")
	if err := EncryptFile(inputPath, encPath, password); err != nil {
		t.Fatalf("EncryptFile: %v", err)
	}
	if err := DecryptFile(encPath, decPath, password); err != nil {
		t.Fatalf("DecryptFile: %v", err)
	}

	decrypted, err := os.ReadFile(decPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(decrypted) != string(large) {
		t.Error("large file round-trip failed: content mismatch")
	}
}
