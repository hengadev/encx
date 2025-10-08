# Production Deployment Guide

**Version**: 1.0.0
**Last Updated**: 2025-10-05

This guide provides step-by-step instructions for deploying encx to production environments with proper security, performance, and reliability configurations.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Architecture Overview](#architecture-overview)
3. [Infrastructure Setup](#infrastructure-setup)
4. [Security Configuration](#security-configuration)
5. [Application Deployment](#application-deployment)
6. [Monitoring & Observability](#monitoring--observability)
7. [Operational Procedures](#operational-procedures)
8. [Troubleshooting](#troubleshooting)
9. [Production Checklist](#production-checklist)

---

## Prerequisites

### System Requirements

**Minimum**:
- Go 1.24.6 or later (required for stdlib security fixes)
- 2 CPU cores
- 4 GB RAM
- 20 GB disk space

**Recommended**:
- Go 1.24.6+
- 4+ CPU cores
- 8+ GB RAM
- SSD storage
- Linux OS (Ubuntu 22.04 LTS or similar)

### Infrastructure Requirements

- [ ] **KMS Service**: AWS KMS, HashiCorp Vault, or compatible KMS provider
- [ ] **Database**: PostgreSQL 13+, MySQL 8+, or compatible (for key metadata)
- [ ] **Secret Management**: AWS Secrets Manager, HashiCorp Vault, Kubernetes Secrets, or equivalent
- [ ] **Monitoring**: Prometheus, Grafana, CloudWatch, or compatible metrics system
- [ ] **Logging**: Centralized logging system (ELK stack, CloudWatch Logs, etc.)

### Security Requirements

- [ ] TLS certificates for all connections
- [ ] Network security groups/firewalls configured
- [ ] IAM roles/policies for least-privilege access
- [ ] Secret rotation procedures defined
- [ ] Backup and disaster recovery plan

---

## Architecture Overview

### Component Architecture

```
┌─────────────┐
│ Application │
└──────┬──────┘
       │
       ├───────> encx Library
       │         └─> Crypto Operations (AES-GCM, Argon2id)
       │
       ├───────> KMS Provider (AWS KMS / Vault)
       │         └─> KEK Encryption/Decryption
       │
       ├───────> Database
       │         └─> Encrypted Data + DEKs
       │
       └───────> Secret Manager
                 └─> Pepper Storage
```

### Data Flow

1. **Encryption**:
   - Application provides plaintext data
   - encx generates DEK (Data Encryption Key)
   - Data encrypted with DEK using AES-256-GCM
   - DEK encrypted with KEK via KMS
   - Encrypted data + encrypted DEK stored in database

2. **Decryption**:
   - Application retrieves encrypted data + encrypted DEK
   - DEK decrypted with KEK via KMS
   - Data decrypted with DEK
   - Plaintext returned to application

---

## Infrastructure Setup

### 1. KMS Setup

#### Option A: AWS KMS

**1.1 Create KMS Key**

```bash
# Create customer master key
aws kms create-key \
  --description "encx Production KEK" \
  --key-usage ENCRYPT_DECRYPT \
  --origin AWS_KMS \
  --multi-region false

# Capture the key ID
export KMS_KEY_ID="<key-id-from-above>"

# Create alias for easier reference
aws kms create-alias \
  --alias-name alias/encx-production-kek \
  --target-key-id $KMS_KEY_ID

# Enable automatic key rotation (recommended)
aws kms enable-key-rotation \
  --key-id $KMS_KEY_ID
```

**1.2 Configure IAM Policy**

Create policy file: `encx-kms-policy.json`

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowEncxKMSOperations",
      "Effect": "Allow",
      "Action": [
        "kms:Encrypt",
        "kms:Decrypt",
        "kms:DescribeKey",
        "kms:GenerateDataKey"
      ],
      "Resource": "arn:aws:kms:us-east-1:123456789012:key/${KMS_KEY_ID}",
      "Condition": {
        "StringEquals": {
          "kms:EncryptionContext:application": "encx-production"
        }
      }
    }
  ]
}
```

Apply policy:

```bash
# Create IAM policy
aws iam create-policy \
  --policy-name encx-kms-production \
  --policy-document file://encx-kms-policy.json

# Attach to application role/user
aws iam attach-role-policy \
  --role-name encx-application-role \
  --policy-arn arn:aws:iam::123456789012:policy/encx-kms-production
```

**1.3 Verify Access**

```bash
# Test KMS access
aws kms describe-key --key-id alias/encx-production-kek

# Test encryption
aws kms encrypt \
  --key-id alias/encx-production-kek \
  --plaintext "test" \
  --encryption-context application=encx-production \
  --query CiphertextBlob \
  --output text
```

#### Option B: HashiCorp Vault

**1.1 Enable Transit Secrets Engine**

```bash
# Enable transit engine
vault secrets enable transit

# Create encryption key
vault write -f transit/keys/encx-production-kek \
  type=aes256-gcm96 \
  exportable=false \
  allow_plaintext_backup=false

# Enable key rotation
vault write transit/keys/encx-production-kek/config \
  auto_rotate_period=2592000  # 30 days
```

**1.2 Create Vault Policy**

Create policy file: `encx-vault-policy.hcl`

```hcl
# Policy for encx application
path "transit/encrypt/encx-production-kek" {
  capabilities = ["update"]
}

path "transit/decrypt/encx-production-kek" {
  capabilities = ["update"]
}

path "transit/keys/encx-production-kek" {
  capabilities = ["read"]
}
```

Apply policy:

```bash
# Create policy
vault policy write encx-production encx-vault-policy.hcl

# Create token for application (use AppRole in production)
vault token create -policy=encx-production
```

**1.3 Setup AppRole (Recommended)**

```bash
# Enable AppRole
vault auth enable approle

# Create role
vault write auth/approle/role/encx-production \
  token_policies=encx-production \
  token_ttl=1h \
  token_max_ttl=4h

# Get role ID
vault read auth/approle/role/encx-production/role-id

# Generate secret ID
vault write -f auth/approle/role/encx-production/secret-id
```

---

### 2. Secret Management

#### Generate and Store Pepper

**2.1 Generate Pepper**

```bash
# Generate cryptographically secure 32-byte pepper
openssl rand -base64 32 | head -c 32 > pepper.txt

# Verify length (must be exactly 32 bytes)
wc -c pepper.txt  # Should output: 32
```

**2.2 Store in AWS Secrets Manager**

```bash
# Store pepper
aws secretsmanager create-secret \
  --name encx/production/pepper \
  --description "Pepper for encx production encryption" \
  --secret-string file://pepper.txt

# Enable automatic rotation (optional, complex)
# aws secretsmanager rotate-secret \
#   --secret-id encx/production/pepper \
#   --rotation-lambda-arn <lambda-arn> \
#   --rotation-rules AutomaticallyAfterDays=365

# Secure the file
shred -u pepper.txt
```

**2.3 Store in HashiCorp Vault**

```bash
# Store pepper in KV v2
vault kv put secret/encx/production/pepper \
  value="$(cat pepper.txt)"

# Secure the file
shred -u pepper.txt

# Verify storage
vault kv get secret/encx/production/pepper
```

**2.4 IAM Policy for Secrets Access (AWS)**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue"
      ],
      "Resource": "arn:aws:secretsmanager:us-east-1:123456789012:secret:encx/production/pepper-*"
    }
  ]
}
```

---

### 3. Database Setup

#### PostgreSQL Setup

**3.1 Create Database**

```sql
-- Create dedicated database
CREATE DATABASE encx_production;

-- Create dedicated user
CREATE USER encx_app WITH ENCRYPTED PASSWORD 'secure-password-here';

-- Grant privileges
GRANT CONNECT ON DATABASE encx_production TO encx_app;
```

**3.2 Create Schema**

```sql
-- Connect to database
\c encx_production

-- Create key metadata table
CREATE TABLE IF NOT EXISTS kek_versions (
    alias TEXT NOT NULL,
    version INTEGER NOT NULL,
    kms_key_id TEXT NOT NULL,
    is_deprecated BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (alias, version)
);

-- Create index for fast lookups
CREATE INDEX idx_kek_versions_active
ON kek_versions(alias, is_deprecated, version DESC)
WHERE is_deprecated = FALSE;

-- Grant table privileges
GRANT SELECT, INSERT, UPDATE ON kek_versions TO encx_app;

-- Insert initial KEK version
INSERT INTO kek_versions (alias, version, kms_key_id, is_deprecated)
VALUES ('alias/encx-production-kek', 1, '<your-kms-key-id>', FALSE);
```

**3.3 Enable Encryption at Rest**

```bash
# For AWS RDS PostgreSQL
aws rds modify-db-instance \
  --db-instance-identifier encx-production \
  --storage-encrypted \
  --apply-immediately

# For self-hosted PostgreSQL (using LUKS)
cryptsetup luksFormat /dev/sdb
cryptsetup luksOpen /dev/sdb encx_encrypted
mkfs.ext4 /dev/mapper/encx_encrypted
```

**3.4 Configure Connection Pooling**

```sql
-- Adjust PostgreSQL settings
ALTER SYSTEM SET max_connections = 200;
ALTER SYSTEM SET shared_buffers = '2GB';
ALTER SYSTEM SET effective_cache_size = '6GB';
ALTER SYSTEM SET work_mem = '16MB';

-- Reload configuration
SELECT pg_reload_conf();
```

#### MySQL Setup

**3.1 Create Database**

```sql
-- Create database
CREATE DATABASE encx_production CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Create user
CREATE USER 'encx_app'@'%' IDENTIFIED BY 'secure-password-here';

-- Grant privileges
GRANT SELECT, INSERT, UPDATE ON encx_production.* TO 'encx_app'@'%';
FLUSH PRIVILEGES;
```

**3.2 Create Schema**

```sql
USE encx_production;

CREATE TABLE IF NOT EXISTS kek_versions (
    alias VARCHAR(255) NOT NULL,
    version INT NOT NULL,
    kms_key_id VARCHAR(255) NOT NULL,
    is_deprecated BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (alias, version),
    INDEX idx_kek_active (alias, is_deprecated, version DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Insert initial KEK version
INSERT INTO kek_versions (alias, version, kms_key_id, is_deprecated)
VALUES ('alias/encx-production-kek', 1, '<your-kms-key-id>', FALSE);
```

---

## Security Configuration

### 1. Network Security

**1.1 Security Groups / Firewall Rules**

```bash
# Application to KMS (AWS)
# Allow HTTPS outbound to AWS KMS endpoints

# Application to Vault
# Allow TCP 8200 to Vault cluster

# Application to Database
# Allow TCP 5432 (PostgreSQL) or 3306 (MySQL) to database

# Application to Secrets Manager
# Allow HTTPS outbound to AWS Secrets Manager endpoints
```

**1.2 TLS Configuration**

```go
// Enable TLS for database connections
dbConfig := &pgx.ConnConfig{
    TLSConfig: &tls.Config{
        MinVersion: tls.VersionTLS12,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        },
    },
}
```

### 2. Environment Configuration

**2.1 Production Environment Variables**

```bash
# Application
export APP_ENV=production
export APP_DEBUG=false

# AWS Configuration (if using AWS KMS)
export AWS_REGION=us-east-1
export AWS_SDK_LOAD_CONFIG=1

# Vault Configuration (if using Vault)
export VAULT_ADDR=https://vault.production.example.com:8200
export VAULT_NAMESPACE=encx-production

# Database
export DB_HOST=db.production.example.com
export DB_PORT=5432
export DB_NAME=encx_production
export DB_USER=encx_app
export DB_SSL_MODE=require

# Logging
export LOG_LEVEL=info
export LOG_FORMAT=json
```

**2.2 Secret References (Not Values!)**

```bash
# These should reference secret storage, not contain actual secrets

# AWS Secrets Manager
export PEPPER_SECRET_ARN=arn:aws:secretsmanager:us-east-1:123456789012:secret:encx/production/pepper-xxxxx

# Vault
export PEPPER_VAULT_PATH=secret/encx/production/pepper

# Environment-specific
export DB_PASSWORD_SECRET_ID=/encx/production/db-password
```

---

## Application Deployment

### 1. Application Setup

**1.1 Load Secrets from Secret Manager**

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
    "github.com/hengadev/encx"
    "github.com/hengadev/encx/providers/awskms"
)

// loadPepperFromAWS loads pepper from AWS Secrets Manager
func loadPepperFromAWS(ctx context.Context) ([]byte, error) {
    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to load AWS config: %w", err)
    }

    client := secretsmanager.NewFromConfig(cfg)

    secretID := os.Getenv("PEPPER_SECRET_ARN")
    result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
        SecretId: &secretID,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get secret: %w", err)
    }

    pepper := []byte(*result.SecretString)
    if len(pepper) != 32 {
        return nil, fmt.Errorf("invalid pepper length: %d (expected 32)", len(pepper))
    }

    return pepper, nil
}

// loadPepperFromVault loads pepper from HashiCorp Vault
func loadPepperFromVault(ctx context.Context) ([]byte, error) {
    client, err := vault.NewClient(&vault.Config{
        Address: os.Getenv("VAULT_ADDR"),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create vault client: %w", err)
    }

    // Authenticate with AppRole
    roleID := os.Getenv("VAULT_ROLE_ID")
    secretID := os.Getenv("VAULT_SECRET_ID")

    loginData := map[string]interface{}{
        "role_id":   roleID,
        "secret_id": secretID,
    }

    resp, err := client.Logical().Write("auth/approle/login", loginData)
    if err != nil {
        return nil, fmt.Errorf("failed to authenticate: %w", err)
    }

    client.SetToken(resp.Auth.ClientToken)

    // Read pepper
    secret, err := client.Logical().Read(os.Getenv("PEPPER_VAULT_PATH"))
    if err != nil {
        return nil, fmt.Errorf("failed to read secret: %w", err)
    }

    pepperStr, ok := secret.Data["value"].(string)
    if !ok {
        return nil, fmt.Errorf("pepper not found in vault")
    }

    pepper := []byte(pepperStr)
    if len(pepper) != 32 {
        return nil, fmt.Errorf("invalid pepper length: %d (expected 32)", len(pepper))
    }

    return pepper, nil
}
```

**1.2 Initialize encx**

```go
func initializeEncx(ctx context.Context) (*encx.Crypto, error) {
    // 1. Load pepper from secret manager
    var pepper []byte
    var err error

    if os.Getenv("USE_VAULT") == "true" {
        pepper, err = loadPepperFromVault(ctx)
    } else {
        pepper, err = loadPepperFromAWS(ctx)
    }

    if err != nil {
        return nil, fmt.Errorf("failed to load pepper: %w", err)
    }

    // 2. Initialize KMS provider
    kmsProvider, err := awskms.NewAWSKMSProvider(ctx, awskms.Config{
        Region: os.Getenv("AWS_REGION"),
        KeyID:  "alias/encx-production-kek",
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create KMS provider: %w", err)
    }

    // 3. Create encx instance
    crypto, err := encx.New(pepper, kmsProvider)
    if err != nil {
        return nil, fmt.Errorf("failed to create encx: %w", err)
    }

    return crypto, nil
}

func main() {
    ctx := context.Background()

    // Initialize encx
    crypto, err := initializeEncx(ctx)
    if err != nil {
        log.Fatalf("Failed to initialize encx: %v", err)
    }

    // Your application logic here
    log.Println("encx initialized successfully")
    log.Printf("Version: %s", encx.VersionInfo())

    // Example: Encrypt user data
    type User struct {
        Email         string `encx:"deterministic"`
        EmailHash     []byte
        Name          string `encx:"encrypt"`
        NameEncrypted []byte
        DEK           []byte
        DEKEncrypted  []byte
    }

    user := &User{
        Email: "user@example.com",
        Name:  "John Doe",
    }

    if err := crypto.Encrypt(ctx, user); err != nil {
        log.Fatalf("Encryption failed: %v", err)
    }

    log.Printf("User encrypted successfully")
}
```

### 2. Docker Deployment

**2.1 Dockerfile**

```dockerfile
# Multi-stage build
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w \
    -X 'github.com/hengadev/encx.Version=1.0.0' \
    -X 'github.com/hengadev/encx.GitCommit=$(git rev-parse HEAD)' \
    -X 'github.com/hengadev/encx.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
    -o /app/main ./cmd/yourapp

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Use non-root user
USER appuser

# Expose port (if needed)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/main", "healthcheck"] || exit 1

# Run application
ENTRYPOINT ["/app/main"]
```

**2.2 Docker Compose (Development/Testing)**

```yaml
version: '3.9'

services:
  app:
    build: .
    environment:
      - APP_ENV=production
      - AWS_REGION=${AWS_REGION}
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_NAME=encx_production
      - DB_USER=encx_app
      - PEPPER_SECRET_ARN=${PEPPER_SECRET_ARN}
    depends_on:
      - postgres
    ports:
      - "8080:8080"

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=encx_production
      - POSTGRES_USER=encx_app
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

volumes:
  postgres_data:
```

### 3. Kubernetes Deployment

**3.1 Secret Configuration**

```yaml
# encx-secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: encx-secrets
  namespace: production
type: Opaque
stringData:
  pepper: "<base64-encoded-pepper>"
  db-password: "<database-password>"
---
# Or use External Secrets Operator with AWS Secrets Manager
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: encx-secrets
  namespace: production
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager
    kind: SecretStore
  target:
    name: encx-secrets
  data:
    - secretKey: pepper
      remoteRef:
        key: encx/production/pepper
```

**3.2 Deployment Configuration**

```yaml
# encx-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: encx-app
  namespace: production
  labels:
    app: encx-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: encx-app
  template:
    metadata:
      labels:
        app: encx-app
    spec:
      serviceAccountName: encx-app
      containers:
      - name: app
        image: your-registry/encx-app:1.0.0
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: APP_ENV
          value: "production"
        - name: AWS_REGION
          value: "us-east-1"
        - name: DB_HOST
          value: "postgres.production.svc.cluster.local"
        - name: DB_PORT
          value: "5432"
        - name: DB_NAME
          value: "encx_production"
        - name: DB_USER
          value: "encx_app"
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: encx-secrets
              key: db-password
        - name: ENCX_PEPPER
          valueFrom:
            secretKeyRef:
              name: encx-secrets
              key: pepper
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        securityContext:
          runAsNonRoot: true
          runAsUser: 1000
          readOnlyRootFilesystem: true
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
```

---

## Monitoring & Observability

### 1. Metrics

**1.1 Prometheus Metrics**

```go
// Add to your application
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    encryptionCounter = promauto.NewCounter(prometheus.CounterOpts{
        Name: "encx_encryptions_total",
        Help: "Total number of encryption operations",
    })

    encryptionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
        Name:    "encx_encryption_duration_seconds",
        Help:    "Encryption operation duration in seconds",
        Buckets: prometheus.ExponentialBuckets(0.0001, 2, 15),
    })

    kmsCallsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "encx_kms_calls_total",
        Help: "Total number of KMS API calls",
    }, []string{"operation", "status"})
)

