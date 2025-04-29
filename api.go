package encx

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"reflect"

	"golang.org/x/crypto/argon2"
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

// EncryptData encrypts the provided data using the provided DEK.
func (c *Crypto) EncryptData(plaintext []byte, dek []byte) ([]byte, error) {
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptData decrypts the provided ciphertext using the provided DEK.
func (c *Crypto) DecryptData(ciphertext []byte, dek []byte) ([]byte, error) {
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("invalid ciphertext size")
	}
	nonce, ciphertextBytes := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return plaintext, nil
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

// DecryptDEK decrypts the DEK using the KEK version it was encrypted with.
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

// RotateKEK generates a new KEK and updates the metadata database.
func (c *Crypto) RotateKEK(ctx context.Context) error {
	currentVersion, err := c.getCurrentKEKVersion(ctx, c.kekAlias)
	if err != nil {
		return err
	}
	newVersion := currentVersion + 1
	kmsKeyID, err := c.kmsService.CreateKey(ctx, c.kekAlias) // KMS might handle rotation internally based on alias
	if err != nil {
		return fmt.Errorf("failed to create new KEK version in KMS: %w", err)
	}

	// Mark the previous version as deprecated
	_, err = c.keyMetadataDB.Exec(`
		UPDATE kek_versions SET is_deprecated = TRUE
		WHERE alias = ? AND version = ?
	`, c.kekAlias, currentVersion)
	if err != nil {
		return fmt.Errorf("failed to deprecate old KEK version: %w", err)
	}

	// Record the new KEK version
	_, err = c.keyMetadataDB.Exec(`
		INSERT INTO kek_versions (alias, version, kms_key_id) VALUES (?, ?, ?)
	`, c.kekAlias, newVersion, kmsKeyID)
	if err != nil {
		return fmt.Errorf("failed to record new KEK version in metadata DB: %w", err)
	}

	log.Printf("KEK rotated for alias '%s'. New version: %d, KMS ID: '%s'", c.kekAlias, newVersion, kmsKeyID)
	return nil
}

// HashBasic performs a basic SHA256 hash on the byte representation of the input.
func (c *Crypto) HashBasic(value []byte) string {
	valueHash := sha256.Sum256(value)
	return hex.EncodeToString(valueHash[:])
}

// HashSecure performs a secure Argon2id hash on the byte representation of the input,
// incorporating the configured Argon2 parameters and pepper.
func (c *Crypto) HashSecure(value []byte) (string, error) {
	if isZeroPepper(c.pepper) {
		return "", NewUninitalizedPepperError()
	}

	// Generate random salt
	salt := make([]byte, c.argon2Params.SaltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Combine value with pepper
	peppered := append(value, c.pepper[:]...)

	// Generate hash using Argon2id
	hash := argon2.IDKey(
		peppered,
		salt,
		c.argon2Params.Iterations,
		c.argon2Params.Memory,
		c.argon2Params.Parallelism,
		c.argon2Params.KeyLength,
	)

	// Encode params, salt, and hash into a string
	params := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		c.argon2Params.Memory,
		c.argon2Params.Iterations,
		c.argon2Params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return params, nil
}

func isZeroPepper(pepper []byte) bool {
	for _, b := range pepper {
		if b != 0 {
			return false
		}
	}
	return true
}

func (c *Crypto) CompareSecureHashAndValue(value any, hashValue string) (bool, error) {
	if value == nil {
		return false, fmt.Errorf("%w: value cannot be nil", ErrNilPointer)
	}
	v, err := c.serializer.Serialize(reflect.ValueOf(value))
	if err != nil {
		return false, fmt.Errorf("failed to serialize field value : %w", err)
	}
	valueHashed, err := c.HashSecure(v)
	if err != nil {
		return false, fmt.Errorf("secure hashing failed for value : %w", err)
	}
	return valueHashed == hashValue, nil
}

func (c *Crypto) CompareBasicHashAndValue(value any, hashValue string) (bool, error) {
	if value == nil {
		return false, fmt.Errorf("%w: value cannot be nil", ErrNilPointer)
	}
	v, err := c.serializer.Serialize(reflect.ValueOf(value))
	if err != nil {
		return false, fmt.Errorf("failed to serialize field value : %w", err)
	}
	return c.HashBasic(v) == hashValue, nil
}
