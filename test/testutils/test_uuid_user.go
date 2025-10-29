package testutils

import "github.com/google/uuid"

// UUIDUser represents a user with UUID and encrypted fields
type UUIDUser struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email" encx:"encrypt"`
	Name  string    `json:"name"`
}