// Expose metrics endpoint
http.Handle("/metrics", promhttp.Handler())
```

**1.2 Grafana Dashboard**

```json
{
  "dashboard": {
    "title": "encx Production Metrics",
    "panels": [
      {
        "title": "Encryption Operations/sec",
        "targets": [
          {
            "expr": "rate(encx_encryptions_total[5m])"
          }
        ]
      },
      {
        "title": "Encryption Latency (P95)",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(encx_encryption_duration_seconds_bucket[5m]))"
          }
        ]
      },
      {
        "title": "KMS Call Rate",
        "targets": [
          {
            "expr": "rate(encx_kms_calls_total[5m])"
          }
        ]
      }
    ]
  }
}
```

### 2. Logging

**2.1 Structured Logging**

```go
import (
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

func initLogger() (*zap.Logger, error) {
    config := zap.Config{
        Level:            zap.NewAtomicLevelAt(zapcore.InfoLevel),
        Encoding:         "json",
        OutputPaths:      []string{"stdout"},
        ErrorOutputPaths: []string{"stderr"},
        EncoderConfig: zapcore.EncoderConfig{
            MessageKey:     "message",
            LevelKey:       "level",
            TimeKey:        "timestamp",
            NameKey:        "logger",
            CallerKey:      "caller",
            StacktraceKey:  "stacktrace",
            LineEnding:     zapcore.DefaultLineEnding,
            EncodeLevel:    zapcore.LowercaseLevelEncoder,
            EncodeTime:     zapcore.ISO8601TimeEncoder,
            EncodeDuration: zapcore.SecondsDurationEncoder,
            EncodeCaller:   zapcore.ShortCallerEncoder,
        },
    }

    if os.Getenv("APP_ENV") == "production" {
        config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
    }

    return config.Build()
}

// Usage
logger, _ := initLogger()
logger.Info("Encryption started",
    zap.String("operation", "encrypt"),
    zap.Int64("user_id", userID),
    // DO NOT log sensitive data!
)
```

### 3. Health Checks

**3.1 Application Health Check**

```go
type HealthStatus struct {
    Status    string            `json:"status"`
    Version   string            `json:"version"`
    Timestamp time.Time         `json:"timestamp"`
    Checks    map[string]string `json:"checks"`
}

func healthCheckHandler(crypto encx.CryptoService, db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        status := HealthStatus{
            Timestamp: time.Now(),
            Version:   encx.Version,
            Checks:    make(map[string]string),
        }

        // Check database
        if err := db.PingContext(ctx); err != nil {
            status.Checks["database"] = fmt.Sprintf("unhealthy: %v", err)
        } else {
            status.Checks["database"] = "healthy"
        }

        // Check KMS (optional - can be expensive)
        // if err := crypto.TestKMSConnection(ctx); err != nil {
        //     status.Checks["kms"] = fmt.Sprintf("unhealthy: %v", err)
        // } else {
        //     status.Checks["kms"] = "healthy"
        // }

        // Determine overall status
        allHealthy := true
        for _, check := range status.Checks {
            if !strings.Contains(check, "healthy") {
                allHealthy = false
                break
            }
        }

        if allHealthy {
            status.Status = "healthy"
            w.WriteHeader(http.StatusOK)
        } else {
            status.Status = "unhealthy"
            w.WriteHeader(http.StatusServiceUnavailable)
        }

        json.NewEncoder(w).Encode(status)
    }
}
```

---

## Operational Procedures

### 1. KEK Rotation

**1.1 Rotation Steps**

```bash
# 1. Create new KMS key
aws kms create-key --description "encx KEK v2"
NEW_KEY_ID="<new-key-id>"

