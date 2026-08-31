package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/carlmjohnson/be"
	"github.com/davenathanael/patchwork/internal/core"
	"github.com/google/uuid"
)

func TestRegisterCreatesUser(t *testing.T) {
	svc, f, _ := newTestService(t)

	user, err := svc.Register(context.Background(), "a@b.c", "password123")
	be.NilErr(t, err)
	be.Equal(t, "a@b.c", user.Email)
	be.True(t, user.ID != uuid.Nil)

	stored, ok := f.users["a@b.c"]
	be.True(t, ok)
	be.Equal(t, user.ID, stored.ID)
	// Stored value is a hash, not the plaintext, and the hash verifies.
	be.Unequal(t, "password123", f.hashes["a@b.c"])
	be.True(t, verifyPassword("password123", f.hashes["a@b.c"]))
}

func TestLoginSuccess(t *testing.T) {
	svc, f, _ := newTestService(t)
	u := f.seedUser(t, "a@b.c", "swordfish")
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", nil)

	err := svc.Login(w, r, "a@b.c", "swordfish")
	be.NilErr(t, err)

	cookies := w.Result().Cookies()
	be.Equal(t, 1, len(cookies))
	be.Equal(t, cookieName, cookies[0].Name)

	be.Equal(t, 1, len(f.sessions))
	for _, s := range f.sessions {
		be.Equal(t, u.ID, s.UserID)
		be.True(t, s.ExpiresAt.After(time.Now()))
	}
}

func TestLoginRejectsBadCredentials(t *testing.T) {
	svc, f, _ := newTestService(t)
	f.seedUser(t, "a@b.c", "right-password")

	// Wrong password and unknown user must produce the same error (no user enumeration).
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", nil)
	wrongPass := svc.Login(rec1, req1, "a@b.c", "wrong")

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", nil)
	unknownUser := svc.Login(rec2, req2, "nobody@x.y", "whatever")

	be.Equal(t, ErrInvalidCredentials, wrongPass)
	be.Equal(t, ErrInvalidCredentials, unknownUser)
	be.Equal(t, 0, len(f.sessions)) // no session created on failure
}

func TestLoginStoreErrorPropagates(t *testing.T) {
	svc, f, _ := newTestService(t)
	f.err = errors.New("db down")
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", nil)

	err := svc.Login(w, r, "a@b.c", "x")
	be.Nonzero(t, err)
	be.Unequal(t, ErrInvalidCredentials, err) // must not masquerade as bad creds
}

func TestLogoutDeletesSession(t *testing.T) {
	svc, f, cfg := newTestService(t)
	sessionID := uuid.New()
	_, _ = f.CreateSession(context.Background(), sessionID, uuid.New(), time.Now().Add(time.Hour))

	r := authedRequest(sessionID.String(), cfg)
	err := svc.Logout(httptest.NewRecorder(), r)
	be.NilErr(t, err)
	be.AllEqual(t, []uuid.UUID{sessionID}, f.deleted)
}

func TestLogoutWithoutCookieIsNoop(t *testing.T) {
	svc, _, _ := newTestService(t)
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

	err := svc.Logout(httptest.NewRecorder(), r)
	be.NilErr(t, err)
}

func TestLogoutWithGarbageCookieIsNoop(t *testing.T) {
	svc, f, cfg := newTestService(t)
	r := authedRequest("not-a-uuid", cfg)

	err := svc.Logout(httptest.NewRecorder(), r)
	be.NilErr(t, err)
	be.Equal(t, 0, len(f.deleted))
}

func TestGetUserFromCookieValid(t *testing.T) {
	svc, f, cfg := newTestService(t)
	u := f.seedUser(t, "a@b.c", "pw")
	s := core.Session{ID: uuid.New(), UserID: u.ID, ExpiresAt: time.Now().Add(time.Hour)}
	f.sessions[s.ID] = s

	user, ok := svc.GetUserFromCookie(authedRequest(s.ID.String(), cfg))
	be.True(t, ok)
	be.Equal(t, u.ID, user.ID)
}

func TestGetUserFromCookieExpiredDeletesSession(t *testing.T) {
	svc, f, cfg := newTestService(t)
	s := core.Session{ID: uuid.New(), UserID: uuid.New(), ExpiresAt: time.Now().Add(-time.Hour)}
	f.sessions[s.ID] = s

	_, ok := svc.GetUserFromCookie(authedRequest(s.ID.String(), cfg))
	be.False(t, ok)
	be.AllEqual(t, []uuid.UUID{s.ID}, f.deleted) // expired session cleaned up
}

