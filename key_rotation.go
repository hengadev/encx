package encx

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Key rotation operations

// RotateKEK generates a new KEK and updates the metadata database.
func (c *Crypto) RotateKEK(ctx context.Context) error {
	// Monitoring: Start key operation
	start := time.Now()
	metadata := map[string]interface{}{
		"key_alias": c.kekAlias,
		"operation_type": "key_rotation",
	}
	c.observabilityHook.OnProcessStart(ctx, "RotateKEK", metadata)
	
	currentVersion, err := c.getCurrentKEKVersion(ctx, c.kekAlias)
	if err != nil {
		c.observabilityHook.OnError(ctx, "RotateKEK", err, metadata)
		c.observabilityHook.OnProcessComplete(ctx, "RotateKEK", time.Since(start), err, metadata)
		return err
	}
	
	newVersion := currentVersion + 1
	metadata["old_version"] = currentVersion
	metadata["new_version"] = newVersion
	
	kmsKeyID, err := c.kmsService.CreateKey(ctx, c.kekAlias) // KMS might handle rotation internally based on alias
	if err != nil {
		err = fmt.Errorf("failed to create new KEK version in KMS: %w", err)
		c.observabilityHook.OnError(ctx, "RotateKEK", err, metadata)
		c.observabilityHook.OnProcessComplete(ctx, "RotateKEK", time.Since(start), err, metadata)
		return err
	}

	// Mark the previous version as deprecated
	_, err = c.keyMetadataDB.Exec(`
		UPDATE kek_versions SET is_deprecated = TRUE
		WHERE alias = ? AND version = ?
	`, c.kekAlias, currentVersion)
	if err != nil {
		err = fmt.Errorf("failed to deprecate old KEK version: %w", err)
		c.observabilityHook.OnError(ctx, "RotateKEK", err, metadata)
		c.observabilityHook.OnProcessComplete(ctx, "RotateKEK", time.Since(start), err, metadata)
		return err
	}

	// Record the new KEK version
	_, err = c.keyMetadataDB.Exec(`
		INSERT INTO kek_versions (alias, version, kms_key_id) VALUES (?, ?, ?)
	`, c.kekAlias, newVersion, kmsKeyID)
	if err != nil {
		err = fmt.Errorf("failed to record new KEK version in metadata DB: %w", err)
		c.observabilityHook.OnError(ctx, "RotateKEK", err, metadata)
		c.observabilityHook.OnProcessComplete(ctx, "RotateKEK", time.Since(start), err, metadata)
		return err
	}

	// Monitoring: Record successful key operation
	c.observabilityHook.OnKeyOperation(ctx, "rotate", c.kekAlias, newVersion, metadata)
	c.observabilityHook.OnProcessComplete(ctx, "RotateKEK", time.Since(start), nil, metadata)

	log.Printf("KEK rotated for alias '%s'. New version: %d, KMS ID: '%s'", c.kekAlias, newVersion, kmsKeyID)
	return nil
}

// ensureInitialKEK checks if a KEK exists for the given alias and creates one if not.
func (c *Crypto) ensureInitialKEK(ctx context.Context) error {
	kmsKeyID, err := c.kmsService.GetKeyID(ctx, c.kekAlias)
	if err != nil {
		// Assume an error here likely means the key doesn't exist in KMS
		// (depending on the KMS implementation's error handling).
		// We'll proceed to create a new key.
		log.Printf("No KEK found in KMS for alias '%s', creating a new one.", c.kekAlias)
		kmsKeyID, err = c.kmsService.CreateKey(ctx, c.kekAlias) // Use the alias as a description
		if err != nil {
			return fmt.Errorf("failed to create initial KEK in KMS: %w", err)
		}
		_, err = c.keyMetadataDB.Exec(`
			INSERT INTO kek_versions (alias, version, kms_key_id) VALUES (?, ?, ?)
		`, c.kekAlias, 1, kmsKeyID)
		if err != nil {
			return fmt.Errorf("failed to record initial KEK in metadata DB: %w", err)
		}
		log.Printf("Initial KEK created for alias '%s' with KMS ID '%s'", c.kekAlias, kmsKeyID)
		return nil
	}
	currentVersion, err := c.getCurrentKEKVersion(ctx, c.kekAlias)
	if err != nil {
		return err
	}
	if currentVersion == 0 {
		// Key exists in KMS but not in our DB (this could happen if the DB was wiped or is new)
		// We should record it, assuming it's the first version.
		_, err = c.keyMetadataDB.Exec(`
			INSERT INTO kek_versions (alias, version, kms_key_id) VALUES (?, ?, ?)
		`, c.kekAlias, 1, kmsKeyID)
		if err != nil {
			return fmt.Errorf("failed to record initial KEK in metadata DB: %w", err)
		}
		log.Printf("Initial KEK created for alias '%s' with KMS ID '%s'", c.kekAlias, kmsKeyID)
	}
	// Key exists in KMS and is recorded in our DB. We don't need to do anything.
	log.Printf("KEK found in KMS for alias '%s' with KMS ID '%s', current version is %d.", c.kekAlias, kmsKeyID, currentVersion)
	return nil
}