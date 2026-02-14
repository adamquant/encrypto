package crypto

import (
	"crypto/aes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/xts"
)

const (
	KeySize        = 32
	RandomSaltSize = 16
	SecretKeySize  = 32
	NonceSize      = 12
	TagSize        = 16
	HeaderMagic    = "ENCrypto"
	HeaderVersion  = 1
)

var (
	ErrDecryptionFailed = errors.New("crypto: decryption failed")
	ErrInvalidKey       = errors.New("crypto: invalid key")
)

type Crypto struct{}

func New() *Crypto {
	return &Crypto{}
}

func (c *Crypto) Encrypt(data, key []byte, sectorNum uint64) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}

	xtsCipher, err := xts.NewCipher(aes.NewCipher, key)
	if err != nil {
		return nil, fmt.Errorf("failed to create XTS cipher: %w", err)
	}

	ciphertext := make([]byte, len(data))
	xtsCipher.Encrypt(ciphertext, data, sectorNum)

	return ciphertext, nil
}

func (c *Crypto) Decrypt(data, key []byte, sectorNum uint64) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}

	xtsCipher, err := xts.NewCipher(aes.NewCipher, key)
	if err != nil {
		return nil, fmt.Errorf("failed to create XTS cipher: %w", err)
	}

	plaintext := make([]byte, len(data))
	xtsCipher.Decrypt(plaintext, data, sectorNum)

	return plaintext, nil
}

func DeriveKey(password string, salt []byte) ([]byte, error) {
	if len(salt) != RandomSaltSize {
		return nil, errors.New("crypto: salt must be 16 bytes")
	}

	key := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, KeySize)
	return key, nil
}

func GenerateSalt() ([]byte, error) {
	salt := make([]byte, RandomSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

func GenerateRandomBytes(size int) ([]byte, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return bytes, nil
}

func HashKey(key []byte) [32]byte {
	return sha256.Sum256(key)
}

type Header struct {
	Magic           [8]byte
	Version         uint32
	Salt            [16]byte
	HashedKey       [32]byte
	Nonce           [12]byte
	Tag             [16]byte
	Flags           uint32
	CreatedAt       uint64
	HiddenSalt      [16]byte
	HashedHiddenKey [32]byte
	HasHidden       bool
}

func (h *Header) Encode() []byte {
	data := make([]byte, 512)
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
	if len(data) < 512 {
		return errors.New("crypto: header too small")
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

func NewHeader() *Header {
	h := &Header{}
	copy(h.Magic[:], HeaderMagic)
	h.Version = HeaderVersion
	return h
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
