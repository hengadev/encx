# Complete Web Application Example

This example demonstrates a complete web application using encx code generation for encrypting user data.

## Overview

This example includes:
- User registration and authentication
- Encrypted personal information storage
- Database integration with PostgreSQL
- REST API with encrypted data handling
- Code generation for high-performance encryption
- Comprehensive error handling

## Architecture

```
examples/complete-webapp/
├── README.md                 # This file
├── main.go                   # Application entry point
├── encx.yaml                 # Code generation configuration
├── go.mod                    # Go module definition
├── models/                   # Data models
│   ├── user.go              # User model with encx tags
│   └── user_encx.go         # Generated encryption code
├── handlers/                 # HTTP handlers
│   ├── auth.go              # Authentication endpoints
│   └── users.go             # User management endpoints
├── database/                 # Database layer
│   ├── migrations/          # SQL migrations
│   └── connection.go        # Database connection
├── config/                  # Configuration
│   └── config.go            # Application configuration
└── docker-compose.yml       # PostgreSQL setup
```

## Features Demonstrated

1. **User Registration**: Encrypt PII during user signup
2. **User Authentication**: Hash-based user lookup
3. **Profile Management**: Update encrypted user data
4. **Search Functionality**: Search by hashed email addresses
5. **Data Export**: Decrypt user data for reporting
6. **Key Rotation**: Handle encryption key version upgrades

## Quick Start

### 1. Start Database

```bash
# Start PostgreSQL with Docker
docker-compose up -d postgres

# Wait for database to be ready
sleep 5
```

### 2. Build and Run

```bash
# Install dependencies
go mod tidy

# Generate encx code
encx-gen generate -v .

# Run database migrations
go run migrations/migrate.go

# Start the application
go run main.go
```

### 3. Test the API

```bash
# Register a new user
curl -X POST http://localhost:8080/api/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "phone": "+1234567890",
    "ssn": "123-45-6789",
    "first_name": "Alice",
    "last_name": "Smith"
  }'

# Login
curl -X POST http://localhost:8080/api/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "securepassword"
  }'

# Get user profile (requires authentication)
curl -X GET http://localhost:8080/api/users/profile \
  -H "Authorization: Bearer <token>"

# Search users by email hash
curl -X GET "http://localhost:8080/api/users/search?email=alice@example.com" \
  -H "Authorization: Bearer <admin_token>"
```

## Code Generation Usage

### 1. User Model with Encx Tags

```go
// models/user.go
package models

//go:generate encx-gen validate -v .
//go:generate encx-gen generate -v .

import "time"

type User struct {
    ID        int       `json:"id" db:"id"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

    // Authentication fields
    Email        string `json:"email" encx:"encrypt,hash_basic" db:"email"`
    PasswordHash string `json:"-" db:"password_hash"`

    // Personal Information (encrypted)
    FirstName string `json:"first_name" encx:"encrypt" db:"first_name"`
    LastName  string `json:"last_name" encx:"encrypt" db:"last_name"`
    Phone     string `json:"phone" encx:"encrypt,hash_basic" db:"phone"`
    SSN       string `json:"ssn" encx:"hash_secure" db:"ssn"`

    // Address (encrypted)
    Address string `json:"address" encx:"encrypt" db:"address"`
    City    string `json:"city" encx:"encrypt" db:"city"`
    State   string `json:"state" encx:"encrypt" db:"state"`
    ZipCode string `json:"zip_code" encx:"encrypt" db:"zip_code"`

    // Companion fields for encryption/hashing
    EmailEncrypted    []byte `json:"-" db:"email_encrypted"`
    EmailHash         string `json:"-" db:"email_hash"`
    FirstNameEncrypted []byte `json:"-" db:"first_name_encrypted"`
    LastNameEncrypted  []byte `json:"-" db:"last_name_encrypted"`
    PhoneEncrypted    []byte `json:"-" db:"phone_encrypted"`
    PhoneHash         string `json:"-" db:"phone_hash"`
    SSNHashSecure     string `json:"-" db:"ssn_hash_secure"`
    AddressEncrypted  []byte `json:"-" db:"address_encrypted"`
    CityEncrypted     []byte `json:"-" db:"city_encrypted"`
    StateEncrypted    []byte `json:"-" db:"state_encrypted"`
    ZipCodeEncrypted  []byte `json:"-" db:"zip_code_encrypted"`

    // Essential encryption fields
    DEKEncrypted []byte `json:"-" db:"dek_encrypted"`
    KeyVersion   int    `json:"-" db:"key_version"`
    Metadata     string `json:"-" db:"metadata"`
}

