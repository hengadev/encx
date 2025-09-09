package crypto

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
)

// DEKOperations handles Data Encryption Key operations
type DEKOperations struct {
	kmsService KeyManagementService
	kekAlias   string
}

// KeyManagementService defines the interface for KMS operations needed by crypto package
type KeyManagementService interface {
	EncryptDEK(ctx context.Context, keyID string, plaintext []byte) ([]byte, error)
	DecryptDEK(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error)
}

// KMSVersionManager handles KEK version management for crypto operations
type KMSVersionManager interface {
	GetCurrentKEKVersion(ctx context.Context, alias string) (int, error)
	GetKMSKeyIDForVersion(ctx context.Context, alias string, version int) (string, error)
}

// NewDEKOperations creates a new DEKOperations instance
func NewDEKOperations(kmsService KeyManagementService, kekAlias string) *DEKOperations {
	return &DEKOperations{
		kmsService: kmsService,
		kekAlias:   kekAlias,
	}
}

// GenerateDEK generates a new Data Encryption Key.
func (d *DEKOperations) GenerateDEK() ([]byte, error) {
	dek := make([]byte, 32) // AES-256 key size
	_, err := io.ReadFull(rand.Reader, dek)
	if err != nil {
		return nil, fmt.Errorf("failed to generate DEK: %w", err)
	}
	return dek, nil
}

// EncryptDEK encrypts the DEK using the current active KEK.
func (d *DEKOperations) EncryptDEK(ctx context.Context, plaintextDEK []byte, versionManager KMSVersionManager) ([]byte, error) {
	currentVersion, err := versionManager.GetCurrentKEKVersion(ctx, d.kekAlias)
	if err != nil {
		return nil, err
	}
	kmsKeyID, err := versionManager.GetKMSKeyIDForVersion(ctx, d.kekAlias, currentVersion)
	if err != nil {
		return nil, err
	}
	ciphertextDEK, err := d.kmsService.EncryptDEK(ctx, kmsKeyID, plaintextDEK)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt DEK with KMS (version %d): %w", currentVersion, err)
	}
	return ciphertextDEK, nil
}

// DecryptDEKWithVersion decrypts the DEK using the KEK version it was encrypted with.
// You'll need to store the KEKVersion alongside the EncryptedDEK in your data records.
func (d *DEKOperations) DecryptDEKWithVersion(ctx context.Context, ciphertextDEK []byte, kekVersion int, versionManager KMSVersionManager) ([]byte, error) {
	kmsKeyID, err := versionManager.GetKMSKeyIDForVersion(ctx, d.kekAlias, kekVersion)
	if err != nil {
		return nil, err
	}
	plaintextDEK, err := d.kmsService.DecryptDEK(ctx, kmsKeyID, ciphertextDEK)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt DEK with KMS (version %d): %w", kekVersion, err)
	}
	return plaintextDEK, nil
}

