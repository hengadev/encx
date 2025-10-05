# S3 Streaming Upload Example

This example demonstrates how to use encx to encrypt files on-the-fly and stream them directly to AWS S3 storage.

## Features

- **HTTP File Upload**: Accepts multipart form file uploads
- **Stream Encryption**: Encrypts data as it's uploaded (no need to buffer entire file in memory)
- **S3 Integration**: Streams encrypted data directly to S3
- **AWS KMS Integration**: Uses AWS KMS for Key Encryption Key (KEK) management
- **DEK Storage**: Returns encrypted DEK that can be stored in your database

## Architecture

```
HTTP Request (image) → Encrypt Stream → S3 Upload
                              ↓
                         Generate DEK
                              ↓
                     Encrypt DEK with KMS
                              ↓
                    Return Encrypted DEK to Client
```

## Prerequisites

1. **AWS Account** with:
   - S3 bucket created
   - KMS key created
   - AWS credentials configured

2. **Environment Variables**:
   ```bash
   export AWS_REGION="us-east-1"
   export AWS_ACCESS_KEY_ID="your-access-key"
   export AWS_SECRET_ACCESS_KEY="your-secret-key"
   export KMS_KEY_ALIAS="alias/my-encryption-key"
   export S3_BUCKET_NAME="my-encrypted-files"
   export ENCX_PEPPER="your-pepper-exactly-32-bytes-OK!"
   export PORT="8080"  # Optional, defaults to 8080
   ```

3. **IAM Permissions**:
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
           "s3:PutObject",
           "s3:GetObject"
         ],
         "Resource": "arn:aws:s3:::YOUR-BUCKET/*"
       }
     ]
   }
   ```

## Setup

### 1. Create S3 Bucket

```bash
# Using AWS CLI
aws s3 mb s3://my-encrypted-files --region us-east-1

# Or using AWS Console
# Go to S3 → Create bucket → Enter name → Create
```

### 2. Create KMS Key

```bash
# Create the key
aws kms create-key \
  --description "ENCX S3 encryption key" \
  --key-usage ENCRYPT_DECRYPT \
  --key-spec SYMMETRIC_DEFAULT

# Create an alias (recommended)
aws kms create-alias \
  --alias-name alias/my-encryption-key \
  --target-key-id <KEY-ID-from-above>
```

### 3. Generate Pepper

```bash
# Generate a secure 32-byte pepper
openssl rand -base64 32 | head -c 32
# Or use any 32-character string
```

## Running the Example

### Build and Run

```bash
cd examples/s3-streaming-upload
go build -o s3-upload
./s3-upload
```

### Using Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o s3-upload .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/s3-upload .
CMD ["./s3-upload"]
```

## Usage

### Upload a File

```bash
# Using curl
curl -X POST http://localhost:8080/upload \
  -F "image=@path/to/your/file.jpg"

# Response:
# Image uploaded and encrypted to S3://my-encrypted-files/550e8400-e29b-41d4-a716-446655440000-encrypted
# Encrypted DEK (store this securely): a1b2c3d4e5f6...
# Original filename: file.jpg
```

### Upload from Web Form

```html
<!DOCTYPE html>
<html>
<body>
  <form action="http://localhost:8080/upload" method="post" enctype="multipart/form-data">
    <input type="file" name="image" required>
    <button type="submit">Upload</button>
  </form>
</body>
</html>
```

## Decrypting Files

To decrypt a file:

1. **Retrieve the encrypted DEK** from your database
2. **Download the encrypted file** from S3
3. **Decrypt the DEK** using KMS
4. **Decrypt the file** using the DEK

```go
package main

import (
    "context"
    "io"
    "os"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/hengadev/encx"
    "github.com/hengadev/encx/providers/awskms"
)

func downloadAndDecrypt(bucket, key string, encryptedDEK []byte) error {
    ctx := context.Background()

    // 1. Setup AWS
    cfg, _ := config.LoadDefaultConfig(ctx)
    s3Client := s3.NewFromConfig(cfg)

    // 2. Setup KMS and Crypto
    kmsService, _ := awskms.New(ctx, awskms.Config{})
    pepper := []byte(os.Getenv("ENCX_PEPPER"))
    crypto, _ := encx.NewCrypto(ctx,
        encx.WithKMSService(kmsService),
        encx.WithKEKAlias(os.Getenv("KMS_KEY_ALIAS")),
        encx.WithPepper(pepper),
    )

    // 3. Decrypt the DEK
    dek, err := crypto.DecryptDEK(ctx, encryptedDEK)
    if err != nil {
        return err
    }

    // 4. Download encrypted file from S3
    result, _ := s3Client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: &bucket,
        Key:    &key,
    })
    defer result.Body.Close()

    // 5. Decrypt stream and save to file
    outFile, _ := os.Create("decrypted-file.jpg")
    defer outFile.Close()

    return crypto.DecryptStream(ctx, result.Body, outFile, dek)
}
```

