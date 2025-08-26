package hashicorpvault

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"

	"github.com/hashicorp/vault/api"
)

// VaultService is an implementation of KeyManagementService for HashiCorp Vault.
type VaultService struct {
	client     *api.Client
	renewalCtx context.Context
	cancelFunc context.CancelFunc
}

// New creates a new VaultService instance.
func New() (*VaultService, error) {
	config := api.DefaultConfig()
	addr := os.Getenv("VAULT_ADDR") // Or load from config
	if addr != "" {
		config.Address = addr
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	config.HttpClient.Transport = transport

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	// Set namespace if using HCP Vault
	namespace := os.Getenv("VAULT_NAMESPACE") // Should be "admin/example"
	if namespace != "" {
		client.SetNamespace(namespace)
	}

	// AppRole authentication
	roleID := os.Getenv("VAULT_ROLE_ID")
	secretID := os.Getenv("VAULT_SECRET_ID")
	if roleID != "" && secretID != "" {
		data := map[string]interface{}{
			"role_id":   roleID,
			"secret_id": secretID,
		}

		resp, err := client.Logical().Write("auth/approle/login", data)
		if err != nil {
			return nil, fmt.Errorf("failed to login with AppRole: %w", err)
		}

		if resp.Auth == nil {
			return nil, fmt.Errorf("no auth info returned from AppRole login")
		}

		// Set the token from AppRole authentication response
		client.SetToken(resp.Auth.ClientToken)
	}
	ctx, cancelFunc := context.WithCancel(context.Background())

	return &VaultService{
		client:     client,
		renewalCtx: ctx,
		cancelFunc: cancelFunc,
	}, nil
}

func (v *VaultService) GetKey(ctx context.Context, keyID string) ([]byte, error) {
	// For Vault's transit engine, directly retrieving the raw key is generally not allowed.
	// You would use the encrypt/decrypt APIs.
	return nil, fmt.Errorf("getting raw key material is not supported for Vault's transit engine")
}

func (v *VaultService) GetKeyID(ctx context.Context, alias string) (string, error) {
	// For Vault's transit engine, the 'alias' is the key name.
	return alias, nil
}

func (v *VaultService) CreateKey(ctx context.Context, description string) (string, error) {
	_, err := v.client.Logical().Write(fmt.Sprintf("transit/keys/%s", description), map[string]interface{}{
		"type": "aes256-gcm96", // Or your desired key type
	})
	if err != nil {
		return "", fmt.Errorf("failed to create transit key '%s': %w", description, err)
	}
	return description, nil // The description acts as the KeyID/alias
}

func (v *VaultService) RotateKey(ctx context.Context, keyID string) error {
	_, err := v.client.Logical().Write(fmt.Sprintf("transit/keys/%s/rotate", keyID), nil)
	if err != nil {
		return fmt.Errorf("failed to rotate key '%s': %w", keyID, err)
	}
	return nil
}

func (v *VaultService) GetSecret(ctx context.Context, path string) ([]byte, error) {
	secret, err := v.client.Logical().Read(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret from Vault at %s: %w", path, err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("secret not found at %s", path)
	}
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid secret format at %s", path)
	}
	value, ok := data["value"].(string) // Assuming your pepper is stored under the key "value"
	if !ok {
		return nil, fmt.Errorf("pepper not found or invalid format at %s", path)
	}
	return []byte(value), nil
}

func (v *VaultService) SetSecret(ctx context.Context, path string, value []byte) error {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"value": string(value),
		},
	}
	_, err := v.client.Logical().Write(path, data)
	if err != nil {
		return fmt.Errorf("failed to write secret to Vault at %s: %w", path, err)
	}
	return nil
}

func (v *VaultService) EncryptDEK(ctx context.Context, keyID string, plaintextDEK []byte) ([]byte, error) {
	resp, err := v.client.Logical().Write(fmt.Sprintf("transit/encrypt/%s", keyID), map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString(plaintextDEK),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt with key '%s': %w", keyID, err)
	}
	ciphertext, ok := resp.Data["ciphertext"].(string)
	if !ok {
		return nil, fmt.Errorf("ciphertext not found in response")
	}
	return []byte(ciphertext), nil
}

func (v *VaultService) DecryptDEK(ctx context.Context, keyID string, ciphertextDEK []byte) ([]byte, error) {
	resp, err := v.client.Logical().Write(fmt.Sprintf("transit/decrypt/%s", keyID), map[string]any{
		"ciphertext": string(ciphertextDEK),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt with key '%s': %w", keyID, err)
	}
	plaintextBase64, ok := resp.Data["plaintext"].(string)
	if !ok {
		return nil, fmt.Errorf("plaintext not found in response")
	}
	plaintext, err := base64.StdEncoding.DecodeString(plaintextBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode plaintext: %w", err)
	}
	return plaintext, nil
}

func (v *VaultService) EncryptDEKWithVersion(ctx context.Context, plaintextDEK []byte, version int) ([]byte, error) {
	return v.EncryptDEK(ctx, "", plaintextDEK)
}

func (v *VaultService) DecryptDEKWithVersion(ctx context.Context, ciphertextDEK []byte, version int) ([]byte, error) {
	return v.DecryptDEK(ctx, "", ciphertextDEK)
}
