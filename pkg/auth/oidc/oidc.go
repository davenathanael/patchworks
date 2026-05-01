package oidc

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// TokenClaims holds the claims extracted from an OIDC ID token.
type TokenClaims struct {
	Subject string
	Email   string
}

// OIDCClient defines the interface for OIDC operations.
// Implemented by Provider.
type OIDCClient interface {
	AuthURL(state string) string
	AuthURLWithPKCE(state, verifier string) string
	Exchange(ctx context.Context, code string) (TokenClaims, error)
	ExchangeWithVerifier(ctx context.Context, code, verifier string) (TokenClaims, error)
}

// Provider wraps go-oidc and oauth2 for OIDC authentication.
type Provider struct {
	provider *oidc.Provider
	config   oauth2.Config
	verifier *oidc.IDTokenVerifier
}

// NewProvider initializes an OIDC provider by fetching the discovery document.
// issuerURL should be the base URL (e.g., "https://pocketid.localhost") without trailing path.
// go-oidc automatically appends /.well-known/openid-configuration.
func NewProvider(ctx context.Context, issuerURL, clientID, clientSecret, redirectURL string) (*Provider, error) {
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("oidc provider init: %w", err)
	}

	cfg := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: clientID,
	})

	return &Provider{
		provider: provider,
		config:   cfg,
		verifier: verifier,
	}, nil
}

// AuthURL returns the authorization URL for redirecting the user to the OIDC provider.
func (p *Provider) AuthURL(state string) string {
	return p.config.AuthCodeURL(state)
}

// AuthURLWithPKCE returns the authorization URL with PKCE code challenge.
func (p *Provider) AuthURLWithPKCE(state, verifier string) string {
	return p.config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
}

// Exchange exchanges an authorization code for an ID token and extracts claims.
func (p *Provider) Exchange(ctx context.Context, code string) (TokenClaims, error) {
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return TokenClaims{}, fmt.Errorf("code exchange: %w", err)
	}

	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		return TokenClaims{}, fmt.Errorf("id_token not in token response")
	}

	verifiedToken, err := p.verifier.Verify(ctx, idToken)
	if err != nil {
		return TokenClaims{}, fmt.Errorf("token verification: %w", err)
	}

	var claims struct {
		Subject string `json:"sub"`
		Email   string `json:"email"`
	}
	if err := verifiedToken.Claims(&claims); err != nil {
		return TokenClaims{}, fmt.Errorf("claims extraction: %w", err)
	}

	return TokenClaims{
		Subject: claims.Subject,
		Email:   claims.Email,
	}, nil
}

// ExchangeWithVerifier exchanges an authorization code for an ID token using PKCE verifier.
func (p *Provider) ExchangeWithVerifier(ctx context.Context, code, verifier string) (TokenClaims, error) {
	token, err := p.config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return TokenClaims{}, fmt.Errorf("code exchange: %w", err)
	}

	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		return TokenClaims{}, fmt.Errorf("id_token not in token response")
	}

	verifiedToken, err := p.verifier.Verify(ctx, idToken)
	if err != nil {
		return TokenClaims{}, fmt.Errorf("token verification: %w", err)
	}

	var claims struct {
		Subject string `json:"sub"`
		Email   string `json:"email"`
	}
	if err := verifiedToken.Claims(&claims); err != nil {
		return TokenClaims{}, fmt.Errorf("claims extraction: %w", err)
	}

	return TokenClaims{
		Subject: claims.Subject,
		Email:   claims.Email,
	}, nil
}