func TestGetUserFromCookieMissingSession(t *testing.T) {
	svc, f, cfg := newTestService(t)
	r := authedRequest(uuid.New().String(), cfg)

	_, ok := svc.GetUserFromCookie(r)
	be.False(t, ok)
	be.Equal(t, 0, len(f.deleted))
}

func TestGetUserFromCookieMissingUser(t *testing.T) {
	svc, f, cfg := newTestService(t)
	// Session exists but points at a user that no longer does.
	s := core.Session{ID: uuid.New(), UserID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour)}
	f.sessions[s.ID] = s

	_, ok := svc.GetUserFromCookie(authedRequest(s.ID.String(), cfg))
	be.False(t, ok)
}

func TestGetUserFromCookieNoCookie(t *testing.T) {
	svc, _, _ := newTestService(t)
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

	_, ok := svc.GetUserFromCookie(r)
	be.False(t, ok)
}

// --- fakes & helpers ---

// fakeStores implements the auth service's private store interfaces.
type fakeStores struct {
	users    map[string]core.User    // email -> user
	hashes   map[string]string       // email -> password hash
	byID     map[uuid.UUID]core.User // id -> user
	sessions map[uuid.UUID]core.Session
	deleted  []uuid.UUID
	err      error // injected store error, returned by every read
}

func newFakeStores() *fakeStores {
	return &fakeStores{
		users:    map[string]core.User{},
		hashes:   map[string]string{},
		byID:     map[uuid.UUID]core.User{},
		sessions: map[uuid.UUID]core.Session{},
	}
}

func (f *fakeStores) CreateUser(ctx context.Context, email, passwordHash string) (core.User, error) {
	u := core.User{ID: uuid.New(), Email: email}
	f.users[email] = u
	f.hashes[email] = passwordHash
	f.byID[u.ID] = u
	return u, nil
}

func (f *fakeStores) GetUserByEmail(ctx context.Context, email string) (core.User, string, bool, error) {
	if f.err != nil {
		return core.User{}, "", false, f.err
	}
	u, ok := f.users[email]
	if !ok {
		return core.User{}, "", false, nil
	}
	return u, f.hashes[email], true, nil
}

func (f *fakeStores) GetUserByID(ctx context.Context, id uuid.UUID) (core.User, error) {
	if f.err != nil {
		return core.User{}, f.err
	}
	u, ok := f.byID[id]
	if !ok {
		return core.User{}, errors.New("user not found")
	}
	return u, nil
}

func (f *fakeStores) CreateSession(ctx context.Context, id, userID uuid.UUID, expiresAt time.Time) (core.Session, error) {
	s := core.Session{ID: id, UserID: userID, ExpiresAt: expiresAt}
	f.sessions[id] = s
	return s, nil
}

func (f *fakeStores) GetSessionByID(ctx context.Context, id uuid.UUID) (core.Session, bool, error) {
	if f.err != nil {
		return core.Session{}, false, f.err
	}
	s, ok := f.sessions[id]
	return s, ok, nil
}

func (f *fakeStores) DeleteSession(ctx context.Context, id uuid.UUID) error {
	delete(f.sessions, id)
	f.deleted = append(f.deleted, id)
	return nil
}

// seedUser registers a user directly in the fake store.
func (f *fakeStores) seedUser(t *testing.T, email, password string) core.User {
	t.Helper()
	hash, err := hashPassword(password)
	be.NilErr(t, err)
	u := core.User{ID: uuid.New(), Email: email}
	f.users[email] = u
	f.hashes[email] = hash
	f.byID[u.ID] = u
	return u
}

// newTestService returns a service wired to a fresh fake store.
func newTestService(t *testing.T) (*Service, *fakeStores, CookieConfig) {
	f := newFakeStores()
	cfg := CookieConfig{Key: testKey(t)}
	return NewService(f, f, cfg), f, cfg
}

// authedRequest returns a request carrying an encrypted session cookie.
func authedRequest(sessionID string, cfg CookieConfig) *http.Request {
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	enc, err := encryptAES(cfg.Key, sessionID)
	if err != nil {
		panic(err)
	}
	r.AddCookie(&http.Cookie{
		Name:     cookieName,
		Value:    enc,
		Path:     cookiePath,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	return r
}
