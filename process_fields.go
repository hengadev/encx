package encx

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"

	"github.com/hengadev/errsx"
)

func (s CryptoEngine) ProcessFields(obj any) error {
	errs := make(errsx.Map)

	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return NewInvalidKindError("Must be a pointer to a struct.")
	}

	v = v.Elem()
	t := v.Type()

	if _, found := t.FieldByName(DEK_FIELD); !found {
		errs.Set("missing field", fmt.Sprintf("%s field is required", DEK_FIELD))
		return errs
	}

	dekValue := v.FieldByName(DEK_FIELD)
	if !dekValue.CanSet() {
		errs.Set("field access", fmt.Sprintf("cannot set %s field", DEK_FIELD))
		return errs
	}

	// Decode encryption key
	dekStr := dekValue.String()
	encryptionKey, err := hex.DecodeString(dekStr)
	if err != nil {
		errs.Set("key decode", fmt.Sprintf("failed to decode %s: %v", DEK_FIELD, err))
		return errs
	}

	// Create cipher only once for all fields
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		errs.Set("cipher creation", err)
		return errs
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		errs.Set("GCM creation", err)
		return errs
	}

	for i := range t.NumField() {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		tagValue := field.Tag.Get(STRUCT_TAG)
		if tagValue == "" {
			continue
		}

		tags := strings.Split(tagValue, ",")
		for _, tag := range tags {
			switch tag {
			case "encrypt":
				handleEncryption(field, fieldValue, v, gcm, &errs)
			case "hash_basic":
				handleBasicHashing(field, fieldValue, v, &errs)
			case "hash_secure":
				handleSecureHashing(field, fieldValue, v, s.Argon2Params, s.Pepper, &errs)
			}
		}
	}

	// Encrypt DEK with KEK
	encryptedDek, err := encryptString(s.KeyEncryptionKey, dekStr)
	if err != nil {
		errs.Set(fmt.Sprintf("encrypt field %s", DEK_FIELD), err)
	} else {
		dekValue.SetString(encryptedDek)
	}
	return errs.AsError()

}
