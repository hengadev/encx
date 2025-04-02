package encx

type CryptoEngineOption func(e *CryptoEngine)

func WithKeyEncryptionKey(key []byte) CryptoEngineOption {
	return func(e *CryptoEngine) {
		e.KeyEncryptionKey = key
	}
}

func WithPepper(key []byte) CryptoEngineOption {
	return func(e *CryptoEngine) {
		e.Pepper = key
	}
}

func WithArgon2Params(params *Argon2Params) CryptoEngineOption {
	return func(e *CryptoEngine) {
		e.Argon2Params = params
	}
}
