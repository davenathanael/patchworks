package db

import (
	"context"

	"github.com/davenathanael/patchwork/internal/core"
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
