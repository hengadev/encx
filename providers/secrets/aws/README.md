# AWS Secrets Manager Provider for encx

AWS Secrets Manager implementation of the `SecretManagementService` interface for encx.

## Overview

This provider enables encx to use AWS Secrets Manager for secure pepper storage. Peppers are secret values used in Argon2id password hashing to add an additional layer of security beyond salts. AWS Secrets Manager provides centralized secret storage with IAM-based access control, automatic replication, and comprehensive audit logging via CloudTrail.

## Features

- **Secure Pepper Storage**: Store peppers in AWS Secrets Manager with encryption at rest
- **Automatic Secret Management**: Automatically create secrets if they don't exist
- **IAM Access Control**: Fine-grained permissions via IAM policies
- **Audit Logging**: All operations logged in CloudTrail
- **Multi-Region Support**: Optional replication for disaster recovery
- **Secret Versioning**: Automatic versioning of all secret updates
- **Base64 Encoding**: Peppers are base64-encoded for storage compatibility

## Installation

```bash
go get github.com/hengadev/encx
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/secretsmanager
```

## Configuration

### Option 1: Region-Based Configuration

```go
import (
    "context"
    awssecrets "github.com/hengadev/encx/providers/secrets/aws"
)

secrets, err := awssecrets.NewSecretsManagerStore(ctx, awssecrets.Config{
    Region: "us-east-1",
})
```

### Option 2: Default AWS Configuration

```go
// Uses AWS_REGION environment variable or ~/.aws/config
secrets, err := awssecrets.NewSecretsManagerStore(ctx, awssecrets.Config{})
```

### Option 3: Custom AWS Config

```go
import (
    "github.com/aws/aws-sdk-go-v2/config"
    awssecrets "github.com/hengadev/encx/providers/secrets/aws"
)

awsCfg, err := config.LoadDefaultConfig(ctx,
    config.WithRegion("us-east-1"),
    config.WithSharedConfigProfile("my-profile"),
)

secrets, err := awssecrets.NewSecretsManagerStore(ctx, awssecrets.Config{
    AWSConfig: &awsCfg,
})
```

## Usage with encx

AWS Secrets Manager is a SecretManagementService implementation and must be paired with a KeyManagementService.

### With AWS KMS

```go
import (
    "github.com/hengadev/encx"
    awskms "github.com/hengadev/encx/providers/keys/aws"
    awssecrets "github.com/hengadev/encx/providers/secrets/aws"
)

// Initialize providers (using same region for both)
kms, err := awskms.NewKMSService(ctx, awskms.Config{Region: "us-east-1"})
secrets, err := awssecrets.NewSecretsManagerStore(ctx, awssecrets.Config{Region: "us-east-1"})

// Create encx.Crypto instance
crypto, err := encx.NewCrypto(ctx, kms, secrets, encx.Config{
    KEKAlias:    "alias/my-app-kek",
    PepperAlias: "my-app-pepper",
})

// Hash password (uses pepper from Secrets Manager)
hashed, err := crypto.HashPassword("user-password")

// Verify password
valid, err := crypto.VerifyPassword("user-password", hashed)
```

### With HashiCorp Vault Transit (Mix-and-Match)

```go
import (
    "github.com/hengadev/encx"
    vaulttransit "github.com/hengadev/encx/providers/keys/hashicorp"
    awssecrets "github.com/hengadev/encx/providers/secrets/aws"
)

// Vault Transit for key encryption, AWS Secrets Manager for pepper storage
kms, err := vaulttransit.NewTransitService(vaulttransit.Config{
    Address: "https://vault.example.com",
    Token:   os.Getenv("VAULT_TOKEN"),
})
secrets, err := awssecrets.NewSecretsManagerStore(ctx, awssecrets.Config{Region: "us-east-1"})

crypto, err := encx.NewCrypto(ctx, kms, secrets, encx.Config{
    KEKAlias:    "my-app-transit-key",
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
            "Sid": "EncxSecretsManagerOperations",
            "Effect": "Allow",
            "Action": [
                "secretsmanager:GetSecretValue",
                "secretsmanager:DescribeSecret",
                "secretsmanager:CreateSecret",
                "secretsmanager:PutSecretValue"
            ],
            "Resource": "arn:aws:secretsmanager:us-east-1:123456789012:secret:encx/*"
        }
    ]
}
```

