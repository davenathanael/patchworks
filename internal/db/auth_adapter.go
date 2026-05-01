package db

import (
	"context"
	"errors"
	"time"

	"github.com/davenathanael/patchwork/pkg/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AuthAdapter wraps *DB and provides methods that satisfy pkg/auth's private interfaces.
// It translates between auth.User/auth.Session (string IDs) and core.User/core.Session (uuid IDs).
type AuthAdapter struct {
	db *DB
}

// NewAuthAdapter creates a new auth adapter for the given database.
func NewAuthAdapter(db *DB) *AuthAdapter {
	return &AuthAdapter{db: db}
}

// CreateSession creates a session, translating from string IDs to UUIDs.
func (a *AuthAdapter) CreateSession(ctx context.Context, id, userID string, expiresAt time.Time) (auth.Session, error) {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return auth.Session{}, err
	}
	userIDUUID, err := uuid.Parse(userID)
	if err != nil {
		return auth.Session{}, err
	}

	coreSession, err := a.db.CreateSession(ctx, idUUID, userIDUUID, expiresAt)
	if err != nil {
		return auth.Session{}, err
	}

	return auth.Session{
		ID:        coreSession.ID.String(),
		UserID:    coreSession.UserID.String(),
		ExpiresAt: coreSession.ExpiresAt,
	}, nil
}

// GetSessionByID retrieves a session by its ID, translating from string to UUID.
func (a *AuthAdapter) GetSessionByID(ctx context.Context, id string) (auth.Session, bool, error) {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return auth.Session{}, false, err
	}

	coreSession, found, err := a.db.GetSessionByID(ctx, idUUID)
	if err != nil || !found {
		return auth.Session{}, found, err
	}

	return auth.Session{
		ID:        coreSession.ID.String(),
		UserID:    coreSession.UserID.String(),
		ExpiresAt: coreSession.ExpiresAt,
	}, true, nil
}

// DeleteSession deletes a session by its ID, translating from string to UUID.
func (a *AuthAdapter) DeleteSession(ctx context.Context, id string) error {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return a.db.DeleteSession(ctx, idUUID)
}

// GetUserByID retrieves a user by their ID, translating from string to UUID.
func (a *AuthAdapter) GetUserByID(ctx context.Context, id string) (auth.User, bool, error) {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return auth.User{}, false, err
	}

	coreUser, err := a.db.GetUserByID(ctx, idUUID)
	if err != nil {
		// Check if it's a "not found" error; if so, return false with no error.
		// SQLC's GetUserById returns pgx.ErrNoRows on no results.
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, false, nil
		}
		return auth.User{}, false, err
	}

	return auth.User{
		ID:    coreUser.ID.String(),
		Email: coreUser.Email,
	}, true, nil
}

// UpsertUser inserts or updates a user by identity_id, translating UUIDs to strings.
func (a *AuthAdapter) UpsertUser(ctx context.Context, id, email, identityID string) (auth.User, error) {
	idUUID, err := uuid.Parse(id)
	if err != nil {
		return auth.User{}, err
	}

	coreUser, err := a.db.UpsertUser(ctx, idUUID, email, identityID)
	if err != nil {
		return auth.User{}, err
	}

	return auth.User{
		ID:    coreUser.ID.String(),
		Email: coreUser.Email,
	}, nil
}
