package encx

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
)

// DEK (Data Encryption Key) operations

// GenerateDEK generates a new Data Encryption Key.
func (c *Crypto) GenerateDEK() ([]byte, error) {
	dek := make([]byte, 32) // AES-256 key size
	_, err := io.ReadFull(rand.Reader, dek)
	if err != nil {
		return nil, fmt.Errorf("failed to generate DEK: %w", err)
	}
	return dek, nil
}

// EncryptDEK encrypts the DEK using the current active KEK.
func (c *Crypto) EncryptDEK(ctx context.Context, plaintextDEK []byte) ([]byte, error) {
	currentVersion, err := c.getCurrentKEKVersion(ctx, c.kekAlias)
	if err != nil {
		return nil, err
	}
	kmsKeyID, err := c.getKMSKeyIDForVersion(ctx, c.kekAlias, currentVersion)
	if err != nil {
		return nil, err
	}
	ciphertextDEK, err := c.kmsService.EncryptDEK(ctx, kmsKeyID, plaintextDEK)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt DEK with KMS (version %d): %w", currentVersion, err)
	}
	return ciphertextDEK, nil
}

// DecryptDEKWithVersion decrypts the DEK using the KEK version it was encrypted with.
// You'll need to store the KEKVersion alongside the EncryptedDEK in your data records.
func (c *Crypto) DecryptDEKWithVersion(ctx context.Context, ciphertextDEK []byte, kekVersion int) ([]byte, error) {
	kmsKeyID, err := c.getKMSKeyIDForVersion(ctx, c.kekAlias, kekVersion)
	if err != nil {
		return nil, err
	}
	plaintextDEK, err := c.kmsService.DecryptDEK(ctx, kmsKeyID, ciphertextDEK)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt DEK with KMS (version %d): %w", kekVersion, err)
	}
	return plaintextDEK, nil
}