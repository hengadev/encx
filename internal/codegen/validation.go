package codegen

import (
	"fmt"
	"strings"
)

// TagValidator handles validation of encx tags
type TagValidator struct {
	knownTags     []string
	invalidCombos map[string][]string
}

// NewTagValidator creates a new tag validator
func NewTagValidator() *TagValidator {
	return &TagValidator{
		knownTags: []string{"encrypt", "hash_basic", "hash_secure"},
		invalidCombos: map[string][]string{
			"hash_basic,hash_secure": {"hash_basic", "hash_secure"},
			"hash_secure,hash_basic": {"hash_basic", "hash_secure"},
		},
	}
}

// ValidateFieldTags validates the tags for a single field
func (tv *TagValidator) ValidateFieldTags(fieldName string, tags []string) []string {
	var errors []string

	// Check for unknown tags
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if !tv.isKnownTag(tag) {
			errors = append(errors, fmt.Sprintf("unknown tag '%s' on field '%s'", tag, fieldName))
		}
	}

	// Check for invalid combinations
	if err := tv.validateTagCombinations(fieldName, tags); err != nil {
		errors = append(errors, err.Error())
	}

	// Check for duplicate tags
	if duplicates := tv.findDuplicateTags(tags); len(duplicates) > 0 {
		errors = append(errors, fmt.Sprintf("duplicate tags on field '%s': %v", fieldName, duplicates))
	}

	return errors
}

// isKnownTag checks if a tag is in the list of known tags
func (tv *TagValidator) isKnownTag(tag string) bool {
	for _, known := range tv.knownTags {
		if tag == known {
			return true
		}
	}
	return false
}

// validateTagCombinations checks for invalid tag combinations
func (tv *TagValidator) validateTagCombinations(fieldName string, tags []string) error {
	// Normalize and sort tags for comparison
	normalizedTags := make([]string, len(tags))
	for i, tag := range tags {
		normalizedTags[i] = strings.TrimSpace(tag)
	}

	// Check each invalid combination
	for combo, invalidTags := range tv.invalidCombos {
		if tv.hasTagCombination(normalizedTags, invalidTags) {
			return fmt.Errorf("invalid tag combination on field '%s': %s (these tags cannot be used together)", fieldName, combo)
		}
	}

	return nil
}

// hasTagCombination checks if the field has a specific combination of tags
func (tv *TagValidator) hasTagCombination(fieldTags, combination []string) bool {
	for _, requiredTag := range combination {
		found := false
		for _, fieldTag := range fieldTags {
			if fieldTag == requiredTag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// findDuplicateTags finds duplicate tags in a list
func (tv *TagValidator) findDuplicateTags(tags []string) []string {
	seen := make(map[string]bool)
	var duplicates []string

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if seen[tag] {
			duplicates = append(duplicates, tag)
		} else {
			seen[tag] = true
		}
	}

	return duplicates
}

// CompanionFieldValidator validates companion fields
type CompanionFieldValidator struct{}

// NewCompanionFieldValidator creates a new companion field validator
func NewCompanionFieldValidator() *CompanionFieldValidator {
	return &CompanionFieldValidator{}
}

// ValidateCompanionFields validates that companion fields exist and have correct types
func (cfv *CompanionFieldValidator) ValidateCompanionFields(structInfo *StructInfo) []ValidationError {
	var errors []ValidationError

	// Create a map of existing fields for quick lookup
	fieldMap := make(map[string]FieldInfo)
	for _, field := range structInfo.Fields {
		fieldMap[field.Name] = field
	}

	// Check each field with encx tags
	for _, field := range structInfo.Fields {
		if len(field.EncxTags) == 0 {
			continue
		}

		for _, tag := range field.EncxTags {
			switch tag {
			case "encrypt":
				expectedName := field.Name + "Encrypted"
				if companion, exists := fieldMap[expectedName]; exists {
					if companion.Type != "[]byte" {
						errors = append(errors, ValidationError{
							Field:   field.Name,
							Tag:     tag,
							Message: fmt.Sprintf("companion field '%s' must be of type []byte, got %s", expectedName, companion.Type),
						})
					}
				} else {
					errors = append(errors, ValidationError{
						Field:   field.Name,
						Tag:     tag,
						Message: fmt.Sprintf("missing companion field '%s' of type []byte", expectedName),
					})
				}

			case "hash_basic":
				expectedName := field.Name + "Hash"
				if companion, exists := fieldMap[expectedName]; exists {
					if companion.Type != "string" {
						errors = append(errors, ValidationError{
							Field:   field.Name,
							Tag:     tag,
							Message: fmt.Sprintf("companion field '%s' must be of type string, got %s", expectedName, companion.Type),
						})
					}
				} else {
					errors = append(errors, ValidationError{
						Field:   field.Name,
						Tag:     tag,
						Message: fmt.Sprintf("missing companion field '%s' of type string", expectedName),
					})
				}

			case "hash_secure":
				expectedName := field.Name + "HashSecure"
				if companion, exists := fieldMap[expectedName]; exists {
					if companion.Type != "string" {
						errors = append(errors, ValidationError{
							Field:   field.Name,
							Tag:     tag,
							Message: fmt.Sprintf("companion field '%s' must be of type string, got %s", expectedName, companion.Type),
						})
					}
				} else {
					errors = append(errors, ValidationError{
						Field:   field.Name,
						Tag:     tag,
						Message: fmt.Sprintf("missing companion field '%s' of type string", expectedName),
					})
				}
			}
		}
	}

	return errors
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Tag     string
	Message string
}

// Error implements the error interface
func (ve ValidationError) Error() string {
	return fmt.Sprintf("field '%s' tag '%s': %s", ve.Field, ve.Tag, ve.Message)
}