package encx

import (
	"crypto/rand"
	"encoding/hex"
	"io"
)

type Encryptor struct {
	KeyEncryptionKey  []byte // 32 bytes for AES-256
	Pepper            []byte // Additional security for password hashing
	Argon2Params      *Argon2Params
}

func New(encryptionKey string) (*Encryptor, error) {
	// Convert hex-encoded key to bytes
	key, err := hex.DecodeString(encryptionKey)
	if err != nil {
		return nil, err
	}
	// Generate random pepper
	pepper := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, pepper); err != nil {
		return nil, err
	}
	return &Encryptor{
		KeyEncryptionKey: key,
		Pepper:           pepper,
		Argon2Params:     DefaultArgon2Params(),
	}, nil
}