# 2. Update alias to point to new key
aws kms update-alias \
  --alias-name alias/encx-production-kek \
  --target-key-id $NEW_KEY_ID

# 3. Update database (mark old key as deprecated)
psql -h $DB_HOST -U encx_app -d encx_production <<EOF
-- Insert new key version
INSERT INTO kek_versions (alias, version, kms_key_id, is_deprecated)
VALUES ('alias/encx-production-kek', 2, '$NEW_KEY_ID', FALSE);

-- Mark old key as deprecated (but keep for decryption)
UPDATE kek_versions
SET is_deprecated = TRUE
WHERE alias = 'alias/encx-production-kek' AND version = 1;
EOF

# 4. Restart application (will use new KEK for new encryptions)
kubectl rollout restart deployment/encx-app -n production

# 5. Re-encrypt existing data (optional, can be done gradually)
# Run re-encryption script or use application endpoint
```

### 2. Pepper Rotation

**2.1 Rotation Steps** (Complex - requires re-hashing all data)

```bash
# 1. Generate new pepper
NEW_PEPPER=$(openssl rand -base64 32 | head -c 32)

# 2. Store new pepper with version suffix
aws secretsmanager create-secret \
  --name encx/production/pepper-v2 \
  --secret-string "$NEW_PEPPER"

