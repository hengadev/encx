# Security Guide

This document provides comprehensive security guidance for using and deploying encx in production environments.

## Table of Contents

1. [Security Architecture](#security-architecture)
2. [Cryptographic Implementation](#cryptographic-implementation)
3. [Secret Management](#secret-management)
4. [Security Audit Results](#security-audit-results)
5. [Best Practices](#best-practices)
6. [Threat Model](#threat-model)
7. [Incident Response](#incident-response)

---

## Security Architecture

### Encryption Model

encx uses envelope encryption with a two-tier key hierarchy:

```
┌─────────────────────────────────────────────────────────┐
│                    Key Hierarchy                         │
├─────────────────────────────────────────────────────────┤
│                                                           │
│  KEK (Key Encryption Key)                                │
│  └─ Managed by KMS (AWS KMS, HashiCorp Vault)           │
│  └─ Never leaves KMS                                     │
│  └─ Used to encrypt/decrypt DEKs                         │
│                                                           │
│  DEK (Data Encryption Key)                               │
│  └─ Generated per record/user                            │
│  └─ 256-bit AES key (32 bytes)                          │
│  └─ Stored encrypted in your database                    │
│  └─ Used to encrypt actual data                          │
│                                                           │
│  Pepper (Application Secret)                             │
│  └─ 256-bit secret (32 bytes)                           │
│  └─ Mixed into hashing operations                        │
│  └─ Stored in environment/secrets manager                │
│  └─ Should be rotated periodically                       │
│                                                           │
└─────────────────────────────────────────────────────────┘
```

### Data Flow

**Encryption Flow:**
```
1. Plaintext data → 2. Generate DEK → 3. Encrypt with DEK
                         ↓
                    Encrypt DEK with KEK
                         ↓
                    Store encrypted DEK + encrypted data
```

**Decryption Flow:**
```
1. Retrieve encrypted DEK → 2. Decrypt DEK with KEK → 3. Decrypt data with DEK
```

---

## Cryptographic Implementation

### Algorithms and Standards

| Operation | Algorithm | Key Size | Notes |
|-----------|-----------|----------|-------|
| **Symmetric Encryption** | AES-256-GCM | 256 bits | NIST approved, authenticated encryption |
| **Key Generation** | crypto/rand | 256 bits | Cryptographically secure RNG |
| **Password Hashing** | Argon2id | 256 bits | Winner of Password Hashing Competition |
| **Basic Hashing** | SHA-256 | 256 bits | For searchable hashes, not passwords |
| **Nonce Generation** | crypto/rand | 96 bits | Never reused with same key |
| **Salt Generation** | crypto/rand | 128 bits | Unique per hash operation |

### AES-GCM Configuration

```go
// Nonce: 12 bytes (96 bits) - recommended GCM nonce size
// Key: 32 bytes (256 bits) - AES-256
// Auth Tag: 16 bytes (128 bits) - maximum GCM tag size

// Implementation (internal/crypto/encryption.go)
cipher, _ := aes.NewCipher(dek)  // 256-bit key
gcm, _ := cipher.NewGCM()        // 128-bit auth tag
nonce := make([]byte, 12)        // 96-bit nonce
rand.Read(nonce)                  // Cryptographically secure random
```

**Security Properties:**
- ✅ Confidentiality (AES-256)
- ✅ Authenticity (GCM authentication tag)
- ✅ Integrity (GCM detects tampering)
- ✅ Unique nonces (randomly generated per encryption)

### Argon2id Configuration

```go
// Default parameters (argon2params.go)
DefaultArgon2Params = &Argon2Params{
    Memory:      64 * 1024,  // 64 MB
    Iterations:  3,          // Time cost
    Parallelism: 2,          // Threads
    SaltLength:  16,         // 128 bits
    KeyLength:   32,         // 256 bits
}
```

**Security Analysis:**
- ✅ Resistant to GPU attacks (memory-hard)
- ✅ Resistant to side-channel attacks (data-independent)
- ✅ OWASP recommended parameters exceeded
- ✅ Unique salt per hash

**Performance vs Security:**
- Current: ~30ms per hash on modern CPU
- Adjust `Memory` and `Iterations` based on threat model
- Higher values = slower but more secure

### Random Number Generation

**All cryptographic randomness uses `crypto/rand`:**

```go
// DEK Generation (internal/crypto/dek.go:39)
dek := make([]byte, 32)
io.ReadFull(rand.Reader, dek)  // ✅ crypto/rand

// Nonce Generation (internal/crypto/encryption.go:31)
nonce := make([]byte, 12)
io.ReadFull(rand.Reader, nonce)  // ✅ crypto/rand

// Salt Generation (internal/crypto/hashing.go:61)
salt := make([]byte, 16)
io.ReadFull(rand.Reader, salt)  // ✅ crypto/rand
```

**Never uses `math/rand` for cryptographic purposes.**

Note: `math/rand` is used in `internal/reliability/retry.go:115` for retry jitter, which is acceptable as it's not security-sensitive.

### Timing Attack Protection

**Constant-time operations using `crypto/subtle`:**

```go
// Hash comparison (internal/security/timing_protection.go:81)
func SecureCompare(a, b string) bool {
    return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// Memory operations (internal/security/secure_memory.go:116)
result[i] = byte(subtle.ConstantTimeSelect(condition, int(a[i]), int(b[i])))

// Copy operations (internal/security/secure_memory.go:127)
subtle.ConstantTimeCopy(1, dst, src)
```

**Protection against:**
- ✅ Timing attacks on hash comparison
- ✅ Branch prediction attacks
- ✅ Cache timing attacks

---

## Secret Management

### Pepper Management

The pepper is a 32-byte application-level secret used in hashing operations.

#### Storage Options

**1. Environment Variables (Development)**
```bash
export ENCX_PEPPER="your-pepper-exactly-32-bytes-OK!"
```

**2. AWS Secrets Manager (Production)**
```go
import "github.com/aws/aws-sdk-go-v2/service/secretsmanager"

// Retrieve pepper from Secrets Manager
secretName := "encx/pepper"
result, err := svc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
    SecretId: aws.String(secretName),
})
pepper := []byte(*result.SecretString)
```

**3. HashiCorp Vault (Production)**
```go
import vault "github.com/hashicorp/vault/api"

// Retrieve pepper from Vault
secret, err := client.Logical().Read("secret/data/encx/pepper")
pepper := []byte(secret.Data["data"].(map[string]interface{})["value"].(string))
```

**4. Kubernetes Secrets (Production)**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: encx-secrets
type: Opaque
data:
  pepper: <base64-encoded-32-bytes>
```

```go
// Read from mounted secret
pepper, err := os.ReadFile("/var/secrets/pepper")
```

#### Pepper Generation

```bash
# Generate secure 32-byte pepper
openssl rand -base64 32 | head -c 32

# Or using Python
python3 -c "import secrets; print(secrets.token_urlsafe(32)[:32])"

# Or using Go
go run -c 'package main; import("crypto/rand"; "encoding/base64"; "fmt"); func main() { b := make([]byte, 32); rand.Read(b); fmt.Println(base64.StdEncoding.EncodeToString(b)[:32]) }'
```

#### Pepper Rotation

**Strategy:**
1. Generate new pepper
2. Store both old and new peppers temporarily
3. Re-hash all secure hashes with new pepper
4. Remove old pepper once migration complete

**Implementation:**
```go
// Support multiple peppers during rotation
crypto, err := encx.NewCrypto(ctx,
    encx.WithKMSService(kms),
    encx.WithKEKAlias(kekAlias),
    encx.WithPepper(newPepper),
    // Configure to try old pepper on verification failure
)

// Migration pseudocode
for each user {
    // Verify with old pepper
    if VerifyPassword(password, oldHash, oldPepper) {
        // Re-hash with new pepper
        newHash := HashPassword(password, newPepper)
        UpdateUser(user.ID, newHash)
    }
}
```

### KEK Management

KEKs are managed by your KMS provider and never leave the KMS.

#### AWS KMS Best Practices

```hcl
# Terraform configuration
resource "aws_kms_key" "encx" {
  description             = "encx encryption key"
  deletion_window_in_days = 30
  enable_key_rotation     = true  # ✅ Enable automatic rotation

  tags = {
    Application = "encx"
    Environment = "production"
  }
}

resource "aws_kms_alias" "encx" {
  name          = "alias/encx-production"
  target_key_id = aws_kms_key.encx.key_id
}
```

**IAM Policy (Least Privilege):**
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
      "Resource": "arn:aws:kms:REGION:ACCOUNT:key/KEY-ID",
      "Condition": {
        "StringEquals": {
          "kms:ViaService": "ec2.REGION.amazonaws.com"
        }
      }
    }
  ]
}
```

#### HashiCorp Vault Best Practices

```bash
# Enable transit engine
vault secrets enable transit

# Create encryption key
vault write -f transit/keys/encx-production \
  type=aes256-gcm96 \
  auto_rotate_period=2160h  # 90 days

# Allow key rotation
vault write transit/keys/encx-production/rotate

# Create policy
vault policy write encx-policy - <<EOF
path "transit/encrypt/encx-production" {
  capabilities = ["update"]
}
path "transit/decrypt/encx-production" {
  capabilities = ["update"]
}
path "transit/keys/encx-production" {
  capabilities = ["read"]
}
EOF
```

### DEK Storage

**Never store DEKs in plaintext.**

Database schema:
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email_encrypted BYTEA NOT NULL,      -- Encrypted email
    email_hash VARCHAR(64) NOT NULL,      -- Searchable hash
    password_hash VARCHAR(128) NOT NULL,  -- Argon2id hash
    dek_encrypted BYTEA NOT NULL,         -- ✅ Encrypted DEK
    kek_version INT NOT NULL,             -- KEK version used
    created_at TIMESTAMP DEFAULT NOW()
);

-- Index for searching
CREATE INDEX idx_users_email_hash ON users(email_hash);
```

**Security Properties:**
- ✅ DEK always stored encrypted
- ✅ KEK version tracked for rotation
- ✅ Hash indexed for fast lookup
- ✅ Encrypted data not indexed

---

## Security Audit Results

### Vulnerability Scan (govulncheck)

**Status:** ⚠️ 5 vulnerabilities found in Go 1.24.2 standard library

**Recommendation:** Upgrade to Go 1.24.6 or later

| Vulnerability | Component | Severity | Fixed In | Impact |
|---------------|-----------|----------|----------|--------|
| GO-2025-3956 | os/exec | Medium | go1.24.6 | Unexpected LookPath results |
| GO-2025-3849 | database/sql | Medium | go1.24.6 | Incorrect Rows.Scan results |
| GO-2025-3751 | net/http | High | go1.24.4 | Sensitive headers leak |
| GO-2025-3750 | os | Low | go1.24.4 | File creation inconsistency |
| GO-2025-3749 | crypto/x509 | Medium | go1.24.4 | Policy validation bypass |

**Action Required:**
```bash
# Update Go version
go get go@1.24.6
go mod tidy
```

### Static Analysis (gosec)

**Status:** 80 issues found (0 critical in core crypto)

#### High Severity Issues

1. **G404: Weak Random in Retry Logic**
   - Location: `internal/reliability/retry.go:115`
   - Finding: Uses `math/rand` for jitter
   - Assessment: ✅ **ACCEPTED** - Non-cryptographic use (retry timing)
   - Rationale: Jitter doesn't require cryptographic security

2. **G108: Profiling Endpoint Exposed**
   - Location: `internal/profiling/profiler.go:7`
   - Finding: pprof automatically exposed on `/debug/pprof`
   - Assessment: ⚠️ **MITIGATED** - Only enabled in development
   - Remediation: Ensure profiling disabled in production

3. **G115: Integer Overflow Conversions**
   - Locations: Multiple files (internal/crypto, internal/serialization)
   - Finding: Potential integer overflow in type conversions
   - Assessment: ✅ **LOW RISK** - Sizes are controlled and validated
   - Context: Converting lengths for serialization

#### Medium Severity Issues

4. **G114: Missing HTTP Timeouts**
   - Location: `examples/s3-streaming-upload/main.go:222`
   - Finding: HTTP server without timeouts
   - Assessment: ⚠️ **EXAMPLE CODE** - Should be fixed for production use
   - Remediation:
   ```go
   server := &http.Server{
       Addr:              ":8080",
       ReadHeaderTimeout: 10 * time.Second,
       ReadTimeout:       30 * time.Second,
       WriteTimeout:      30 * time.Second,
       IdleTimeout:       120 * time.Second,
   }
   ```

5. **G304: Potential File Inclusion**
   - Locations: config.go, generator.go, profiler.go
   - Finding: File operations with variable paths
   - Assessment: ✅ **ACCEPTED** - Paths validated, controlled input
   - Context: Configuration and code generation tools

6. **G104: Unhandled Errors**
   - Locations: Various (templates.go, main.go)
   - Finding: Some errors not explicitly checked
   - Assessment: ⚠️ **MINOR** - Should be fixed for robustness
   - Priority: Low

### Cryptographic Implementation Review

**Status:** ✅ **PASSED**

| Component | Status | Notes |
|-----------|--------|-------|
| Random Number Generation | ✅ | Uses crypto/rand throughout |
| AES-GCM Implementation | ✅ | Correct nonce handling, auth tags |
| Argon2id Parameters | ✅ | Exceeds OWASP recommendations |
| Timing Attack Protection | ✅ | Uses crypto/subtle for comparisons |
| Key Sizes | ✅ | 256-bit keys for all operations |
| Salt Uniqueness | ✅ | Generated per operation |
| Nonce Uniqueness | ✅ | Generated per encryption |

---

## Best Practices

### Development

1. **Never commit secrets**
   ```gitignore
   # .gitignore
   .env
   secrets/
   *.pem
   *.key
   ```

2. **Use test crypto for development**
   ```go
   // Development
   crypto, _ := encx.NewTestCrypto(nil)

   // Production
   crypto, _ := encx.NewCrypto(ctx,
       encx.WithKMSService(kms),
       encx.WithKEKAlias(alias),
       encx.WithPepper(pepper),
   )
   ```

3. **Validate input before encryption**
   ```go
   if len(data) == 0 {
       return errors.New("cannot encrypt empty data")
   }
   if len(data) > MaxDataSize {
       return errors.New("data too large")
   }
   ```

### Deployment

1. **Use TLS for all connections**
   - Database connections
   - KMS API calls
   - Application endpoints

2. **Enable KMS key rotation**
   ```bash
   # AWS KMS
   aws kms enable-key-rotation --key-id <KEY-ID>

   # Vault
   vault write transit/keys/my-key/rotate
   ```

3. **Implement secrets rotation**
   - Rotate pepper every 90 days
   - Track rotation in audit logs
   - Test rotation procedure in staging

4. **Monitor KMS usage**
   ```go
   // Log KMS operations (without sensitive data)
   log.Printf("operation=encrypt_dek kek_version=%d status=success", version)
   ```

### Logging

**DO NOT log:**
- ❌ Plaintext data
- ❌ DEKs (encrypted or decrypted)
- ❌ Pepper
- ❌ Passwords
- ❌ Encrypted values (can leak size)

**DO log:**
- ✅ Operation type
- ✅ Field names
- ✅ Success/failure
- ✅ KEK version used
- ✅ Error types (without sensitive details)

**Example:**
```go
// ❌ BAD
log.Printf("Encrypted email: %s -> %x", email, encrypted)

// ✅ GOOD
log.Printf("operation=encrypt field=email status=success size=%d", len(encrypted))
```

---

## Threat Model

### Threats Mitigated

| Threat | Mitigation | Status |
|--------|------------|--------|
| **Data breach (database dump)** | Envelope encryption, all data encrypted at rest | ✅ |
| **KMS compromise** | KEK rotation, version tracking | ✅ |
| **Timing attacks** | Constant-time comparisons | ✅ |
| **Rainbow table attacks** | Unique salts, Argon2id, pepper | ✅ |
| **Replay attacks** | Unique nonces, GCM auth tags | ✅ |
| **Man-in-the-middle** | TLS required, authenticated encryption | ✅ |
| **Memory dumps** | Secure memory zeroing (configurable) | ⚠️ |
| **Side-channel attacks** | Argon2id (data-independent), constant-time ops | ✅ |

### Residual Risks

| Risk | Severity | Mitigation Strategy |
|------|----------|---------------------|
| **Application compromise** | High | Defense in depth, WAF, monitoring |
| **Insider threat** | Medium | Audit logging, access controls, least privilege |
| **KMS service outage** | Medium | Circuit breakers, caching, failover |
| **Pepper exposure** | High | Secrets management, rotation, monitoring |

### Attack Scenarios

**Scenario 1: Database Breach**
- Attacker gains access to database
- All data encrypted with DEKs
- DEKs encrypted with KEK (in KMS)
- **Result:** Data unreadable without KMS access

**Scenario 2: Application Compromise**
- Attacker gains application access
- Can decrypt data for authorized operations
- Audit logs record all operations
- **Result:** Limited to authorized decryption scope

**Scenario 3: KMS Compromise**
- Attacker gains KMS access
- Can decrypt DEKs
- **Mitigation:** Rotate KEK, re-encrypt all DEKs

---

## Incident Response

### Security Incident Checklist

If you suspect a security incident:

1. **Immediate Actions** (0-1 hour)
   - [ ] Isolate affected systems
   - [ ] Enable enhanced logging
   - [ ] Notify security team
   - [ ] Preserve evidence

2. **Assessment** (1-4 hours)
   - [ ] Determine scope of breach
   - [ ] Identify compromised data
   - [ ] Check audit logs
   - [ ] Review KMS access logs

3. **Containment** (4-24 hours)
   - [ ] Rotate compromised keys/secrets
   - [ ] Revoke compromised credentials
   - [ ] Update firewall rules
   - [ ] Deploy security patches

4. **Recovery** (1-7 days)
   - [ ] Re-encrypt affected data
   - [ ] Restore from clean backup
   - [ ] Update security policies
   - [ ] Conduct post-mortem

### KEK Rotation Procedure

If KEK is compromised:

```bash
# 1. Create new KEK
aws kms create-key --description "encx-new-key"
aws kms create-alias --alias-name alias/encx-new --target-key-id <NEW-KEY-ID>

# 2. Update application to use new KEK for new data
# (Configure dual KEK support temporarily)

# 3. Re-encrypt all DEKs
go run ./scripts/reencrypt-deks --old-kek=old --new-kek=new

# 4. Verify all DEKs re-encrypted

# 5. Remove old KEK from application config

# 6. Schedule old KEK deletion (30-day window)
aws kms schedule-key-deletion --key-id <OLD-KEY-ID> --pending-window-in-days 30
```

### Pepper Rotation Procedure

If pepper is compromised:

```bash
# 1. Generate new pepper
NEW_PEPPER=$(openssl rand -base64 32 | head -c 32)

# 2. Store new pepper in secrets manager
aws secretsmanager create-secret \
  --name encx/pepper-new \
  --secret-string "$NEW_PEPPER"

# 3. Update application to try both peppers
# (Old for verification, new for new hashes)

# 4. Re-hash all secure hashes (requires user re-authentication)
# This requires users to reset passwords or re-authenticate

# 5. After migration period, remove old pepper
```

### Contact Information

**Security Issues:**
- Email: security@example.com (replace with your security contact)
- PGP Key: [Link to PGP key]

**Vulnerability Disclosure:**
- Follow responsible disclosure practices
- Report security vulnerabilities privately
- Expected response time: 48 hours

---

## Compliance

### GDPR

- ✅ Data encryption at rest (Article 32)
- ✅ Pseudonymization support (Article 25)
- ✅ Right to be forgotten (delete user records)
- ✅ Data portability (export decrypted data)

### HIPAA

- ✅ Encryption required (164.312(a)(2)(iv))
- ✅ Integrity controls (164.312(c)(1))
- ✅ Audit controls (164.312(b))
- ✅ Access control (164.312(a)(1))

### PCI DSS

- ✅ Requirement 3: Protect stored cardholder data
- ✅ Requirement 4: Encrypt transmission
- ✅ Requirement 8: Strong cryptography
- ✅ Requirement 10: Track and monitor access

---

## Updates

This security documentation should be reviewed:
- After each security audit
- After dependency updates
- After major feature additions
- At least quarterly

**Last Updated:** 2025-10-05
**Version:** 1.0.0
**Audit Date:** 2025-10-05
