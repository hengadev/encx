package encx

import (
	"fmt"
	"reflect"
)
// validateObjectForProcessing checks if the provided object is a non-nil pointer to a struct.
// It returns an error if the object is nil, not a pointer, or not pointing to a struct,
// or if the pointer is not settable.
func validateObjectForProcessing(object any) error {
	if object == nil {
		return fmt.Errorf("nil object encountered: the object can not be processed for encryption")
	}
	v := reflect.ValueOf(object)
	if v.Kind() != reflect.Ptr {
		return NewInvalidKindError("Must be a pointer to a struct.")
	}
	if v.IsNil() { // Check for nil pointer after getting Value
		return fmt.Errorf("nil pointer to struct encountered: the object cannot be processed")
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return NewInvalidKindError("Must be a pointer to a struct.")
	}
	if !v.CanSet() { // Check if the pointer's value can be modified
		return fmt.Errorf("cannot set value on the provided pointer")
	}
	return nil
}
