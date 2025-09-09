package encx_test

import (
	"github.com/hengadev/encx/internal/processor"
	"testing"
)

// Test struct with correct encx tags
type ValidUser struct {
	Name          string `encx:"encrypt"`
	NameEncrypted []byte
	Email         string `encx:"hash_basic"`
	EmailHash     string
	Password      string `encx:"hash_secure"`
	PasswordHash  string
	DEK           []byte
	DEKEncrypted  []byte
	KeyVersion    int
}

// Test struct with invalid encx tag
type InvalidTagUser struct {
	Name       string `encx:"invalid_tag"`
	Email      string
	DEK        []byte
	KeyVersion int
}

// Test struct missing companion fields
type MissingCompanionUser struct {
	Name string `encx:"encrypt"`
	// Missing NameEncrypted []byte
	Email string `encx:"hash_basic"`
	// Missing EmailHash string
	DEK          []byte
	DEKEncrypted []byte
	KeyVersion   int
}

// Test struct missing required fields
type MissingRequiredFieldsUser struct {
	Name          string `encx:"encrypt"`
	NameEncrypted []byte
	// Missing DEK, DEKEncrypted, KeyVersion
}

func TestValidateStruct_ValidStruct(t *testing.T) {
	user := &ValidUser{}
	err := processor.ValidateStruct(user)
	if err != nil {
		t.Errorf("Expected no error for valid struct, got: %v", err)
	}
}

func TestValidateStruct_MissingRequiredFields(t *testing.T) {
	user := &MissingRequiredFieldsUser{}
	err := processor.ValidateStruct(user)
	if err == nil {
		t.Error("Expected error for struct missing required fields")
	}

	if !contains(err.Error(), "missing required field: DEK") {
		t.Errorf("Expected error about missing DEK field, got: %v", err)
	}
}

func TestValidateStruct_MissingCompanionFields(t *testing.T) {
	user := &MissingCompanionUser{}
	err := processor.ValidateStruct(user)
	if err == nil {
		t.Error("Expected error for struct missing companion fields")
	}

	if !contains(err.Error(), "requires companion field 'NameEncrypted []byte'") {
		t.Errorf("Expected error about missing NameEncrypted field, got: %v", err)
	}

	if !contains(err.Error(), "requires companion field 'EmailHash string'") {
		t.Errorf("Expected error about missing EmailHash field, got: %v", err)
	}
}

func TestValidateStruct_InvalidTag(t *testing.T) {
	user := &InvalidTagUser{}
	err := processor.ValidateStruct(user)
	if err == nil {
		t.Error("Expected error for struct with invalid encx tag")
	}

	if !contains(err.Error(), "invalid encx tag 'invalid_tag'") {
		t.Errorf("Expected error about invalid tag, got: %v", err)
	}
}

func TestValidateStruct_WrongCompanionType(t *testing.T) {
	// Test struct with wrong companion field types
	type WrongTypeUser struct {
		Name          string `encx:"encrypt"`
		NameEncrypted string // Should be []byte
		Email         string `encx:"hash_basic"`
		EmailHash     []byte // Should be string
		DEK           []byte
		DEKEncrypted  []byte
		KeyVersion    int
	}

	user := &WrongTypeUser{}
	err := processor.ValidateStruct(user)
	if err == nil {
		t.Error("Expected error for struct with wrong companion field types")
	}

	if !contains(err.Error(), "must be of type []byte") {
		t.Errorf("Expected error about wrong NameEncrypted type, got: %v", err)
	}

	if !contains(err.Error(), "must be of type string") {
		t.Errorf("Expected error about wrong EmailHash type, got: %v", err)
	}
}

func TestValidateStruct_NilPointer(t *testing.T) {
	err := processor.ValidateStruct(nil)
	if err == nil {
		t.Error("Expected error for nil pointer")
	}

	if !contains(err.Error(), "non-nil object") {
		t.Errorf("Expected error about nil pointer, got: %v", err)
	}
}

func TestValidateStruct_NotPointer(t *testing.T) {
	user := ValidUser{}
	err := processor.ValidateStruct(user) // Not a pointer
	if err == nil {
		t.Error("Expected error for non-pointer argument")
	}

	if !contains(err.Error(), "requires a pointer") {
		t.Errorf("Expected error about non-pointer argument, got: %v", err)
	}
}

func TestNewStructTagValidator(t *testing.T) {
	validator := processor.NewStructTagValidator()
	if validator == nil {
		t.Error("Expected non-nil validator")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				containsAt(s, substr)))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

