# HashiCorp Vault Provider for ENCX

HashiCorp Vault provider for the encx encryption library, implementing both key management and secret storage.

## Overview

The HashiCorp Vault provider includes two services that work together:

1. **TransitService** - Implements `encx.KeyManagementService` using Vault Transit Engine for encryption/decryption operations
2. **KVStore** - Implements `encx.SecretManagementService` using Vault KV v2 for pepper storage

This separation follows the single responsibility principle: Transit Engine handles cryptographic operations while KV Store handles secret storage.

## Prerequisites

### 1. Vault Server

- HashiCorp Vault server running (self-hosted or HCP Vault)
- Vault CLI installed (for setup)
- Transit Engine enabled
- KV v2 Engine enabled

### 2. Enable Vault Engines

```bash
# Enable Transit Engine
vault secrets enable transit

# Enable KV v2 Engine (usually enabled by default at "secret/")
vault secrets enable -path=secret kv-v2
```

### 3. Authentication

Configure one of these authentication methods:

**Direct Token (Development)**:
```bash
export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_TOKEN="your-vault-token"
```

**AppRole (Production)**:
```bash
export VAULT_ADDR="https://vault.example.com"
export VAULT_ROLE_ID="your-role-id"
export VAULT_SECRET_ID="your-secret-id"
```

**HCP Vault**:
```bash
export VAULT_ADDR="https://your-cluster.vault.hashicorp.cloud:8200"
export VAULT_NAMESPACE="admin"
export VAULT_TOKEN="your-hcp-token"
```

### 4. Vault Policies

Create a policy for ENCX operations:

```hcl
# encx-policy.hcl
path "transit/encrypt/my-encryption-key" {
  capabilities = ["update"]
}

path "transit/decrypt/my-encryption-key" {
  capabilities = ["update"]
}

path "transit/keys/my-encryption-key" {
  capabilities = ["read"]
}

path "secret/data/encx/*" {
  capabilities = ["create", "read", "update"]
}

path "secret/metadata/encx/*" {
  capabilities = ["read", "list"]
}
```

Apply the policy:
```bash
vault policy write encx-policy encx-policy.hcl
```

## Creating a Transit Key

### Using Vault CLI

```bash
# Create encryption key
vault write -f transit/keys/my-encryption-key

# Verify key creation
vault read transit/keys/my-encryption-key

# Enable key rotation (optional)
vault write -f transit/keys/my-encryption-key/rotate
```

### Using Vault API

```bash
curl \
    --header "X-Vault-Token: $VAULT_TOKEN" \
    --request POST \
    $VAULT_ADDR/v1/transit/keys/my-encryption-key
```

### Using Terraform

```hcl
resource "vault_mount" "transit" {
  path = "transit"
  type = "transit"
}

resource "vault_transit_secret_backend_key" "encx" {
  backend = vault_mount.transit.path
  name    = "my-encryption-key"

  # Enable automatic key rotation
  auto_rotate_period = 7776000  # 90 days
}
```

## Usage

### Basic Example

```go
package main

import (
    "context"
    "log"

    "github.com/hengadev/encx"
    "github.com/hengadev/encx/providers/hashicorp"
)

func main() {
    ctx := context.Background()

    // Initialize Transit Engine for cryptographic operations
    transit, err := hashicorp.NewTransitService()
    if err != nil {
        log.Fatalf("Failed to create Transit service: %v", err)
    }

    // Initialize KV Store for pepper storage
    kvStore, err := hashicorp.NewKVStore()
    if err != nil {
        log.Fatalf("Failed to create KV store: %v", err)
    }

    // Create explicit configuration
    cfg := encx.Config{
        KEKAlias:    "my-encryption-key",  // Transit key name
        PepperAlias: "my-app-service",     // Service identifier
    }

    // Create encx crypto service
    crypto, err := encx.NewCrypto(ctx, transit, kvStore, cfg)
    if err != nil {
        log.Fatalf("Failed to create crypto service: %v", err)
    }

    // Encrypt data
    plaintext := []byte("sensitive data")
    dek, _ := crypto.GenerateDEK()
    ciphertext, err := crypto.EncryptData(ctx, plaintext, dek)
    if err != nil {
        log.Fatalf("Encryption failed: %v", err)
    }

    log.Printf("Encrypted: %x", ciphertext)
}
```

### Environment-based Configuration

For 12-factor apps, use environment variables:

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/hengadev/encx"
    "github.com/hengadev/encx/providers/hashicorp"
)

