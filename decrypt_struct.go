package encx

import (
	"context"
	"encoding/base64"
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

	v := reflect.ValueOf(object).Elem()
	t := v.Type()

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

	var decryptErrs errsx.Map

	// iterate through the fields to find encrypted ones
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("encx")
		fieldVal := v.FieldByName(field.Name)

		operations := strings.Split(tag, ",")
		for _, op := range operations {
			op = strings.TrimSpace(op)
			if op == ENCRYPT {
				if err := c.decryptField(field, v, fieldVal, dek); err != nil {
					decryptErrs.Set(fmt.Sprintf("decrypt field '%s'", field.Name), err)
				}
			}
			// might want to handle other operations in the future (e.g., verifying hashes)
		}
	}
	if !decryptErrs.IsEmpty() {
		return fmt.Errorf("decryption errors: %w", decryptErrs.AsError())
	}

	return nil
}

func getKeyVersion(object any) (int, error) {
	var errs errsx.Map
	v := reflect.ValueOf(object).Elem()
	keyVersionValue := v.FieldByName(VERSION_FIELD)
	if !keyVersionValue.IsValid() {
		errs.Set(fmt.Sprintf("'%s' value not valid", VERSION_FIELD), NewMissingFieldError(DEK_ENCRYPTED_FIELD, Decrypt))
	}
	if keyVersionValue.Kind() != reflect.Int {
		errs.Set(fmt.Sprintf("'%s' kind not valid", VERSION_FIELD), NewInvalidFieldTypeError(DEK_ENCRYPTED_FIELD, "int", keyVersionValue.Type().String(), Decrypt))
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

	encryptedDEKFieldValue := v.FieldByName(DEK_ENCRYPTED_FIELD)
	if !encryptedDEKFieldValue.IsValid() {
		errs.Set(fmt.Sprintf("'%s' value not valid", DEK_ENCRYPTED_FIELD), NewMissingFieldError(DEK_ENCRYPTED_FIELD, Decrypt))
	}
	if encryptedDEKFieldValue.Kind() != reflect.Slice || encryptedDEKFieldValue.Type().Elem().Kind() != reflect.Uint8 {
		errs.Set(fmt.Sprintf("'%s' kind not valid", DEK_ENCRYPTED_FIELD), NewInvalidFieldTypeError(DEK_ENCRYPTED_FIELD, "[]byte", encryptedDEKFieldValue.Type().String(), Decrypt))
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

func (c *Crypto) decryptField(field reflect.StructField, v, fieldVal reflect.Value, dek []byte) error {
	for _, fieldToSkip := range FIELDS_TO_SKIP {
		if field.Name == fieldToSkip {
			log.Printf("Warning: Skipping decrypting for field '%s'.", field.Name)
			return nil
		}
	}
	encryptedFieldName := field.Name + ENCRYPTED_FIELD_SUFFIX
	encryptedField := v.FieldByName(encryptedFieldName)
	if encryptedField.IsValid() && encryptedField.Kind() == reflect.String && fieldVal.CanSet() {
		encryptedBase64 := encryptedField.String()
		ciphertext, err := base64.StdEncoding.DecodeString(encryptedBase64)
		if err != nil {
			return fmt.Errorf("failed to base64 decode field '%s': %w", encryptedFieldName, err)
		}

		plaintextBytes, err := c.DecryptData(ciphertext, dek)
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
