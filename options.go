package encx

import (
	"fmt"
)

type CryptoOption func(e *Crypto) error

func WithArgon2Params(params *Argon2Params) CryptoOption {
	return func(e *Crypto) error {
		if err := params.Validate(); err != nil {
			return fmt.Errorf("validate Argon2Params: %w", err)
		}
		e.argon2Params = params
		return nil
	}
}
