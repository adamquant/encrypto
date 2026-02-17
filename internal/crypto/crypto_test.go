package crypto

import (
	"bytes"
	"testing"
)

func TestDeriveKey(t *testing.T) {
	salt := make([]byte, RandomSaltSize)
	for i := range salt {
		salt[i] = byte(i + 1)
	}

	key1, err := DeriveKey("password", salt)
	if err != nil {
		t.Fatal(err)
	}
	if len(key1) != KeySize {
		t.Errorf("key length = %d, want %d", len(key1), KeySize)
	}

	key2, err := DeriveKey("password", salt)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(key1, key2) {
		t.Error("same password and salt should produce same key")
	}

	key3, err := DeriveKey("different", salt)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(key1, key3) {
		t.Error("different passwords should produce different keys")
	}
}

func TestDeriveKeyInvalidSalt(t *testing.T) {
	_, err := DeriveKey("password", []byte("tooshort"))
	if err == nil {
		t.Error("expected error for invalid salt length")
	}
}

func TestGenerateSalt(t *testing.T) {
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}
	if len(salt) != RandomSaltSize {
		t.Errorf("salt length = %d, want %d", len(salt), RandomSaltSize)
	}

	salt2, err := GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(salt, salt2) {
		t.Error("two generated salts should not be equal")
	}
}

func TestGenerateRandomBytes(t *testing.T) {
	b, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 32 {
		t.Errorf("length = %d, want 32", len(b))
	}

	b2, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(b, b2) {
		t.Error("two random byte slices should not be equal")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	salt := make([]byte, RandomSaltSize)
	for i := range salt {
		salt[i] = byte(i + 1)
	}

	key, err := DeriveKey("testpassword", salt)
	if err != nil {
		t.Fatal(err)
	}

	plaintext := make([]byte, 512)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}

	c := New()
	ciphertext, err := c.Encrypt(plaintext, key, 0)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(plaintext, ciphertext) {
		t.Error("ciphertext should differ from plaintext")
	}

	decrypted, err := c.Decrypt(ciphertext, key, 0)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Error("decrypted data should match original plaintext")
	}
}

func TestEncryptSectorNumberAffectsOutput(t *testing.T) {
	salt := make([]byte, RandomSaltSize)
	key, err := DeriveKey("pass", salt)
	if err != nil {
		t.Fatal(err)
	}

	plaintext := make([]byte, 512)
	c := New()

	ct0, err := c.Encrypt(plaintext, key, 0)
	if err != nil {
		t.Fatal(err)
	}
	ct1, err := c.Encrypt(plaintext, key, 1)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(ct0, ct1) {
		t.Error("different sector numbers should produce different ciphertext")
	}
}

func TestEncryptInvalidKey(t *testing.T) {
	c := New()
	_, err := c.Encrypt(make([]byte, 512), make([]byte, 16), 0)
	if err == nil {
		t.Error("expected error for invalid key size")
	}
}

func TestDecryptInvalidKey(t *testing.T) {
	c := New()
	_, err := c.Decrypt(make([]byte, 512), make([]byte, 16), 0)
	if err == nil {
		t.Error("expected error for invalid key size")
	}
}

func TestHeaderEncodeDecodeRoundTrip(t *testing.T) {
	h := NewHeader()

	salt := make([]byte, 16)
	for i := range salt {
		salt[i] = byte(i + 5)
	}
	copy(h.Salt[:], salt)
	h.CreatedAt = 9999999

	encoded := h.Encode()
	if len(encoded) != 512 {
		t.Errorf("encoded length = %d, want 512", len(encoded))
	}

	var decoded Header
	if err := decoded.Decode(encoded); err != nil {
		t.Fatal(err)
	}

	if string(decoded.Magic[:]) != HeaderMagic {
		t.Errorf("magic = %q, want %q", string(decoded.Magic[:]), HeaderMagic)
	}
	if decoded.Version != HeaderVersion {
		t.Errorf("version = %d, want %d", decoded.Version, HeaderVersion)
	}
	if decoded.CreatedAt != h.CreatedAt {
		t.Errorf("created_at = %d, want %d", decoded.CreatedAt, h.CreatedAt)
	}
	if !bytes.Equal(decoded.Salt[:], h.Salt[:]) {
		t.Error("decoded salt should match original")
	}
}

func TestHeaderFlags(t *testing.T) {
	h := NewHeader()

	if h.IsHidden() {
		t.Error("new header should not have hidden flag set")
	}

	h.SetFlags(true)
	if !h.IsHidden() {
		t.Error("header should have hidden flag after SetFlags(true)")
	}

	h.SetFlags(false)
	if h.IsHidden() {
		t.Error("header should not have hidden flag after SetFlags(false)")
	}
}

func TestHeaderDecodeTooSmall(t *testing.T) {
	var h Header
	if err := h.Decode(make([]byte, 100)); err == nil {
		t.Error("expected error for header data smaller than 512 bytes")
	}
}

func TestNewHeader(t *testing.T) {
	h := NewHeader()
	if string(h.Magic[:]) != HeaderMagic {
		t.Errorf("magic = %q, want %q", string(h.Magic[:]), HeaderMagic)
	}
	if h.Version != HeaderVersion {
		t.Errorf("version = %d, want %d", h.Version, HeaderVersion)
	}
}
