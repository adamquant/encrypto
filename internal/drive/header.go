package drive

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/encrypto/encrypto/internal/crypto"
)

const (
	HeaderMagic    = "ENCrypto"
	HeaderVersion  = 1
	HeaderSize     = 512
	HashedKeySize  = 32
	HiddenHashSize = 32
	NonceSize      = 12
	TagSize        = 16
)

type Header struct {
	Magic           [8]byte
	Version         uint32
	Salt            [16]byte
	HashedKey       [HashedKeySize]byte
	Nonce           [NonceSize]byte
	Tag             [TagSize]byte
	Flags           uint32
	CreatedAt       uint64
	HiddenSalt      [16]byte
	HashedHiddenKey [HashedKeySize]byte
	HasHidden       bool
}

func (h *Header) Encode() []byte {
	data := make([]byte, HeaderSize)
	copy(data[0:8], h.Magic[:])
	binary.LittleEndian.PutUint32(data[8:12], h.Version)
	copy(data[12:28], h.Salt[:])
	copy(data[28:60], h.HashedKey[:])
	copy(data[60:72], h.Nonce[:])
	copy(data[72:88], h.Tag[:])
	binary.LittleEndian.PutUint32(data[88:92], h.Flags)
	binary.LittleEndian.PutUint64(data[92:100], h.CreatedAt)

	if h.HasHidden {
		copy(data[100:116], h.HiddenSalt[:])
		copy(data[116:148], h.HashedHiddenKey[:])
	}

	return data
}

func (h *Header) Decode(data []byte) error {
	if len(data) < HeaderSize {
		return fmt.Errorf("header too small: %d bytes", len(data))
	}
	copy(h.Magic[:], data[0:8])
	h.Version = binary.LittleEndian.Uint32(data[8:12])
	copy(h.Salt[:], data[12:28])
	copy(h.HashedKey[:], data[28:60])
	copy(h.Nonce[:], data[60:72])
	copy(h.Tag[:], data[72:88])
	h.Flags = binary.LittleEndian.Uint32(data[88:92])
	h.CreatedAt = binary.LittleEndian.Uint64(data[92:100])

	if h.Flags&1 != 0 {
		h.HasHidden = true
		copy(h.HiddenSalt[:], data[100:116])
		copy(h.HashedHiddenKey[:], data[116:148])
	}

	return nil
}

func (h *Header) SetFlags(hidden bool) {
	if hidden {
		h.Flags |= 1
	} else {
		h.Flags &= ^uint32(1)
	}
}

func (h *Header) IsHidden() bool {
	return (h.Flags & 1) != 0
}

func createHeader(password []byte) (Header, error) {
	salt, err := crypto.GenerateRandomBytes(16)
	if err != nil {
		return Header{}, err
	}
	hashedKey, err := crypto.DeriveKey(string(password), salt)
	if err != nil {
		return Header{}, err
	}

	h := Header{}
	copy(h.Magic[:], HeaderMagic)
	h.Version = HeaderVersion
	copy(h.Salt[:], salt)
	copy(h.HashedKey[:], hashedKey)
	h.Flags = 0
	h.CreatedAt = uint64(1707859200)

	return h, nil
}

func createHeaderWithHidden(primaryPassword, hiddenPassword []byte) (Header, error) {
	salt, err := crypto.GenerateRandomBytes(16)
	if err != nil {
		return Header{}, err
	}
	primaryHashedKey, err := crypto.DeriveKey(string(primaryPassword), salt)
	if err != nil {
		return Header{}, err
	}

	hiddenSalt, err := crypto.GenerateRandomBytes(16)
	if err != nil {
		return Header{}, err
	}
	hashedHiddenKey, err := crypto.DeriveKey(string(hiddenPassword), hiddenSalt)
	if err != nil {
		return Header{}, err
	}

	h := Header{}
	copy(h.Magic[:], HeaderMagic)
	h.Version = HeaderVersion
	copy(h.Salt[:], salt)
	copy(h.HashedKey[:], primaryHashedKey)
	h.Flags = 1
	h.CreatedAt = uint64(1707859200)
	copy(h.HiddenSalt[:], hiddenSalt)
	copy(h.HashedHiddenKey[:], hashedHiddenKey)
	h.HasHidden = true

	return h, nil
}

func deriveKeyFromPassword(password []byte, header Header) ([]byte, bool, error) {
	primaryHashedKey, err := crypto.DeriveKey(string(password), header.Salt[:])
	if err != nil {
		return nil, false, err
	}

	if !verifyKey(primaryHashedKey, header) {
		if !header.HasHidden {
			return nil, false, fmt.Errorf("invalid password")
		}

		hiddenHashedKey, err := crypto.DeriveKey(string(password), header.HiddenSalt[:])
		if err != nil {
			return nil, false, err
		}
		if sha256.Sum256(hiddenHashedKey) == sha256.Sum256(header.HashedHiddenKey[:]) {
			return hiddenHashedKey, true, nil
		}

		return nil, false, fmt.Errorf("invalid password")
	}

	return primaryHashedKey, false, nil
}

func verifyKey(key []byte, header Header) bool {
	return sha256.Sum256(key) == sha256.Sum256(header.HashedKey[:])
}
