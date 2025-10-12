# AWS KMS Provider for ENCX

AWS Key Management Service (KMS) provider for the encx encryption library.

## Overview

This provider implements the `KeyManagementService` interface using AWS KMS, enabling secure key encryption operations (KEK management) for your encx-based applications.

## Prerequisites

### 1. AWS Account Setup

- AWS account with KMS access
- AWS credentials configured
- KMS key created in your desired region

### 2. AWS Credentials

Configure credentials using one of these methods:

**Environment Variables**:
```bash
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"
```

**AWS Config File** (`~/.aws/credentials`):
```ini
[default]
aws_access_key_id = your-access-key
aws_secret_access_key = your-secret-key
region = us-east-1
```

**IAM Role** (recommended for EC2/ECS/Lambda):
```bash
# Automatically uses instance/task/function IAM role
# No credentials needed in code or config files
```

### 3. IAM Permissions

Your AWS credentials/role must have these KMS permissions:

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
    }
  ]
}
```

Optional permissions (if creating keys programmatically):
```json
{
  "Effect": "Allow",
  "Action": [
    "kms:CreateKey",
    "kms:CreateAlias"
  ],
  "Resource": "*"
}
```

## Creating a KMS Key

### Using AWS Console

1. Go to AWS KMS Console
2. Click "Create key"
3. Choose "Symmetric" key type
4. Key usage: "Encrypt and decrypt"
5. Add alias: `alias/my-encryption-key`
6. Define key administrators
7. Define key users (the IAM role/user that will use encx)

### Using AWS CLI

```bash
# Create the key
aws kms create-key \
  --description "ENCX encryption key" \
  --key-usage ENCRYPT_DECRYPT \
  --key-spec SYMMETRIC_DEFAULT

# Output will include KeyId, save it

# Create an alias (recommended)
aws kms create-alias \
  --alias-name alias/my-encryption-key \
  --target-key-id <KEY-ID-from-above>
```

### Using Terraform

```hcl
resource "aws_kms_key" "encx" {
  description             = "ENCX encryption key"
  deletion_window_in_days = 30
  enable_key_rotation     = true
}

resource "aws_kms_alias" "encx" {
  name          = "alias/my-encryption-key"
  target_key_id = aws_kms_key.encx.key_id
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
    "github.com/hengadev/encx/providers/aws"
)

