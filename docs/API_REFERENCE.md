# Encx Code Generation API Reference

This reference provides detailed documentation for all APIs, functions, and types in the encx code generation system.

## Table of Contents

1. [Generated Functions](#generated-functions)
2. [CLI Commands](#cli-commands)
3. [Configuration API](#configuration-api)
4. [Metadata API](#metadata-api)
5. [Schema Helpers](#schema-helpers)
6. [Validation API](#validation-api)
7. [Error Types](#error-types)

## Generated Functions

For each struct with encx tags, the code generator creates processing and decryption functions.

### ProcessStructEncx

Encrypts and hashes struct fields according to their encx tags.

**Signature:**
```go
func ProcessStructEncx(ctx context.Context, crypto encx.CryptoService, source *Struct) (*StructEncx, error)
```

**Parameters:**
- `ctx`: Context for cancellation and deadlines
- `crypto`: encx.CryptoService instance (interface)
- `source`: Original struct with plaintext data

**Returns:**
- `*StructEncx`: Struct with encrypted/hashed fields
- `error`: Processing error, if any

**Example:**
```go
user := &User{
    Email: "user@example.com",
    Phone: "+1234567890",
}

userEncx, err := ProcessUserEncx(ctx, crypto, user)
if err != nil {
    log.Fatal(err)
}

// userEncx contains encrypted data
fmt.Printf("Email hash: %s\n", userEncx.EmailHash)
```

**Error Handling:**
Uses `errsx.Map` for structured error collection:
```go
// Generated error handling
errs := errsx.Map{}

if emailErr != nil {
    errs.Set("Email encryption", emailErr)
}
if phoneErr != nil {
    errs.Set("Phone encryption", phoneErr)
}

return result, errs.AsError() // Returns nil if no errors
```

### DecryptStructEncx

Decrypts struct fields back to their original form.

**Signature:**
```go
func DecryptStructEncx(ctx context.Context, crypto encx.CryptoService, source *StructEncx) (*Struct, error)
```

**Parameters:**
- `ctx`: Context for cancellation and deadlines
- `crypto`: encx.CryptoService instance (interface)
- `source`: Struct with encrypted/hashed fields

**Returns:**
- `*Struct`: Original struct with decrypted data
- `error`: Decryption error, if any

**Example:**
```go
// userEncx loaded from database
userEncx := loadUserFromDB(userID)

user, err := DecryptUserEncx(ctx, crypto, userEncx)
if err != nil {
    log.Fatal(err)
}

// user contains decrypted data
fmt.Printf("Email: %s\n", user.Email)
```

**Note:** Hash-only fields cannot be decrypted and will remain empty in the result.

### Generated Struct Types

For each source struct, an encrypted counterpart is generated.

**Example:**
```go
// Source struct
type User struct {
    Email string `encx:"encrypt,hash_basic"`
    Phone string `encx:"encrypt"`
    SSN   string `encx:"hash_secure"`

    // Companion fields...
}

// Generated encrypted struct
type UserEncx struct {
    EmailEncrypted []byte `db:"email_encrypted" json:"email_encrypted"`
    EmailHash      string `db:"email_hash" json:"email_hash"`
    PhoneEncrypted []byte `db:"phone_encrypted" json:"phone_encrypted"`
    SSNHashSecure  string `db:"ssn_hash_secure" json:"ssn_hash_secure"`

    // Essential encryption fields
    DEKEncrypted []byte `db:"dek_encrypted" json:"dek_encrypted"`
    KeyVersion   int    `db:"key_version" json:"key_version"`
    Metadata     string `db:"metadata" json:"metadata"`
}
```

## CLI Commands

### encx-gen generate

Generates encx code for structs with encx tags.

**Syntax:**
```bash
encx-gen generate [flags] [packages...]
```

**Flags:**
- `-config string`: Configuration file path (default "encx.yaml")
- `-output string`: Override output directory
- `-v, -verbose`: Enable verbose output
- `-dry-run`: Show what would be generated without writing files

**Examples:**
```bash
# Generate for current directory
encx-gen generate .

# Generate for multiple packages
encx-gen generate ./models ./api

# Verbose generation with custom config
encx-gen generate -config=custom.yaml -v ./models

# Dry run to preview changes
encx-gen generate -dry-run .
```

**Exit Codes:**
- `0`: Success
- `1`: Generation failed
- `2`: Validation errors found

### encx-gen validate

Validates configuration and struct tags.

**Syntax:**
```bash
encx-gen validate [flags] [packages...]
```

**Flags:**
- `-config string`: Configuration file path (default "encx.yaml")
- `-v, -verbose`: Enable verbose output

**Examples:**
```bash
# Validate current directory
encx-gen validate .

# Validate with verbose output
encx-gen validate -v ./models

# Validate specific packages
encx-gen validate ./models ./api ./handlers
```

**Output:**
```
Found 2 structs with encx tags in ./models:
  User (user.go)
    ✓ All fields valid
  Order (order.go)
    ✗ Order.CreditCard: missing companion field CreditCardEncrypted []byte for encrypt tag

✗ Validation failed with errors.
```

### encx-gen init

Creates a default configuration file.

**Syntax:**
```bash
encx-gen init [flags]
```

**Flags:**
- `-force`: Overwrite existing configuration file

**Examples:**
```bash
# Create default config
encx-gen init

# Overwrite existing config
encx-gen init -force
```

**Generated Config:**
```yaml
version: "1"

generation:
  output_suffix: "_encx"
  package_name: "encx"

packages: {}
```

### encx-gen version

Shows version information.

**Syntax:**
```bash
encx-gen version
```

**Output:**
```
encx-gen version 1.0.0
Code generator for encx encryption library

Features:
  - AST-based struct discovery
  - Incremental generation with caching
  - Comprehensive tag validation
  - Cross-database JSON metadata support
  - Template-based code generation

Supported tags: encrypt, hash_basic, hash_secure
Supported databases: PostgreSQL, SQLite, MySQL
```

## Configuration API

### Config Structure

```go
type Config struct {
    Version    string                   `yaml:"version"`
    Generation GenerationConfig         `yaml:"generation"`
    Packages   map[string]PackageConfig `yaml:"packages"`
}

type GenerationConfig struct {
    OutputSuffix string `yaml:"output_suffix"`
    PackageName  string `yaml:"package_name"`
}

type PackageConfig struct {
    OutputDir string `yaml:"output_dir"`
    Skip      bool   `yaml:"skip"`
}
```

### LoadConfig

Loads configuration from a YAML file.

**Signature:**
```go
func LoadConfig(path string) (*Config, error)
```

**Example:**
```go
config, err := LoadConfig("encx.yaml")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Output suffix: %s\n", config.Generation.OutputSuffix)
```

### SaveConfig

Saves configuration to a YAML file.

**Signature:**
```go
func SaveConfig(config *Config, path string) error
```

**Example:**
```go
config := DefaultConfig()
config.Generation.OutputSuffix = "_encrypted"

err := SaveConfig(config, "custom.yaml")
if err != nil {
    log.Fatal(err)
}
```

### DefaultConfig

Returns a configuration with default values.

**Signature:**
```go
func DefaultConfig() *Config
```

### Config.Validate

Validates configuration settings.

**Signature:**
```go
func (c *Config) Validate() error
```

**Validation Rules:**
- `output_suffix`: Must not be empty, must start with underscore or letter
- `package_name`: Must be valid Go identifier (or "auto" for automatic detection)

## Metadata API

### EncryptionMetadata

Stores metadata about encrypted data.

```go
type EncryptionMetadata struct {
    PepperVersion    int    `json:"pepper_version"`
    KEKAlias         string `json:"kek_alias"`
    EncryptionTime   int64  `json:"encryption_time"`
    GeneratorVersion string `json:"generator_version"`
}
```

**Methods:**

#### NewEncryptionMetadata
```go
func NewEncryptionMetadata(kekAlias, generatorVersion string, pepperVersion int) *EncryptionMetadata
```

#### ToJSON
```go
func (em *EncryptionMetadata) ToJSON() ([]byte, error)
```

#### FromJSON
```go
func (em *EncryptionMetadata) FromJSON(data []byte) error
```

#### Validate
```go
func (em *EncryptionMetadata) Validate() error
```

**Example:**
```go
metadata := NewEncryptionMetadata("primary", "1.0.0", 1)
jsonData, err := metadata.ToJSON()
if err != nil {
    log.Fatal(err)
}

// Store jsonData in database metadata column
```

## Schema Helpers

### MetadataColumn

Cross-database JSON column support.

```go
type MetadataColumn struct {
    data map[string]interface{}
}
```

**Methods:**

#### NewMetadataColumn
```go
func NewMetadataColumn() *MetadataColumn
```

#### Set
```go
func (mc *MetadataColumn) Set(key string, value interface{})
```

#### Get
```go
func (mc *MetadataColumn) Get(key string) (interface{}, bool)
```

#### GetString
```go
func (mc *MetadataColumn) GetString(key string) (string, bool)
```

#### GetInt
```go
func (mc *MetadataColumn) GetInt(key string) (int, bool)
```

**Example:**
```go
metadata := NewMetadataColumn()
metadata.Set("kek_alias", "primary")
metadata.Set("encryption_time", time.Now().Unix())

// Use in database struct
type UserRecord struct {
    ID       int            `db:"id"`
    Metadata MetadataColumn `db:"metadata"`
}
```

### Database Type Helpers

#### DatabaseType
```go
type DatabaseType string

const (
    PostgreSQL DatabaseType = "postgresql"
    SQLite     DatabaseType = "sqlite"
    MySQL      DatabaseType = "mysql"
)
```

#### ParseDatabaseType
```go
func ParseDatabaseType(s string) DatabaseType
```

#### GetJSONColumnType
```go
func (dt DatabaseType) GetJSONColumnType() string
```

#### GetBlobColumnType
```go
func (dt DatabaseType) GetBlobColumnType() string
```

**Example:**
```go
dbType := ParseDatabaseType("postgresql")
jsonType := dbType.GetJSONColumnType() // Returns "JSONB"
blobType := dbType.GetBlobColumnType() // Returns "BYTEA"
```

## Validation API

### TagValidator

Validates encx struct tags.

```go
type TagValidator struct {
    knownTags     []string
    invalidCombos map[string][]string
}
```

#### NewTagValidator
```go
func NewTagValidator() *TagValidator
```

#### ValidateFieldTags
```go
func (tv *TagValidator) ValidateFieldTags(fieldName string, tags []string) []string
```

**Example:**
```go
validator := NewTagValidator()
errors := validator.ValidateFieldTags("Email", []string{"encrypt", "hash_basic"})
if len(errors) > 0 {
    log.Printf("Validation errors: %v", errors)
}
```

### CompanionFieldValidator

Validates companion fields for encx tags.

#### NewCompanionFieldValidator
```go
func NewCompanionFieldValidator() *CompanionFieldValidator
```

#### ValidateCompanionFields
```go
func (cfv *CompanionFieldValidator) ValidateCompanionFields(structInfo *StructInfo) []ValidationError
```

### StructInfo

Information about discovered structs.

```go
type StructInfo struct {
    PackageName       string
    StructName        string
    SourceFile        string
    Fields            []FieldInfo
    HasEncxTags       bool
    GenerationOptions map[string]string
}

type FieldInfo struct {
    Name             string
    Type             string
    EncxTags         []string
    CompanionFields  map[string]CompanionField
    IsValid          bool
    ValidationErrors []string
}
```

### DiscoverStructs

Discovers structs with encx tags.

**Signature:**
```go
func DiscoverStructs(packagePath string, config *DiscoveryConfig) ([]StructInfo, error)
```

**Example:**
```go
config := &DiscoveryConfig{}
structs, err := DiscoverStructs("./models", config)
if err != nil {
    log.Fatal(err)
}

for _, structInfo := range structs {
    fmt.Printf("Found struct: %s\n", structInfo.StructName)
    for _, field := range structInfo.Fields {
        if len(field.EncxTags) > 0 {
            fmt.Printf("  %s: %v\n", field.Name, field.EncxTags)
        }
    }
}
```

## Error Types

### ValidationError

Represents a validation error.

```go
type ValidationError struct {
    Field   string
    Message string
    Type    string
}

func (ve ValidationError) Error() string
```

### GenerationError

Represents a code generation error.

```go
type GenerationError struct {
    Struct  string
    Field   string
    Message string
    Cause   error
}

func (ge GenerationError) Error() string
func (ge GenerationError) Unwrap() error
```

### ConfigurationError

Represents a configuration error.

```go
type ConfigurationError struct {
    Field   string
    Value   string
    Message string
}

func (ce ConfigurationError) Error() string
```

**Example Error Handling:**
```go
_, err := DiscoverStructs("./models", config)
if err != nil {
    var validationErr ValidationError
    if errors.As(err, &validationErr) {
        log.Printf("Validation error in field %s: %s", validationErr.Field, validationErr.Message)
    } else {
        log.Printf("Discovery error: %v", err)
    }
}
```

## Code Generation Internals

### TemplateEngine

Manages code generation templates.

```go
type TemplateEngine struct {
    processTemplate *template.Template
    // internal fields
}
```

#### NewTemplateEngine
```go
func NewTemplateEngine() (*TemplateEngine, error)
```

#### GenerateCode
```go
func (te *TemplateEngine) GenerateCode(data TemplateData) ([]byte, error)
```

### TemplateData

Data passed to code generation templates.

```go
type TemplateData struct {
    PackageName      string
    StructName       string
    SourceFile       string
    GeneratedTime    string
    GeneratorVersion string
    Imports          []string
    EncryptedFields  []TemplateField
    ProcessingSteps  []string
    DecryptionSteps  []string
}
```

### BuildTemplateData

Builds template data from struct information.

**Signature:**
```go
func BuildTemplateData(structInfo StructInfo, config GenerationConfig) TemplateData
```

## Performance Considerations

### Incremental Generation

The code generator uses SHA256 file hashing to detect changes:

```go
type GenerationCache struct {
    SourceHashes   map[string]string
    ConfigHash     string
    GeneratedFiles map[string]GeneratedFileInfo
    LastGenerated  time.Time
}
```

Cache is automatically maintained in `.encx-gen-cache.json`.

### Memory Usage

Generated functions use minimal memory allocation:
- Direct field access (no reflection)
- Structured error collection with errsx.Map
- Efficient template-based generation

### Performance Benchmarks

Run benchmarks to measure performance:

```bash
go test -bench=BenchmarkDiscoverStructs -benchmem ./internal/codegen
go test -bench=BenchmarkTemplateGeneration -benchmem ./internal/codegen
go test -bench=BenchmarkUserProcessing -benchmem ./models
```

Expected results:
- Struct discovery: ~1000 ns/op for small projects
- Template generation: ~50 μs/op per struct
- User processing: ~50 ns/op (vs 1000+ ns/op with reflection)

This API reference covers all public interfaces and functions in the encx code generation system.
