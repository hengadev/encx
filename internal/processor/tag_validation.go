package processor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
)

// Constants from the main package
const (
	StructTag     = "encx"
	TagEncrypt    = "encrypt"
	TagHashSecure = "hash_secure"
	TagHashBasic  = "hash_basic"

	// Suffix constants
	SuffixEncrypted = "Encrypted"
	SuffixHashed    = "Hash"

	// Field name constants
	FieldKeyVersion   = "KeyVersion"
	FieldDEK          = "DEK"
	FieldDEKEncrypted = FieldDEK + SuffixEncrypted
)

// validateObjectForProcessing checks if the provided object is a non-nil pointer to a struct.
// It returns an error if the object is nil, not a pointer, or not pointing to a struct,
// or if the pointer is not settable.
func validateObjectForProcessing(object any) error {
	if object == nil {
		return fmt.Errorf("ProcessStruct requires a non-nil object. "+
			"Usage: crypto.ProcessStruct(ctx, &myStruct)")
	}
	v := reflect.ValueOf(object)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("ProcessStruct requires a pointer to a struct, got %T. "+
			"Usage: crypto.ProcessStruct(ctx, &myStruct) not crypto.ProcessStruct(ctx, myStruct)",
			object)
	}
	if v.IsNil() { // Check for nil pointer after getting Value
		return fmt.Errorf("ProcessStruct requires a non-nil pointer to a struct. "+
			"Your pointer is nil")
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("ProcessStruct requires a pointer to a struct, got pointer to %s. "+
			"Usage: crypto.ProcessStruct(ctx, &myStruct)", elem.Type())
	}
	if !elem.CanSet() {
		return fmt.Errorf("struct fields must be settable. "+
			"Make sure your struct fields are exported (start with uppercase)")
	}
	return nil
}

// StructTagValidator provides compile-time validation for encx struct tags
type StructTagValidator struct {
	errors []string
}

// NewStructTagValidator creates a new struct tag validator
func NewStructTagValidator() *StructTagValidator {
	return &StructTagValidator{}
}

// ValidateSourceFile validates all encx struct tags in a Go source file
func (v *StructTagValidator) ValidateSourceFile(filename string) error {
	v.errors = nil

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file %s: %w", filename, err)
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.StructType:
			v.validateStruct(fset, x)
		}
		return true
	})

	if len(v.errors) > 0 {
		return fmt.Errorf("struct tag validation errors:\n%s", strings.Join(v.errors, "\n"))
	}

	return nil
}

// validateStruct validates a struct definition for correct encx tags
func (v *StructTagValidator) validateStruct(fset *token.FileSet, structType *ast.StructType) {
	if structType.Fields == nil {
		return
	}

	for _, field := range structType.Fields.List {
		if field.Tag != nil {
			v.validateField(fset, field)
		}
	}
}

// validateField validates a single field's encx tag
func (v *StructTagValidator) validateField(fset *token.FileSet, field *ast.Field) {
	if field.Tag == nil {
		return
	}

	// Parse the tag
	tagValue := field.Tag.Value
	if len(tagValue) < 2 {
		return
	}

	// Remove quotes
	tagValue = tagValue[1 : len(tagValue)-1]

	// Parse struct tags
	tags := reflect.StructTag(tagValue)
	encxTag := tags.Get(StructTag)

	if encxTag == "" {
		return
	}

	pos := fset.Position(field.Pos())
	fieldName := ""
	if len(field.Names) > 0 {
		fieldName = field.Names[0].Name
	}

	// Parse comma-separated tags
	encxTags := strings.Split(strings.TrimSpace(encxTag), ",")
	for i, t := range encxTags {
		encxTags[i] = strings.TrimSpace(t)
	}

	// Validate each tag value
	for _, singleTag := range encxTags {
		if !v.isValidEncxTag(singleTag) {
			v.addError(pos, fieldName, fmt.Sprintf("invalid encx tag '%s' in '%s', supported values: %s, %s, %s",
				singleTag, encxTag, TagEncrypt, TagHashSecure, TagHashBasic))
			return
		}
	}

	// Check for required companion fields based on tag types
	v.validateCompanionFields(pos, fieldName, encxTags, field)
}

