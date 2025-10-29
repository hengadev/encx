package testutils

import (
	"time"

	"github.com/google/uuid"
)

// UUIDUser represents a user with various field types to test zero-value checking
type UUIDUser struct {
	// Basic types - should always be encrypted (no condition check)
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email" encx:"encrypt"`
	Name     string    `json:"name"`
	Age      int       `json:"age" encx:"encrypt"`
	IsActive bool      `json:"is_active" encx:"encrypt"`

	// Struct types with semantic zero values - should check for zero
	CreatedAt time.Time `json:"created_at" encx:"encrypt"`
	UserID    uuid.UUID `json:"user_id" encx:"encrypt"`

	// Pointer types - should check for nil
	NickName  *string    `json:"nickname" encx:"encrypt"`
	UpdatedAt *time.Time `json:"updated_at" encx:"encrypt"`
	TenantID  *uuid.UUID `json:"tenant_id" encx:"encrypt"`
}
