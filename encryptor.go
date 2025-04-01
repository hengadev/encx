package encx

import (
	"crypto/rand"
	"encoding/hex"
	"io"
)

type Encryptor struct {
	KeyEncryptionKey []byte // 32 bytes for AES-256
	Pepper           []byte // Additional security for password hashing
	Argon2Params     *Argon2Params
}

func New(encryptionKey string, argon2params *Argon2Params) (*Encryptor, error) {
	var encryptor Encryptor
	// Convert hex-encoded key to bytes
	key, err := hex.DecodeString(encryptionKey)
	if err != nil {
		return nil, err
	}
	encryptor.KeyEncryptionKey = key

	// Generate random pepper
	pepper := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, pepper); err != nil {
		return nil, err
	}
	encryptor.Pepper = pepper

	if argon2params != nil {
		encryptor.Argon2Params = argon2params
	} else {
		encryptor.Argon2Params = DefaultArgon2Params()
	}

	return &encryptor, nil
}
