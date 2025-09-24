package encx

import (
	"fmt"

	"github.com/hengadev/encx/internal/config"
	"github.com/hengadev/errsx"
)

// Argon2Params defines the parameters for Argon2id (re-exported from internal)
type Argon2Params = config.Argon2Params

// NewArgon2Params creates a new Argon2Params instance with validation
func NewArgon2Params(
	memory uint32,
	iterations uint32,
	parallelism uint8,
	saltLength uint32,
	keyLength uint32,
) (*Argon2Params, error) {
	// Create the internal struct
	params := &config.Argon2Params{
		Memory:      memory,
		Iterations:  iterations,
		Parallelism: parallelism,
		SaltLength:  saltLength,
		KeyLength:   keyLength,
	}

	// Validate using public validation rules
	if err := validateArgon2Params(params); err != nil {
		return nil, fmt.Errorf("validate Argon2 parameters: %w", err)
	}
	return params, nil
}

// validateArgon2Params provides public validation with OWASP recommended minimums
func validateArgon2Params(a *Argon2Params) error {
	var errs errsx.Map
	// OWASP recommended minimums as of 2023
	if a.Memory < 19456 { // 19 MiB minimum
		errs.Set("memory", "parameter too low, minimum recommended is 19456 KiB")
	}
	if a.Iterations < 2 {
		errs.Set("iterations", "parameter too low, minimum recommended is 2")
	}
	if a.Parallelism < 1 {
		errs.Set("parallelism", "parameter must be at least 1")
	}
	if a.SaltLength < 16 {
		errs.Set("saltLength", "parameter too short, minimum recommended is 16 bytes")
	}
	if a.KeyLength < 32 {
		errs.Set("keyLength", "parameter too short, minimum recommended is 32 bytes")
	}
	return errs.AsError()
}


// DefaultArgon2Params provides secure default parameters
var DefaultArgon2Params = &Argon2Params{
	Memory:      64 * 1024, // 64MB
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}
