package metadata

import "errors"

// Metadata validation errors
var (
	ErrMissingKEKAlias         = errors.New("KEK alias is required")
	ErrMissingGeneratorVersion = errors.New("generator version is required")
	ErrInvalidMetadataFormat   = errors.New("invalid metadata format")
)