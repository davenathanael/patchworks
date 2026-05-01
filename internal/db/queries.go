package db

import (
	"context"

	"github.com/davenathanael/patchwork/internal/core"
	"github.com/davenathanael/patchwork/internal/db/sqlc"
	"github.com/google/uuid"
)

// GetUserByID retrieves a user by their ID.
func (db *DB) GetUserByID(ctx context.Context, id uuid.UUID) (core.User, error) {
	row, err := db.querier.GetUserById(ctx, id)
	if err != nil {
		return core.User{}, err
	}
	return ToUser(row), nil
}

// UpsertUser inserts or updates a user by identity_id, returning the user.
func (db *DB) UpsertUser(ctx context.Context, id uuid.UUID, email, identityID string) (core.User, error) {
	row, err := db.querier.UpsertUser(ctx, sqlc.UpsertUserParams{
		ID:         id,
		Email:      email,
		IdentityID: identityID,
	})
	if err != nil {
		return core.User{}, err
	}
	return ToUser(row), nil
}
