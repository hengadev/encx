package encx

import (
	"fmt"

	"github.com/hengadev/errsx"
)

// TODO: document recommended parameter values in the struct definition

// Argon2Params defines the parameters for Argon2id
type Argon2Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func NewArgon2Params(
	memory uint32,
	iterations uint32,
	parallelism uint8,
	saltLength uint32,
	keyLength uint32,
) (*Argon2Params, error) {
	// check for minimum security requirements
	params := &Argon2Params{
		Memory:      memory,
		Iterations:  iterations,
		Parallelism: parallelism,
		SaltLength:  saltLength,
		KeyLength:   keyLength,
	}
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("validate Argon2 parameters: %w", err)
	}
	return params, nil
}

// use a map for the errors here
func (a *Argon2Params) Validate() error {
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

var DefaultArgon2Params = &Argon2Params{
	Memory:      64 * 1024, // 64MB
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

// Interface methods for internal crypto package compatibility
func (a *Argon2Params) GetMemory() uint32      { return a.Memory }
func (a *Argon2Params) GetIterations() uint32  { return a.Iterations }
func (a *Argon2Params) GetParallelism() uint8  { return a.Parallelism }
func (a *Argon2Params) GetSaltLength() uint32  { return a.SaltLength }
func (a *Argon2Params) GetKeyLength() uint32   { return a.KeyLength }
