# HashiCorp Vault KV v2 Provider for encx

HashiCorp Vault KV v2 Engine implementation of the `SecretManagementService` interface for encx.

## Overview

This provider enables encx to use Vault's KV v2 Engine for secure pepper storage. Peppers are secret values used in Argon2id password hashing to add an additional layer of security beyond salts. Vault KV v2 provides versioned secret storage with audit logging, soft deletes, and check-and-set operations for concurrency control.

## Features

- **Secure Pepper Storage**: Store peppers in Vault KV v2 with encryption at rest
- **Secret Versioning**: Automatic versioning of all secret updates (up to 10 versions by default)
- **Automatic Secret Management**: Automatically create secrets if they don't exist
- **Token Renewal**: Background token renewal for long-running applications
- **Audit Logging**: All operations logged in Vault audit logs
- **Soft Deletes**: Deleted secrets can be recovered
- **Check-and-Set**: Prevent concurrent modification conflicts
- **Namespace Support**: Vault Enterprise namespace support

## Installation

```bash
go get github.com/hengadev/encx
go get github.com/hashicorp/vault/api
```

## Configuration

Vault KV is configured entirely via environment variables:

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
    vaultkv "github.com/hengadev/encx/providers/secrets/hashicorp"
)

// Configuration is read from environment variables
kv, err := vaultkv.NewKVStore()
if err != nil {
    log.Fatal(err)
}
defer kv.Close()  // Important: stops token renewal
```

## Vault Setup

### 1. Enable KV v2 Engine

```bash
# Enable KV v2 at default path
vault secrets enable -version=2 kv

# Or enable at custom path
vault secrets enable -path=secret -version=2 kv
```

### 2. Create Vault Policy

Create a policy file `encx-kv-policy.hcl`:

```hcl
# Read, create, and update secrets
path "kv/data/encx/*" {
    capabilities = ["create", "read", "update"]
}

# Read secret metadata (for version info, existence checks)
path "kv/metadata/encx/*" {
    capabilities = ["read", "list"]
}
```

Apply the policy:

```bash
vault policy write encx-kv encx-kv-policy.hcl
```

### 3. Create Token

```bash
# Create token with policy
vault token create -policy=encx-kv

# For longer-lived applications, consider using AppRole:
vault auth enable approle
vault write auth/approle/role/encx-kv policies=encx-kv
```

## Usage with encx

Vault KV is a SecretManagementService implementation and must be paired with a KeyManagementService.

### With Vault Transit

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

// Hash password (uses pepper from Vault KV)
hashed, err := crypto.HashPassword("user-password")

// Verify password
valid, err := crypto.VerifyPassword("user-password", hashed)
```

### With AWS KMS (Mix-and-Match)

```go
import (
    "github.com/hengadev/encx"
    awskms "github.com/hengadev/encx/providers/keys/aws"
    vaultkv "github.com/hengadev/encx/providers/secrets/hashicorp"
)

// AWS KMS for key encryption, Vault KV for pepper storage
kms, err := awskms.NewKMSService(ctx, awskms.Config{Region: "us-east-1"})
kv, err := vaultkv.NewKVStore()
if err != nil {
    log.Fatal(err)
}
defer kv.Close()

crypto, err := encx.NewCrypto(ctx, kms, kv, encx.Config{
    KEKAlias:    "alias/my-app-kek",
    PepperAlias: "my-app-pepper",
})
```

## API Reference

### NewKVStore

```go
func NewKVStore() (*KVStore, error)
```

Creates a new Vault KV v2 store instance. Configuration is read from environment variables.

**Returns:**
- `*KVStore`: Initialized KV store with automatic token renewal
- `error`: Error if Vault configuration fails

**Example:**
```go
kv, err := vaultkv.NewKVStore()
if err != nil {
    log.Fatal(err)
}
defer kv.Close()
```

### StorePepper

```go
func (k *KVStore) StorePepper(ctx context.Context, alias string, pepper []byte) error
```

Stores a pepper in Vault KV v2. Creates the secret if it doesn't exist, updates if it does.

**Parameters:**
- `alias`: Pepper identifier (used to construct secret path)
- `pepper`: 32-byte pepper value

**Returns:**
- `error`: `encx.ErrSecretStorageUnavailable` if operation fails

**Example:**
```go
pepper := make([]byte, 32)
rand.Read(pepper)
err := kv.StorePepper(ctx, "my-app-pepper", pepper)
```

### GetPepper

```go
func (k *KVStore) GetPepper(ctx context.Context, alias string) ([]byte, error)
```

Retrieves a pepper from Vault KV v2.

**Parameters:**
- `alias`: Pepper identifier