### Minimal Permissions (Read-Only)

If peppers are pre-created, you only need:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "secretsmanager:GetSecretValue",
                "secretsmanager:DescribeSecret"
            ],
            "Resource": "arn:aws:secretsmanager:us-east-1:123456789012:secret:encx/*"
        }
    ]
}
```

## Pepper Management

### Automatic Creation

When using `encx.NewCrypto()`, peppers are automatically created if they don't exist:

```go
// If encx/my-app-pepper doesn't exist, it's automatically generated and stored
crypto, err := encx.NewCrypto(ctx, kms, secrets, encx.Config{
    KEKAlias:    "alias/my-app-kek",
    PepperAlias: "my-app-pepper",
})
```

### Manual Creation

You can manually create a pepper via AWS CLI:

```bash
# Generate a 32-byte random pepper
PEPPER=$(openssl rand -base64 32)

# Store in Secrets Manager
aws secretsmanager create-secret \
    --name encx/my-app-pepper \
    --description "ENCX pepper for my-app" \
    --secret-string "$PEPPER" \
    --region us-east-1
```

### Secret Path Format

Peppers are stored using the path format:

```
encx/{alias}/pepper
```

Examples:
- `PepperAlias: "my-app"` → Secret name: `encx/my-app/pepper`
- `PepperAlias: "payment-service"` → Secret name: `encx/payment-service/pepper`

### Checking if Pepper Exists

```go
exists, err := secrets.PepperExists(ctx, "my-app-pepper")
if !exists {
    // Pepper will be auto-created on first NewCrypto() call
}
```

## API Reference

### NewSecretsManagerStore

```go
func NewSecretsManagerStore(ctx context.Context, cfg Config) (*SecretsManagerStore, error)
```

Creates a new AWS Secrets Manager store instance.

**Parameters:**
- `ctx`: Context for AWS SDK operations
- `cfg`: Configuration with Region or AWSConfig

**Returns:**
- `*SecretsManagerStore`: Initialized Secrets Manager store
- `error`: Error if AWS configuration fails

### StorePepper

```go
func (s *SecretsManagerStore) StorePepper(ctx context.Context, alias string, pepper []byte) error
```

Stores a pepper in AWS Secrets Manager. Creates the secret if it doesn't exist, updates if it does.

**Parameters:**
- `alias`: Pepper identifier (used to construct secret name)
- `pepper`: 32-byte pepper value

**Returns:**
- `error`: `encx.ErrSecretStorageUnavailable` if operation fails

### GetPepper

```go
func (s *SecretsManagerStore) GetPepper(ctx context.Context, alias string) ([]byte, error)
```

Retrieves a pepper from AWS Secrets Manager.

**Parameters:**
- `alias`: Pepper identifier

**Returns:**
- `[]byte`: 32-byte pepper value
- `error`: `encx.ErrSecretStorageUnavailable` if pepper not found

### PepperExists

```go
func (s *SecretsManagerStore) PepperExists(ctx context.Context, alias string) (bool, error)
```

Checks if a pepper exists in AWS Secrets Manager.

**Parameters:**
- `alias`: Pepper identifier

**Returns:**
- `bool`: `true` if pepper exists, `false` otherwise
- `error`: Error only for actual failures (not "not found")

### GetStoragePath

```go
func (s *SecretsManagerStore) GetStoragePath(alias string) string
```

Returns the full secret path for a given alias.

**Parameters:**
- `alias`: Pepper identifier

**Returns:**
- `string`: Secret path (e.g., "encx/my-app/pepper")

### Region

```go
func (s *SecretsManagerStore) Region() string
```

Returns the AWS region the store is configured for.

## Error Handling

All operations return wrapped errors from the `encx` package:

```go
pepper, err := secrets.GetPepper(ctx, "my-app-pepper")
if err != nil {
    switch {
    case errors.Is(err, encx.ErrSecretStorageUnavailable):
        // Handle Secrets Manager unavailability or secret not found
    case errors.Is(err, encx.ErrInvalidConfiguration):
        // Handle configuration error (e.g., invalid pepper length)
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

## Multi-Region Replication

For high availability and disaster recovery, replicate peppers to other regions:

```bash
aws secretsmanager replicate-secret-to-regions \
    --secret-id encx/my-app-pepper \
    --add-replica-regions Region=us-west-2 \
    --region us-east-1
```

Then configure your application to use the replica:

```go
// Primary region
secrets, err := awssecrets.NewSecretsManagerStore(ctx, awssecrets.Config{
    Region: "us-east-1",
})

// Failover region
secretsFailover, err := awssecrets.NewSecretsManagerStore(ctx, awssecrets.Config{
    Region: "us-west-2",
})
```

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

    // Test password hashing
    hashed, err := crypto.HashPassword("test-password")
    // ...
}
```

For integration tests with actual AWS Secrets Manager, see `test/integration/kms_providers/aws_kms_integration_test.go`.

## Best Practices

1. **Pepper Immutability**: Never rotate peppers - they must remain constant for password verification
2. **Backup Strategy**: Regularly back up peppers to a secure location
3. **IAM Permissions**: Follow principle of least privilege
4. **Secret Naming**: Use descriptive, service-specific pepper aliases
5. **Multi-Region**: Replicate critical peppers for disaster recovery
6. **Access Logging**: Enable CloudTrail for audit logging
7. **Encryption**: Use KMS encryption for Secrets Manager (enabled by default)
8. **Testing**: Use `encx.NewTestCrypto()` for unit tests to avoid AWS dependencies

## Security Considerations

### Why 32 Bytes?

Peppers must be exactly 32 bytes (256 bits) to provide sufficient entropy for cryptographic security. This length matches the output size of SHA-256 and provides adequate protection against brute-force attacks.

### Pepper vs Salt

- **Salt**: Random value per password, stored with hash, prevents rainbow table attacks
- **Pepper**: Shared secret value, stored separately, adds server-side secret layer

Both are used together in encx's Argon2id implementation for defense-in-depth.

### Storage Separation

Peppers are stored in Secrets Manager (separate from application database) to ensure that:
1. Database compromise doesn't expose peppers
2. Application compromise requires additional access to Secrets Manager
3. Defense-in-depth is maintained

## Cost Considerations

AWS Secrets Manager charges:
- **Storage**: $0.40 per secret per month
- **API Calls**: $0.05 per 10,000 requests

For a typical application:
- 1 pepper secret: ~$0.40/month
- Password hashing operations: Negligible (pepper cached in memory)

Total monthly cost: **~$0.40** (assuming single pepper, minimal API calls)

## Troubleshooting

### "Secret not found"

Ensure the secret exists with the correct name format:

```bash
aws secretsmanager describe-secret \
    --secret-id encx/my-app-pepper \
    --region us-east-1
```

### "AccessDeniedException"

Verify IAM permissions include required Secrets Manager actions on the `encx/*` resource prefix.

### "InvalidParameter"

Check that pepper is exactly 32 bytes before storing:

```go
if len(pepper) != encx.PepperLength {
    // Invalid pepper length
}
```

### "Region not configured"

Set `AWS_REGION` environment variable or specify `Region` in `Config`.

## Related Providers

- **AWS KMS** (`providers/keys/aws`): Pair with AWS Secrets Manager for full AWS integration
- **Vault Transit** (`providers/keys/hashicorp`): Use Vault Transit for KEK management with AWS Secrets Manager

## Documentation

- [encx Documentation](https://github.com/hengadev/encx)
- [AWS Secrets Manager Documentation](https://docs.aws.amazon.com/secretsmanager/)
- [encx API Reference](../../docs/API.md)
- [encx Architecture](../../docs/ARCHITECTURE.md)

## License

See the main encx repository for license information.