func main() {
    ctx := context.Background()

    // Set environment variables:
    // export VAULT_ADDR="https://vault.example.com"
    // export VAULT_TOKEN="your-token"  # or VAULT_ROLE_ID/VAULT_SECRET_ID
    // export ENCX_KEK_ALIAS="my-encryption-key"
    // export ENCX_PEPPER_ALIAS="my-app-service"

    // Initialize providers
    transit, _ := hashicorp.NewTransitService()
    kvStore, _ := hashicorp.NewKVStore()

    // Load configuration from environment
    crypto, err := encx.NewCryptoFromEnv(ctx, transit, kvStore)
    if err != nil {
        log.Fatalf("Failed to create crypto service: %v", err)
    }

    // Ready to use
    dek, _ := crypto.GenerateDEK()
    // ...
}
```

### Production Setup with AppRole

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/hengadev/encx"
    "github.com/hengadev/encx/providers/hashicorp"
)

func main() {
    ctx := context.Background()

    // Environment variables required:
    // export VAULT_ADDR="https://vault.example.com"
    // export VAULT_ROLE_ID="role-id-from-approle"
    // export VAULT_SECRET_ID="secret-id-from-approle"
    // export ENCX_KEK_ALIAS="production-key"
    // export ENCX_PEPPER_ALIAS="production-service"

    // Vault providers (automatically uses AppRole if configured)
    transit, err := hashicorp.NewTransitService()
    if err != nil {
        log.Fatal(err)
    }

    kvStore, err := hashicorp.NewKVStore()
    if err != nil {
        log.Fatal(err)
    }

    // Load configuration from environment
    crypto, err := encx.NewCryptoFromEnv(ctx, transit, kvStore)
    if err != nil {
        log.Fatal(err)
    }

    // Your application logic here
    log.Println("Crypto service initialized successfully")
}
```

## Pepper Storage

The `KVStore` automatically manages pepper storage in Vault KV v2:

### Storage Path

Peppers are stored at: `secret/data/encx/{PepperAlias}/pepper`

For example:
- PepperAlias: `my-app-service` → KV path: `secret/data/encx/my-app-service/pepper`
- PepperAlias: `payment-service` → KV path: `secret/data/encx/payment-service/pepper`

**Note**: The `data` in the path is required for KV v2 access.

### Automatic Pepper Management

The first time you initialize crypto with a new `PepperAlias`:
1. ENCX checks if pepper exists in KV store
2. If not found, generates a secure random 32-byte pepper
3. Stores it in KV v2 at `secret/data/encx/{PepperAlias}/pepper`
4. Subsequent initializations load the existing pepper

### Manual Pepper Inspection

```bash
# View pepper (requires policy permissions)
vault kv get secret/encx/my-app-service/pepper

# List all encx peppers
vault kv list secret/encx

# View specific version
vault kv get -version=1 secret/encx/my-app-service/pepper
```

## Authentication Methods

### Direct Token (Development Only)

```bash
export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_TOKEN="root"  # Never use root token in production!
```

### AppRole (Recommended for Production)

**Setup AppRole:**
```bash
# Enable AppRole auth
vault auth enable approle

# Create role
vault write auth/approle/role/encx \
    token_ttl=1h \
    token_max_ttl=4h \
    policies="encx-policy"

# Get role ID
vault read auth/approle/role/encx/role-id

# Generate secret ID
vault write -f auth/approle/role/encx/secret-id
```

**Use in application:**
```bash
export VAULT_ADDR="https://vault.example.com"
export VAULT_ROLE_ID="<role-id-from-above>"
export VAULT_SECRET_ID="<secret-id-from-above>"
```

### Kubernetes Auth (For K8s Deployments)

```bash
# Enable Kubernetes auth
vault auth enable kubernetes

# Configure Kubernetes auth
vault write auth/kubernetes/config \
    kubernetes_host="https://kubernetes.default.svc:443"

# Create role
vault write auth/kubernetes/role/encx \
    bound_service_account_names=encx-sa \
    bound_service_account_namespaces=default \
    policies=encx-policy \
    ttl=1h
```

Then mount service account token in your pod.

## Multi-Environment Setup

Use unique aliases for different environments:

```go
// Development
cfg := encx.Config{
    KEKAlias:    "dev-encryption-key",
    PepperAlias: "myapp-dev",
}

// Staging
cfg := encx.Config{
    KEKAlias:    "staging-encryption-key",
    PepperAlias: "myapp-staging",
}

// Production
cfg := encx.Config{
    KEKAlias:    "prod-encryption-key",
    PepperAlias: "myapp-prod",
}
```

## Key Rotation

Vault Transit Engine supports automatic key rotation:

```bash
# Manually rotate key
vault write -f transit/keys/my-encryption-key/rotate

# Enable automatic rotation (90 days)
vault write transit/keys/my-encryption-key/config \
    auto_rotate_period=7776000

# View key versions
vault read transit/keys/my-encryption-key
```

**ENCX automatically handles key versioning:**
- Old data encrypted with version 1 can still be decrypted
- New encryptions use the latest key version
- No application changes required

## Error Handling

