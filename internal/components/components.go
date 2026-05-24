package components

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/davenathanael/patchwork/internal/config"
	"github.com/davenathanael/patchwork/internal/db"
	"github.com/davenathanael/patchwork/internal/http/client"
	"github.com/davenathanael/patchwork/pkg/auth"
	"github.com/davenathanael/patchwork/pkg/auth/oidc"
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

	authSvc, err := createOIDCAuthService(ctx, cfg, database)
	if err != nil {
		return nil, err
	}

	// Initialize OIDC provider
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

func createOIDCAuthService(ctx context.Context, cfg config.AppConfig, database *db.DB) (*auth.Service, error) {
	oidcProvider, err := oidc.NewProvider(ctx,
		cfg.OIDC.IssuerURL,
		cfg.OIDC.ClientID,
		cfg.OIDC.ClientSecret,
		cfg.OIDC.RedirectURL,
	)
	if err != nil {
		return nil, fmt.Errorf("oidc provider init: %w", err)
	}

	// Decode session encryption key from base64
	cookieKey, err := base64.StdEncoding.DecodeString(cfg.Session.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decode session encryption key: %w", err)
	}

	// Create auth adapter (implements auth service's private interfaces)
	adapter := db.NewAuthAdapter(database)

	return auth.NewService(
		oidcProvider,
		adapter,
		adapter,
		auth.CookieConfig{Key: cookieKey},
	), nil
}
