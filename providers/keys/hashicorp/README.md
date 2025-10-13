# HashiCorp Vault Transit Provider for encx

HashiCorp Vault Transit Engine implementation of the `KeyManagementService` interface for encx.

## Overview

This provider enables encx to use Vault's Transit Engine for managing Key Encryption Keys (KEKs) and performing DEK encryption/decryption operations. The Transit Engine acts as an "encryption as a service" platform where key material never leaves Vault, providing a secure and centralized cryptographic operations service.

## Features

- **KEK Management**: Create and manage encryption keys using Vault Transit Engine
- **DEK Encryption/Decryption**: Encrypt and decrypt Data Encryption Keys using Vault-managed KEKs
- **Key Versioning**: Support for key rotation with automatic version management
- **Automatic Token Renewal**: Background token renewal for long-running applications
- **Multiple Key Types**: Support for AES-256-GCM96 (default), RSA, ED25519, and more
- **No Key Export**: Cryptographic key material never leaves Vault
- **Audit Logging**: All operations logged in Vault audit logs
- **High Availability**: Works with Vault clusters and load balancers

## Installation

```bash
go get github.com/hengadev/encx
go get github.com/hashicorp/vault/api
```

## Configuration

Vault Transit is configured entirely via environment variables:

### Required Environment Variables

```bash
export VAULT_ADDR="https://vault.example.com:8200"
export VAULT_TOKEN="hvs.your-token-here"
```

### Optional Environment Variables

```bash
# Vault Enterprise namespace
export VAULT_NAMESPACE="my-namespace"

# Skip TLS verification (NOT recommended for production)
export VAULT_SKIP_VERIFY="true"

# Custom CA certificate path
export VAULT_CACERT="/path/to/ca.crt"

# Client certificate for mTLS
export VAULT_CLIENT_CERT="/path/to/client.crt"
export VAULT_CLIENT_KEY="/path/to/client.key"
```

### Usage Example

```go
import (
    "context"
    vaulttransit "github.com/hengadev/encx/providers/keys/hashicorp"
)

// Configuration is read from environment variables
transit, err := vaulttransit.NewTransitService()
if err != nil {
    log.Fatal(err)
}
defer transit.Close()  // Important: stops token renewal
```

## Vault Setup

### 1. Enable Transit Engine

```bash
vault secrets enable transit
```

### 2. Create Transit Key

```bash
# Create key for encryption
vault write -f transit/keys/my-app-transit-key type=aes256-gcm96

# Verify key creation
vault read transit/keys/my-app-transit-key
```

### 3. Create Vault Policy

Create a policy file `encx-transit-policy.hcl`:

```hcl
# Encrypt and decrypt operations
path "transit/encrypt/*" {
    capabilities = ["update"]
}

path "transit/decrypt/*" {
    capabilities = ["update"]
}

# Key management operations
path "transit/keys/*" {
    capabilities = ["create", "read"]
}
```

Apply the policy:

```bash
vault policy write encx-transit encx-transit-policy.hcl
```

### 4. Create Token

```bash
# Create token with policy
vault token create -policy=encx-transit

# For longer-lived applications, consider using AppRole:
vault auth enable approle
vault write auth/approle/role/encx-transit policies=encx-transit
```

## Usage with encx

Vault Transit is a KeyManagementService implementation and must be paired with a SecretManagementService.

### With Vault KV

```go
import (
    "github.com/hengadev/encx"
    vaulttransit "github.com/hengadev/encx/providers/keys/hashicorp"
    vaultkv "github.com/hengadev/encx/providers/secrets/hashicorp"
)

// Initialize providers
transit, err := vaulttransit.NewTransitService()
if err != nil {
    log.Fatal(err)
}
defer transit.Close()

kv, err := vaultkv.NewKVStore()
if err != nil {
    log.Fatal(err)
}
defer kv.Close()

// Create encx.Crypto instance
crypto, err := encx.NewCrypto(ctx, transit, kv, encx.Config{
    KEKAlias:    "my-app-transit-key",
    PepperAlias: "my-app-pepper",
})

// Encrypt data
encrypted, err := crypto.Encrypt([]byte("sensitive data"))

// Decrypt data
decrypted, err := crypto.Decrypt(encrypted)
```

### With AWS Secrets Manager (Mix-and-Match)

```go
import (
    "github.com/hengadev/encx"
    vaulttransit "github.com/hengadev/encx/providers/keys/hashicorp"
    awssecrets "github.com/hengadev/encx/providers/secrets/aws"
)

// Vault Transit for key encryption, AWS Secrets Manager for pepper storage
transit, err := vaulttransit.NewTransitService()
if err != nil {
    log.Fatal(err)
}
defer transit.Close()

secrets, err := awssecrets.NewSecretsManagerStore(ctx, awssecrets.Config{
    Region: "us-east-1",
})

crypto, err := encx.NewCrypto(ctx, transit, secrets, encx.Config{
    KEKAlias:    "my-app-transit-key",
    PepperAlias: "my-app-pepper",
})
```

## API Reference

### NewTransitService

