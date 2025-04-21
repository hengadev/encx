package encx

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/hengadev/encx/internal/types"
)

type Crypto struct {
	kmsService   KeyManagementService
	kekAlias     string
	pepper       [16]byte
	argon2Params *types.Argon2Params
	serializer   Serializer // Add the Serializer field
}

// NewCrypto creates a new Crypto instance, initializing the KMS service and retrieving necessary secrets and KEK ID.
func New(
	ctx context.Context,
	kmsService KeyManagementService,
	kekAlias string,
	argon2Params *types.Argon2Params,
	pepperSecretPath string,
	serializer ...Serializer,
) (*Crypto, error) {
	pepperBytes, err := kmsService.GetSecret(ctx, pepperSecretPath)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve pepper from KMS: %w", err)
	}
	if len(pepperBytes) != 16 {
		return nil, fmt.Errorf("invalid pepper length retrieved from KMS: expected 16, got %d", len(pepperBytes))
	}
	var pepper [16]byte
	copy(pepper[:], pepperBytes)

	// handle nil case for argon2Params
	if argon2Params == nil {
		argon2Params = types.DefaultArgon2Params()
	}

	cryptoInstance := &Crypto{
		kmsService:   kmsService,
		kekAlias:     kekAlias,
		pepper:       pepper,
		argon2Params: argon2Params,
	}

	if len(serializer) > 0 {
		cryptoInstance.serializer = serializer[0]
	} else {
		cryptoInstance.serializer = JSONSerializer{} // Set a default serializer
	}

	// Ensure an initial KEK exists for this alias
	err = cryptoInstance.ensureInitialKEK(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure initial KEK: %w", err)
	}

	return cryptoInstance, nil
}

// ensureInitialKEK checks if a KEK exists for the given alias and creates one if not.
func (c *Crypto) ensureInitialKEK(ctx context.Context) error {
	currentVersion, err := c.getCurrentKEKVersion(ctx, c.kekAlias)
	if err != nil {
		return err
	}
	if currentVersion == 0 {
		// No key exists, create the first one
		kmsKeyID, err := c.kmsService.CreateKey(ctx, c.kekAlias) // Use the alias as a description
		if err != nil {
			return fmt.Errorf("failed to create initial KEK in KMS: %w", err)
		}
		_, err = keyMetadataDB.Exec(`
			INSERT INTO kek_versions (alias, version, kms_key_id) VALUES (?, ?, ?)
		`, c.kekAlias, 1, kmsKeyID)
		if err != nil {
			return fmt.Errorf("failed to record initial KEK in metadata DB: %w", err)
		}
		log.Printf("Initial KEK created for alias '%s' with KMS ID '%s'", c.kekAlias, kmsKeyID)
	}
	return nil
}

// getCurrentKEKVersion retrieves the current active KEK version for a given alias.
func (c *Crypto) getCurrentKEKVersion(ctx context.Context, alias string) (int, error) {
	row := keyMetadataDB.QueryRowContext(ctx, `
		SELECT version FROM kek_versions
		WHERE alias = ? AND is_deprecated = FALSE
		ORDER BY version DESC
		LIMIT 1
	`, alias)
	var version int
	err := row.Scan(&version)
	if err == sql.ErrNoRows {
		return 0, nil // No active key found
	} else if err != nil {
		return 0, fmt.Errorf("failed to get current KEK version for alias '%s': %w", alias, err)
	}
	return version, nil
}
