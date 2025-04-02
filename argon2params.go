package encx

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
) *Argon2Params {
	return &Argon2Params{
		Memory:      memory,
		Iterations:  iterations,
		Parallelism: parallelism,
		SaltLength:  saltLength,
		KeyLength:   keyLength,
	}
}

// DefaultArgon2Params returns recommended parameters for Argon2id
func DefaultArgon2Params() *Argon2Params {
	return &Argon2Params{
		Memory:      64 * 1024, // 64MB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}
