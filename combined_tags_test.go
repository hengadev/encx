package encx

import (
	"context"
	"testing"
)

// Test struct with combined tags
type CombinedTagsUser struct {
	Email             string `encx:"encrypt,hash_basic"`
	EmailEncrypted    []byte
	EmailHash         string
	Password          string `encx:"hash_secure,encrypt"`
	PasswordEncrypted []byte
	PasswordHash      string
	Name              string `encx:"encrypt"`
	NameEncrypted     []byte
	DEK               []byte
	DEKEncrypted      []byte
	KeyVersion        int
}

func TestCombinedTags_ValidateStruct(t *testing.T) {
	user := &CombinedTagsUser{}
	err := ValidateStruct(user)
	if err != nil {
		t.Errorf("Expected no validation errors for combined tags struct, got: %v", err)
	}
}

func TestCombinedTags_ProcessStruct(t *testing.T) {
	// Create test crypto instance
	crypto, _ := NewTestCrypto(t)

	user := &CombinedTagsUser{
		Email:    "test@example.com",
		Password: "secret123",
		Name:     "John Doe",
	}

	// Process the struct
	ctx := context.Background()
	err := crypto.ProcessStruct(ctx, user)
	if err != nil {
		t.Fatalf("Failed to process struct with combined tags: %v", err)
	}

	// Verify encryption worked (original fields should be cleared)
	if user.Email != "" {
		t.Error("Email field should be cleared after encryption")
	}
	if user.Password != "" {
		t.Error("Password field should be cleared after encryption")
	}
	if user.Name != "" {
		t.Error("Name field should be cleared after encryption")
	}

	// Verify encrypted fields are populated
	if len(user.EmailEncrypted) == 0 {
		t.Error("EmailEncrypted should be populated")
	}
	if len(user.PasswordEncrypted) == 0 {
		t.Error("PasswordEncrypted should be populated")
	}
	if len(user.NameEncrypted) == 0 {
		t.Error("NameEncrypted should be populated")
	}

	// Verify hash fields are populated
	if user.EmailHash == "" {
		t.Error("EmailHash should be populated")
	}
	if user.PasswordHash == "" {
		t.Error("PasswordHash should be populated")
	}

	// Verify DEK fields are set
	if len(user.DEK) == 0 {
		t.Error("DEK should be populated")
	}
	if len(user.DEKEncrypted) == 0 {
		t.Error("DEKEncrypted should be populated")
	}
	if user.KeyVersion == 0 {
		t.Error("KeyVersion should be populated")
	}

	t.Logf("Successfully processed struct with combined tags!")
	t.Logf("EmailHash: %s", user.EmailHash[:20]+"...") // Show partial hash for verification
	t.Logf("PasswordHash: %s", user.PasswordHash[:20]+"...") // Show partial hash for verification
}

func TestCombinedTags_MissingCompanionFields(t *testing.T) {
	// Test struct missing companion fields for combined tags
	type MissingCompanionUser struct {
		Email        string `encx:"encrypt,hash_basic"`
		// Missing EmailEncrypted []byte and EmailHash string
		DEK          []byte
		DEKEncrypted []byte
		KeyVersion   int
	}
	
	user := &MissingCompanionUser{}
	err := ValidateStruct(user)
	if err == nil {
		t.Error("Expected validation error for missing companion fields")
	}
	
	errorMsg := err.Error()
	if !contains(errorMsg, "requires companion field 'EmailEncrypted []byte'") {
		t.Errorf("Expected error about missing EmailEncrypted field, got: %v", err)
	}
	
	if !contains(errorMsg, "requires companion field 'EmailHash string'") {
		t.Errorf("Expected error about missing EmailHash field, got: %v", err)
	}
}

func TestCombinedTags_InvalidTagCombination(t *testing.T) {
	// Test struct with invalid tag in combination
	type InvalidCombinationUser struct {
		Email         string `encx:"encrypt,invalid_tag"`
		EmailEncrypted []byte
		DEK           []byte
		DEKEncrypted  []byte
		KeyVersion    int
	}
	
	user := &InvalidCombinationUser{}
	err := ValidateStruct(user)
	if err == nil {
		t.Error("Expected validation error for invalid tag in combination")
	}
	
	if !contains(err.Error(), "invalid encx tag 'invalid_tag'") {
		t.Errorf("Expected error about invalid tag, got: %v", err)
	}
}