# 3. Deploy application update that supports dual-pepper mode
# (Application reads both old and new pepper, writes with new)

# 4. Re-hash all data with new pepper
# This requires application-specific logic

# 5. After all data re-hashed, remove old pepper
aws secretsmanager delete-secret \
  --secret-id encx/production/pepper \
  --force-delete-without-recovery
```

### 3. Backup & Restore

**3.1 Database Backup**

```bash
# PostgreSQL backup
pg_dump -h $DB_HOST -U encx_app -d encx_production \
  --format=custom \
  --file=encx-backup-$(date +%Y%m%d).dump

# Upload to S3
aws s3 cp encx-backup-$(date +%Y%m%d).dump \
  s3://your-backup-bucket/encx/$(date +%Y%m%d)/

# Encrypt backup at rest (S3 server-side encryption)
aws s3api put-object \
  --bucket your-backup-bucket \
  --key encx/$(date +%Y%m%d)/encx-backup.dump \
  --body encx-backup-$(date +%Y%m%d).dump \
  --server-side-encryption AES256
```

**3.2 Database Restore**

```bash
# Download backup from S3
aws s3 cp s3://your-backup-bucket/encx/20251005/encx-backup.dump ./

# Restore database
pg_restore -h $DB_HOST -U encx_app -d encx_production_restored \
  --clean --if-exists \
  encx-backup.dump

