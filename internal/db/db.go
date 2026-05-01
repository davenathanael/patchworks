package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxuuid "github.com/vgarvardt/pgx-google-uuid/v5"

	"github.com/davenathanael/patchwork/internal/db/sqlc"
)

// DB holds the database connection pool and provides query execution.
type DB struct {
	Pool    *pgxpool.Pool
	querier *sqlc.Queries
}

// New creates a new DB instance with the given context and database URL.
func New(ctx context.Context, url string) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	poolConfig.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		pgxuuid.Register(c.TypeMap())
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	return &DB{
		Pool:    pool,
		querier: sqlc.New(pool),
	}, nil
}

// Close closes the database connection pool.
// This is a blocking operation, it waits until all connections are closed in the pool.
func (db *DB) Close() {
	db.Pool.Close()
}
