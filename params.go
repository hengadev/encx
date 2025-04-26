package encx

// GetPepper retrieves the pepper (already loaded during initialization).
func (c *Crypto) GetPepper() []byte {
	return c.pepper
}

// Argon2Params returns the Argon2 hashing parameters.
func (c *Crypto) GetArgon2Params() *Argon2Params {
	return c.argon2Params
}