# Verify data can be decrypted
psql -h $DB_HOST -U encx_app -d encx_production_restored \
  -c "SELECT COUNT(*) FROM your_encrypted_table;"
```

### 4. Disaster Recovery

**4.1 Recovery Plan**

1. **Infrastructure Recovery**:
   - Restore KMS key access
   - Restore database from backup
   - Restore secrets (pepper) from backup

2. **Verify Integrity**:
   - Test KMS connectivity
   - Verify pepper is correct
   - Test decryption of sample data

3. **Application Recovery**:
   - Deploy application to new environment
   - Run health checks
   - Verify encryption/decryption works

---

## Troubleshooting

### Common Issues

#### 1. KMS Permission Errors

**Symptoms**: `AccessDeniedException` when calling KMS

**Solutions**:
```bash
# Check IAM permissions
aws iam get-role-policy --role-name encx-application-role \
  --policy-name encx-kms-production

# Verify KMS key policy
aws kms get-key-policy --key-id alias/encx-production-kek \
  --policy-name default

# Test KMS access
aws kms encrypt --key-id alias/encx-production-kek \
  --plaintext "test" --query CiphertextBlob --output text
```

#### 2. Database Connection Issues

**Symptoms**: `connection refused` or `timeout`

**Solutions**:
```bash
# Test database connectivity
psql -h $DB_HOST -U encx_app -d encx_production -c "SELECT 1;"

