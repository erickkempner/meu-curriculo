package models

import (
	"time"

	"github.com/google/uuid"
)

// Session represents an active user session stored server-side in PostgreSQL.
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	CSRFToken string
	ExpiresAt time.Time
	CreatedAt time.Time
}
