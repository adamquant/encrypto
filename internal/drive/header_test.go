package drive

import (
	"bytes"
	"testing"
)

func TestCreateHeader(t *testing.T) {
	h, err := createHeader([]byte("password"))
	if err != nil {
		t.Fatal(err)
	}

	if string(h.Magic[:]) != HeaderMagic {
		t.Errorf("magic = %q, want %q", string(h.Magic[:]), HeaderMagic)
	}
	if h.Version != HeaderVersion {
		t.Errorf("version = %d, want %d", h.Version, HeaderVersion)
	}
	if h.HasHidden {
		t.Error("standard header should not have hidden flag")
	}
	if h.IsHidden() {
		t.Error("standard header flags should not indicate hidden")
	}
}

func TestCreateHeaderWithHidden(t *testing.T) {
	h, err := createHeaderWithHidden([]byte("primary"), []byte("hidden"))
	if err != nil {
		t.Fatal(err)
	}

	if !h.HasHidden {
		t.Error("header should have HasHidden=true")
	}
	if !h.IsHidden() {
		t.Error("header flags should indicate hidden volume")
	}

	allZero := true
	for _, b := range h.HiddenSalt {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("hidden salt should not be all zeros")
	}
}

func TestHeaderEncodeDecodeRoundTrip(t *testing.T) {
	h, err := createHeader([]byte("roundtrip"))
	if err != nil {
		t.Fatal(err)
	}

	encoded := h.Encode()
	if len(encoded) != HeaderSize {
		t.Errorf("encoded length = %d, want %d", len(encoded), HeaderSize)
	}

	var decoded Header
	if err := decoded.Decode(encoded); err != nil {
		t.Fatal(err)
	}

	if string(decoded.Magic[:]) != HeaderMagic {
		t.Errorf("magic = %q, want %q", string(decoded.Magic[:]), HeaderMagic)
	}
	if decoded.Version != h.Version {
		t.Errorf("version = %d, want %d", decoded.Version, h.Version)
	}
	if !bytes.Equal(decoded.Salt[:], h.Salt[:]) {
		t.Error("decoded salt should match original")
	}
	if !bytes.Equal(decoded.HashedKey[:], h.HashedKey[:]) {
		t.Error("decoded hashed key should match original")
	}
}

func TestHeaderEncodeDecodeHidden(t *testing.T) {
	h, err := createHeaderWithHidden([]byte("primary"), []byte("hidden"))
	if err != nil {
		t.Fatal(err)
	}

	encoded := h.Encode()
	var decoded Header
	if err := decoded.Decode(encoded); err != nil {
		t.Fatal(err)
	}

	if !decoded.HasHidden {
		t.Error("decoded header should have HasHidden=true")
	}
	if !bytes.Equal(decoded.HiddenSalt[:], h.HiddenSalt[:]) {
		t.Error("decoded hidden salt should match original")
	}
	if !bytes.Equal(decoded.HashedHiddenKey[:], h.HashedHiddenKey[:]) {
		t.Error("decoded hashed hidden key should match original")
	}
}

func TestHeaderDecodeTooSmall(t *testing.T) {
	var h Header
	if err := h.Decode(make([]byte, 100)); err == nil {
		t.Error("expected error for header data smaller than HeaderSize")
	}
}

func TestHeaderSetFlags(t *testing.T) {
	var h Header

	if h.IsHidden() {
		t.Error("zero header should not have hidden flag")
	}

	h.SetFlags(true)
	if !h.IsHidden() {
		t.Error("should have hidden flag after SetFlags(true)")
	}

	h.SetFlags(false)
	if h.IsHidden() {
		t.Error("should not have hidden flag after SetFlags(false)")
	}
}

func TestVerifyKey(t *testing.T) {
	h, err := createHeader([]byte("verifytest"))
	if err != nil {
		t.Fatal(err)
	}

	if !verifyKey(h.HashedKey[:], h) {
		t.Error("verifyKey should return true for the stored key")
	}

	wrongKey := make([]byte, HashedKeySize)
	if verifyKey(wrongKey, h) {
		t.Error("verifyKey should return false for a wrong key")
	}
}

func TestDeriveKeyFromPasswordCorrect(t *testing.T) {
	h, err := createHeader([]byte("mypassword"))
	if err != nil {
		t.Fatal(err)
	}

	key, isHidden, err := deriveKeyFromPassword([]byte("mypassword"), h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isHidden {
		t.Error("should not be hidden volume for standard header")
	}
	if len(key) == 0 {
		t.Error("key should not be empty")
	}
}

func TestDeriveKeyFromPasswordWrong(t *testing.T) {
	h, err := createHeader([]byte("mypassword"))
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = deriveKeyFromPassword([]byte("wrongpassword"), h)
	if err == nil {
		t.Error("expected error for wrong password")
	}
}

func TestDeriveKeyFromPasswordHidden(t *testing.T) {
	h, err := createHeaderWithHidden([]byte("primary"), []byte("hidden"))
	if err != nil {
		t.Fatal(err)
	}

	_, isHidden, err := deriveKeyFromPassword([]byte("hidden"), h)
	if err != nil {
		t.Fatalf("unexpected error with hidden password: %v", err)
	}
	if !isHidden {
		t.Error("should detect hidden volume when using hidden password")
	}
}