# Check security groups
aws ec2 describe-security-groups --group-ids sg-xxxxx

# Verify SSL/TLS settings
psql "postgresql://encx_app@$DB_HOST/encx_production?sslmode=require"
```

#### 3. Decryption Failures

**Symptoms**: `decryption failed` errors

**Solutions**:
```bash
# Verify KEK version in database
psql -h $DB_HOST -U encx_app -d encx_production \
  -c "SELECT * FROM kek_versions ORDER BY version DESC LIMIT 5;"

# Check if KMS key is enabled
aws kms describe-key --key-id alias/encx-production-kek \
  --query 'KeyMetadata.Enabled'

# Verify pepper is correct (length check)
aws secretsmanager get-secret-value \
  --secret-id encx/production/pepper \
  --query 'SecretString' --output text | wc -c  # Should be 32
```

#### 4. Performance Issues

**Symptoms**: High latency, timeouts

**Solutions**:
```bash
# Check KMS rate limits (AWS)
aws cloudwatch get-metric-statistics \
  --namespace AWS/KMS \
  --metric-name UserErrorCount \
  --dimensions Name=KeyId,Value=<key-id> \
  --start-time 2025-10-05T00:00:00Z \
  --end-time 2025-10-05T23:59:59Z \
  --period 3600 \
  --statistics Sum

