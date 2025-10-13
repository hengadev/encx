package hashicorp

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/vault/api"
	"github.com/hengadev/encx"
)

// TransitService implements encx.KeyManagementService using HashiCorp Vault Transit Engine.
//
// This service provides cryptographic operations (encrypt/decrypt DEKs) using Vault's
// Transit Engine. It does NOT handle secret storage - use KVStore for that.
type TransitService struct {
	client     *api.Client
	renewalCtx context.Context
	cancelFunc context.CancelFunc
}

// NewTransitService creates a new TransitService instance.
//
// The service uses environment variables for configuration (see createVaultClient).
//
// Usage:
//
//	transit, err := hashicorp.NewTransitService()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer transit.Close()
//
// The Transit Engine must be enabled in Vault before use:
//
//	vault secrets enable transit
func NewTransitService() (*TransitService, error) {
	client, renewalCtx, cancelFunc, err := createVaultClientWithContext()
	if err != nil {
		return nil, err
	}

	return &TransitService{
		client:     client,
		renewalCtx: renewalCtx,
		cancelFunc: cancelFunc,
	}, nil
}

// GetKeyID returns the key ID for a given alias.
//
// In Vault Transit Engine, the alias IS the key ID/name, so this just returns the alias.
func (t *TransitService) GetKeyID(ctx context.Context, alias string) (string, error) {
	if alias == "" {
		return "", fmt.Errorf("%w: alias cannot be empty", encx.ErrInvalidConfiguration)
	}
	// For Vault's transit engine, the 'alias' is the key name
	return alias, nil
}

// CreateKey creates a new Transit Engine key with the given name.
//
// The description parameter is used as the key name in Vault.
// Returns the key name (which serves as the key ID).
//
// Example:
//
//	keyID, err := transit.CreateKey(ctx, "my-app-key")
func (t *TransitService) CreateKey(ctx context.Context, description string) (string, error) {
	if description == "" {
		return "", fmt.Errorf("%w: description (key name) cannot be empty", encx.ErrInvalidConfiguration)
	}

	_, err := t.client.Logical().Write(fmt.Sprintf("transit/keys/%s", description), map[string]interface{}{
		"type": "aes256-gcm96", // AES-256-GCM with 96-bit nonce
	})
	if err != nil {
		return "", fmt.Errorf("%w: failed to create transit key '%s': %w", encx.ErrKMSUnavailable, description, err)
	}

	// The description acts as the KeyID/name in Transit Engine
	return description, nil
}

// EncryptDEK encrypts a Data Encryption Key using the Vault Transit Engine.
//
// The keyID is the name of the Transit Engine key.
// Returns Vault-formatted ciphertext (e.g., "vault:v1:base64...").
//
// Example:
//
//	encryptedDEK, err := transit.EncryptDEK(ctx, "my-app-key", dek)
func (t *TransitService) EncryptDEK(ctx context.Context, keyID string, plaintextDEK []byte) ([]byte, error) {
	if len(plaintextDEK) == 0 {
		return nil, fmt.Errorf("%w: plaintext cannot be empty", encx.ErrEncryptionFailed)
	}
	if keyID == "" {
		return nil, fmt.Errorf("%w: keyID cannot be empty", encx.ErrInvalidConfiguration)
	}

	// Vault Transit expects base64-encoded plaintext
	resp, err := t.client.Logical().Write(fmt.Sprintf("transit/encrypt/%s", keyID), map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString(plaintextDEK),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to encrypt with key '%s': %w", encx.ErrEncryptionFailed, keyID, err)
	}

	if resp == nil || resp.Data == nil {
		return nil, fmt.Errorf("%w: no response from Vault Transit encrypt", encx.ErrEncryptionFailed)
	}

	ciphertext, ok := resp.Data["ciphertext"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: ciphertext not found in response", encx.ErrEncryptionFailed)
	}

	return []byte(ciphertext), nil
}

// DecryptDEK decrypts a Data Encryption Key using the Vault Transit Engine.
//
// The keyID is the name of the Transit Engine key.
// The ciphertext should be in Vault format (e.g., "vault:v1:base64...").
//
// Example:
//
//	dek, err := transit.DecryptDEK(ctx, "my-app-key", encryptedDEK)
func (t *TransitService) DecryptDEK(ctx context.Context, keyID string, ciphertextDEK []byte) ([]byte, error) {
	if len(ciphertextDEK) == 0 {
		return nil, fmt.Errorf("%w: ciphertext cannot be empty", encx.ErrDecryptionFailed)
	}
	if keyID == "" {
		return nil, fmt.Errorf("%w: keyID cannot be empty", encx.ErrInvalidConfiguration)
	}

	resp, err := t.client.Logical().Write(fmt.Sprintf("transit/decrypt/%s", keyID), map[string]interface{}{
		"ciphertext": string(ciphertextDEK),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decrypt with key '%s': %w", encx.ErrDecryptionFailed, keyID, err)
	}

	if resp == nil || resp.Data == nil {
		return nil, fmt.Errorf("%w: no response from Vault Transit decrypt", encx.ErrDecryptionFailed)
	}

	plaintextBase64, ok := resp.Data["plaintext"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: plaintext not found in response", encx.ErrDecryptionFailed)
	}

	// Decode from base64
	plaintext, err := base64.StdEncoding.DecodeString(plaintextBase64)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decode plaintext: %w", encx.ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

// Close cancels the renewal context and cleans up resources.
//
// Call this when shutting down to stop any background token renewal.
func (t *TransitService) Close() {
	if t.cancelFunc != nil {
		t.cancelFunc()
	}
}
