package components

import (
	"context"

	"github.com/davenathanael/patchwork/internal/config"
	"github.com/davenathanael/patchwork/internal/db"
)

// Components holds the application dependencies (DB connections, HTTP clients, Configs, etc).
type Components struct {
	DB     *db.DB
	Config config.AppConfig
}

// New builds a new Components instance and starts all dependencies.
func New(ctx context.Context) (*Components, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	db, err := db.New(ctx, cfg.DB.URL)
	if err != nil {
		return nil, err
	}

	return &Components{
		DB:     db,
		Config: cfg,
	}, nil
}

// Close shuts down all dependencies and releases resources.
func (c *Components) Close(ctx context.Context) error {
	if c.DB != nil {
		c.DB.Close()
	}

	return nil
}