**Returns:**
- `[]byte`: 32-byte pepper value
- `error`: `encx.ErrSecretStorageUnavailable` if pepper not found

**Example:**
```go
pepper, err := kv.GetPepper(ctx, "my-app-pepper")
```

### PepperExists

```go
func (k *KVStore) PepperExists(ctx context.Context, alias string) (bool, error)
```

Checks if a pepper exists in Vault KV v2.

**Parameters:**
- `alias`: Pepper identifier

**Returns:**
- `bool`: `true` if pepper exists, `false` otherwise
- `error`: Error only for actual failures (not "not found")

**Example:**
```go
exists, err := kv.PepperExists(ctx, "my-app-pepper")
if !exists {
    // Pepper will be auto-created on first NewCrypto() call
}
```

### GetStoragePath

```go
func (k *KVStore) GetStoragePath(alias string) string
```

Returns the full KV path for a given alias.

**Parameters:**
- `alias`: Pepper identifier

**Returns:**
- `string`: KV path (e.g., "encx/my-app-pepper/pepper")

### Close

```go
func (k *KVStore) Close()
```

Stops token renewal and cleans up resources. Always call this when shutting down.

**Example:**
```go
kv, _ := vaultkv.NewKVStore()
defer kv.Close()
```

## Pepper Management

### Automatic Creation

When using `encx.NewCrypto()`, peppers are automatically created if they don't exist:

```go
// If encx/my-app-pepper/pepper doesn't exist, it's automatically generated and stored
crypto, err := encx.NewCrypto(ctx, kms, kv, encx.Config{
    KEKAlias:    "my-app-kek",
    PepperAlias: "my-app-pepper",
})
```

### Manual Creation

You can manually create a pepper via Vault CLI:

```bash
# Generate a 32-byte random pepper
PEPPER=$(openssl rand -base64 32)

# Store in Vault KV v2
vault kv put kv/encx/my-app-pepper/pepper value="$PEPPER"
```

### Secret Path Format

Peppers are stored using the path format:

```
kv/data/encx/{alias}/pepper
```

Note the `/data/` component - this is required for KV v2 API access.

Examples:
- `PepperAlias: "my-app"` → Path: `kv/data/encx/my-app/pepper`
- `PepperAlias: "payment-service"` → Path: `kv/data/encx/payment-service/pepper`

### Checking Secret Versions

```bash
# View secret metadata and versions
vault kv metadata get kv/encx/my-app-pepper/pepper

# Get specific version
vault kv get -version=1 kv/encx/my-app-pepper/pepper
```

## Secret Versioning

### View Version History

```bash
vault kv metadata get kv/encx/my-app-pepper/pepper
```

Output shows:
- Current version
- Created time
- Updated time
- Number of versions
- Deletion status

### Rollback to Previous Version

```bash
# Rollback to version 1
vault kv rollback -version=1 kv/encx/my-app-pepper/pepper
```

**Warning**: Only use rollback for disaster recovery. Changing peppers breaks password verification!

### Delete Specific Version

```bash
# Soft delete version 2
vault kv delete -versions=2 kv/encx/my-app-pepper/pepper

# Permanently delete version 2
vault kv destroy -versions=2 kv/encx/my-app-pepper/pepper

# Undelete version 2
vault kv undelete -versions=2 kv/encx/my-app-pepper/pepper
```

### Configure Version Retention

```bash
# Keep only 5 versions
vault kv metadata put -max-versions=5 kv/encx/my-app-pepper/pepper

# Automatically delete secrets after 30 days
vault kv metadata put -delete-version-after=720h kv/encx/my-app-pepper/pepper
```

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
vault write auth/approle/role/encx-kv \
    token_policies=encx-kv \
    token_ttl=1h \
    token_max_ttl=4h

# Get role ID and secret ID
vault read auth/approle/role/encx-kv/role-id
vault write -f auth/approle/role/encx-kv/secret-id
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
vault write auth/kubernetes/role/encx-kv \
    bound_service_account_names=encx \
    bound_service_account_namespaces=default \
    policies=encx-kv \
    ttl=1h
