package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/davenathanael/patchwork/internal/core"
	"github.com/davenathanael/patchwork/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// GetUserByID retrieves a user by their ID.
func (db *DB) GetUserByID(ctx context.Context, id uuid.UUID) (core.User, error) {
	row, err := db.querier.GetUserById(ctx, id)
	if err != nil {
		return core.User{}, err
	}
	return toUser(row), nil
}

// GetUserByEmail retrieves a user and their password hash by email.
// Returns (zero User, "", false, nil) when no user matches.
func (db *DB) GetUserByEmail(ctx context.Context, email string) (core.User, string, bool, error) {
	row, err := db.querier.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return core.User{}, "", false, nil
	}
	if err != nil {
		return core.User{}, "", false, err
	}
	return toUser(row), row.PasswordHash.String, true, nil
}

// CreateUser creates a new user with the given password hash.
func (db *DB) CreateUser(ctx context.Context, email, passwordHash string) (core.User, error) {
	row, err := db.querier.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return core.User{}, fmt.Errorf("create user: %w: %v", core.ErrEmailTaken, pgErr)
		}
		return core.User{}, fmt.Errorf("create user: %w", err)
	}
	return toUser(row), nil
}

// TouchLastLogin records the timestamp of a successful login.
func (db *DB) TouchLastLogin(ctx context.Context, userID uuid.UUID) error {
	return db.querier.SetUserLastLoginAt(ctx, userID)
}
