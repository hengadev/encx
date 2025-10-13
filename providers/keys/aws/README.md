# AWS KMS Provider for encx

AWS Key Management Service (KMS) implementation of the `KeyManagementService` interface for encx.

## Overview

This provider enables encx to use AWS KMS for managing Key Encryption Keys (KEKs) and performing DEK encryption/decryption operations. AWS KMS provides hardware security modules (HSMs) for cryptographic operations with automatic key rotation and comprehensive audit logging via CloudTrail.

## Features

- **KEK Management**: Create and manage encryption keys using AWS KMS
- **DEK Encryption/Decryption**: Encrypt and decrypt Data Encryption Keys using AWS-managed KEKs
- **Key Aliases**: Support for friendly key names via KMS aliases
- **Multi-Region**: Support for AWS regions (single-region keys by default)
- **Automatic Base64 Encoding**: Ciphertext is base64-encoded for storage compatibility
- **IAM Integration**: Fine-grained access control via IAM policies

## Installation

```bash
go get github.com/hengadev/encx
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/kms
```

## Configuration

### Option 1: Region-Based Configuration

```go
import (
    "context"
    awskms "github.com/hengadev/encx/providers/keys/aws"
)

kms, err := awskms.NewKMSService(ctx, awskms.Config{
    Region: "us-east-1",
})
```

### Option 2: Default AWS Configuration

```go
// Uses AWS_REGION environment variable or ~/.aws/config
kms, err := awskms.NewKMSService(ctx, awskms.Config{})
```

### Option 3: Custom AWS Config

```go
import (
    "github.com/aws/aws-sdk-go-v2/config"
    awskms "github.com/hengadev/encx/providers/keys/aws"
)

awsCfg, err := config.LoadDefaultConfig(ctx,
    config.WithRegion("us-east-1"),
    config.WithSharedConfigProfile("my-profile"),
)

kms, err := awskms.NewKMSService(ctx, awskms.Config{
    AWSConfig: &awsCfg,
})
```

## Usage with encx

AWS KMS is a KeyManagementService implementation and must be paired with a SecretManagementService.

### With AWS Secrets Manager

```go
import (
    "github.com/hengadev/encx"
    awskms "github.com/hengadev/encx/providers/keys/aws"
    awssecrets "github.com/hengadev/encx/providers/secrets/aws"
)

// Initialize providers
kms, err := awskms.NewKMSService(ctx, awskms.Config{Region: "us-east-1"})
secrets, err := awssecrets.NewSecretsManagerStore(ctx, awssecrets.Config{Region: "us-east-1"})

// Create encx.Crypto instance
crypto, err := encx.NewCrypto(ctx, kms, secrets, encx.Config{
    KEKAlias:    "alias/my-app-kek",
    PepperAlias: "my-app-pepper",
})

// Encrypt data
encrypted, err := crypto.Encrypt([]byte("sensitive data"))

// Decrypt data
decrypted, err := crypto.Decrypt(encrypted)
```

### With HashiCorp Vault KV (Mix-and-Match)

```go
import (
    "github.com/hengadev/encx"
    awskms "github.com/hengadev/encx/providers/keys/aws"
    vaultkv "github.com/hengadev/encx/providers/secrets/hashicorp"
)

// AWS KMS for key encryption, Vault KV for pepper storage
kms, err := awskms.NewKMSService(ctx, awskms.Config{Region: "us-east-1"})
secrets, err := vaultkv.NewKVStore(vaultkv.Config{
    Address: "https://vault.example.com",
    Token:   os.Getenv("VAULT_TOKEN"),
})

crypto, err := encx.NewCrypto(ctx, kms, secrets, encx.Config{
    KEKAlias:    "alias/my-app-kek",
    PepperAlias: "my-app-pepper",
})
```

## IAM Permissions

Your IAM role or user needs the following permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "EncxKMSOperations",
            "Effect": "Allow",
            "Action": [
                "kms:Encrypt",
                "kms:Decrypt",
                "kms:DescribeKey"
            ],
            "Resource": "arn:aws:kms:us-east-1:123456789012:key/*"
        },
        {
            "Sid": "EncxKMSKeyCreation",
            "Effect": "Allow",
            "Action": [
                "kms:CreateKey",
                "kms:CreateAlias"
            ],
            "Resource": "*"
        }
    ]
}
```

### Minimal Permissions (Read-Only)

If keys are pre-created, you only need:

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
            "Resource": "arn:aws:kms:us-east-1:123456789012:key/*"
        }
    ]
}
```

## Key Management

### Creating a KMS Key

Via AWS CLI:

```bash
# Create KMS key
aws kms create-key \
    --description "encx KEK for my-app" \
    --key-usage ENCRYPT_DECRYPT \
    --region us-east-1

# Create alias
aws kms create-alias \
    --alias-name alias/my-app-kek \
    --target-key-id <key-id> \
    --region us-east-1
```

Via encx API:

```go
keyID, err := kms.CreateKey(ctx, "encx KEK for my-app")
// Note: You'll need to create an alias separately via AWS CLI or Console
```

### Key Aliases

AWS KMS uses aliases with the `alias/` prefix. This provider automatically adds the prefix if not provided:

```go
// These are equivalent:
kms.GetKeyID(ctx, "my-app-kek")        // Becomes "alias/my-app-kek"
kms.GetKeyID(ctx, "alias/my-app-kek")  // Used as-is
```

## API Reference

### NewKMSService

```go
func NewKMSService(ctx context.Context, cfg Config) (*KMSService, error)
```

Creates a new AWS KMS service instance.

**Parameters:**
- `ctx`: Context for AWS SDK operations
- `cfg`: Configuration with Region or AWSConfig

**Returns:**
- `*KMSService`: Initialized KMS service
- `error`: Error if AWS configuration fails

### GetKeyID

```go
func (k *KMSService) GetKeyID(ctx context.Context, alias string) (string, error)
```

Retrieves the key ID for a given alias.

**Parameters:**
- `alias`: Key alias (with or without "alias/" prefix)

**Returns:**
- `string`: KMS key ID or ARN
- `error`: `encx.ErrKMSUnavailable` if key not found

### CreateKey

```go
func (k *KMSService) CreateKey(ctx context.Context, description string) (string, error)
```

Creates a new symmetric encryption key in AWS KMS.

**Parameters:**
- `description`: Human-readable key description

**Returns:**
- `string`: KMS key ID
- `error`: `encx.ErrKMSUnavailable` if creation fails

### EncryptDEK

```go
func (k *KMSService) EncryptDEK(ctx context.Context, keyID string, plaintext []byte) ([]byte, error)
```

Encrypts a Data Encryption Key using AWS KMS.

**Parameters:**
- `keyID`: Key ID, ARN, alias, or alias ARN
- `plaintext`: DEK to encrypt (typically 32 bytes)

**Returns:**
- `[]byte`: Base64-encoded ciphertext
- `error`: `encx.ErrEncryptionFailed` if encryption fails

### DecryptDEK

```go
func (k *KMSService) DecryptDEK(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error)
```

Decrypts a Data Encryption Key encrypted by AWS KMS.

**Parameters:**
- `keyID`: Optional key ID (AWS KMS uses ciphertext metadata)
- `ciphertext`: Base64-encoded ciphertext from EncryptDEK

**Returns:**
- `[]byte`: Decrypted DEK plaintext
- `error`: `encx.ErrDecryptionFailed` if decryption fails

### Region

```go
func (k *KMSService) Region() string
```

Returns the AWS region the KMS service is configured for.

## Error Handling

All operations return wrapped errors from the `encx` package:

```go
encrypted, err := kms.EncryptDEK(ctx, keyID, dek)
if err != nil {
    switch {
    case errors.Is(err, encx.ErrKMSUnavailable):
        // Handle KMS unavailability or key not found
    case errors.Is(err, encx.ErrEncryptionFailed):
        // Handle encryption failure
    case errors.Is(err, encx.ErrInvalidConfiguration):
        // Handle configuration error
    }
}
```

## Environment Variables

AWS SDK respects standard AWS environment variables:

- `AWS_REGION`: Default region
- `AWS_PROFILE`: AWS CLI profile name
- `AWS_ACCESS_KEY_ID`: AWS access key
- `AWS_SECRET_ACCESS_KEY`: AWS secret key
- `AWS_SESSION_TOKEN`: Session token for temporary credentials

## Testing

For unit tests without AWS dependencies, use encx's test utilities:

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

For integration tests with actual AWS KMS, see `test/integration/kms_providers/aws_kms_integration_test.go`.

## Best Practices

1. **Use Key Aliases**: Prefer aliases over key IDs for better maintainability
2. **IAM Permissions**: Follow principle of least privilege
3. **Key Rotation**: Enable automatic key rotation in AWS KMS
4. **CloudTrail Logging**: Enable CloudTrail for audit logging
5. **Multi-Region**: Use multi-region keys for disaster recovery if needed
6. **Caching**: Consider caching decrypted DEKs at the application level for performance
7. **Error Handling**: Always check for `encx.ErrKMSUnavailable` to handle transient failures

## Cost Considerations

AWS KMS charges per API call:
- Encrypt/Decrypt: $0.03 per 10,000 requests
- Key storage: $1 per key per month

For high-volume applications, consider:
- Caching decrypted DEKs
- Using longer-lived DEKs
- Batching encryption operations where possible

## Troubleshooting

### "KMS key not found"

Ensure the key alias exists and is correctly formatted:

```bash
aws kms describe-key --key-id alias/my-app-kek --region us-east-1
```

### "AccessDeniedException"

Verify IAM permissions include required KMS actions on the target key.

### "Region not configured"

Set `AWS_REGION` environment variable or specify `Region` in `Config`.

## Related Providers

- **AWS Secrets Manager** (`providers/secrets/aws`): Pair with AWS KMS for full AWS integration
- **Vault KV** (`providers/secrets/hashicorp`): Use Vault for pepper storage with AWS KMS

## Documentation

- [encx Documentation](https://github.com/hengadev/encx)
- [AWS KMS Documentation](https://docs.aws.amazon.com/kms/)
- [encx API Reference](../../docs/API.md)
- [encx Architecture](../../docs/ARCHITECTURE.md)

## License

See the main encx repository for license information.
