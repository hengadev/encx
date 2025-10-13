package hashicorp

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/hashicorp/vault/api"
	"github.com/hengadev/encx"
)

// createVaultClient creates a configured Vault client using environment variables.
//
// Environment Variables:
//   - VAULT_ADDR: Vault server address (required, e.g., "https://vault.example.com")
//   - VAULT_NAMESPACE: Vault namespace for HCP Vault (optional, e.g., "admin/example")
//   - VAULT_TOKEN: Direct Vault token (optional, alternative to AppRole)
//   - VAULT_ROLE_ID: AppRole role ID for authentication (optional, requires VAULT_SECRET_ID)
//   - VAULT_SECRET_ID: AppRole secret ID for authentication (optional, requires VAULT_ROLE_ID)
//
// Authentication Priority:
//  1. If VAULT_TOKEN is set, uses token directly
//  2. If VAULT_ROLE_ID and VAULT_SECRET_ID are set, uses AppRole authentication
//  3. Otherwise, returns error (no authentication method available)
//
// Usage:
//
//	client, err := createVaultClient()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// The client can be used for both Transit Engine and KV v2 operations.
func createVaultClient() (*api.Client, error) {
	// Create default config
	config := api.DefaultConfig()

	// Set Vault address
	addr := os.Getenv("VAULT_ADDR")
	if addr != "" {
		config.Address = addr
	}
	if config.Address == "" {
		return nil, fmt.Errorf("%w: VAULT_ADDR environment variable is required", encx.ErrInvalidConfiguration)
	}

	// Configure HTTP transport with proxy support
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	config.HttpClient.Transport = transport

	// Create client
	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create Vault client: %w", encx.ErrKMSUnavailable, err)
	}

	// Set namespace if using HCP Vault
	namespace := os.Getenv("VAULT_NAMESPACE")
	if namespace != "" {
		client.SetNamespace(namespace)
	}

	// Authentication: Check for token first, then AppRole
	token := os.Getenv("VAULT_TOKEN")
	if token != "" {
		// Use direct token authentication
		client.SetToken(token)
		return client, nil
	}

	// Try AppRole authentication
	roleID := os.Getenv("VAULT_ROLE_ID")
	secretID := os.Getenv("VAULT_SECRET_ID")
	if roleID != "" && secretID != "" {
		// Perform AppRole login
		data := map[string]interface{}{
			"role_id":   roleID,
			"secret_id": secretID,
		}

		resp, err := client.Logical().Write("auth/approle/login", data)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to login with AppRole: %w", encx.ErrAuthenticationFailed, err)
		}

		if resp == nil || resp.Auth == nil {
			return nil, fmt.Errorf("%w: no auth info returned from AppRole login", encx.ErrAuthenticationFailed)
		}

		// Set the token from AppRole authentication response
		client.SetToken(resp.Auth.ClientToken)
		return client, nil
	}

	// No authentication method available
	return nil, fmt.Errorf("%w: no Vault authentication method configured (set VAULT_TOKEN or VAULT_ROLE_ID+VAULT_SECRET_ID)",
		encx.ErrInvalidConfiguration)
}

// createVaultClientWithContext creates a Vault client with a context for token renewal.
//
// This is used for long-running services that need automatic token renewal.
// Returns the client, a context for renewal operations, and a cancel function.
//
// Usage:
//
//	client, renewalCtx, cancelFunc, err := createVaultClientWithContext()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cancelFunc() // Cancel renewal when service shuts down
func createVaultClientWithContext() (*api.Client, context.Context, context.CancelFunc, error) {
	client, err := createVaultClient()
	if err != nil {
		return nil, nil, nil, err
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	return client, ctx, cancelFunc, nil
}
