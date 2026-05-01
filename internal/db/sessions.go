package db

import (
	"context"
	"errors"
	"time"

	"github.com/davenathanael/patchwork/internal/core"
	"github.com/davenathanael/patchwork/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// CreateSession creates a new session in the database.
func (db *DB) CreateSession(ctx context.Context, id, userID uuid.UUID, expiresAt time.Time) (core.Session, error) {
	row, err := db.querier.CreateSession(ctx, sqlc.CreateSessionParams{
		ID:        id,
		UserID:    userID,
		ExpiresAt: pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return core.Session{}, err
	}
	return ToSession(row), nil
}

// GetSessionByID retrieves a session by its ID, returning (zero, false, nil) if not found.
func (db *DB) GetSessionByID(ctx context.Context, id uuid.UUID) (core.Session, bool, error) {
	row, err := db.querier.GetSessionById(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return core.Session{}, false, nil
	} else if err != nil {
		return core.Session{}, false, err
	}
	return ToSession(row), true, nil
}

// DeleteSession deletes a session by its ID.
func (db *DB) DeleteSession(ctx context.Context, id uuid.UUID) error {
	return db.querier.DeleteSession(ctx, id)
}
