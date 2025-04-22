package encx

import (
	"context"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// exemple d'alias : user_data_kek

const defaultDBFileName = ".key_metadata.db" // Name of your SQLite database file

type KEKVersion struct {
	Alias        string    `gorm:"primaryKey"`
	Version      int       `gorm:"primaryKey;autoIncrement:false"`
	CreationTime time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	IsDeprecated bool      `gorm:"default:false"`
	KMSKeyID     string    // Identifier of the key in your KMS (e.g., Vault's transit key name, AWS KMS ARN)
}

// getKMSKeyIDForVersion retrieves the KMS Key ID for a specific KEK version and alias.
func (c *Crypto) getKMSKeyIDForVersion(ctx context.Context, alias string, version int) (string, error) {
	row := c.keyMetadataDB.QueryRowContext(ctx, `
		SELECT kms_key_id FROM kek_versions
		WHERE alias = ? AND version = ?
	`, alias, version)
	var kmsKeyID string
	err := row.Scan(&kmsKeyID)
	if err != nil {
		return "", fmt.Errorf("failed to get KMS Key ID for alias '%s' version %d: %w", alias, version, err)
	}
	return kmsKeyID, nil
}

// rotateKEK generates a new KEK and updates the metadata database.
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
