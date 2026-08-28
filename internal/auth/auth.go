package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/davenathanael/patchwork/internal/core"
	"github.com/google/uuid"
)

const sessionDuration = 30 * 24 * time.Hour

// ErrInvalidCredentials is returned when login fails due to a bad email or password.
var ErrInvalidCredentials = errors.New("invalid email or password")

// sessionStore defines the methods needed to manage sessions.
// Implemented by *db.DB.
type sessionStore interface {
	CreateSession(ctx context.Context, id, userID uuid.UUID, expiresAt time.Time) (core.Session, error)
	GetSessionByID(ctx context.Context, id uuid.UUID) (core.Session, bool, error)
	DeleteSession(ctx context.Context, id uuid.UUID) error
}

// userStore defines the methods needed to manage users.
// Implemented by *db.DB.
type userStore interface {
	CreateUser(ctx context.Context, email, passwordHash string) (core.User, error)
	GetUserByEmail(ctx context.Context, email string) (core.User, string, bool, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (core.User, error)
}

// Service orchestrates registration, password login, sessions, and logout.
type Service struct {
	sessions  sessionStore
	users     userStore
	cookieCfg CookieConfig
}

// NewService creates a new auth service.
func NewService(sessions sessionStore, users userStore, cookieCfg CookieConfig) *Service {
	return &Service{
		sessions:  sessions,
		users:     users,
		cookieCfg: cookieCfg,
	}
}

// Register creates a new user with a hashed password.
func (s *Service) Register(ctx context.Context, email, password string) (core.User, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return core.User{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.CreateUser(ctx, email, hash)
	if err != nil {
		return core.User{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

// Login verifies credentials, creates a session, and sets the session cookie.
func (s *Service) Login(w http.ResponseWriter, r *http.Request, email, password string) error {
	user, passwordHash, found, err := s.users.GetUserByEmail(r.Context(), email)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if !found || !verifyPassword(password, passwordHash) {
		return ErrInvalidCredentials
	}

	sessionID := uuid.New()
	expiresAt := time.Now().Add(sessionDuration)
	session, err := s.sessions.CreateSession(r.Context(), sessionID, user.ID, expiresAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	if err := SetSessionCookie(w, s.cookieCfg, session.ID.String()); err != nil {
		return fmt.Errorf("set session cookie: %w", err)
	}
	return nil
}

// Logout deletes the session from the database and clears the session cookie.
func (s *Service) Logout(w http.ResponseWriter, r *http.Request) error {
	sessionID, err := GetSessionCookie(r, s.cookieCfg)
	if err != nil {
		// No session cookie is fine for logout.
		DeleteSessionCookie(w)
		return nil
	}

	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		DeleteSessionCookie(w)
		return nil
	}

	if err := s.sessions.DeleteSession(r.Context(), sessionUUID); err != nil {
		return fmt.Errorf("delete session failed: %w", err)
	}

	DeleteSessionCookie(w)
	return nil
}

// GetUserFromCookie retrieves the authenticated user from the session cookie.
// Returns (zero User, false) if no valid session exists.
func (s *Service) GetUserFromCookie(r *http.Request) (core.User, bool) {
	sessionID, err := GetSessionCookie(r, s.cookieCfg)
	if err != nil {
		return core.User{}, false
	}

	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return core.User{}, false
	}

	session, found, err := s.sessions.GetSessionByID(r.Context(), sessionUUID)
	if err != nil || !found {
		return core.User{}, false
	}

	if session.IsExpired() {
		// Clean up expired session (non-blocking, ignore errors).
		_ = s.sessions.DeleteSession(r.Context(), session.ID)
		return core.User{}, false
	}

	user, err := s.users.GetUserByID(r.Context(), session.UserID)
	if err != nil {
		return core.User{}, false
	}

	return user, true
}