// isValidEncxTag checks if an encx tag value is valid
func (v *StructTagValidator) isValidEncxTag(tag string) bool {
	switch tag {
	case TagEncrypt, TagHashSecure, TagHashBasic:
		return true
	default:
		return false
	}
}

// validateCompanionFields validates that required companion fields exist for encx tags
// Note: This is a simplified validation - full validation requires the entire struct context
func (v *StructTagValidator) validateCompanionFields(pos token.Position, fieldName string, encxTags []string, field *ast.Field) {
	// This is a compile-time check, so we can only validate the tag syntax
	// Runtime validation of companion fields happens in the actual processing

	for _, encxTag := range encxTags {
		switch encxTag {
		case TagEncrypt:
			// Would need companion field: fieldName + SuffixEncrypted ([]byte)
			v.addWarning(pos, fieldName, fmt.Sprintf("field with 'encrypt' tag requires companion field '%s%s []byte'",
				fieldName, SuffixEncrypted))
		case TagHashSecure, TagHashBasic:
			// Would need companion field: fieldName + SuffixHashed (string)
			v.addWarning(pos, fieldName, fmt.Sprintf("field with '%s' tag requires companion field '%s%s string'",
				encxTag, fieldName, SuffixHashed))
		}
	}
}

// addError adds a validation error
func (v *StructTagValidator) addError(pos token.Position, fieldName, message string) {
	v.errors = append(v.errors, fmt.Sprintf("%s: field '%s': %s", pos, fieldName, message))
}

// addWarning adds a validation warning (treated as error for now)
func (v *StructTagValidator) addWarning(pos token.Position, fieldName, message string) {
	v.errors = append(v.errors, fmt.Sprintf("%s: field '%s': warning: %s", pos, fieldName, message))
}

// ValidateStruct performs runtime validation of a struct value for correct encx usage
func ValidateStruct(object any) error {
	if err := validateObjectForProcessing(object); err != nil {
		return err
	}

	v := reflect.ValueOf(object).Elem()
	t := v.Type()

	var errors []string

	// Check for required fields
	requiredFields := []string{FieldDEK, FieldDEKEncrypted, FieldKeyVersion}
	for _, fieldName := range requiredFields {
		if _, exists := t.FieldByName(fieldName); !exists {
			errors = append(errors, fmt.Sprintf("missing required field: %s", fieldName))
		}
	}

	// Validate encx tagged fields and their companions
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get(StructTag)

		if tag == "" {
			continue
		}

		// Parse comma-separated tags
		tags := strings.Split(strings.TrimSpace(tag), ",")
		for j, singleTag := range tags {
			tags[j] = strings.TrimSpace(singleTag)
		}

		// Validate each tag value and check companion fields
		for _, singleTag := range tags {
			switch singleTag {
			case TagEncrypt:
				companionName := field.Name + SuffixEncrypted
				companionField, exists := t.FieldByName(companionName)
				if !exists {
					errors = append(errors, fmt.Sprintf("field '%s' with 'encrypt' tag requires companion field '%s []byte'",
						field.Name, companionName))
				} else if companionField.Type.Kind() != reflect.Slice || companionField.Type.Elem().Kind() != reflect.Uint8 {
					errors = append(errors, fmt.Sprintf("companion field '%s' must be of type []byte, got %s",
						companionName, companionField.Type))
				}
			case TagHashSecure, TagHashBasic:
				companionName := field.Name + SuffixHashed
				companionField, exists := t.FieldByName(companionName)
				if !exists {
					errors = append(errors, fmt.Sprintf("field '%s' with '%s' tag requires companion field '%s string'",
						field.Name, singleTag, companionName))
				} else if companionField.Type.Kind() != reflect.String {
					errors = append(errors, fmt.Sprintf("companion field '%s' must be of type string, got %s",
						companionName, companionField.Type))
				}
			default:
				errors = append(errors, fmt.Sprintf("field '%s' has invalid encx tag '%s' in '%s', supported values: %s, %s, %s",
					field.Name, singleTag, tag, TagEncrypt, TagHashSecure, TagHashBasic))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("struct validation errors:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}
