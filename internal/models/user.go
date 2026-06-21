package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a registered user in the system.
type User struct {
	ID           uuid.UUID
	Name         string
	Email        string
	PasswordHash string
	Provider     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
