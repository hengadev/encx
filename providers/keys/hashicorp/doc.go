// Package hashicorp provides HashiCorp Vault Transit Engine integration for encx.
//
// This package implements the encx.KeyManagementService interface using Vault's Transit Engine,
// enabling secure encryption and decryption of Data Encryption Keys (DEKs) using Vault-managed
// Key Encryption Keys (KEKs).
//
// # Features
//
//   - KEK management via Vault Transit Engine
//   - DEK encryption and decryption operations
//   - Automatic token renewal for long-running applications
//   - Support for multiple encryption key types (AES-256-GCM96 default)
//   - Key versioning and rotation support
//   - No key material ever leaves Vault
//
// # Basic Usage
//
//	import (
//	    "context"
//	    "github.com/hengadev/encx"
//	    vaulttransit "github.com/hengadev/encx/providers/keys/hashicorp"
//	)
//
//	// Initialize Vault Transit service
//	transit, err := vaulttransit.NewTransitService()
//	if err != nil {
//	    // handle error
//	}
//	defer transit.Close()
//
//	// Use with encx.NewCrypto() along with a SecretManagementService
//	crypto, err := encx.NewCrypto(ctx, transit, secretsStore, encx.Config{
//	    KEKAlias: "my-app-transit-key",
//	    PepperAlias: "my-app-pepper",
//	})
//
// # Configuration
//
// Vault Transit is configured via environment variables:
//
//	// Required
//	export VAULT_ADDR="https://vault.example.com:8200"
//	export VAULT_TOKEN="hvs.your-token-here"
//
//	// Optional
//	export VAULT_NAMESPACE="my-namespace"  // For Vault Enterprise
//	export VAULT_SKIP_VERIFY="true"       // Skip TLS verification (NOT recommended for production)
//
// # Vault Setup
//
// Before using this provider, enable the Transit Engine in Vault:
//
//	vault secrets enable transit
//
// Create a transit key:
//
//	vault write -f transit/keys/my-app-transit-key type=aes256-gcm96
//
// # IAM/Policy Permissions
//
// The Vault token needs the following policy:
//
//	path "transit/encrypt/*" {
//	    capabilities = ["update"]
//	}
//
//	path "transit/decrypt/*" {
//	    capabilities = ["update"]
//	}
//
//	path "transit/keys/*" {
//	    capabilities = ["read", "create"]
//	}
//
// Save this as transit-policy.hcl and apply:
//
//	vault policy write encx-transit transit-policy.hcl
//
// Create a token with this policy:
//
//	vault token create -policy=encx-transit
//
// # Error Handling
//
// Operations return wrapped errors from the encx package:
//
//   - encx.ErrKMSUnavailable: Vault is unavailable or key not found
//   - encx.ErrEncryptionFailed: Encryption operation failed
//   - encx.ErrDecryptionFailed: Decryption operation failed
//   - encx.ErrInvalidConfiguration: Invalid configuration (e.g., empty key name)
//
// # Mix-and-Match Providers
//
// Vault Transit can be combined with different SecretManagementService implementations:
//
//	// Vault Transit + Vault KV
//	import (
//	    vaulttransit "github.com/hengadev/encx/providers/keys/hashicorp"
//	    vaultkv "github.com/hengadev/encx/providers/secrets/hashicorp"
//	)
//
//	// Vault Transit + AWS Secrets Manager
//	import (
//	    vaulttransit "github.com/hengadev/encx/providers/keys/hashicorp"
//	    awssecrets "github.com/hengadev/encx/providers/secrets/aws"
//	)
//
// # Key Naming
//
// Unlike AWS KMS, Vault Transit doesn't use aliases with prefixes.
// The key name you provide is used directly:
//
//	transit.GetKeyID(ctx, "my-app-key")  // Uses "my-app-key" as-is
//
// # Automatic Token Renewal
//
// The TransitService automatically renews Vault tokens in the background.
// Always call Close() when shutting down to stop the renewal goroutine:
//
//	transit, _ := vaulttransit.NewTransitService()
//	defer transit.Close()  // IMPORTANT: Stops token renewal
//
// # Key Rotation
//
// Vault Transit supports key rotation:
//
//	vault write -f transit/keys/my-app-key/rotate
//
// After rotation, Vault automatically uses the latest version for encryption
// while maintaining the ability to decrypt with older versions.
//
// # Testing
//
// For testing without Vault dependencies, use encx.NewTestCrypto():
//
//	func TestMyCode(t *testing.T) {
//	    crypto, _ := encx.NewTestCrypto(t)
//	    // Test code using crypto...
//	}
//
// For integration tests with an actual Vault instance, see
// test/integration/kms_providers/vault_integration_test.go
//
// # High Availability
//
// For production deployments:
//   - Use Vault clusters with multiple nodes
//   - Configure load balancer for VAULT_ADDR
//   - Use Vault Enterprise with namespaces for multi-tenancy
//   - Enable audit logging for compliance
//   - Use AppRole or Kubernetes auth instead of static tokens
//
// For more information, see https://github.com/hengadev/encx
package hashicorp
