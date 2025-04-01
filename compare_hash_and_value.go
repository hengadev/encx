package encx

func (s Encryptor) CompareHashAndValue(value string, hash string) (bool, error) {
	hashedValue, err := s.Hash(value)
	if err != nil {
		return false, NewInternalError(err.Error())
	}
	return hashedValue == hash, nil
}
