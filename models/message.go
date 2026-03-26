package models

import (
	"time"
)

// StoredInitialMessage holds an initial (session-opening, type 1) message
// for a recipient who is offline.
type StoredInitialMessage struct {
	ID          uint   `gorm:"primaryKey;autoIncrement"`
	RecipientID string `gorm:"type:text;index;not null"` // UUID of the recipient
	Payload     string `gorm:"type:text;not null"`       // Full JSON of the ConversationOpen message
	CreatedAt   time.Time
}

// StoredMessage holds a regular (type 2/3) message for a recipient who is offline.
type StoredMessage struct {
	ID          uint   `gorm:"primaryKey;autoIncrement"`
	RecipientID string `gorm:"type:text;index;not null"` // UUID of the recipient
	Payload     string `gorm:"type:text;not null"`       // Full JSON of the ConversationMessage/KeyRotation message
	CreatedAt   time.Time
}