```go
func NewTransitService() (*TransitService, error)
```

Creates a new Vault Transit service instance. Configuration is read from environment variables.

**Returns:**
- `*TransitService`: Initialized Transit service with automatic token renewal
- `error`: Error if Vault configuration fails

**Example:**
```go
transit, err := vaulttransit.NewTransitService()
if err != nil {
    log.Fatal(err)
}
defer transit.Close()
```

### GetKeyID

```go
func (t *TransitService) GetKeyID(ctx context.Context, alias string) (string, error)
```

Returns the key ID for a given alias. For Vault Transit, the alias IS the key name.

**Parameters:**
- `alias`: Key name

**Returns:**
- `string`: Key name (same as input)
- `error`: `encx.ErrInvalidConfiguration` if alias is empty

### CreateKey

```go
func (t *TransitService) CreateKey(ctx context.Context, description string) (string, error)
```

Creates a new Transit Engine key.

**Parameters:**
- `description`: Key name (used directly as key ID)

**Returns:**
- `string`: Key name
- `error`: `encx.ErrKMSUnavailable` if creation fails

**Example:**
```go
keyID, err := transit.CreateKey(ctx, "my-app-key")
```

### EncryptDEK

```go
func (t *TransitService) EncryptDEK(ctx context.Context, keyID string, plaintext []byte) ([]byte, error)
```

Encrypts a Data Encryption Key using Vault Transit Engine.

**Parameters:**
- `keyID`: Transit key name
- `plaintext`: DEK to encrypt (typically 32 bytes)

**Returns:**
- `[]byte`: Vault-formatted ciphertext (e.g., "vault:v1:base64...")
- `error`: `encx.ErrEncryptionFailed` if encryption fails

### DecryptDEK

```go
func (t *TransitService) DecryptDEK(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error)
```

Decrypts a Data Encryption Key using Vault Transit Engine.

**Parameters:**
- `keyID`: Transit key name
- `ciphertext`: Vault-formatted ciphertext from EncryptDEK

**Returns:**
- `[]byte`: Decrypted DEK plaintext
- `error`: `encx.ErrDecryptionFailed` if decryption fails

### Close

```go
func (t *TransitService) Close()
```

Stops token renewal and cleans up resources. Always call this when shutting down.

**Example:**
```go
transit, _ := vaulttransit.NewTransitService()
defer transit.Close()
```

## Key Management

### Key Types

Vault Transit supports multiple key types:

```bash
# AES-256-GCM96 (default, recommended for encx)
vault write -f transit/keys/my-key type=aes256-gcm96

# Other supported types
vault write -f transit/keys/my-key type=chacha20-poly1305
vault write -f transit/keys/my-key type=rsa-2048
vault write -f transit/keys/my-key type=rsa-4096
vault write -f transit/keys/my-key type=ecdsa-p256
vault write -f transit/keys/my-key type=ed25519
```

For encx, use `aes256-gcm96` (default) or `chacha20-poly1305`.

### Key Rotation

Rotate keys to comply with security policies:

```bash
# Rotate key (creates new version)
vault write -f transit/keys/my-app-transit-key/rotate

# Check current version
vault read transit/keys/my-app-transit-key

# Set minimum decryption version (prevents use of old versions)
vault write transit/keys/my-app-transit-key/config min_decryption_version=2
```

After rotation:
- Encryption uses the latest version automatically
- Decryption works with all versions (unless min_decryption_version is set)
- No application code changes needed

### Key Deletion

```bash
# Enable deletion (keys are protected by default)
vault write transit/keys/my-app-transit-key/config deletion_allowed=true

# Delete key
vault delete transit/keys/my-app-transit-key
```

**Warning**: Deleting a key makes all data encrypted with it unrecoverable!

## Authentication Methods

### 1. Token Authentication (Simplest)

```bash
export VAULT_TOKEN="hvs.your-token-here"
```

**Pros**: Simple, good for development
**Cons**: Token may expire, requires rotation

### 2. AppRole Authentication (Recommended for Production)

```bash
# Enable AppRole
vault auth enable approle

# Create role
vault write auth/approle/role/encx-transit \
    token_policies=encx-transit \
    token_ttl=1h \
    token_max_ttl=4h

# Get role ID and secret ID
vault read auth/approle/role/encx-transit/role-id
vault write -f auth/approle/role/encx-transit/secret-id
```

In your application:
```go
import "github.com/hashicorp/vault/api"

// Login with AppRole
client, _ := api.NewClient(api.DefaultConfig())
client.SetAddress(os.Getenv("VAULT_ADDR"))

secret, _ := client.Logical().Write("auth/approle/login", map[string]interface{}{
    "role_id":   roleID,
    "secret_id": secretID,
})

client.SetToken(secret.Auth.ClientToken)
```

### 3. Kubernetes Authentication (For K8s Deployments)

