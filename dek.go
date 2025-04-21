package encx

import (
	"crypto/rand"
	"fmt"
	"io"
)

// GenerateDEK generates a new Data Encryption Key.
func (c *Crypto) GenerateDEK() ([]byte, error) {
	dek := make([]byte, 32) // AES-256 key size
	_, err := io.ReadFull(rand.Reader, dek)
	if err != nil {
		return nil, fmt.Errorf("failed to generate DEK: %w", err)
	}
	return dek, nil
}
