# AWS Provider for ENCX

AWS provider for the encx encryption library, implementing both key management and secret storage.

## Overview

The AWS provider includes two services that work together:

1. **KMSService** - Implements `encx.KeyManagementService` using AWS KMS for encryption/decryption operations
2. **SecretsManagerStore** - Implements `encx.SecretManagementService` using AWS Secrets Manager for pepper storage

This separation follows the single responsibility principle: KMS handles cryptographic operations while Secrets Manager handles secret storage.

## Prerequisites

### 1. AWS Account Setup

- AWS account with KMS and Secrets Manager access
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

Your AWS credentials/role must have these permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "KMSPermissions",
      "Effect": "Allow",
      "Action": [
        "kms:Encrypt",
        "kms:Decrypt",
        "kms:DescribeKey"
      ],
      "Resource": "arn:aws:kms:REGION:ACCOUNT:key/KEY-ID"
    },
    {
      "Sid": "SecretsManagerPermissions",
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue",
        "secretsmanager:CreateSecret",
        "secretsmanager:PutSecretValue",
        "secretsmanager:DescribeSecret"
      ],
      "Resource": "arn:aws:secretsmanager:REGION:ACCOUNT:secret:encx/*"
    }
  ]
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

    // Initialize KMS for cryptographic operations
    kms, err := aws.NewKMSService(ctx, aws.Config{
        Region: "us-east-1",
    })
    if err != nil {
        log.Fatalf("Failed to create KMS service: %v", err)
    }

    // Initialize Secrets Manager for pepper storage
    secrets, err := aws.NewSecretsManagerStore(ctx, aws.Config{
        Region: "us-east-1",
    })
    if err != nil {
        log.Fatalf("Failed to create Secrets Manager store: %v", err)
    }

    // Create explicit configuration
    cfg := encx.Config{
        KEKAlias:    "alias/my-encryption-key",  // KMS key identifier
        PepperAlias: "my-app-service",           // Service identifier
    }

    // Create encx crypto service
    crypto, err := encx.NewCrypto(ctx, kms, secrets, cfg)
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

    "github.com/hengadev/encx"
    "github.com/hengadev/encx/providers/aws"
)

func main() {
    ctx := context.Background()

    // Set environment variables:
    // export ENCX_KEK_ALIAS="alias/my-encryption-key"
    // export ENCX_PEPPER_ALIAS="my-app-service"

    // Initialize providers
    kms, _ := aws.NewKMSService(ctx, aws.Config{Region: "us-east-1"})
    secrets, _ := aws.NewSecretsManagerStore(ctx, aws.Config{Region: "us-east-1"})

    // Load configuration from environment
    crypto, err := encx.NewCryptoFromEnv(ctx, kms, secrets)
    if err != nil {
        log.Fatalf("Failed to create crypto service: %v", err)
    }

    // Ready to use
    dek, _ := crypto.GenerateDEK()
    // ...
}
```

### Production Setup with Database

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

    // AWS providers
    kms, err := aws.NewKMSService(ctx, aws.Config{
        Region: os.Getenv("AWS_REGION"),
    })
    if err != nil {
        log.Fatal(err)
    }

    secrets, err := aws.NewSecretsManagerStore(ctx, aws.Config{
        Region: os.Getenv("AWS_REGION"),
    })
    if err != nil {
        log.Fatal(err)
    }

    // Load configuration from environment
    crypto, err := encx.NewCryptoFromEnv(ctx, kms, secrets)
    if err != nil {
        log.Fatal(err)
    }

    // Your application logic here
    log.Println("Crypto service initialized successfully")
}
```

## Pepper Storage

The `SecretsManagerStore` automatically manages pepper storage in AWS Secrets Manager:

### Storage Path

Peppers are stored at: `encx/{PepperAlias}/pepper`

For example:
- PepperAlias: `my-app-service` → Secret path: `encx/my-app-service/pepper`
- PepperAlias: `payment-service` → Secret path: `encx/payment-service/pepper`

### Automatic Pepper Management

The first time you initialize crypto with a new `PepperAlias`:
1. ENCX checks if pepper exists in Secrets Manager
2. If not found, generates a secure random 32-byte pepper
3. Stores it in Secrets Manager at `encx/{PepperAlias}/pepper`
4. Subsequent initializations load the existing pepper

### Manual Pepper Inspection

```bash
# View pepper (requires IAM permissions)
aws secretsmanager get-secret-value \
  --secret-id encx/my-app-service/pepper \
  --region us-east-1

# List all encx peppers
aws secretsmanager list-secrets \
  --filters Key=name,Values=encx/ \
  --region us-east-1
```

## Key Alias vs Key ID