func main() {
    ctx := context.Background()

    // Create AWS KMS provider
    kmsService, err := aws.NewKMSService(ctx, aws.Config{
        Region: "us-east-1",
    })
    if err != nil {
        log.Fatalf("Failed to create KMS service: %v", err)
    }

    // Get pepper from secure storage (e.g., AWS Secrets Manager)
    pepper := []byte("your-pepper-exactly-32-bytes-OK!")

    // Create encx crypto service
    crypto, err := encx.NewCrypto(ctx,
        encx.WithKMSService(kmsService),
        encx.WithKEKAlias("alias/my-encryption-key"),
        encx.WithPepper(pepper),
    )
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

### With Custom AWS Config

```go
import (
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/hengadev/encx/providers/aws"
)

// Load custom AWS config
awsCfg, err := config.LoadDefaultConfig(ctx,
    config.WithRegion("us-west-2"),
    config.WithRetryMaxAttempts(3),
)
if err != nil {
    log.Fatal(err)
}

// Use custom config
kmsService, err := aws.NewKMSService(ctx, aws.Config{
    AWSConfig: &awsCfg,
})
```

### Production Setup

```go
package main

import (
    "context"
    "database/sql"
    "log"
    "os"

    _ "github.com/lib/pq"
    "github.com/hengadev/encx"
    "github.com/hengadev/encx/providers/aws"
)

func main() {
    ctx := context.Background()

    // AWS KMS provider
    kmsService, err := aws.NewKMSService(ctx, aws.Config{
        // Region from environment or AWS config
    })
    if err != nil {
        log.Fatal(err)
    }

    // Database for key versioning
    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Get pepper from AWS Secrets Manager (example)
    pepper, err := getPepperFromSecretsManager(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Create crypto service
    crypto, err := encx.NewCrypto(ctx,
        encx.WithKMSService(kmsService),
        encx.WithKEKAlias(os.Getenv("KMS_KEY_ALIAS")),
        encx.WithPepper(pepper),
        encx.WithDatabase(db),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Your application logic here
}
```

## Key Alias vs Key ID

**Recommended: Use Key Aliases**
```go
// Good: Using alias
encx.WithKEKAlias("alias/my-encryption-key")

// Works but not recommended: Using key ID directly
encx.WithKEKAlias("1234abcd-12ab-34cd-56ef-1234567890ab")
```

**Why use aliases?**
- Easier key rotation (update alias target, not application code)
- More readable and maintainable
- Supports multiple environments (dev/staging/prod with same alias name)

## Multi-Region Setup

For multi-region applications:

**Option 1: Separate keys per region**
```go
// US East
kmsUSEast, _ := aws.NewKMSService(ctx, aws.Config{Region: "us-east-1"})
cryptoUSEast, _ := encx.NewCrypto(ctx,
    encx.WithKMSService(kmsUSEast),
    encx.WithKEKAlias("alias/my-key"),
    ...
)

// EU West
kmsEUWest, _ := aws.NewKMSService(ctx, aws.Config{Region: "eu-west-1"})
cryptoEUWest, _ := encx.NewCrypto(ctx,
    encx.WithKMSService(kmsEUWest),
    encx.WithKEKAlias("alias/my-key"),
    ...
)
```

**Option 2: Multi-region KMS keys** (Automatic replication)
```bash
# Create multi-region key
aws kms create-key --multi-region --region us-east-1

# Replicate to other regions
aws kms replicate-key \
  --key-id <primary-key-id> \
  --replica-region eu-west-1
```

## Performance Optimization

### DEK Caching

For high-throughput applications, cache DEKs to reduce KMS API calls:

```go
type DEKCache struct {
    mu sync.RWMutex
    cache map[string][]byte
    ttl time.Duration
}

func (c *DEKCache) GetOrGenerate(ctx context.Context, crypto *encx.Crypto, recordID string) ([]byte, error) {
    c.mu.RLock()
    if dek, ok := c.cache[recordID]; ok {
        c.mu.RUnlock()
        return dek, nil
    }
    c.mu.RUnlock()

    // Generate new DEK
    dek, err := crypto.GenerateDEK()
    if err != nil {
        return nil, err
    }

    c.mu.Lock()
    c.cache[recordID] = dek
    c.mu.Unlock()

    return dek, nil
}
```

### Connection Pooling

The AWS SDK automatically handles connection pooling. For custom tuning:

```go
import "github.com/aws/aws-sdk-go-v2/aws/retry"

awsCfg, _ := config.LoadDefaultConfig(ctx,
    config.WithRetryer(func() aws.Retryer {
        return retry.AddWithMaxAttempts(retry.NewStandard(), 3)
    }),
)
```

## Error Handling

```go
ciphertext, err := crypto.EncryptData(ctx, plaintext, dek)
if err != nil {
    switch {
    case errors.Is(err, encx.ErrKMSUnavailable):
        // KMS service unavailable - retry with backoff
        log.Println("KMS unavailable, retrying...")
    case errors.Is(err, encx.ErrEncryptionFailed):
        // Encryption failed - check key permissions
        log.Println("Encryption failed, check KMS permissions")
    default:
        // Other error
        log.Printf("Unexpected error: %v", err)
    }
}
```

## Cost Optimization

AWS KMS pricing (as of 2024):
- **Key storage**: ~$1/month per key
- **API requests**: $0.03 per 10,000 requests

**Cost reduction strategies:**
1. Cache DEKs (reduces API calls)
2. Use single key for multiple applications (if appropriate)
3. Batch operations when possible
4. Monitor usage with CloudWatch

Example: Encrypting 1M records/day
- Without caching: 1M encryptions = ~$3/day = $90/month
- With caching (1 DEK per 1000 records): 1K encryptions = ~$0.003/day = ~$0.09/month

## Troubleshooting

### Access Denied Errors

```
Error: AccessDeniedException: User is not authorized to perform: kms:Encrypt
```

**Solution**: Add KMS permissions to your IAM role/user (see IAM Permissions section)

### Key Not Found

```
Error: NotFoundException: Key 'alias/my-key' does not exist
```

**Solutions**:
- Verify key exists: `aws kms describe-key --key-id alias/my-key`
- Check you're in the correct region
- Verify alias name is correct (includes "alias/" prefix)

### Region Mismatch

```
Error: The key ARN is from a different region
```

**Solution**: Ensure KMS provider region matches your key region:
```go
kmsService, _ := aws.NewKMSService(ctx, aws.Config{
    Region: "us-east-1", // Must match key region
})
```

## Security Best Practices

1. **Use IAM roles** instead of access keys when possible
2. **Enable key rotation** in KMS console
3. **Use separate keys** for dev/staging/prod
4. **Monitor key usage** with CloudWatch
5. **Set key deletion window** (30 days recommended)
6. **Store pepper in AWS Secrets Manager**, not in code
7. **Use VPC endpoints** for KMS to keep traffic private
8. **Enable CloudTrail** logging for audit compliance

## Testing

See `awskms_test.go` for unit tests and examples.

For integration testing with real AWS KMS:
```bash
export AWS_REGION=us-east-1
export TEST_KMS_KEY_ID=alias/test-key
go test -tags=integration ./providers/awskms
```

## References

- [AWS KMS Documentation](https://docs.aws.amazon.com/kms/)
- [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/)
- [ENCX Documentation](../../README.md)
- [Envelope Encryption Pattern](https://docs.aws.amazon.com/encryption-sdk/latest/developer-guide/how-it-works.html)
