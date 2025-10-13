package hashicorp

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/vault/api"
	"github.com/hengadev/encx"
)

// KVStore implements encx.SecretManagementService using HashiCorp Vault KV v2 Engine.
//
// This service stores peppers (secret values) in Vault's KV v2 secrets engine for
// secure, versioned secret storage with audit logging.
type KVStore struct {
	client *api.Client
}

// NewKVStore creates a new KVStore instance.
//
// The service uses environment variables for configuration (see createVaultClient).
//
// Usage:
//
//	kv, err := hashicorp.NewKVStore()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// The KV v2 engine must be enabled in Vault before use:
//
//	vault secrets enable -path=secret kv-v2
func NewKVStore() (*KVStore, error) {
	client, err := createVaultClient()
	if err != nil {
		return nil, err
	}

	return &KVStore{
		client: client,
	}, nil
}

// GetStoragePath returns the Vault KV v2 path for a given alias.
//
// Path format: "secret/data/encx/{alias}/pepper"
//
// Note: The "/data/" segment is required for KV v2 API reads/writes.
//
// Examples:
//   - alias "my-service" → "secret/data/encx/my-service/pepper"
//   - alias "payment-api" → "secret/data/encx/payment-api/pepper"
func (k *KVStore) GetStoragePath(alias string) string {
	return fmt.Sprintf(encx.VaultPepperPathTemplate, alias)
}

// StorePepper stores a pepper in Vault KV v2 engine.
//
// If a pepper already exists for this alias, it will be versioned (KV v2 keeps history).
// The pepper must be exactly 32 bytes (encx.PepperLength).
//
// Example:
//
//	pepper := []byte("your-32-byte-pepper-secret-here!")
//	err := kv.StorePepper(ctx, "my-service", pepper)
func (k *KVStore) StorePepper(ctx context.Context, alias string, pepper []byte) error {
	if len(pepper) != encx.PepperLength {
		return fmt.Errorf("%w: pepper must be exactly %d bytes, got %d",
			encx.ErrInvalidConfiguration, encx.PepperLength, len(pepper))
	}

	path := k.GetStoragePath(alias)

	// Encode pepper to base64 for storage
	pepperBase64 := base64.StdEncoding.EncodeToString(pepper)

	// KV v2 requires data to be wrapped in a "data" key
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"value": pepperBase64,
		},
	}

	_, err := k.client.Logical().Write(path, data)
	if err != nil {
		return fmt.Errorf("%w: failed to store pepper in Vault KV: %w",
			encx.ErrSecretStorageUnavailable, err)
	}

	return nil
}

// GetPepper retrieves a pepper from Vault KV v2 engine.
//
// Returns an error if the pepper doesn't exist or has invalid length.
//
// Example:
//
//	pepper, err := kv.GetPepper(ctx, "my-service")
//	if err != nil {
//	    log.Fatalf("Failed to get pepper: %v", err)
//	}
func (k *KVStore) GetPepper(ctx context.Context, alias string) ([]byte, error) {
	path := k.GetStoragePath(alias)

	secret, err := k.client.Logical().Read(path)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read pepper from Vault KV: %w",
			encx.ErrSecretStorageUnavailable, err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("%w: pepper not found for alias: %s",
			encx.ErrSecretStorageUnavailable, alias)
	}

	// KV v2 wraps the actual data in a "data" key
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%w: invalid KV v2 secret format for alias: %s",
			encx.ErrSecretStorageUnavailable, alias)
	}

	// Get the pepper value
	pepperBase64, ok := data["value"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: pepper value not found or invalid format for alias: %s",
			encx.ErrSecretStorageUnavailable, alias)
	}

	// Decode from base64
	pepper, err := base64.StdEncoding.DecodeString(pepperBase64)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decode pepper: %w",
			encx.ErrSecretStorageUnavailable, err)
	}

	// Validate pepper length
	if len(pepper) != encx.PepperLength {
		return nil, fmt.Errorf("%w: invalid pepper length: expected %d bytes, got %d",
			encx.ErrSecretStorageUnavailable, encx.PepperLength, len(pepper))
	}

	return pepper, nil
}

// PepperExists checks if a pepper exists in Vault KV v2 engine.
//
// Returns true if the pepper exists, false if it doesn't.
// Returns an error only for actual failures (not for "secret not found").
//
// Example:
//
//	exists, err := kv.PepperExists(ctx, "my-service")
//	if err != nil {
//	    log.Fatalf("Failed to check pepper: %v", err)
//	}
//	if !exists {
//	    // Generate and store new pepper
//	}
func (k *KVStore) PepperExists(ctx context.Context, alias string) (bool, error) {
	path := k.GetStoragePath(alias)

	secret, err := k.client.Logical().Read(path)
	if err != nil {
		// Vault returns an error for read failures, but nil secret for "not found"
		return false, fmt.Errorf("%w: failed to check if pepper exists: %w",
			encx.ErrSecretStorageUnavailable, err)
	}

	// If secret is nil or has no data, it doesn't exist
	if secret == nil || secret.Data == nil {
		return false, nil
	}

	// Check if the data structure is valid KV v2 format
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return false, nil
	}

	// Check if the pepper value exists
	_, ok = data["value"].(string)
	return ok, nil
}