```bash
# Enable Kubernetes auth
vault auth enable kubernetes

# Configure
vault write auth/kubernetes/config \
    kubernetes_host="https://$KUBERNETES_SERVICE_HOST:$KUBERNETES_SERVICE_PORT"

# Create role
vault write auth/kubernetes/role/encx-transit \
    bound_service_account_names=encx \
    bound_service_account_namespaces=default \
    policies=encx-transit \
    ttl=1h
```

## Error Handling

All operations return wrapped errors from the `encx` package:

```go
encrypted, err := transit.EncryptDEK(ctx, keyID, dek)
if err != nil {
    switch {
    case errors.Is(err, encx.ErrKMSUnavailable):
        // Handle Vault unavailability or key not found
    case errors.Is(err, encx.ErrEncryptionFailed):
        // Handle encryption failure
    case errors.Is(err, encx.ErrInvalidConfiguration):
        // Handle configuration error
    }
}
```

## Testing

For unit tests without Vault dependencies, use encx's test utilities:

```go
import (
    "testing"
    "github.com/hengadev/encx"
)

func TestMyApplication(t *testing.T) {
    // Creates in-memory KMS and secret store
    crypto, err := encx.NewTestCrypto(t)
    if err != nil {
        t.Fatal(err)
    }

    // Test your application code
    result, err := MyFunction(crypto)
    // ...
}
```

For integration tests with actual Vault, see `test/integration/kms_providers/vault_integration_test.go`.

### Local Vault Development

Start Vault in dev mode for testing:

```bash
# Start Vault dev server
vault server -dev -dev-root-token-id="root"

# In another terminal
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='root'

# Enable transit
vault secrets enable transit

# Create test key
vault write -f transit/keys/test-key type=aes256-gcm96
```

## Best Practices

1. **Use AppRole**: Prefer AppRole authentication over static tokens for production
2. **Enable Audit Logging**: Track all cryptographic operations
3. **Key Rotation**: Rotate keys regularly according to security policy
4. **High Availability**: Use Vault clusters with multiple nodes
5. **Network Security**: Use TLS for Vault communication (`VAULT_SKIP_VERIFY=false`)
6. **Resource Cleanup**: Always call `Close()` to stop token renewal
7. **Error Handling**: Implement retry logic for transient Vault failures
8. **Monitoring**: Monitor Vault health and token expiration

## High Availability Setup

For production deployments:

### 1. Vault Cluster

Deploy Vault with HA storage backend:

```hcl
storage "consul" {
  address = "127.0.0.1:8500"
  path    = "vault/"
}

listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_cert_file = "/path/to/cert.pem"
  tls_key_file  = "/path/to/key.pem"
}

api_addr = "https://vault-1.example.com:8200"
cluster_addr = "https://vault-1.example.com:8201"
```

### 2. Load Balancer

Point `VAULT_ADDR` to load balancer:

```bash
export VAULT_ADDR="https://vault.example.com:8200"
```

### 3. Health Checks

Implement health checks in your application:

```go
func checkVaultHealth(transit *TransitService) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Try to get key info
    _, err := transit.GetKeyID(ctx, "health-check-key")
    return err
}
```

## Performance Considerations

### Latency

Vault Transit adds network latency:
- Typical encrypt/decrypt: 5-20ms (local network)
- With TLS: +2-5ms
- Cross-region: +50-200ms

### Throughput

Vault can handle high throughput:
- Single Vault node: ~1000-5000 ops/sec
- Vault cluster: Scales linearly with nodes
- Consider batching for high-volume workloads

### Optimization Tips

1. **Connection Pooling**: Reuse Vault client connections
2. **Caching**: Cache decrypted DEKs at application level (with rotation)
3. **Batching**: Use Vault's batch encrypt/decrypt endpoints for bulk operations
4. **Regional Deployment**: Deploy Vault close to your application

## Troubleshooting

### "connection refused"

Verify Vault address and network connectivity:

```bash
curl $VAULT_ADDR/v1/sys/health
```

### "permission denied"

Check token has required policy:

```bash
vault token lookup
vault token capabilities transit/encrypt/my-key
```

### "key not found"

Create the transit key:

```bash
vault write -f transit/keys/my-app-transit-key type=aes256-gcm96
```

### "token expired"

Token renewal failed. Check token TTL:

```bash
vault token lookup
```

Recreate token with longer TTL or use AppRole.

## Cost Considerations

Vault is open source and free to use. For HashiCorp Cloud Platform (HCP) Vault:

- Development tier: Free (limited requests)
- Standard tier: ~$0.50/hour per cluster
- Plus/Premium: Higher tiers for enterprise features

For self-hosted Vault:
- Infrastructure costs only (compute, storage, network)
- No per-request charges

## Related Providers

- **Vault KV** (`providers/secrets/hashicorp`): Pair with Vault Transit for full Vault integration
- **AWS KMS** (`providers/keys/aws`): Alternative key management service

## Documentation

- [encx Documentation](https://github.com/hengadev/encx)
- [Vault Transit Engine Documentation](https://developer.hashicorp.com/vault/docs/secrets/transit)
- [encx API Reference](../../docs/API.md)
- [encx Architecture](../../docs/ARCHITECTURE.md)

## License

See the main encx repository for license information.
