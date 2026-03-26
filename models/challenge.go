package models

import (
	"time"
)

// AuthChallenge stores a pending authentication challenge nonce.
type AuthChallenge struct {
	Username  string    `gorm:"type:text;primaryKey"`
	Nonce     string    `gorm:"type:text;not null"` // base64-encoded 24-byte nonce
	ExpiresAt time.Time `gorm:"not null"`
}