**Recommended: Use Key Aliases**
```go
// Good: Using alias
cfg := encx.Config{
    KEKAlias: "alias/my-encryption-key",
    // ...
}

// Works but not recommended: Using key ID directly
cfg := encx.Config{
    KEKAlias: "1234abcd-12ab-34cd-56ef-1234567890ab",
    // ...
}
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
secretsUSEast, _ := aws.NewSecretsManagerStore(ctx, aws.Config{Region: "us-east-1"})
cryptoUSEast, _ := encx.NewCrypto(ctx, kmsUSEast, secretsUSEast, cfg)

// EU West
kmsEUWest, _ := aws.NewKMSService(ctx, aws.Config{Region: "eu-west-1"})
secretsEUWest, _ := aws.NewSecretsManagerStore(ctx, aws.Config{Region: "eu-west-1"})
cryptoEUWest, _ := encx.NewCrypto(ctx, kmsEUWest, secretsEUWest, cfg)
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

## Error Handling

```go
crypto, err := encx.NewCrypto(ctx, kms, secrets, cfg)
if err != nil {
    switch {
    case errors.Is(err, encx.ErrKMSUnavailable):
        // KMS service unavailable - retry with backoff
        log.Println("KMS unavailable, retrying...")
    case errors.Is(err, encx.ErrSecretStorageUnavailable):
        // Secrets Manager unavailable
        log.Println("Secrets Manager unavailable")
    case errors.Is(err, encx.ErrInvalidConfiguration):
        // Configuration validation failed
        log.Printf("Invalid configuration: %v", err)
    default:
        // Other error
        log.Printf("Unexpected error: %v", err)
    }
}
```

## Cost Optimization

### AWS KMS Pricing (as of 2024)
- **Key storage**: ~$1/month per key
- **API requests**: $0.03 per 10,000 requests

### AWS Secrets Manager Pricing
- **Secret storage**: $0.40/month per secret
- **API requests**: $0.05 per 10,000 requests

**Cost reduction strategies:**
1. Cache DEKs (reduces KMS API calls)
2. Use single key for multiple applications (if appropriate)
3. Share peppers across environments using different PepperAlias values
4. Monitor usage with CloudWatch

## Troubleshooting

### Access Denied - KMS

```
Error: AccessDeniedException: User is not authorized to perform: kms:Encrypt
```

**Solution**: Add KMS permissions to your IAM role/user (see IAM Permissions section)

### Access Denied - Secrets Manager

```
Error: AccessDeniedException: User is not authorized to perform: secretsmanager:GetSecretValue
```

**Solution**: Add Secrets Manager permissions to your IAM role/user

### Key Not Found

```
Error: NotFoundException: Key 'alias/my-key' does not exist
```

**Solutions**:
- Verify key exists: `aws kms describe-key --key-id alias/my-key`
- Check you're in the correct region
- Verify alias name is correct (includes "alias/" prefix)

### Secret Not Found (First Run)

This is normal! On first run, ENCX will automatically create the pepper secret. If you see errors:
- Check Secrets Manager permissions (CreateSecret, PutSecretValue)
- Verify the secret name doesn't already exist with different ownership

### Region Mismatch

```
Error: The key ARN is from a different region
```

**Solution**: Ensure KMS and Secrets Manager regions match:
```go
kms, _ := aws.NewKMSService(ctx, aws.Config{Region: "us-east-1"})
secrets, _ := aws.NewSecretsManagerStore(ctx, aws.Config{Region: "us-east-1"})
```

## Security Best Practices

1. **Use IAM roles** instead of access keys when possible
2. **Enable key rotation** in KMS console (automatic annual rotation)
3. **Use separate keys and peppers** for dev/staging/prod
4. **Monitor key usage** with CloudWatch and AWS CloudTrail
5. **Set key deletion window** (30 days recommended)
6. **Use unique PepperAlias** for each service/environment
7. **Use VPC endpoints** for KMS and Secrets Manager to keep traffic private
8. **Enable CloudTrail** logging for audit compliance
9. **Restrict Secrets Manager access** to specific secret paths (encx/*)

## Testing

For unit tests with mock services:
```go
func TestEncryption(t *testing.T) {
    crypto, _ := encx.NewTestCrypto(t)
    // Test your encryption logic
}
```

For integration testing with real AWS services:
```bash
export AWS_REGION=us-east-1
export ENCX_KEK_ALIAS=alias/test-key
export ENCX_PEPPER_ALIAS=test-service
go test -tags=integration ./providers/aws
```

## References

- [AWS KMS Documentation](https://docs.aws.amazon.com/kms/)
- [AWS Secrets Manager Documentation](https://docs.aws.amazon.com/secretsmanager/)
- [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/)
- [ENCX Documentation](../../README.md)
- [Envelope Encryption Pattern](https://docs.aws.amazon.com/encryption-sdk/latest/developer-guide/how-it-works.html)
