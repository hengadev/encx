# ENCX v0.6.0 Migration Guide

> **For users upgrading from ENCX v0.5.x to v0.6.0+**
>
> This guide helps you migrate to the new architecture with separated KeyManagementService and SecretManagementService interfaces.

## Table of Contents

1. [Overview of Changes](#overview-of-changes)
2. [Breaking Changes](#breaking-changes)
3. [Before You Begin](#before-you-begin)
4. [Migration Steps](#migration-steps)
5. [Code Changes Required](#code-changes-required)
6. [Pepper Migration](#pepper-migration)
7. [Troubleshooting](#troubleshooting)
8. [FAQs](#faqs)
9. [Rollback Plan](#rollback-plan)

## Overview of Changes

ENCX v0.6.0 introduces a major architectural change: **separation of cryptographic operations from secret storage**.

### What Changed?

| Component | v0.5.x | v0.6.0+ |
|-----------|--------|---------|
| **Interfaces** | Single KMS interface | Separated: KeyManagementService + SecretManagementService |
| **Provider Imports** | `providers/awskms`, `providers/hashicorpvault` | `providers/aws`, `providers/hashicorp` |
| **Initialization** | `NewCrypto(ctx, options...)` | `NewCrypto(ctx, kms, secrets, cfg, options...)` |
| **Configuration** | Functional options only | Config struct + optional environment-based |
| **Pepper Storage** | Filesystem (`.encx/pepper.secret`) | Cloud storage (AWS Secrets Manager, Vault KV) |
| **Environment Variables** | `ENCX_PEPPER_SECRET_PATH` | `ENCX_PEPPER_ALIAS` |

### Why These Changes?

1. **Single Responsibility Principle**: Cryptographic operations and secret storage are now separate concerns
2. **Improved Security**: Peppers stored in cloud secret management services, not filesystem
3. **Better Flexibility**: Mix and match providers (e.g., AWS KMS + Vault KV)
4. **Clearer Dependencies**: Explicit dependency injection makes testing and configuration easier
5. **Cloud-Native**: First-class support for cloud secret management services

## Breaking Changes

### ⚠️ **All Changes are Breaking**

This is a major version update with significant breaking changes:

1. **Provider Import Paths Changed**
   - `github.com/hengadev/encx/providers/awskms` → `github.com/hengadev/encx/providers/aws`
   - `github.com/hengadev/encx/providers/hashicorpvault` → `github.com/hengadev/encx/providers/hashicorp`

2. **NewCrypto Signature Changed**
   - Old: `NewCrypto(ctx, options...)`
   - New: `NewCrypto(ctx, kms, secrets, cfg, options...)`

3. **Provider Constructors Changed**
   - AWS: `awskms.New()` → `aws.NewKMSService()` + `aws.NewSecretsManagerStore()`
   - Vault: `hashicorpvault.New()` → `hashicorp.NewTransitService()` + `hashicorp.NewKVStore()`

4. **Pepper Storage Changed**
   - Old: Filesystem at `ENCX_PEPPER_SECRET_PATH`
   - New: Cloud storage identified by `ENCX_PEPPER_ALIAS`

5. **Configuration Approach Changed**
   - Old: Functional options only
   - New: Config struct (recommended) or NewCryptoFromEnv (convenience)

### ✅ **What's NOT Breaking**

- Database schema (backward compatible)
- Encrypted data format (can decrypt old data)
- KEK versions (existing keys still work)
- Core encryption/decryption APIs

## Before You Begin

### Prerequisites

1. **Backup Your Data**
   - Backup your `.encx/keys.db` database
   - Backup your pepper file (usually at `.encx/pepper.secret`)
   - Backup your encrypted data

2. **Set Up Cloud Secret Storage**

   **For AWS Users:**
   ```bash
   # Ensure you have AWS credentials configured
   aws secretsmanager create-secret \
     --name encx/my-app-service/pepper \
     --secret-string "$(cat .encx/pepper.secret | base64)"
   ```

   **For Vault Users:**
   ```bash
   # Ensure Vault is accessible
   vault kv put secret/encx/my-app-service/pepper \
     value="$(cat .encx/pepper.secret | base64)"
   ```

3. **Review IAM/Vault Policies**
   - Ensure your application has permissions for both KMS and secret storage
   - See [AWS Provider README](../providers/aws/README.md) for IAM policies
   - See [HashiCorp Provider README](../providers/hashicorp/README.md) for Vault policies

4. **Plan Downtime**
   - This migration requires application restart
   - Plan for testing in staging before production

## Migration Steps

### Step 1: Update Dependencies

```bash
# Update to v0.6.0+
go get github.com/hengadev/encx@v0.6.0

# Update module
go mod tidy
```

### Step 2: Update Import Paths

**Before:**
```go
import (
    "github.com/hengadev/encx"
    "github.com/hengadev/encx/providers/awskms"
    // or
    "github.com/hengadev/encx/providers/hashicorpvault"
)
```

**After:**
```go
import (
    "github.com/hengadev/encx"
    "github.com/hengadev/encx/providers/aws"
    // or
    "github.com/hengadev/encx/providers/hashicorp"
)
```

### Step 3: Update Provider Initialization

**Before (AWS):**
```go
kms, err := awskms.New(ctx, awskms.Config{
    Region: "us-east-1",
})
```

**After (AWS):**
```go
// Initialize both services
kms, err := aws.NewKMSService(ctx, aws.Config{
    Region: "us-east-1",
})
if err != nil {
    log.Fatal(err)
}

secrets, err := aws.NewSecretsManagerStore(ctx, aws.Config{
    Region: "us-east-1",
})
if err != nil {
    log.Fatal(err)
}
```

**Before (Vault):**
```go
vault, err := hashicorpvault.New()
```

**After (Vault):**
```go
// Initialize both services
transit, err := hashicorp.NewTransitService()
if err != nil {
    log.Fatal(err)
}

kvStore, err := hashicorp.NewKVStore()
if err != nil {
    log.Fatal(err)
}
```

### Step 4: Update Crypto Initialization

**Before:**
```go
crypto, err := encx.NewCrypto(ctx,
    encx.WithKMSService(kms),
    encx.WithKEKAlias("my-app-kek"),
    encx.WithPepper(pepper),
)
```

**After (Explicit Configuration - Recommended for Libraries):**
```go
cfg := encx.Config{
    KEKAlias:    "my-app-kek",
    PepperAlias: "my-app-service",  // NEW: Service identifier
}

crypto, err := encx.NewCrypto(ctx, kms, secrets, cfg)
if err != nil {
    log.Fatal(err)
}
```

**After (Environment-Based - Recommended for Applications):**
```go
// Set environment variables:
// export ENCX_KEK_ALIAS="my-app-kek"
// export ENCX_PEPPER_ALIAS="my-app-service"

crypto, err := encx.NewCryptoFromEnv(ctx, kms, secrets)
if err != nil {
    log.Fatal(err)
}
```

### Step 5: Update Environment Variables

**Before:**
```bash
export ENCX_KEK_ALIAS="my-app-kek"
export ENCX_PEPPER_SECRET_PATH="/etc/encx/pepper.secret"
```

**After:**
```bash
export ENCX_KEK_ALIAS="my-app-kek"
export ENCX_PEPPER_ALIAS="my-app-service"  # Service identifier, not file path
```

### Step 6: Migrate Pepper to Cloud Storage

See [Pepper Migration](#pepper-migration) section below for detailed instructions.

### Step 7: Update Tests

**Before:**
```go
func TestEncryption(t *testing.T) {
    crypto, _ := encx.NewCrypto(ctx,
        encx.WithKMSService(mockKMS),
        encx.WithPepper(testPepper),
    )
    // ...
}
```

**After:**
```go
func TestEncryption(t *testing.T) {
    // Option 1: Use test helper (simplest)
    crypto := encx.NewTestCrypto(t)

    // Option 2: Explicit setup
    kms := encx.NewSimpleTestKMS()
    secrets := encx.NewInMemorySecretStore()
    cfg := encx.Config{
        KEKAlias:    "test-kek",
        PepperAlias: "test-service",
    }
    crypto, _ := encx.NewCrypto(ctx, kms, secrets, cfg)
    // ...
}
```

### Step 8: Deploy and Verify

1. Deploy to staging environment
2. Verify pepper is accessible from cloud storage
3. Test encryption/decryption operations
4. Test existing encrypted data can be decrypted
5. Monitor logs for errors
6. Deploy to production with rollback plan ready

## Code Changes Required

### Complete Before/After Example (AWS)

**Before (v0.5.x):**
```go
package main

import (
    "context"
    "log"

    "github.com/hengadev/encx"
    "github.com/hengadev/encx/providers/awskms"
)

func main() {
    ctx := context.Background()

    // Initialize KMS
    kms, err := awskms.New(ctx, awskms.Config{
        Region: "us-east-1",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Load pepper from filesystem
    pepper, err := os.ReadFile(".encx/pepper.secret")
    if err != nil {
        log.Fatal(err)
    }

    // Initialize crypto
    crypto, err := encx.NewCrypto(ctx,
        encx.WithKMSService(kms),
        encx.WithKEKAlias("my-app-kek"),
        encx.WithPepper(pepper),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Use crypto...
}
```

**After (v0.6.0+):**
```go
package main

import (
    "context"
    "log"

    "github.com/hengadev/encx"
    "github.com/hengadev/encx/providers/aws"
)

func main() {
    ctx := context.Background()

    // Initialize KMS for cryptographic operations
    kms, err := aws.NewKMSService(ctx, aws.Config{
        Region: "us-east-1",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Initialize Secrets Manager for pepper storage
    secrets, err := aws.NewSecretsManagerStore(ctx, aws.Config{
        Region: "us-east-1",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create configuration
    cfg := encx.Config{
        KEKAlias:    "my-app-kek",
        PepperAlias: "my-app-service",  // Service identifier
    }

    // Initialize crypto (pepper loaded automatically from Secrets Manager)
    crypto, err := encx.NewCrypto(ctx, kms, secrets, cfg)
    if err != nil {
        log.Fatal(err)
    }

    // Use crypto...
}
```

### Complete Before/After Example (HashiCorp Vault)

**Before (v0.5.x):**
```go
package main

import (
    "context"
    "log"

    "github.com/hengadev/encx"
    "github.com/hengadev/encx/providers/hashicorpvault"
)

func main() {
    ctx := context.Background()

    // Initialize Vault
    vault, err := hashicorpvault.New()
    if err != nil {
        log.Fatal(err)
    }

    // Load pepper from filesystem
    pepper, err := os.ReadFile(".encx/pepper.secret")
    if err != nil {
        log.Fatal(err)
    }

    // Initialize crypto
    crypto, err := encx.NewCrypto(ctx,
        encx.WithKMSService(vault),
        encx.WithKEKAlias("my-app-kek"),
        encx.WithPepper(pepper),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Use crypto...
}
```

**After (v0.6.0+):**
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
        log.Fatal(err)
    }

    // Initialize KV Store for pepper storage
    kvStore, err := hashicorp.NewKVStore()
    if err != nil {
        log.Fatal(err)
    }

    // Create configuration
    cfg := encx.Config{
        KEKAlias:    "my-app-kek",
        PepperAlias: "my-app-service",  // Service identifier
    }

    // Initialize crypto (pepper loaded automatically from Vault KV)
    crypto, err := encx.NewCrypto(ctx, transit, kvStore, cfg)
    if err != nil {
        log.Fatal(err)
    }

    // Use crypto...
}
```

## Pepper Migration

### Why Migrate Peppers to Cloud Storage?

| Aspect | Filesystem | Cloud Storage |
|--------|-----------|---------------|
| **Deployment** | Manual file management | Automatic retrieval |
| **Backup** | Separate backup process | Included in cloud backups |
| **Access Control** | Filesystem permissions | IAM/Vault policies |
| **Multi-Region** | File replication needed | Native replication |
| **Auditing** | No audit trail | Full audit logs |
| **Rotation** | Manual file updates | Centralized management |

### Migration Procedure

#### Option 1: Automatic Migration (Recommended)

The first time you initialize ENCX v0.6.0+ with a new `PepperAlias`, it will:
1. Check if pepper exists in cloud storage
2. If not found, generate a new 32-byte pepper
3. Store it automatically

**If you want to preserve your existing pepper:**

**AWS:**
```bash
# Read existing pepper
PEPPER=$(cat .encx/pepper.secret)

# Store in AWS Secrets Manager
aws secretsmanager create-secret \
    --name encx/my-app-service/pepper \
    --secret-string "$PEPPER" \
    --region us-east-1

# Verify
aws secretsmanager get-secret-value \
    --secret-id encx/my-app-service/pepper \
    --region us-east-1
```

**Vault:**
```bash
# Read existing pepper
PEPPER=$(cat .encx/pepper.secret)

# Store in Vault KV v2
vault kv put secret/encx/my-app-service/pepper value="$PEPPER"

# Verify
vault kv get secret/encx/my-app-service/pepper
```

#### Option 2: Let ENCX Generate New Pepper

If you don't need to preserve the existing pepper (e.g., fresh deployment or non-production):

1. Remove or ignore `.encx/pepper.secret`
2. Initialize ENCX v0.6.0+ with your chosen `PepperAlias`
3. ENCX will automatically generate and store a new pepper

**⚠️ Important:** If you let ENCX generate a new pepper, **all existing password hashes will be invalid** and users will need to reset passwords.

### Post-Migration Cleanup

After successful migration and verification:

```bash
# Optional: Remove old pepper file
rm .encx/pepper.secret

# Update .gitignore (if you had pepper.secret tracked)
# Remove: .encx/pepper.secret
# Keep: .encx/keys.db
```

## Troubleshooting

### Error: "KeyManagementService is required"

**Cause**: You're not passing the KMS service to `NewCrypto`.

**Solution**:
```go
// ❌ Wrong
crypto, err := encx.NewCrypto(ctx, cfg)

// ✅ Correct
crypto, err := encx.NewCrypto(ctx, kms, secrets, cfg)
```

### Error: "SecretManagementService is required"

**Cause**: You're not passing the secret storage service to `NewCrypto`.

**Solution**:
```go
// ❌ Wrong
crypto, err := encx.NewCrypto(ctx, kms, cfg)

// ✅ Correct
crypto, err := encx.NewCrypto(ctx, kms, secrets, cfg)
```

### Error: "KEKAlias is required"

**Cause**: Config struct doesn't have KEKAlias set.

**Solution**:
```go
// ❌ Wrong
cfg := encx.Config{
    PepperAlias: "my-service",
}

// ✅ Correct
cfg := encx.Config{
    KEKAlias:    "my-app-kek",
    PepperAlias: "my-service",
}
```

### Error: "PepperAlias is required"

**Cause**: Config struct doesn't have PepperAlias set.

**Solution**:
```go
// ❌ Wrong
cfg := encx.Config{
    KEKAlias: "my-app-kek",
}

// ✅ Correct
cfg := encx.Config{
    KEKAlias:    "my-app-kek",
    PepperAlias: "my-service",
}
```

### Error: "environment variable ENCX_KEK_ALIAS is required"

**Cause**: Using `NewCryptoFromEnv` without setting required environment variables.

**Solution**:
```bash
# Set required environment variables
export ENCX_KEK_ALIAS="my-app-kek"
export ENCX_PEPPER_ALIAS="my-app-service"
```

### Error: "pepper not found for alias: my-service"

**Cause**: Pepper doesn't exist in cloud storage and automatic generation failed.

**Solution**:
1. Check cloud service connectivity
2. Verify IAM/Vault permissions for CreateSecret/Write
3. Manually create the pepper (see [Pepper Migration](#pepper-migration))

### Error: "permission denied" (AWS)

**Cause**: Missing IAM permissions for Secrets Manager.

**Solution**:
```json
{
  "Effect": "Allow",
  "Action": [
    "secretsmanager:GetSecretValue",
    "secretsmanager:CreateSecret",
    "secretsmanager:PutSecretValue"
  ],
  "Resource": "arn:aws:secretsmanager:REGION:ACCOUNT:secret:encx/*"
}
```

### Error: "permission denied" (Vault)

**Cause**: Missing Vault policy for KV v2.

**Solution**:
```hcl
path "secret/data/encx/*" {
  capabilities = ["create", "read", "update"]
}
```

### Imports Not Found After Update

**Cause**: Old import paths cached in Go module cache.

**Solution**:
```bash
# Clear module cache
go clean -modcache

# Re-download dependencies
go mod tidy
go mod download
```

## FAQs

### Q: Why separate KeyManagementService and SecretManagementService?

**A:** Following the Single Responsibility Principle:
- **KeyManagementService**: Handles cryptographic operations (encrypt/decrypt DEKs)
- **SecretManagementService**: Handles secret storage (store/retrieve pepper)

This separation provides better flexibility, security, and testability.

### Q: Can I still use filesystem for peppers?

**A:** No. v0.6.0+ requires cloud-based secret storage. This provides better security, automatic backups, audit trails, and proper access control.

For testing, use `InMemorySecretStore`:
```go
secrets := encx.NewInMemorySecretStore()
```

### Q: Do I need to migrate existing peppers?

**A:** **Yes, if you want to preserve existing password hashes.** If you generate a new pepper, all existing password hashes will become invalid.

### Q: Will this migration break my existing encrypted data?

**A:** **No.** The database schema and encrypted data format are unchanged. You can decrypt existing data with v0.6.0+.

### Q: Can I mix providers (e.g., AWS KMS + Vault KV)?

**A:** **Yes!** This is one of the benefits of the separated architecture:

```go
kms, _ := aws.NewKMSService(ctx, aws.Config{Region: "us-east-1"})
secrets, _ := hashicorp.NewKVStore()  // Vault KV for secrets

crypto, _ := encx.NewCrypto(ctx, kms, secrets, cfg)
```

### Q: What happens if cloud storage is unavailable?

**A:** Initialization will fail with `ErrSecretStorageUnavailable`. Your application should handle this error and retry with exponential backoff or alert operations.

### Q: How do I test without cloud services?

**A:** Use the in-memory test implementations:

```go
kms := encx.NewSimpleTestKMS()
secrets := encx.NewInMemorySecretStore()

crypto, _ := encx.NewCrypto(ctx, kms, secrets, cfg)
```

Or use the convenience helper:
```go
crypto := encx.NewTestCrypto(t)
```

### Q: Can I use the old API temporarily?

**A:** **No.** The old API is completely removed in v0.6.0. You must update to the new API. Follow this migration guide carefully.

### Q: Will there be more breaking changes?

**A:** This is a major architectural refactoring. Future v0.6.x versions will maintain backward compatibility. The next major version (v0.7.0) may introduce new breaking changes, but you'll have a migration guide.

## Rollback Plan

If you encounter critical issues and need to rollback:

### Step 1: Downgrade ENCX

```bash
go get github.com/hengadev/encx@v0.5.3
go mod tidy
```

### Step 2: Revert Code Changes

```bash
git revert <migration-commit-hash>
# or
git checkout <previous-working-commit>
```

### Step 3: Restore Filesystem Pepper

```bash
# If you migrated pepper to cloud, restore from backup
cp .encx/pepper.secret.backup .encx/pepper.secret

# Or retrieve from cloud storage
# AWS:
aws secretsmanager get-secret-value \
    --secret-id encx/my-app-service/pepper \
    --query SecretString \
    --output text > .encx/pepper.secret

# Vault:
vault kv get -field=value secret/encx/my-app-service/pepper > .encx/pepper.secret
```

### Step 4: Revert Environment Variables

```bash
export ENCX_PEPPER_SECRET_PATH=".encx/pepper.secret"
unset ENCX_PEPPER_ALIAS
```

### Step 5: Restart Application

```bash
# Restart with v0.5.3 and old configuration
./your-app
```

## Migration Checklist

Use this checklist to track your migration progress:

- [ ] **Pre-Migration**
  - [ ] Backup `.encx/keys.db` database
  - [ ] Backup `.encx/pepper.secret` file
  - [ ] Backup encrypted data
  - [ ] Review IAM/Vault policies
  - [ ] Set up cloud secret storage
  - [ ] Test in staging environment

- [ ] **Code Changes**
  - [ ] Update ENCX to v0.6.0+
  - [ ] Update import paths
  - [ ] Update provider initialization (separate KMS and secrets)
  - [ ] Update crypto initialization (Config struct or NewCryptoFromEnv)
  - [ ] Update environment variables
  - [ ] Update tests

- [ ] **Pepper Migration**
  - [ ] Migrate pepper to cloud storage
  - [ ] Verify pepper accessibility
  - [ ] Test password hashing with migrated pepper

- [ ] **Testing**
  - [ ] Run unit tests
  - [ ] Run integration tests
  - [ ] Test encryption/decryption of new data
  - [ ] Test decryption of existing data
  - [ ] Test password verification with existing hashes

- [ ] **Deployment**
  - [ ] Deploy to staging
  - [ ] Verify staging functionality
  - [ ] Monitor staging logs
  - [ ] Deploy to production
  - [ ] Monitor production logs
  - [ ] Verify production functionality

- [ ] **Post-Migration**
  - [ ] Remove old pepper file (optional)
  - [ ] Update documentation
  - [ ] Update runbooks
  - [ ] Train team on new architecture
  - [ ] Update CI/CD pipelines

## Benefits of Upgrading

1. **Better Security**: Peppers in cloud secret stores with audit trails
2. **Clearer Architecture**: Separated concerns are easier to understand and maintain
3. **Improved Flexibility**: Mix and match providers as needed
4. **Better Testing**: Mock only what you need with in-memory implementations
5. **Cloud-Native**: First-class support for AWS and HashiCorp Vault
6. **Explicit Dependencies**: No hidden configuration, everything is explicit
7. **12-Factor App Support**: Environment-based configuration with `NewCryptoFromEnv`

## Support and Resources

- **Architecture Documentation**: [docs/ARCHITECTURE.md](./ARCHITECTURE.md)
- **API Reference**: [docs/API.md](./API.md)
- **AWS Provider Guide**: [providers/aws/README.md](../providers/aws/README.md)
- **HashiCorp Provider Guide**: [providers/hashicorp/README.md](../providers/hashicorp/README.md)
- **Troubleshooting Guide**: [docs/TROUBLESHOOTING.md](./TROUBLESHOOTING.md)
- **GitHub Issues**: https://github.com/hengadev/encx/issues

---

**Good luck with your migration! The new architecture provides better security and flexibility for the long term.**
