package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/davenathanael/patchwork/pkg/auth/oidc"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

const (
	stateParamKey      = "state"
	stateCookieName    = "oauth_state"
	verifierCookieName = "oauth_verifier"
	stateCookieMaxAge  = 300 // 5 minutes
	sessionDuration    = 30 * 24 * time.Hour
)

// sessionStore defines the methods needed to manage sessions.
// Implemented by AuthAdapter in internal/db.
type sessionStore interface {
	CreateSession(ctx context.Context, id, userID string, expiresAt time.Time) (Session, error)
	GetSessionByID(ctx context.Context, id string) (Session, bool, error)
	DeleteSession(ctx context.Context, id string) error
}

// userStore defines the methods needed to manage users.
// Implemented by AuthAdapter in internal/db.
type userStore interface {
	GetUserByID(ctx context.Context, id string) (User, bool, error)
	UpsertUser(ctx context.Context, id, email, identityID string) (User, error)
}

// oidcClient defines the OIDC provider interface.
// Implemented by oidc.Provider.
type oidcClient interface {
	AuthURL(state string) string
	AuthURLWithPKCE(state, verifier string) string
	Exchange(ctx context.Context, code string) (oidc.TokenClaims, error)
	ExchangeWithVerifier(ctx context.Context, code, verifier string) (oidc.TokenClaims, error)
}

// Service orchestrates OIDC authentication, user management, and session lifecycle.
type Service struct {
	oidc      oidcClient
	sessions  sessionStore
	users     userStore
	cookieCfg CookieConfig
}

// NewService creates a new auth service.
func NewService(oidc oidcClient, sessions sessionStore, users userStore, cookieCfg CookieConfig) *Service {
	return &Service{
		oidc:      oidc,
		sessions:  sessions,
		users:     users,
		cookieCfg: cookieCfg,
	}
}

// InitiateLogin generates CSRF state and PKCE verifier, stores them in cookies, and redirects to the OIDC provider.
func (s *Service) InitiateLogin(w http.ResponseWriter, r *http.Request) {
	state, err := generateState()
	if err != nil {
		http.Error(w, "failed to generate state", http.StatusInternalServerError)
		return
	}

	verifier := oauth2.GenerateVerifier()

	// Store state in a short-lived cookie
	stateCookie := &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Path:     "/",
		MaxAge:   stateCookieMaxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, stateCookie)

	// Store verifier in a short-lived cookie
	verifierCookie := &http.Cookie{
		Name:     verifierCookieName,
		Value:    verifier,
		Path:     "/",
		MaxAge:   stateCookieMaxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, verifierCookie)

	// Redirect to OIDC authorization endpoint with PKCE
	authURL := s.oidc.AuthURLWithPKCE(state, verifier)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// HandleCallback validates the state, exchanges the code for tokens, upserts the user, creates a session, and sets the session cookie.
// Returns the authenticated user and session on success.
func (s *Service) HandleCallback(w http.ResponseWriter, r *http.Request) (User, Session, error) {
	// Read state from query param
	stateParam := r.URL.Query().Get(stateParamKey)
	if stateParam == "" {
		return User{}, Session{}, fmt.Errorf("missing state param")
	}

	// Read state from cookie
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil {
		return User{}, Session{}, fmt.Errorf("missing state cookie: %w", err)
	}

	// Validate state matches
	if stateParam != stateCookie.Value {
		return User{}, Session{}, fmt.Errorf("state mismatch")
	}

	// Read PKCE verifier from cookie
	verifierCookie, err := r.Cookie(verifierCookieName)
	if err != nil {
		return User{}, Session{}, fmt.Errorf("missing PKCE verifier cookie: %w", err)
	}

	// Clear the state and verifier cookies immediately
	DeleteStateCookie(w)
	DeleteVerifierCookie(w)

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		return User{}, Session{}, fmt.Errorf("missing authorization code")
	}

	// Exchange code for tokens with PKCE verifier
	claims, err := s.oidc.ExchangeWithVerifier(r.Context(), code, verifierCookie.Value)
	if err != nil {
		return User{}, Session{}, fmt.Errorf("code exchange failed: %w", err)
	}

	// Upsert user in database
	userID := uuid.New().String()
	user, err := s.users.UpsertUser(r.Context(), userID, claims.Email, claims.Subject)
	if err != nil {
		return User{}, Session{}, fmt.Errorf("upsert user failed: %w", err)
	}

	// Create session in database
	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(sessionDuration)
	session, err := s.sessions.CreateSession(r.Context(), sessionID, user.ID, expiresAt)
	if err != nil {
		return User{}, Session{}, fmt.Errorf("create session failed: %w", err)
	}

	// Set session cookie
	if err := SetSessionCookie(w, s.cookieCfg, session.ID); err != nil {
		return User{}, Session{}, fmt.Errorf("set session cookie failed: %w", err)
	}

	return user, session, nil
}

// Logout deletes the session from the database and clears the session cookie.
func (s *Service) Logout(w http.ResponseWriter, r *http.Request) error {
	// Get session ID from cookie
	sessionID, err := GetSessionCookie(r, s.cookieCfg)
	if err != nil {
		// No session cookie is fine for logout
		DeleteSessionCookie(w)
		return nil
	}

	// Delete session from database
	if err := s.sessions.DeleteSession(r.Context(), sessionID); err != nil {
		return fmt.Errorf("delete session failed: %w", err)
	}

	// Clear session cookie
	DeleteSessionCookie(w)
	return nil
}

// GetUserFromCookie retrieves the authenticated user from the session cookie.
// Returns (zero User, false) if no valid session exists.
func (s *Service) GetUserFromCookie(r *http.Request) (User, bool) {
	// Get session ID from cookie
	sessionID, err := GetSessionCookie(r, s.cookieCfg)
	if err != nil {
		return User{}, false
	}

	// Fetch session from database
	session, found, err := s.sessions.GetSessionByID(r.Context(), sessionID)
	if err != nil || !found {
		return User{}, false
	}

	// Check expiry
	if session.IsExpired() {
		// Clean up expired session (non-blocking, ignore errors)
		_ = s.sessions.DeleteSession(r.Context(), session.ID)
		return User{}, false
	}

	// Fetch user from database
	user, found, err := s.users.GetUserByID(r.Context(), session.UserID)
	if err != nil || !found {
		return User{}, false
	}

	return user, true
}

// generateState creates a random base64url-encoded state string for CSRF protection.
func generateState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// DeleteStateCookie clears the CSRF state cookie.
func DeleteStateCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     stateCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

// DeleteVerifierCookie clears the PKCE verifier cookie.
func DeleteVerifierCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     verifierCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}
