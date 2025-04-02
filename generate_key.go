package encx

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateStringEncryptionKey() (string, error) {
	key, err := GenerateEncryptionKey()
	if err != nil {
		return "", nil
	}

	// Convert to hex string (equivalent to -hex flag in openssl)
	hexKey := hex.EncodeToString(key)

	return hexKey, nil

}

func GenerateEncryptionKey() ([]byte, error) {
	// Create a byte slice of length 32 (256 bits)
	key := make([]byte, 32)

	// Fill it with cryptographically secure random bytes
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}
