package encx

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/hengadev/errsx"
)

func (c *Crypto) DecryptStruct(ctx context.Context, object any) error {
	var validErrs errsx.Map
	if err := validateObjectForProcessing(object); err != nil {
		validErrs.Set("validate object for struct decryption", err)
	}

	// get key version
	keyVersion, err := getKeyVersion(object)
	if err != nil {
		validErrs.Set("get key version", err)
	}

	// get DEK
	dek, err := c.getDEK(ctx, object, keyVersion)
	if err != nil {
		validErrs.Set("get DEK", err)
	}

	if !validErrs.IsEmpty() {
		return validErrs.AsError()
	}

	// Create a new context with the DEK value
	ctxWithDEK := context.WithValue(ctx, dekContextKey{}, dek)

	v := reflect.ValueOf(object).Elem()
	t := v.Type()

	var decryptErrs errsx.Map
	// iterate through the fields to find encrypted ones
	for i := range t.NumField() {
		field := t.Field(i)
		if tag := field.Tag.Get(StructTag); tag != "" {
			fieldVal := v.FieldByName(field.Name)
			operations := strings.Split(tag, ",")
			for _, op := range operations {
				op = strings.TrimSpace(op)
				if op == TagEncrypt {
					if err := c.decryptField(ctxWithDEK, field, v, fieldVal, dek); err != nil {
						decryptErrs.Set(fmt.Sprintf("decrypt field '%s'", field.Name), err)
					}
				}
				// might want to handle other operations in the future (e.g., verifying hashes)
			}
		} else if field.Type.Kind() == reflect.Struct {
			embeddedVal := v.Field(i)
			embeddedType := field.Type
			// Recursively call DecryptEmbeddedStruct passing the context
			if err := c.decryptEmbeddedStruct(ctxWithDEK, embeddedVal, embeddedType); err != nil {
				decryptErrs.Set(fmt.Sprintf("decrypt embedded field '%s'", field.Name), err)
			}
		}

	}
	// TODO: I need to decrypt the DEK using the
	if !decryptErrs.IsEmpty() {
		return fmt.Errorf("decryption errors: %w", decryptErrs.AsError())
	}

	return nil
}

// Helper function to decrypt embedded structs recursively
func (c *Crypto) decryptEmbeddedStruct(ctx context.Context, v reflect.Value, t reflect.Type) error {
	var decryptErrs errsx.Map
	for i := range t.NumField() {
		field := t.Field(i)
		if tag := field.Tag.Get(StructTag); tag != "" {
			fieldVal := v.FieldByName(field.Name)
			operations := strings.Split(tag, ",")
			for _, op := range operations {
				op = strings.TrimSpace(op)
				if op == TagEncrypt {
					dek, ok := ctx.Value(dekContextKey{}).([]byte)
					if !ok {
						return fmt.Errorf("DEK not found in context for field '%s'", field.Name)
					}
					if len(dek) != 32 {
						return NewInvalidFormatError(FieldDEK, "32-byte []byte", Encrypt)
					}
					if err := c.decryptField(ctx, field, v, fieldVal, dek); err != nil { // Note: Using the context with DEK
						decryptErrs.Set(fmt.Sprintf("decrypt embedded field '%s'", field.Name), err)
					}
				}
			}
		} else if field.Type.Kind() == reflect.Struct {
			embeddedVal := v.Field(i)
			embeddedType := field.Type
			if err := c.decryptEmbeddedStruct(ctx, embeddedVal, embeddedType); err != nil {
				decryptErrs.Set(fmt.Sprintf("decrypt deeply embedded field '%s'", field.Name), err)
			}
		}
	}
	return decryptErrs.AsError()
}

func getKeyVersion(object any) (int, error) {
	var errs errsx.Map
	v := reflect.ValueOf(object).Elem()
	keyVersionValue := v.FieldByName(FieldKeyVersion)
	if !keyVersionValue.IsValid() {
		errs.Set(fmt.Sprintf("'%s' value not valid", FieldKeyVersion), NewMissingFieldError(FieldDEKEncrypted, Decrypt))
	}
	if keyVersionValue.Kind() != reflect.Int {
		errs.Set(fmt.Sprintf("'%s' kind not valid", FieldKeyVersion), NewInvalidFieldTypeError(FieldDEKEncrypted, "int", keyVersionValue.Type().String(), Decrypt))
	}
	if !errs.IsEmpty() {
		return 0, errs.AsError()
	}
	keyVersion := int(keyVersionValue.Int())
	return keyVersion, nil
}

func (c *Crypto) getDEK(ctx context.Context, object any, keyVersion int) ([]byte, error) {
	var errs errsx.Map
	v := reflect.ValueOf(object).Elem()

	encryptedDEKFieldValue := v.FieldByName(FieldDEKEncrypted)
	if !encryptedDEKFieldValue.IsValid() {
		errs.Set(fmt.Sprintf("'%s' value not valid", FieldDEKEncrypted), NewMissingFieldError(FieldDEKEncrypted, Decrypt))
	}
	if encryptedDEKFieldValue.Kind() != reflect.Slice || encryptedDEKFieldValue.Type().Elem().Kind() != reflect.Uint8 {
		errs.Set(fmt.Sprintf("'%s' kind not valid", FieldDEKEncrypted), NewInvalidFieldTypeError(FieldDEKEncrypted, "[]byte", encryptedDEKFieldValue.Type().String(), Decrypt))
	}
	encryptedDEKBytes := encryptedDEKFieldValue.Bytes()

	dek, err := c.DecryptDEKWithVersion(ctx, encryptedDEKBytes, keyVersion)
	if err != nil {
		errs.Set("decrypt DEK", err)
	}
	if len(dek) != 32 {
		errs.Set("DEK length", fmt.Errorf("decrypted DEK has incorrect length: expected 32, got %d", len(dek)))
	}

	return dek, errs.AsError()
}

func (c *Crypto) decryptField(ctx context.Context, field reflect.StructField, v, fieldVal reflect.Value, dek []byte) error {
	for _, fieldToSkip := range fieldsToSkip {
		if field.Name == fieldToSkip {
			log.Printf("Warning: Skipping decrypting for field '%s'.", field.Name)
			return nil
		}
	}
	encryptedFieldName := field.Name + SuffixEncrypted
	encryptedField := v.FieldByName(encryptedFieldName)
	if encryptedField.IsValid() && encryptedField.Kind() == reflect.Slice && fieldVal.CanSet() {
		ciphertext := encryptedField.Bytes()
		plaintextBytes, err := c.DecryptData(ctx, ciphertext, dek)
		if err != nil {
			return fmt.Errorf("decryption failed for field '%s': %w", field.Name, err)
		}

		// Deserialize back to the original type
		if err := c.serializer.Deserialize(plaintextBytes, fieldVal); err != nil {
			return fmt.Errorf("failed to deserialize field '%s': %w", field.Name, err)
		}
	}
	return nil
}
