package components

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/davenathanael/patchwork/internal/auth"
	"github.com/davenathanael/patchwork/internal/config"
	"github.com/davenathanael/patchwork/internal/db"
	"github.com/davenathanael/patchwork/internal/http/client"
)

// Components holds the application dependencies (DB connections, HTTP clients, Configs, etc).
type Components struct {
	DB          *db.DB
	HTTPClient  *client.Client
	Config      config.AppConfig
	AuthService *auth.Service
}

// New builds a new Components instance and starts all dependencies.
func New(ctx context.Context) (*Components, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	database, err := db.New(ctx, cfg.DB.URL)
	if err != nil {
		return nil, err
	}

	authSvc, err := createAuthService(cfg, database)
	if err != nil {
		return nil, err
	}

	return &Components{
		DB:          database,
		HTTPClient:  client.New(),
		Config:      cfg,
		AuthService: authSvc,
	}, nil
}

// Close shuts down all dependencies and releases resources.
func (c *Components) Close(ctx context.Context) error {
	if c.DB != nil {
		c.DB.Close()
	}

	return nil
}

func createAuthService(cfg config.AppConfig, database *db.DB) (*auth.Service, error) {
	cookieKey, err := decodeSessionKey(cfg.Session.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("session encryption key: %w", err)
	}

	return auth.NewService(
		database,
		database,
		auth.CookieConfig{Key: cookieKey},
	), nil
}

// decodeSessionKey base64-decodes the session encryption key and requires
// exactly 32 bytes (AES-256). Fails fast at boot instead of on first login.
func decodeSessionKey(raw string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("must decode to 32 bytes (AES-256), got %d", len(key))
	}
	return key, nil
}
