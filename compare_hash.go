package encx

import (
	"fmt"
	"reflect"
)

func (c *Crypto) CompareSecureHashAndValue(value any, hashValue string) (bool, error) {
	if value == nil {
		return false, fmt.Errorf("%w: value cannot be nil", ErrNilPointer)
	}
	v, err := c.serializer.Serialize(reflect.ValueOf(value))
	if err != nil {
		return false, fmt.Errorf("failed to serialize field value : %w", err)
	}
	valueHashed, err := c.HashSecure(v)
	if err != nil {
		return false, fmt.Errorf("secure hashing failed for value : %w", err)
	}
	return valueHashed == hashValue, nil
}

func (c *Crypto) CompareBasicHashAndValue(value any, hashValue string) (bool, error) {
	if value == nil {
		return false, fmt.Errorf("%w: value cannot be nil", ErrNilPointer)
	}
	v, err := c.serializer.Serialize(reflect.ValueOf(value))
	if err != nil {
		return false, fmt.Errorf("failed to serialize field value : %w", err)
	}
	return c.HashBasic(v) == hashValue, nil
}