```go
crypto, err := encx.NewCrypto(ctx, transit, kvStore, cfg)
if err != nil {
    switch {
    case errors.Is(err, encx.ErrKMSUnavailable):
        // Vault unavailable - check connectivity
        log.Println("Vault unavailable, retrying...")
    case errors.Is(err, encx.ErrSecretStorageUnavailable):
        // KV store unavailable
        log.Println("Vault KV unavailable")
    case errors.Is(err, encx.ErrInvalidConfiguration):
        // Configuration validation failed
        log.Printf("Invalid configuration: %v", err)
    default:
        // Other error
        log.Printf("Unexpected error: %v", err)
    }
}
```

## Performance Optimization

### Connection Reuse

Both `TransitService` and `KVStore` share the same Vault client connection, minimizing overhead.

### DEK Caching

For high-throughput applications, cache DEKs to reduce Vault API calls:

```go
// Cache DEKs by record ID
type DEKCache struct {
    mu    sync.RWMutex
    cache map[string][]byte
}

func (c *DEKCache) GetOrGenerate(ctx context.Context, crypto *encx.Crypto, id string) ([]byte, error) {
    c.mu.RLock()
    if dek, ok := c.cache[id]; ok {
        c.mu.RUnlock()
        return dek, nil
    }
    c.mu.RUnlock()

    dek, err := crypto.GenerateDEK()
    if err != nil {
        return nil, err
    }

    c.mu.Lock()
    c.cache[id] = dek
    c.mu.Unlock()

    return dek, nil
}
```

## Troubleshooting

### Permission Denied - Transit

```
Error: permission denied
```

**Solution**: Verify Transit Engine policy:
```bash
vault policy read encx-policy
# Should include transit/encrypt and transit/decrypt paths
```

### Permission Denied - KV

```
Error: permission denied on path "secret/data/encx/my-app/pepper"
```

**Solution**: Add KV permissions to policy:
```hcl
path "secret/data/encx/*" {
  capabilities = ["create", "read", "update"]
}
```

### Key Not Found

```
Error: transit key not found
```

**Solutions**:
- Create key: `vault write -f transit/keys/my-encryption-key`
- Verify Transit Engine is enabled: `vault secrets list`
- Check key name matches `KEKAlias` in config

### Vault Unavailable

```
Error: failed to connect to Vault
```

**Solutions**:
- Verify `VAULT_ADDR` is correct
- Check network connectivity
- Verify Vault server is running
- Check if using HTTPS with self-signed cert (set `VAULT_SKIP_VERIFY=true` for testing only)

### Token Expired

```
Error: permission denied (expired token)
```

**Solutions**:
- Renew token: `vault token renew`
- Use AppRole for automatic token renewal
- Increase `token_ttl` in AppRole configuration

## Security Best Practices

1. **Use AppRole** for production (not direct tokens)
2. **Enable audit logging** in Vault for compliance
3. **Use least-privilege policies** - grant only necessary permissions
4. **Use unique PepperAlias** for each service/environment
5. **Enable TLS** for Vault communication
6. **Rotate tokens regularly** (automatic with AppRole)
7. **Monitor Vault usage** with audit logs
8. **Backup Vault data** regularly (especially KV secrets)
9. **Use separate keys** for dev/staging/prod
10. **Enable automatic key rotation** for Transit keys

## HCP Vault

For HashiCorp Cloud Platform Vault:

```bash
export VAULT_ADDR="https://your-cluster.vault.hashicorp.cloud:8200"
export VAULT_NAMESPACE="admin"
export VAULT_TOKEN="your-hcp-token"
```

**Namespace support is automatic** - the provider will use the `VAULT_NAMESPACE` environment variable if set.

## Testing

For unit tests with mock services:
```go
func TestEncryption(t *testing.T) {
    crypto, _ := encx.NewTestCrypto(t)
    // Test your encryption logic
}
```

For integration testing with real Vault:
```bash
export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN=root
export ENCX_KEK_ALIAS=test-key
export ENCX_PEPPER_ALIAS=test-service
go test -tags=integration ./providers/hashicorp
```

## Docker Compose for Development

```yaml
version: '3.8'
services:
  vault:
    image: hashicorp/vault:latest
    ports:
      - "8200:8200"
    environment:
      VAULT_DEV_ROOT_TOKEN_ID: "root"
      VAULT_DEV_LISTEN_ADDRESS: "0.0.0.0:8200"
    cap_add:
      - IPC_LOCK
    command: server -dev
```

Start and configure:
```bash
docker-compose up -d
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='root'

# Enable engines
vault secrets enable transit
vault write -f transit/keys/my-encryption-key
```

## References

- [HashiCorp Vault Documentation](https://developer.hashicorp.com/vault/docs)
- [Vault Transit Engine](https://developer.hashicorp.com/vault/docs/secrets/transit)
- [Vault KV v2](https://developer.hashicorp.com/vault/docs/secrets/kv/kv-v2)
- [Vault AppRole Auth](https://developer.hashicorp.com/vault/docs/auth/approle)
- [ENCX Documentation](../../README.md)