// UserRegistration represents registration request data
type UserRegistration struct {
    Email     string `json:"email" validate:"required,email"`
    Password  string `json:"password" validate:"required,min=8"`
    FirstName string `json:"first_name" validate:"required"`
    LastName  string `json:"last_name" validate:"required"`
    Phone     string `json:"phone" validate:"required"`
    SSN       string `json:"ssn" validate:"required"`
    Address   string `json:"address"`
    City      string `json:"city"`
    State     string `json:"state"`
    ZipCode   string `json:"zip_code"`
}

// UserProfile represents user profile response (without sensitive data)
type UserProfile struct {
    ID        int       `json:"id"`
    Email     string    `json:"email"`
    FirstName string    `json:"first_name"`
    LastName  string    `json:"last_name"`
    Phone     string    `json:"phone"`
    Address   string    `json:"address"`
    City      string    `json:"city"`
    State     string    `json:"state"`
    ZipCode   string    `json:"zip_code"`
    CreatedAt time.Time `json:"created_at"`
}
```

### 2. Using Generated Code in Handlers

```go
// handlers/users.go
package handlers

func (h *UserHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
    var req models.UserRegistration
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    // Validate input
    if err := h.validator.Struct(req); err != nil {
        http.Error(w, "Validation failed", http.StatusBadRequest)
        return
    }

    // Create user model
    user := &models.User{
        Email:     req.Email,
        FirstName: req.FirstName,
        LastName:  req.LastName,
        Phone:     req.Phone,
        SSN:       req.SSN,
        Address:   req.Address,
        City:      req.City,
        State:     req.State,
        ZipCode:   req.ZipCode,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    // Hash password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }
    user.PasswordHash = string(hashedPassword)

    // Encrypt user data using generated code
    userEncx, err := models.ProcessUserEncx(r.Context(), h.crypto, user)
    if err != nil {
        log.Printf("Failed to encrypt user data: %v", err)
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }

    // Store in database
    userID, err := h.userService.CreateUser(r.Context(), userEncx)
    if err != nil {
        log.Printf("Failed to create user: %v", err)
        http.Error(w, "Failed to create user", http.StatusInternalServerError)
        return
    }

    user.ID = userID

    // Return success response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": "User created successfully",
        "user_id": userID,
    })
}

