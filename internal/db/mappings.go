package db

import (
	"github.com/davenathanael/patchwork/internal/core"
	"github.com/davenathanael/patchwork/internal/db/sqlc"
)

// ToUser converts a sqlc.User row to a core.User struct.
func ToUser(row sqlc.User) core.User {
	return core.User{
		ID:    row.ID,
		Email: row.Email,
	}
}

// ToSession converts a sqlc.Session row to a core.Session struct.
func ToSession(row sqlc.Session) core.Session {
	return core.Session{
		ID:        row.ID,
		UserID:    row.UserID,
		ExpiresAt: row.ExpiresAt.Time,
	}
}