```

## Error Handling

All operations return wrapped errors from the `encx` package:

```go
pepper, err := kv.GetPepper(ctx, "my-app-pepper")
if err != nil {
    switch {
    case errors.Is(err, encx.ErrSecretStorageUnavailable):
        // Handle Vault unavailability or secret not found
    case errors.Is(err, encx.ErrInvalidConfiguration):
        // Handle configuration error (e.g., invalid pepper length)
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

    // Test password hashing
    hashed, err := crypto.HashPassword("test-password")
    // ...
}
```

For integration tests with actual Vault KV, see `test/integration/kms_providers/vault_integration_test.go`.

### Local Vault Development

Start Vault in dev mode for testing:

```bash
# Start Vault dev server
vault server -dev -dev-root-token-id="root"

# In another terminal
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='root'

# KV v2 is enabled by default at "secret/" in dev mode
# You can enable it at "kv/" for consistency:
vault secrets enable -version=2 kv

# Test pepper storage
PEPPER=$(openssl rand -base64 32)
vault kv put kv/encx/test/pepper value="$PEPPER"
vault kv get kv/encx/test/pepper
```

## Best Practices

1. **Use AppRole**: Prefer AppRole authentication over static tokens for production
2. **Enable Audit Logging**: Track all secret access operations
3. **Version Retention**: Configure appropriate max-versions for your use case
4. **Backup Strategy**: Regularly backup Vault data for disaster recovery
5. **High Availability**: Use Vault clusters with multiple nodes
6. **Network Security**: Use TLS for Vault communication (`VAULT_SKIP_VERIFY=false`)
7. **Resource Cleanup**: Always call `Close()` to stop token renewal
8. **Namespace Isolation**: Use Vault Enterprise namespaces for multi-tenancy
9. **Pepper Immutability**: Never rotate peppers intentionally (breaks password verification)

## Security Considerations

### Why 32 Bytes?

Peppers must be exactly 32 bytes (256 bits) to provide sufficient entropy for cryptographic security. This length matches the output size of SHA-256 and provides adequate protection against brute-force attacks.

### Pepper vs Salt

- **Salt**: Random value per password, stored with hash, prevents rainbow table attacks
- **Pepper**: Shared secret value, stored separately, adds server-side secret layer

Both are used together in encx's Argon2id implementation for defense-in-depth.

### Storage Separation

Peppers are stored in Vault (separate from application database) to ensure that:
1. Database compromise doesn't expose peppers
2. Application compromise requires additional access to Vault
3. Defense-in-depth is maintained

### Vault Security Best Practices

1. **Enable TLS**: Always use HTTPS for Vault communication
2. **Seal/Unseal**: Properly manage Vault seal keys
3. **Audit Logs**: Enable audit logging and monitor for suspicious activity
4. **Token TTL**: Use short-lived tokens with automatic renewal
5. **Least Privilege**: Grant minimum required permissions
6. **Network Segmentation**: Isolate Vault in secure network zone

## Cost Considerations

Vault is open source and free to use. For HashiCorp Cloud Platform (HCP) Vault:

- Development tier: Free (limited requests)
- Standard tier: ~$0.50/hour per cluster (~$360/month)
- Plus/Premium: Higher tiers for enterprise features

For self-hosted Vault:
- Infrastructure costs only (compute, storage, network)
- No per-secret or per-request charges
- Typical single pepper: Negligible storage cost

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
vault token capabilities kv/data/encx/my-app-pepper/pepper
```

### "Upgrading from root module 1 to 2 is not supported"

You're trying to use KV v1 path with KV v2 provider. Upgrade to KV v2:

```bash
vault kv enable-versioning kv/
```

### "secret not found"

Create the pepper secret:

```bash
PEPPER=$(openssl rand -base64 32)
vault kv put kv/encx/my-app-pepper/pepper value="$PEPPER"
```

### "token expired"

Token renewal failed. Check token TTL:

```bash
vault token lookup
```

Recreate token with longer TTL or use AppRole.

### "Invalid pepper length"

Ensure pepper is exactly 32 bytes:

```bash
# Correct: 32 bytes (base64-encoded will be longer)
openssl rand -base64 32

# Verify length after decoding
PEPPER=$(openssl rand -base64 32)
echo -n "$PEPPER" | base64 -d | wc -c  # Should output: 32
```

## Migration from KV v1 to KV v2

If you're currently using KV v1:

```bash
# Enable versioning on existing KV v1 mount
vault kv enable-versioning kv/

# Or create new KV v2 mount and migrate
vault secrets enable -version=2 -path=kv2 kv
vault kv get -format=json kv/encx/my-app/pepper | \
    jq -r '.data' | \
    vault kv put kv2/encx/my-app/pepper -
```

Update your import paths to use the new provider structure.

## Related Providers

- **Vault Transit** (`providers/keys/hashicorp`): Pair with Vault KV for full Vault integration
- **AWS Secrets Manager** (`providers/secrets/aws`): Alternative secret storage service

## Documentation

- [encx Documentation](https://github.com/hengadev/encx)
- [Vault KV v2 Documentation](https://developer.hashicorp.com/vault/docs/secrets/kv/kv-v2)
- [encx API Reference](../../docs/API.md)
- [encx Architecture](../../docs/ARCHITECTURE.md)

## License

See the main encx repository for license information.