func (h *UserHandler) GetUserProfile(w http.ResponseWriter, r *http.Request) {
    userID := getUserIDFromToken(r) // Extract from JWT token

    // Load encrypted user data
    userEncx, err := h.userService.GetUserByID(r.Context(), userID)
    if err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    // Decrypt user data using generated code
    user, err := models.DecryptUserEncx(r.Context(), h.crypto, userEncx)
    if err != nil {
        log.Printf("Failed to decrypt user data: %v", err)
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }

    // Create safe profile response
    profile := models.UserProfile{
        ID:        user.ID,
        Email:     user.Email,
        FirstName: user.FirstName,
        LastName:  user.LastName,
        Phone:     user.Phone,
        Address:   user.Address,
        City:      user.City,
        State:     user.State,
        ZipCode:   user.ZipCode,
        CreatedAt: user.CreatedAt,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(profile)
}

func (h *UserHandler) SearchUsersByEmail(w http.ResponseWriter, r *http.Request) {
    email := r.URL.Query().Get("email")
    if email == "" {
        http.Error(w, "Email parameter required", http.StatusBadRequest)
        return
    }

    // Create temporary user to generate hash
    tempUser := &models.User{Email: email}
    userEncx, err := models.ProcessUserEncx(r.Context(), h.crypto, tempUser)
    if err != nil {
        log.Printf("Failed to hash email for search: %v", err)
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }

    // Search by email hash
    users, err := h.userService.SearchUsersByEmailHash(r.Context(), userEncx.EmailHash)
    if err != nil {
        log.Printf("Failed to search users: %v", err)
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }

    // Decrypt and return results
    var profiles []models.UserProfile
    for _, userEncx := range users {
        user, err := models.DecryptUserEncx(r.Context(), h.crypto, userEncx)
        if err != nil {
            log.Printf("Failed to decrypt user %d: %v", userEncx.ID, err)
            continue
        }

        profiles = append(profiles, models.UserProfile{
            ID:        user.ID,
            Email:     user.Email,
            FirstName: user.FirstName,
            LastName:  user.LastName,
            CreatedAt: user.CreatedAt,
        })
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(profiles)
}
```

## Database Schema

### PostgreSQL Migration

```sql
-- database/migrations/001_create_users.sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Authentication
    password_hash TEXT NOT NULL,

    -- Encrypted data columns
    email_encrypted BYTEA,
    first_name_encrypted BYTEA,
    last_name_encrypted BYTEA,
    phone_encrypted BYTEA,
    address_encrypted BYTEA,
    city_encrypted BYTEA,
    state_encrypted BYTEA,
    zip_code_encrypted BYTEA,

    -- Hash columns for searching
    email_hash VARCHAR(64) UNIQUE,
    phone_hash VARCHAR(64),
    ssn_hash_secure TEXT,

    -- Essential encryption fields
    dek_encrypted BYTEA NOT NULL,
    key_version INTEGER NOT NULL DEFAULT 1,
    metadata JSONB NOT NULL DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_users_email_hash ON users (email_hash);
CREATE INDEX idx_users_phone_hash ON users (phone_hash);
CREATE INDEX idx_users_key_version ON users (key_version);
CREATE INDEX idx_users_metadata_gin ON users USING GIN (metadata);

-- Update trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE
    ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

## Configuration

### encx.yaml

```yaml
version: "1.0"

generation:
  output_suffix: "_encx"
  function_prefix: "Process"
  package_name: "models"
  default_serializer: "json"

packages:
  "./models":
    skip: false
    serializer: "json"
  "./handlers":
    skip: true
  "./database":
    skip: true
```

### Application Configuration

```go
// config/config.go
package config

type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Database DatabaseConfig `yaml:"database"`
    Encx     EncxConfig     `yaml:"encx"`
}

type ServerConfig struct {
    Port string `yaml:"port" env:"PORT" env-default:"8080"`
    Host string `yaml:"host" env:"HOST" env-default:"localhost"`
}

type DatabaseConfig struct {
    Host     string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
    Port     string `yaml:"port" env:"DB_PORT" env-default:"5432"`
    Database string `yaml:"database" env:"DB_NAME" env-default:"webapp_example"`
    Username string `yaml:"username" env:"DB_USER" env-default:"postgres"`
    Password string `yaml:"password" env:"DB_PASSWORD" env-default:"password"`
}

type EncxConfig struct {
    KEKPath       string `yaml:"kek_path" env:"KEK_PATH" env-default:"./keys/kek.key"`
    PepperPath    string `yaml:"pepper_path" env:"PEPPER_PATH" env-default:"./keys/pepper.key"`
    KeyVersion    int    `yaml:"key_version" env:"KEY_VERSION" env-default:"1"`
}
```

## Testing

### Unit Tests

```go
// models/user_test.go
func TestUserEncryptDecrypt(t *testing.T) {
    ctx := context.Background()
    crypto := setupTestCrypto(t)

    // Original user data
    user := &User{
        Email:     "test@example.com",
        FirstName: "John",
        LastName:  "Doe",
        Phone:     "+1234567890",
        SSN:       "123-45-6789",
        Address:   "123 Main St",
        City:      "Anytown",
        State:     "CA",
        ZipCode:   "12345",
    }

    // Encrypt using generated code
    userEncx, err := ProcessUserEncx(ctx, crypto, user)
    require.NoError(t, err)

    // Verify encrypted fields are populated
    assert.NotEmpty(t, userEncx.EmailEncrypted)
    assert.NotEmpty(t, userEncx.EmailHash)
    assert.NotEmpty(t, userEncx.FirstNameEncrypted)
    assert.NotEmpty(t, userEncx.PhoneEncrypted)
    assert.NotEmpty(t, userEncx.SSNHashSecure)

    // Decrypt using generated code
    decryptedUser, err := DecryptUserEncx(ctx, crypto, userEncx)
    require.NoError(t, err)

    // Verify data integrity
    assert.Equal(t, user.Email, decryptedUser.Email)
    assert.Equal(t, user.FirstName, decryptedUser.FirstName)
    assert.Equal(t, user.LastName, decryptedUser.LastName)
    assert.Equal(t, user.Phone, decryptedUser.Phone)
    assert.Equal(t, user.SSN, decryptedUser.SSN)
    assert.Equal(t, user.Address, decryptedUser.Address)
}
```

### Integration Tests

```go
// handlers/integration_test.go
func TestUserRegistrationFlow(t *testing.T) {
    // Setup test server
    server := setupTestServer(t)
    defer server.Close()

    // Register user
    regData := models.UserRegistration{
        Email:     "test@example.com",
        Password:  "securepassword",
        FirstName: "John",
        LastName:  "Doe",
        Phone:     "+1234567890",
        SSN:       "123-45-6789",
    }

    // Test registration
    resp := testRequest(t, server, "POST", "/api/users/register", regData)
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    // Test login
    loginData := map[string]string{
        "email":    "test@example.com",
        "password": "securepassword",
    }
    resp = testRequest(t, server, "POST", "/api/users/login", loginData)
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    // Extract token and test profile access
    var loginResp map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&loginResp)
    token := loginResp["token"].(string)

    // Test profile retrieval
    req := httptest.NewRequest("GET", "/api/users/profile", nil)
    req.Header.Set("Authorization", "Bearer "+token)
    resp = httptest.NewRecorder()
    server.ServeHTTP(resp, req)

    assert.Equal(t, http.StatusOK, resp.Code)

    var profile models.UserProfile
    json.NewDecoder(resp.Body).Decode(&profile)
    assert.Equal(t, "test@example.com", profile.Email)
    assert.Equal(t, "John", profile.FirstName)
}
```

## Performance Benchmarks

```go
// models/benchmark_test.go
func BenchmarkUserProcessing(b *testing.B) {
    ctx := context.Background()
    crypto := setupBenchmarkCrypto(b)

    user := &User{
        Email:     "benchmark@example.com",
        FirstName: "Benchmark",
        LastName:  "User",
        Phone:     "+1234567890",
        SSN:       "123-45-6789",
        Address:   "123 Benchmark St",
        City:      "Testville",
        State:     "CA",
        ZipCode:   "12345",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        userEncx, err := ProcessUserEncx(ctx, crypto, user)
        if err != nil {
            b.Fatal(err)
        }
        _ = userEncx
    }
}

func BenchmarkUserDecryption(b *testing.B) {
    ctx := context.Background()
    crypto := setupBenchmarkCrypto(b)

    user := &User{Email: "benchmark@example.com", FirstName: "Test"}
    userEncx, _ := ProcessUserEncx(ctx, crypto, user)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        decryptedUser, err := DecryptUserEncx(ctx, crypto, userEncx)
        if err != nil {
            b.Fatal(err)
        }
        _ = decryptedUser
    }
}
```

## Production Deployment

### Docker Configuration

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go run cmd/encx-gen/main.go generate -v .
RUN go build -o webapp ./main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/webapp .
COPY --from=builder /app/config.yaml .

EXPOSE 8080
CMD ["./webapp"]
```

### Environment Variables

```bash
# Production environment
export PORT=8080
export DB_HOST=prod-db.example.com
export DB_PASSWORD=secure_db_password
export KEK_PATH=/secure/kek.key
export PEPPER_PATH=/secure/pepper.key
export KEY_VERSION=1
```

This complete example demonstrates all aspects of using encx code generation in a real-world web application, from model definition to API endpoints to database integration.