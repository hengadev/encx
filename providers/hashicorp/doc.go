// Package hashicorp provides HashiCorp Vault integration for the encx library.
//
// This package implements two encx interfaces:
//  - encx.KeyManagementService: Cryptographic operations using Vault Transit Engine
//  - encx.SecretManagementService: Secret storage using Vault KV v2 Engine
//
// # Overview
//
// HashiCorp Vault provides two separate engines that encx leverages:
//
//  - Transit Engine: "Encryption as a Service" - performs cryptographic operations without exposing keys
//  - KV v2 Engine: Versioned secret storage with audit logging and access control
//
// This separation matches encx's architecture:
//  - Transit Engine encrypts and decrypts Data Encryption Keys (DEKs)
//  - KV Engine stores the pepper (secret value) securely with versioning
//
// # Setup
//
// Before using this provider, you need:
//
//  1. HashiCorp Vault server (self-hosted or HCP Vault)
//  2. Vault authentication configured (Token or AppRole)
//  3. Transit Engine enabled at "transit/" path
//  4. KV v2 Engine enabled at "secret/" path
//
// ## Enabling Engines
//
//	# Enable Transit Engine
//	vault secrets enable transit
//
//	# Enable KV v2 Engine (usually enabled by default)
//	vault secrets enable -path=secret kv-v2
//
// # Vault Policies Required
//
// The authentication token/role must have these permissions:
//
//	# Transit Engine permissions (for TransitService)
//	path "transit/encrypt/*" {
//	  capabilities = ["create", "update"]
//	}
//	path "transit/decrypt/*" {
//	  capabilities = ["create", "update"]
//	}
//	path "transit/keys/*" {
//	  capabilities = ["create", "read", "update"]
//	}
//
//	# KV v2 permissions (for KVStore)
//	path "secret/data/encx/*" {
//	  capabilities = ["create", "read", "update"]
//	}
//	path "secret/metadata/encx/*" {
//	  capabilities = ["list", "read"]
//	}
//
// # Environment Variables
//
// Both services use the following environment variables:
//
//  - VAULT_ADDR: Vault server address (required, e.g., "https://vault.example.com:8200")
//  - VAULT_NAMESPACE: Vault namespace for HCP Vault (optional, e.g., "admin/example")
//  - VAULT_TOKEN: Direct Vault token (option 1 for auth)
//  - VAULT_ROLE_ID: AppRole role ID (option 2 for auth, requires VAULT_SECRET_ID)
//  - VAULT_SECRET_ID: AppRole secret ID (option 2 for auth, requires VAULT_ROLE_ID)
//
// # Authentication Methods
//
// ## Option 1: Direct Token (Development)
//
//	export VAULT_ADDR="http://127.0.0.1:8200"
//	export VAULT_TOKEN="hvs.CAES..."
//
// ## Option 2: AppRole (Production)
//
//	export VAULT_ADDR="https://vault.example.com:8200"
//	export VAULT_ROLE_ID="your-role-id"
//	export VAULT_SECRET_ID="your-secret-id"
//
// # Usage Example
//
// Complete setup with both Transit and KV engines:
//
//	import (
//	    "context"
//	    "github.com/hengadev/encx"
//	    "github.com/hengadev/encx/providers/hashicorp"
//	)
//
//	func main() {
//	    ctx := context.Background()
//
//	    // Create Transit Engine service for cryptographic operations
//	    transit, err := hashicorp.NewTransitService()
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    defer transit.Close()
//
//	    // Create KV v2 service for pepper storage
//	    kv, err := hashicorp.NewKVStore()
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Create encx crypto service with both Vault services
//	    crypto, err := encx.NewCrypto(ctx, transit, kv, encx.Config{
//	        KEKAlias:    "my-app-key",     // Transit Engine key name
//	        PepperAlias: "my-service",     // Service identifier for pepper
//	    })
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Use crypto service for encryption
//	    dek, _ := crypto.GenerateDEK()
//	    ciphertext, _ := crypto.EncryptData(ctx, plaintext, dek)
//	}
//
// # Transit Engine Keys
//
// Create Transit Engine keys before use:
//
//	# Create a new Transit key
//	vault write -f transit/keys/my-app-key type=aes256-gcm96
//
//	# Or use the TransitService programmatically
//	keyID, err := transit.CreateKey(ctx, "my-app-key")
//
// Transit Engine keys support automatic rotation and versioning.
//
// # KV v2 Pepper Storage
//
// Peppers are automatically stored at: "secret/data/encx/{pepperAlias}/pepper"
//
//	// If PepperAlias is "my-service", pepper is stored at:
//	// "secret/data/encx/my-service/pepper"
//
// KV v2 maintains version history of all secrets for audit and rollback.
//
// # Performance Considerations
//
// - Transit operations involve network calls to Vault (typically 1-5ms latency)
//  - KV reads are cached by Vault client (configurable TTL)
//  - Transit Engine has no rate limits (governed by Vault server capacity)
//  - Consider DEK caching for high-throughput applications
//  - Both services use the same Vault connection (efficient)
//
// # Security Best Practices
//
//  1. Use AppRole authentication in production (not direct tokens)
//  2. Enable Vault audit logging for compliance
//  3. Use TLS for Vault communication (https://)
//  4. Rotate AppRole secret IDs regularly
//  5. Use separate Transit keys per environment (dev/staging/prod)
//  6. Enable Transit key rotation for automatic key cycling
//  7. Use Vault namespaces for multi-tenant deployments (HCP Vault)
//  8. Restrict Vault policies to minimum required permissions
//
// # High Availability
//
// For production deployments:
//
//  - Use Vault clusters for HA (3+ nodes recommended)
//  - Both Transit and KV engines replicate automatically in HA mode
//  - Peppers are available across all Vault nodes
//  - Transit operations are stateless and can use any Vault node
//
// # Error Handling
//
// All methods return encx-compatible errors:
//
//  - encx.ErrKMSUnavailable: Vault Transit Engine is unavailable or inaccessible
//  - encx.ErrSecretStorageUnavailable: Vault KV engine is unavailable or inaccessible
//  - encx.ErrAuthenticationFailed: Vault authentication failed
//  - encx.ErrEncryptionFailed: Transit encryption operation failed
//  - encx.ErrDecryptionFailed: Transit decryption operation failed
//  - encx.ErrInvalidConfiguration: Invalid configuration provided
//
// # Vault Namespaces (HCP Vault)
//
// For HCP Vault or Enterprise with namespaces:
//
//	export VAULT_ADDR="https://my-cluster.vault.hashicorp.cloud:8200"
//	export VAULT_NAMESPACE="admin/my-namespace"
//	export VAULT_TOKEN="hvs.CAES..."
//
// # Separation of Concerns
//
// This package follows the Single Responsibility Principle by separating:
//
//  - Cryptographic operations (Transit) from secret storage (KV)
//  - This matches Vault's engine architecture and allows independent auditing
//  - Both services use the same Vault authentication and connection
//
// # Comparison with AWS Provider
//
//  - Transit Engine ≈ AWS KMS (cryptographic operations)
//  - KV v2 Engine ≈ AWS Secrets Manager (secret storage)
//  - Both provide envelope encryption pattern support
//  - Vault can be self-hosted; AWS services are cloud-only
//
// For more information, see:
//  - Vault Transit Engine: https://developer.hashicorp.com/vault/docs/secrets/transit
//  - Vault KV v2 Engine: https://developer.hashicorp.com/vault/docs/secrets/kv/kv-v2
//  - Vault Authentication: https://developer.hashicorp.com/vault/docs/auth
package hashicorp
