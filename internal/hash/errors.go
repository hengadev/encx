package hash

import (
	"errors"
)

var (
	ErrUninitializedPepper = errors.New("pepper value appears to be uninitialized (all zeros)")
)

func NewUninitalizedPepperError() error {
	return ErrUninitializedPepper
}
