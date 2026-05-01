package core

import (
	"time"

	"github.com/google/uuid"
)

// Session represents a database-backed session with UUID identifiers.
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
}

// IsExpired reports whether the session has expired.
func (s Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