# Check database connection pool
psql -h $DB_HOST -U encx_app -d encx_production \
  -c "SELECT count(*) FROM pg_stat_activity WHERE datname='encx_production';"

# Review application metrics
curl http://localhost:8080/metrics | grep encx_
```

---

## Production Checklist

### Pre-Deployment
- [ ] All tests pass (unit, integration, e2e)
- [ ] Security scans complete (govulncheck, gosec)
- [ ] Go version is 1.24.6 or later
- [ ] KMS provider configured and tested
- [ ] Pepper generated and securely stored
- [ ] Database schema created and tested
- [ ] All secrets stored in secret manager
- [ ] TLS configured for all connections
- [ ] Network security groups configured
- [ ] IAM roles/policies configured (least privilege)

### Deployment
- [ ] Application deployed to production environment
- [ ] Health checks passing
- [ ] Metrics being collected
- [ ] Logs being shipped to centralized system
- [ ] Alerts configured
- [ ] Backup procedures tested
- [ ] Disaster recovery plan documented

### Post-Deployment
- [ ] Monitor for errors in first 24 hours
- [ ] Verify encryption/decryption works
- [ ] Check performance metrics
- [ ] Review security logs
- [ ] Schedule first KEK rotation (30-90 days)
- [ ] Document any issues or lessons learned

### Ongoing
- [ ] Weekly: Review error logs and metrics
- [ ] Monthly: Review and update documentation
- [ ] Quarterly: KEK rotation, security audit
- [ ] Annually: Pepper rotation (complex, plan carefully)

---

## Support & Resources

### Documentation
- [Security Guide](./SECURITY.md)
- [Performance Guide](./PERFORMANCE.md)
- [Validation Guide](./VALIDATION_GUIDE.md)
- [API Reference](./API_REFERENCE.md)

### Monitoring
- Prometheus metrics: `http://localhost:8080/metrics`
- Grafana dashboard: `encx-production-dashboard`
- Health check: `http://localhost:8080/health`

### Contact
- Issues: https://github.com/hengadev/encx/issues
- Security: security@yourcompany.com

---

**Last Updated**: 2025-10-05
**Version**: 1.0.0
**Status**: Production Ready
