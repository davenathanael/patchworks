package auth

import "time"

// Session represents an authenticated session.
type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
}

// IsExpired reports whether the session has expired.
func (s Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
