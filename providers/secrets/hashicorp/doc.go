// Package hashicorp provides HashiCorp Vault KV v2 Engine integration for encx.
//
// This package implements the encx.SecretManagementService interface using Vault's KV v2 Engine,
// enabling secure storage and retrieval of peppers (secret values) used in Argon2id password
// hashing operations.
//
// # Features
//
//   - Secure pepper storage in Vault KV v2
//   - Automatic secret creation and updates
//   - Secret versioning (up to 10 versions by default)
//   - Automatic token renewal for long-running applications
//   - Audit logging of all operations
//   - Support for Vault namespaces (Enterprise)
//   - Check-and-Set (CAS) operations for concurrency control
//
// # Basic Usage
//
//	import (
//	    "context"
//	    "github.com/hengadev/encx"
//	    vaultkv "github.com/hengadev/encx/providers/secrets/hashicorp"
//	)
//
//	// Initialize Vault KV store
//	kv, err := vaultkv.NewKVStore()
//	if err != nil {
//	    // handle error
//	}
//	defer kv.Close()
//
//	// Use with encx.NewCrypto() along with a KeyManagementService
//	crypto, err := encx.NewCrypto(ctx, kmsService, kv, encx.Config{
//	    KEKAlias: "my-app-kek",
//	    PepperAlias: "my-app-pepper",
//	})
//
// # Configuration
//
// Vault KV is configured via environment variables:
//
//	// Required
//	export VAULT_ADDR="https://vault.example.com:8200"
//	export VAULT_TOKEN="hvs.your-token-here"
//
//	// Optional
//	export VAULT_NAMESPACE="my-namespace"  // For Vault Enterprise
//	export VAULT_SKIP_VERIFY="true"       // Skip TLS verification (NOT recommended)
//
// # Vault Setup
//
// Before using this provider, enable the KV v2 Engine in Vault:
//
//	vault secrets enable -version=2 kv
//
// # Pepper Storage
//
// Peppers are stored in Vault KV using the path format:
//
//	kv/data/encx/{alias}/pepper
//
// For example, if your PepperAlias is "payment-api", the secret will be stored at:
//
//	kv/data/encx/payment-api/pepper
//
// # IAM/Policy Permissions
//
// The Vault token needs the following policy:
//
//	path "kv/data/encx/*" {
//	    capabilities = ["create", "read", "update"]
//	}
//
//	path "kv/metadata/encx/*" {
//	    capabilities = ["read", "list"]
//	}
//
// Save this as kv-policy.hcl and apply:
//
//	vault policy write encx-kv kv-policy.hcl
//
// Create a token with this policy:
//
//	vault token create -policy=encx-kv
//
// # Error Handling
//
// Operations return wrapped errors from the encx package:
//
//   - encx.ErrSecretStorageUnavailable: Vault is unavailable or secret not found
//   - encx.ErrInvalidConfiguration: Invalid pepper length or configuration
//
// # Mix-and-Match Providers
//
// Vault KV can be combined with different KeyManagementService implementations:
//
//	// Vault KV + Vault Transit
//	import (
//	    vaulttransit "github.com/hengadev/encx/providers/keys/hashicorp"
//	    vaultkv "github.com/hengadev/encx/providers/secrets/hashicorp"
//	)
//
//	// Vault KV + AWS KMS
//	import (
//	    awskms "github.com/hengadev/encx/providers/keys/aws"
//	    vaultkv "github.com/hengadev/encx/providers/secrets/hashicorp"
//	)
//
// # Automatic Pepper Management
//
// When using encx.NewCrypto(), peppers are automatically managed:
//
//   - If a pepper doesn't exist, it's automatically generated and stored
//   - If a pepper exists, it's retrieved and used
//   - Peppers are exactly 32 bytes (encx.PepperLength)
//   - Peppers are base64-encoded before storage
//
// # Secret Versioning
//
// KV v2 automatically versions all secrets. You can:
//
//   - View secret history: vault kv metadata get kv/encx/my-app-pepper/pepper
//   - Rollback to previous version: vault kv rollback -version=1 kv/encx/my-app-pepper/pepper
//   - Delete specific version: vault kv delete -versions=2 kv/encx/my-app-pepper/pepper
//
// However, for peppers, versioning should be used for disaster recovery only.
// Never intentionally rotate peppers as it invalidates all existing password hashes.
//
// # Automatic Token Renewal
//
// The KVStore automatically renews Vault tokens in the background.
// Always call Close() when shutting down to stop the renewal goroutine:
//
//	kv, _ := vaultkv.NewKVStore()
//	defer kv.Close()  // IMPORTANT: Stops token renewal
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
// # High Availability
//
// For production deployments:
//   - Use Vault clusters with multiple nodes
//   - Configure load balancer for VAULT_ADDR
//   - Use Vault Enterprise with namespaces for multi-tenancy
//   - Enable audit logging for compliance
//   - Use AppRole or Kubernetes auth instead of static tokens
//   - Enable KV v2 versioning for disaster recovery
//
// # KV v1 vs KV v2
//
// This provider requires KV v2, which provides:
//   - Secret versioning
//   - Soft deletes
//   - Check-and-Set operations
//   - Metadata tracking
//
// To upgrade from KV v1 to KV v2:
//
//	vault kv enable-versioning kv/
//
// For more information, see https://github.com/hengadev/encx
package hashicorp
