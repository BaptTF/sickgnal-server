package models

import (
	"time"
)

// User represents a registered user account.
type User struct {
	ID          string `gorm:"type:text;primaryKey"` // UUID
	Username    string `gorm:"type:text;uniqueIndex;not null"`
	IdentityKey string `gorm:"type:text;not null"` // base64-encoded 64-byte public key (x25519 || ed25519)
	Token       string `gorm:"type:text;uniqueIndex;not null"`
	CreatedAt   time.Time
}
