# ENCX Architecture

This document explains the architectural design of ENCX, focusing on the separation of concerns between key management and secret storage.

## Table of Contents

- [Overview](#overview)
- [Core Architecture](#core-architecture)
- [Interface Separation](#interface-separation)
- [Provider Implementations](#provider-implementations)
- [Pepper Management](#pepper-management)
- [Data Flow](#data-flow)
- [Design Principles](#design-principles)
- [Security Model](#security-model)

## Overview

ENCX is a cryptographic library designed for Go applications that need to encrypt sensitive data at the field level. It follows a dual-service architecture where **cryptographic operations** and **secret storage** are handled by separate, specialized components.

### Key Concepts

- **DEK (Data Encryption Key)**: 32-byte symmetric key used to encrypt/decrypt application data
- **KEK (Key Encryption Key)**: Master key stored in KMS used to encrypt DEKs
- **Pepper**: 32-byte secret value used to strengthen password hashing
- **Envelope Encryption**: DEKs are encrypted with KEK before storage

## Core Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Application                          │
└────────────────────────┬────────────────────────────────────┘
                         │
                         │ encx.NewCrypto(ctx, kms, secrets, cfg)
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                       Crypto Service                        │
│  ┌───────────────────────────────────────────────────┐     │
│  │   • EncryptData / DecryptData                     │     │
│  │   • EncryptDEK / DecryptDEK                       │     │
│  │   • HashBasic / HashSecure                        │     │
│  │   • ProcessStruct / DecryptStruct                 │     │
│  │   • RotateKEK                                     │     │
│  └───────────────────────────────────────────────────┘     │
└───────────┬─────────────────────────────┬───────────────────┘
            │                             │
            │                             │
   ┌────────▼─────────┐         ┌────────▼──────────┐
   │ KeyManagement    │         │ SecretManagement  │
   │ Service          │         │ Service           │
   ├──────────────────┤         ├───────────────────┤
   │ • GetKeyID       │         │ • StorePepper     │
   │ • CreateKey      │         │ • GetPepper       │
   │ • EncryptDEK     │         │ • PepperExists    │
   │ • DecryptDEK     │         │ • GetStoragePath  │
   └────────┬─────────┘         └────────┬──────────┘
            │                            │
            │                            │
   ┌────────▼─────────┐         ┌────────▼──────────┐
   │  Cloud KMS       │         │  Secret Store     │
   │                  │         │                   │
   │ • AWS KMS        │         │ • AWS Secrets Mgr │
   │ • Vault Transit  │         │ • Vault KV v2     │
   │ • Test Mock      │         │ • In-Memory       │
   └──────────────────┘         └───────────────────┘
```

## Interface Separation

### Why Separate Interfaces?

ENCX v0.6.0+ separates cryptographic operations from secret storage following the **Single Responsibility Principle**:

1. **KeyManagementService**: Handles cryptographic operations (encrypt/decrypt DEKs)
2. **SecretManagementService**: Handles secret storage (store/retrieve pepper)

### Benefits of Separation

| Benefit | Description |
|---------|-------------|
| **Clear Responsibilities** | Each interface has a single, well-defined purpose |
| **Flexible Deployment** | Mix and match providers (e.g., AWS KMS + Vault KV) |
| **Independent Scaling** | Scale cryptographic and storage operations independently |
| **Simplified Testing** | Mock only what you need for each test scenario |
| **Better Security** | Principle of least privilege - different IAM permissions for each service |
| **Provider Agnostic** | Easy to switch providers without changing application code |

## Provider Implementations

### AWS Provider

The AWS provider implements both interfaces using separate AWS services:

```go
// KeyManagementService implementation
type KMSService struct {
    client *kms.Client
    region string
}

// SecretManagementService implementation
type SecretsManagerStore struct {
    client *secretsmanager.Client
    region string
}
```

**Usage**:
```go
kms, _ := aws.NewKMSService(ctx, aws.Config{Region: "us-east-1"})
secrets, _ := aws.NewSecretsManagerStore(ctx, aws.Config{Region: "us-east-1"})

crypto, _ := encx.NewCrypto(ctx, kms, secrets, cfg)
```

**Services Used**:
- **AWS KMS**: Encrypt/decrypt DEKs using customer master keys
- **AWS Secrets Manager**: Store/retrieve peppers securely

**IAM Permissions Required**:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "kms:Encrypt",
        "kms:Decrypt",
        "kms:DescribeKey"
      ],
      "Resource": "arn:aws:kms:REGION:ACCOUNT:key/KEY-ID"
    },
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue",
        "secretsmanager:CreateSecret",
        "secretsmanager:PutSecretValue"
      ],
      "Resource": "arn:aws:secretsmanager:REGION:ACCOUNT:secret:encx/*"
    }
  ]
}
```

### HashiCorp Vault Provider

The HashiCorp provider implements both interfaces using separate Vault engines:

```go
// KeyManagementService implementation
type TransitService struct {
    client *api.Client
}

// SecretManagementService implementation
type KVStore struct {
    client *api.Client
}
```

**Usage**:
```go
transit, _ := hashicorp.NewTransitService()
kvStore, _ := hashicorp.NewKVStore()

crypto, _ := encx.NewCrypto(ctx, transit, kvStore, cfg)
```

**Vault Engines Used**:
- **Transit Engine**: Encrypt/decrypt DEKs using Vault-managed keys
- **KV v2 Engine**: Store/retrieve peppers with versioning

**Vault Policies Required**:
```hcl
# Transit Engine permissions
path "transit/encrypt/my-key" {
  capabilities = ["update"]
}
path "transit/decrypt/my-key" {
  capabilities = ["update"]
}

# KV v2 permissions
path "secret/data/encx/*" {
  capabilities = ["create", "read", "update"]
}
```

### Test Provider

For testing, ENCX provides in-memory implementations:

```go
kms := encx.NewSimpleTestKMS()              // Mock KMS
secrets := encx.NewInMemorySecretStore()    // In-memory storage

crypto, _ := encx.NewCrypto(ctx, kms, secrets, cfg)
```

## Pepper Management

### What is a Pepper?

A **pepper** is a 32-byte secret value used to strengthen password hashing:

- Added to passwords before hashing with Argon2id
- Stored separately from the database
- Same pepper used across all password hashes
- Never changes (unlike salts, which are per-password)

### Pepper Storage Evolution

**Before v0.6.0 (Filesystem)**:
```
.encx/
└── pepper.secret    # Stored on filesystem
```

**After v0.6.0 (Cloud Secret Store)**:
```
AWS Secrets Manager:
└── encx/{PepperAlias}/pepper

HashiCorp Vault KV:
└── secret/data/encx/{PepperAlias}/pepper
```

### Why Move Peppers to Cloud Storage?

| Challenge | Old (Filesystem) | New (Cloud Storage) |
|-----------|------------------|---------------------|
| **Deployment** | Manual file management | Automatic retrieval |
| **Rotation** | Manual file updates | Centralized management |
| **Backup** | Separate backup process | Included in cloud backups |
| **Access Control** | Filesystem permissions | IAM/Vault policies |
| **Multi-Region** | File replication needed | Native replication support |
| **Auditing** | No audit trail | Full audit logs |
| **Secret Rotation** | Requires app restart | Seamless updates |

### Automatic Pepper Management

ENCX automatically handles pepper lifecycle:

```go
crypto, _ := encx.NewCrypto(ctx, kms, secrets, encx.Config{
    KEKAlias:    "my-kek",
    PepperAlias: "my-service",  // Service identifier
})
```

**First initialization**:
1. Check if pepper exists at `encx/{PepperAlias}/pepper`
2. If not found, generate secure 32-byte pepper
3. Store in SecretManagementService
4. Use for hashing operations

**Subsequent initializations**:
1. Check if pepper exists
2. Load existing pepper
3. Use for hashing operations

## Data Flow

### Encryption Flow

```
┌─────────────┐
│ Application │
└──────┬──────┘
       │ plaintext
       ▼
┌──────────────────────────────────────────┐
│ Crypto.EncryptData(plaintext, dek)       │
│                                          │
│ 1. Generate DEK (32 bytes)               │
│ 2. Encrypt plaintext with DEK (AES-GCM)  │
│ 3. Encrypt DEK with KEK (via KMS)        │
│ 4. Store encrypted DEK + ciphertext      │
└──────────────────────────────────────────┘
       │
       ├─── DEK Encryption ────►  ┌─────────────────────┐
       │                          │ KeyManagementService│
       │                          │ (AWS KMS / Vault)   │
       │                          └─────────────────────┘
       │
       └─► ciphertext + encrypted_dek
```

### Decryption Flow

```
┌─────────────┐
│ Application │
└──────┬──────┘
       │ ciphertext + encrypted_dek
       ▼
┌──────────────────────────────────────────┐
│ Crypto.DecryptData(ciphertext, dek)      │
│                                          │
│ 1. Decrypt DEK with KEK (via KMS)        │
│ 2. Decrypt ciphertext with DEK (AES-GCM) │
│ 3. Return plaintext                      │
└──────────────────────────────────────────┘
       │
       ├─── DEK Decryption ────► ┌─────────────────────┐
       │                          │ KeyManagementService│
       │                          │ (AWS KMS / Vault)   │
       │                          └─────────────────────┘
       │
       └─► plaintext
```

### Password Hashing Flow

```
┌─────────────┐
│ Application │
└──────┬──────┘
       │ password
       ▼
┌──────────────────────────────────────────┐
│ Crypto.HashSecure(password)              │
│                                          │
│ 1. Load pepper from SecretManagement     │
│ 2. Generate random salt                  │
│ 3. Hash: Argon2id(password + pepper)     │
│ 4. Return encoded hash                   │
└──────────────────────────────────────────┘
       │
       ├─── Pepper Retrieval ──► ┌─────────────────────┐
       │                          │SecretManagement     │
       │                          │Service              │
       │                          └─────────────────────┘
       │
       └─► encoded_hash
```

## Design Principles

### 1. Single Responsibility Principle

Each interface has one clear responsibility:
- **KeyManagementService**: Cryptographic operations only
- **SecretManagementService**: Secret storage only
- **Crypto**: Orchestration and high-level operations

### 2. Dependency Injection

All dependencies are explicitly passed:

```go
// Explicit - full control
crypto, err := encx.NewCrypto(ctx, kms, secrets, cfg)

// Implicit - convenience
crypto, err := encx.NewCryptoFromEnv(ctx, kms, secrets)
```

### 3. Interface Segregation

Small, focused interfaces:
- Easier to implement
- Easier to test
- Easier to understand
- Easier to maintain

### 4. Open/Closed Principle

Open for extension (new providers), closed for modification (core logic):

```go
// Add new provider without changing core
type CustomKMS struct { /* ... */ }
func (k *CustomKMS) EncryptDEK(...) { /* ... */ }
func (k *CustomKMS) DecryptDEK(...) { /* ... */ }

crypto, _ := encx.NewCrypto(ctx, customKMS, secrets, cfg)
```

### 5. Fail-Fast Validation

All configuration is validated at initialization:

```go
crypto, err := encx.NewCrypto(ctx, kms, secrets, cfg)
// If this succeeds, all configuration is valid
// No runtime configuration errors
```

## Security Model

### Threat Model

ENCX protects against:

1. **Database Compromise**: Encrypted data remains secure
2. **Application Memory Dump**: DEKs are not cached
3. **Insider Threats**: Encrypted data requires both database and KMS access
4. **Password Database Breach**: Peppered hashes are more resistant to cracking

### Security Layers

```
┌─────────────────────────────────────────────┐
│           Application Layer                 │
│  ┌───────────────────────────────────────┐ │
│  │ Encrypted Data (AES-GCM with DEK)     │ │
│  └───────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
                    │
                    │ Requires DEK
                    ▼
┌─────────────────────────────────────────────┐
│           Database Layer                    │
│  ┌───────────────────────────────────────┐ │
│  │ Encrypted DEK (Encrypted with KEK)    │ │
│  └───────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
                    │
                    │ Requires KEK
                    ▼
┌─────────────────────────────────────────────┐
│           KMS Layer                         │
│  ┌───────────────────────────────────────┐ │
│  │ KEK (Managed by Cloud Provider)       │ │
│  └───────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
                    │
                    │ Requires IAM/Vault Auth
                    ▼
┌─────────────────────────────────────────────┐
│           Access Control                    │
│  • IAM Policies (AWS)                       │
│  • Vault Policies (HashiCorp)               │
│  • Network Security Groups                  │
│  • Audit Logs                               │
└─────────────────────────────────────────────┘
```

### Key Rotation

ENCX supports KEK rotation without re-encrypting data:

```go
crypto.RotateKEK(ctx)
```

**Process**:
1. Create new KEK version in KMS
2. Update metadata database with new version
3. New DEKs encrypted with new KEK
4. Old DEKs still decryptable with old KEK versions
5. No data re-encryption required

### Pepper Security

Peppers are protected by:
- Cloud provider's secret management service
- IAM/Vault access policies
- Audit logging
- Automatic backup and replication
- Encryption at rest and in transit

## Migration from v0.5.x

The architecture change from v0.5.x to v0.6.0+ involved:

| Component | v0.5.x | v0.6.0+ |
|-----------|--------|---------|
| **Pepper Storage** | Filesystem | Cloud Secret Store |
| **API** | Functional options | Explicit dependency injection |
| **Interfaces** | Single KMS interface | Separated KMS + SecretManagement |
| **Configuration** | Environment-only | Config struct + Environment |
| **Initialization** | `NewCrypto(ctx, options...)` | `NewCrypto(ctx, kms, secrets, cfg)` |

See [Migration Guide](./MIGRATION_GUIDE.md) for detailed upgrade instructions.

## Best Practices

### Production Deployment

1. **Use IAM Roles**: Don't use access keys in production
2. **Separate Environments**: Different KEK and PepperAlias per environment
3. **Monitor KMS Usage**: Set up CloudWatch alarms for unusual activity
4. **Enable Audit Logging**: AWS CloudTrail or Vault audit logs
5. **Use VPC Endpoints**: Keep KMS traffic within AWS network
6. **Test Disaster Recovery**: Verify backup/restore procedures

### Testing Strategy

```go
// Unit tests - use mocks
kms := encx.NewSimpleTestKMS()
secrets := encx.NewInMemorySecretStore()
crypto, _ := encx.NewCrypto(ctx, kms, secrets, cfg)

// Integration tests - use real services
kms, _ := aws.NewKMSService(ctx, awsCfg)
secrets, _ := aws.NewSecretsManagerStore(ctx, awsCfg)
crypto, _ := encx.NewCrypto(ctx, kms, secrets, cfg)
```

### Configuration Management

```go
// Library code - explicit config
cfg := encx.Config{
    KEKAlias:    kekAlias,
    PepperAlias: pepperAlias,
}
crypto, _ := encx.NewCrypto(ctx, kms, secrets, cfg)

// Application code - environment config
crypto, _ := encx.NewCryptoFromEnv(ctx, kms, secrets)
```

## Related Documentation

- [API Reference](./API.md) - Complete API documentation
- [Security Guide](./SECURITY.md) - Security best practices
- [Migration Guide](./MIGRATION_GUIDE.md) - Upgrading from v0.5.x
- [AWS Provider](../providers/aws/README.md) - AWS-specific documentation
- [HashiCorp Provider](../providers/hashicorp/README.md) - Vault-specific documentation
