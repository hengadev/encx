package encx

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/hengadev/errsx"
)

type dekContextKey struct{}

// ProcessStruct encrypts, hashes, and processes fields in a struct based on `encx` tags.
//
// Supported tags:
//   - encrypt: AES-GCM encryption, requires companion *Encrypted field
//   - hash_secure: Argon2id hashing with pepper, requires companion *Hash field
//   - hash_basic: SHA-256 hashing, requires companion *Hash field
//   - Combined tags: comma-separated for multiple operations, e.g. "encrypt,hash_basic"
//
// Required struct fields:
//   - DEK []byte: Data Encryption Key (auto-generated if nil)
//   - DEKEncrypted []byte: Encrypted DEK (set automatically)
//   - KeyVersion int: KEK version used (set automatically)
//
// Examples:
//
// Single operation tags:
//
//	type User struct {
//	    Email        string `encx:"hash_basic"`
//	    EmailHash    string
//	    Password     string `encx:"hash_secure"`
//	    PasswordHash string
//	    Address      string `encx:"encrypt"`
//	    AddressEncrypted []byte
//	    DEK          []byte
//	    DEKEncrypted []byte
//	    KeyVersion   int
//	}
//
// Combined operation tags (encrypt AND hash same field):
//
//	type User struct {
//	    Email             string `encx:"encrypt,hash_basic"`
//	    EmailEncrypted    []byte // For secure storage
//	    EmailHash         string // For fast lookups
//	    Password          string `encx:"hash_secure,encrypt"`
//	    PasswordHash      string // For authentication
//	    PasswordEncrypted []byte // For recovery scenarios
//	    DEK               []byte
//	    DEKEncrypted      []byte
//	    KeyVersion        int
//	}
//
//	user := &User{Email: "test@example.com", Password: "secret"}
//	if err := crypto.ProcessStruct(ctx, user); err != nil {
//	    return fmt.Errorf("processing failed: %w", err)
//	}
//	// All hash and encrypted fields are now populated
//
// Use cases for combined tags:
//   - Email: encrypt for privacy + hash for fast user lookups
//   - Password: secure hash for authentication + encrypt for recovery
//   - Phone: hash for deduplication + encrypt for storage
//   - SSN: encrypt only (no hashing for sensitive data)
func (c *Crypto) ProcessStruct(ctx context.Context, object any) error {
	// Monitoring: Start processing
	start := time.Now()
	metadata := map[string]interface{}{
		"operation_type": "struct_processing",
		"struct_type":    reflect.TypeOf(object).String(),
	}
	c.observabilityHook.OnProcessStart(ctx, "ProcessStruct", metadata)
	
	var validErrs errsx.Map
	if err := validateObjectForProcessing(object); err != nil {
		validErrs.Set("validate object for struct encryption", err)
	}

	dek, err := c.validateDEKField(object)
	if err != nil {
		validErrs.Set("validate DEK related field for struct encryption", err)
	}

	if !validErrs.IsEmpty() {
		finalErr := validErrs.AsError()
		// Monitoring: Record error and completion
		c.observabilityHook.OnError(ctx, "ProcessStruct", finalErr, metadata)
		c.observabilityHook.OnProcessComplete(ctx, "ProcessStruct", time.Since(start), finalErr, metadata)
		return finalErr
	}

	// Create a new context with the DEK value
	ctxWithDEK := context.WithValue(ctx, dekContextKey{}, dek)

	v := reflect.ValueOf(object).Elem()
	t := v.Type()

	var processErrs errsx.Map
	for i := range t.NumField() {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip fields that cannot be processed
		if shouldSkipField(field.Name) {
			continue
		}

		// Skip unexported fields that cannot be set
		if !fieldValue.CanSet() {
			continue
		}

		if tag := field.Tag.Get(StructTag); tag != "" {
			if err := c.processField(ctxWithDEK, v, field, tag); err != nil {
				processErrs.Set(fmt.Sprintf("processing field '%s' with tag '%s' in struct type %s",
					field.Name, tag, t.String()), err)
			}
		} else if field.Type.Kind() == reflect.Struct {
			embeddedVal := v.Field(i)
			embeddedType := field.Type
			// Recursively call ProcessStruct (or a similar function) passing the context
			if err := c.processEmbeddedStruct(ctxWithDEK, embeddedVal, embeddedType); err != nil {
				processErrs.Set(fmt.Sprintf("processing embedded struct '%s' of type %s in struct type %s",
					field.Name, embeddedType.String(), t.String()), err)
			}
		}
	}

	if err := c.setEncryptedDEK(ctxWithDEK, v); err != nil {
		processErrs.Set(fmt.Sprintf("setting encrypted DEK field in struct type %s", t.String()), err)
	}

	if err := c.setKeyVersion(ctxWithDEK, v); err != nil {
		processErrs.Set(fmt.Sprintf("setting key version field in struct type %s", t.String()), err)
	}

	finalErr := processErrs.AsError()
	// Monitoring: Record completion (success or failure)
	if finalErr != nil {
		c.observabilityHook.OnError(ctx, "ProcessStruct", finalErr, metadata)
	}
	c.observabilityHook.OnProcessComplete(ctx, "ProcessStruct", time.Since(start), finalErr, metadata)
	
	return finalErr
}

