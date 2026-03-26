package models

// SignedPreKey represents a user's signed mid-term pre-key.
type SignedPreKey struct {
	UserID    string `gorm:"type:text;primaryKey"` // UUID of the owning user
	Key       string `gorm:"type:text;not null"`   // base64-encoded x25519 public key
	Signature string `gorm:"type:text;not null"`   // base64-encoded Ed25519 signature
}

// EphemeralPreKey represents a one-time pre-key for X3DH.
type EphemeralPreKey struct {
	ID     string `gorm:"type:text;primaryKey"`     // UUID of this key
	UserID string `gorm:"type:text;index;not null"` // UUID of the owning user
	Key    string `gorm:"type:text;not null"`       // base64-encoded x25519 public key
}
