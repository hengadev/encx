package encx

import (
	"fmt"

	"github.com/hengadev/encx/internal/types"
)

type CryptoOption func(e *Crypto) error

func WithArgon2Params(params *types.Argon2Params) CryptoOption {
	return func(e *Crypto) error {
		if err := params.Validate(); err != nil {
			return fmt.Errorf("validate Argon2Params: %w", err)
		}
		e.argon2Params = params
		return nil
	}
}