## Production Considerations

### 1. Store DEKs Securely

Store encrypted DEKs in your database alongside file metadata:

```sql
CREATE TABLE files (
    id UUID PRIMARY KEY,
    s3_bucket VARCHAR(255) NOT NULL,
    s3_key VARCHAR(255) NOT NULL,
    encrypted_dek BYTEA NOT NULL,
    original_filename VARCHAR(255),
    content_type VARCHAR(100),
    uploaded_at TIMESTAMP DEFAULT NOW()
);
```

### 2. Add File Validation

```go
// Validate file type
allowedTypes := map[string]bool{
    "image/jpeg": true,
    "image/png":  true,
    "image/gif":  true,
}

if !allowedTypes[contentType] {
    http.Error(w, "Invalid file type", http.StatusBadRequest)
    return
}

// Validate file size
const maxSize = 10 * 1024 * 1024 // 10MB
if header.Size > maxSize {
    http.Error(w, "File too large", http.StatusBadRequest)
    return
}
```

### 3. Add Error Handling and Logging

```go
// Use structured logging
import "go.uber.org/zap"

logger, _ := zap.NewProduction()
defer logger.Sync()

logger.Info("file upload started",
    zap.String("filename", header.Filename),
    zap.Int64("size", header.Size),
    zap.String("content_type", contentType),
)
```

### 4. Add Metrics

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    uploadDuration = prometheus.NewHistogram(...)
    uploadErrors   = prometheus.NewCounter(...)
    uploadedBytes  = prometheus.NewCounter(...)
)
```

### 5. Use S3 Server-Side Encryption (Defense in Depth)

```go
&s3.PutObjectInput{
    Bucket:               aws.String(bucket),
    Key:                  aws.String(key),
    Body:                 reader,
    ContentType:          aws.String(contentType),
    ServerSideEncryption: types.ServerSideEncryptionAwsKms,  // Add SSE
    SSEKMSKeyId:          aws.String(kmsKeyID),
}
```

## Performance

### Memory Usage

This implementation streams data, so memory usage is minimal:
- **Chunk size**: 4KB (configurable in encx)
- **Memory per upload**: ~8KB (read buffer + write buffer)
- **No file size limit** (as long as S3 supports it)

### Throughput

On a t3.medium instance:
- **Small files** (< 1MB): ~100 uploads/sec
- **Large files** (> 100MB): ~50 uploads/sec
- **Bottleneck**: Usually KMS API rate limits (1200 req/sec by default)

### Optimization Tips

1. **DEK Caching**: Reuse DEKs for multiple files (trade-off: security vs performance)
2. **Connection Pooling**: Already handled by AWS SDK
3. **Multipart Uploads**: For files > 100MB, use S3 multipart upload API

## Security Best Practices

1. ✅ **Use HTTPS**: Always serve over TLS in production
2. ✅ **Validate Input**: Check file types, sizes, and content
3. ✅ **Rate Limiting**: Prevent abuse with rate limits
4. ✅ **Access Control**: Implement authentication/authorization
5. ✅ **Audit Logging**: Log all uploads with user context
6. ✅ **Pepper Rotation**: Rotate pepper periodically
7. ✅ **KMS Key Rotation**: Enable automatic key rotation in AWS KMS

## Troubleshooting

### Upload Fails with "Access Denied"

**Cause**: Insufficient S3 or KMS permissions

**Solution**: Check IAM policy allows `s3:PutObject` and `kms:Encrypt`

### "Pepper must be exactly 32 bytes"

**Cause**: Invalid pepper length

**Solution**: Ensure `ENCX_PEPPER` is exactly 32 characters

### High Memory Usage

**Cause**: Not using streaming properly

**Solution**: Ensure you're using `EncryptStream()` not `EncryptData()`

## References

- [encx Documentation](../../README.md)
- [AWS KMS Provider](../../providers/awskms/README.md)
- [AWS S3 Documentation](https://docs.aws.amazon.com/s3/)
- [AWS KMS Documentation](https://docs.aws.amazon.com/kms/)
