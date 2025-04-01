package encx

import (
	"errors"
	"fmt"
)

var (
	InternalError    = errors.New("")
	InvalidKindError = errors.New("Invalid kind for object to encrypt")
)

func NewInternalError(err string) error {
	return fmt.Errorf("%s: %s", InternalError, err)
}

func NewInvalidKindError(err string) error {
	return fmt.Errorf("%s: %s", InvalidKindError, err)
}
