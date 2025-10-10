package crypto

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// KeyRotationOperations handles key rotation operations
type KeyRotationOperations struct {
	kmsService    KeyRotationService
	kekAlias      string
	keyMetadataDB *sql.DB
	observability ObservabilityHook
}

// KeyRotationService extends KeyManagementService for rotation operations
type KeyRotationService interface {
	KeyManagementService
	CreateKey(ctx context.Context, alias string) (string, error)
	GetKeyID(ctx context.Context, alias string) (string, error)
}

// ObservabilityHook defines observability operations for monitoring
type ObservabilityHook interface {
	OnProcessStart(ctx context.Context, operation string, metadata map[string]any)
	OnProcessComplete(ctx context.Context, operation string, duration time.Duration, err error, metadata map[string]any)
	OnError(ctx context.Context, operation string, err error, metadata map[string]any)
	OnKeyOperation(ctx context.Context, operation string, alias string, version int, metadata map[string]any)
}

// NewKeyRotationOperations creates a new KeyRotationOperations instance
func NewKeyRotationOperations(kmsService KeyRotationService, kekAlias string, keyMetadataDB *sql.DB, observability ObservabilityHook) (*KeyRotationOperations, error) {
	if kmsService == nil {
		return nil, fmt.Errorf("KMS service cannot be nil")
	}
	return &KeyRotationOperations{
		kmsService:    kmsService,
		kekAlias:      kekAlias,
		keyMetadataDB: keyMetadataDB,
		observability: observability,
	}, nil
}

// RotateKEK generates a new KEK and updates the metadata database.
func (kr *KeyRotationOperations) RotateKEK(ctx context.Context, versionManager KMSVersionManager) error {
	// Monitoring: Start key operation
	start := time.Now()
	metadata := map[string]any{
		"key_alias":      kr.kekAlias,
		"operation_type": "key_rotation",
	}
	if kr.observability != nil {
		kr.observability.OnProcessStart(ctx, "RotateKEK", metadata)
	}

	currentVersion, err := versionManager.GetCurrentKEKVersion(ctx, kr.kekAlias)
	if err != nil {
		if kr.observability != nil {
			kr.observability.OnError(ctx, "RotateKEK", err, metadata)
			kr.observability.OnProcessComplete(ctx, "RotateKEK", time.Since(start), err, metadata)
		}
		return err
	}

	newVersion := currentVersion + 1
	metadata["old_version"] = currentVersion
	metadata["new_version"] = newVersion

	kmsKeyID, err := kr.kmsService.CreateKey(ctx, kr.kekAlias) // KMS might handle rotation internally based on alias
	if err != nil {
		err = fmt.Errorf("failed to create new KEK version in KMS: %w", err)
		if kr.observability != nil {
			kr.observability.OnError(ctx, "RotateKEK", err, metadata)
			kr.observability.OnProcessComplete(ctx, "RotateKEK", time.Since(start), err, metadata)
		}
		return err
	}

	// Mark the previous version as deprecated
	_, err = kr.keyMetadataDB.Exec(`
		UPDATE kek_versions SET is_deprecated = TRUE
		WHERE alias = ? AND version = ?
	`, kr.kekAlias, currentVersion)
	if err != nil {
		err = fmt.Errorf("failed to deprecate old KEK version: %w", err)
		if kr.observability != nil {
			kr.observability.OnError(ctx, "RotateKEK", err, metadata)
			kr.observability.OnProcessComplete(ctx, "RotateKEK", time.Since(start), err, metadata)
		}
		return err
	}

	// Record the new KEK version
	_, err = kr.keyMetadataDB.Exec(`
		INSERT INTO kek_versions (alias, version, kms_key_id) VALUES (?, ?, ?)
	`, kr.kekAlias, newVersion, kmsKeyID)
	if err != nil {
		err = fmt.Errorf("failed to record new KEK version in metadata DB: %w", err)
		if kr.observability != nil {
			kr.observability.OnError(ctx, "RotateKEK", err, metadata)
			kr.observability.OnProcessComplete(ctx, "RotateKEK", time.Since(start), err, metadata)
		}
		return err
	}

	// Monitoring: Record successful key operation
	if kr.observability != nil {
		kr.observability.OnKeyOperation(ctx, "rotate", kr.kekAlias, newVersion, metadata)
		kr.observability.OnProcessComplete(ctx, "RotateKEK", time.Since(start), nil, metadata)
	}

	log.Printf("KEK rotated for alias '%s'. New version: %d, KMS ID: '%s'", kr.kekAlias, newVersion, kmsKeyID)
	return nil
}

// EnsureInitialKEK checks if a KEK exists for the given alias and creates one if not.
func (kr *KeyRotationOperations) EnsureInitialKEK(ctx context.Context, versionManager KMSVersionManager) error {
	kmsKeyID, err := kr.kmsService.GetKeyID(ctx, kr.kekAlias)
	if err != nil {
		// Assume an error here likely means the key doesn't exist in KMS
		// (depending on the KMS implementation's error handling).
		// We'll proceed to create a new key.
		log.Printf("No KEK found in KMS for alias '%s', creating a new one.", kr.kekAlias)
		kmsKeyID, err = kr.kmsService.CreateKey(ctx, kr.kekAlias) // Use the alias as a description
		if err != nil {
			return fmt.Errorf("failed to create initial KEK in KMS: %w", err)
		}
		_, err = kr.keyMetadataDB.Exec(`
			INSERT INTO kek_versions (alias, version, kms_key_id) VALUES (?, ?, ?)
		`, kr.kekAlias, 1, kmsKeyID)
		if err != nil {
			return fmt.Errorf("failed to record initial KEK in metadata DB: %w", err)
		}
		log.Printf("Initial KEK created for alias '%s' with KMS ID '%s'", kr.kekAlias, kmsKeyID)
		return nil
	}
	currentVersion, err := versionManager.GetCurrentKEKVersion(ctx, kr.kekAlias)
	if err != nil {
		return err
	}
	if currentVersion == 0 {
		// Key exists in KMS but not in our DB (this could happen if the DB was wiped or is new)
		// We should record it, assuming it's the first version.
		_, err = kr.keyMetadataDB.Exec(`
			INSERT INTO kek_versions (alias, version, kms_key_id) VALUES (?, ?, ?)
		`, kr.kekAlias, 1, kmsKeyID)
		if err != nil {
			return fmt.Errorf("failed to record initial KEK in metadata DB: %w", err)
		}
		log.Printf("Initial KEK created for alias '%s' with KMS ID '%s'", kr.kekAlias, kmsKeyID)
	}
	// Key exists in KMS and is recorded in our DB. We don't need to do anything.
	log.Printf("KEK found in KMS for alias '%s' with KMS ID '%s', current version is %d.", kr.kekAlias, kmsKeyID, currentVersion)
	return nil
}
