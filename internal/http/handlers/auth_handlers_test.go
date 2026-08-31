package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/carlmjohnson/be"
	"github.com/davenathanael/patchwork/internal/auth"
	"github.com/davenathanael/patchwork/internal/core"
)

func TestGetLoginPage(t *testing.T) {
	rec := httptest.NewRecorder()
	handleGetLogin().ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/login", nil))

	be.Equal(t, http.StatusOK, rec.Code)
}

func TestPostLoginSuccess(t *testing.T) {
	svc := &fakeAuth{}
	rec := httptest.NewRecorder()
	handlePostLogin(svc).ServeHTTP(rec, mustFormRequest(t, "email=a%40b.c&password=pw"))

	be.Equal(t, http.StatusFound, rec.Code)
	be.Equal(t, "/", rec.Header().Get("Location"))
	be.Equal(t, "a@b.c", svc.loginEmail)
}

func TestPostLoginInvalidCredentials(t *testing.T) {
	svc := &fakeAuth{loginErr: auth.ErrInvalidCredentials}
	rec := httptest.NewRecorder()
	handlePostLogin(svc).ServeHTTP(rec, mustFormRequest(t, "email=a%40b.c&password=wrong"))

	be.Equal(t, http.StatusUnauthorized, rec.Code)
	be.True(t, containsBody(rec, "invalid email or password"))
	be.False(t, containsBody(rec, "boom")) // no internal error details leaked
}

func TestPostLoginServiceError(t *testing.T) {
	svc := &fakeAuth{loginErr: errors.New("db down")}
	rec := httptest.NewRecorder()
	handlePostLogin(svc).ServeHTTP(rec, mustFormRequest(t, "email=a%40b.c&password=pw"))

	be.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestPostLoginEmptyCredentials(t *testing.T) {
	svc := &fakeAuth{}
	rec := httptest.NewRecorder()
	handlePostLogin(svc).ServeHTTP(rec, mustFormRequest(t, "email=&password="))

	be.Equal(t, http.StatusBadRequest, rec.Code)
	be.Equal(t, "", svc.loginEmail) // Login not called
}

func TestPostRegisterEmptyCredentials(t *testing.T) {
	svc := &fakeAuth{}
	rec := httptest.NewRecorder()
	handlePostRegister(svc).ServeHTTP(rec, mustFormRequest(t, "email=&password="))

	be.Equal(t, http.StatusBadRequest, rec.Code)
	be.Equal(t, 0, len(svc.registered))
}

func TestPostRegisterSuccess(t *testing.T) {
	svc := &fakeAuth{}
	rec := httptest.NewRecorder()
	handlePostRegister(svc).ServeHTTP(rec, mustFormRequest(t, "email=a%40b.c&password=pw"))

	be.Equal(t, http.StatusFound, rec.Code)
	be.Equal(t, "/auth/login", rec.Header().Get("Location"))
	be.AllEqual(t, []string{"a@b.c"}, svc.registered)
}

func TestPostRegisterDuplicateEmail(t *testing.T) {
	svc := &fakeAuth{registerErr: core.ErrEmailTaken}
	rec := httptest.NewRecorder()
	handlePostRegister(svc).ServeHTTP(rec, mustFormRequest(t, "email=a%40b.c&password=pw"))

	be.Equal(t, http.StatusBadRequest, rec.Code)
	be.True(t, containsBody(rec, "already registered"))
}

func TestPostRegisterServiceError(t *testing.T) {
	svc := &fakeAuth{registerErr: errors.New("db down")}
	rec := httptest.NewRecorder()
	handlePostRegister(svc).ServeHTTP(rec, mustFormRequest(t, "email=a%40b.c&password=pw"))

	be.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetLogout(t *testing.T) {
	svc := &fakeAuth{}
	rec := httptest.NewRecorder()
	handleGetLogout(svc).ServeHTTP(rec, mustAuthedRequest(t, http.MethodGet, "/auth/logout", nil))

	be.Equal(t, http.StatusFound, rec.Code)
	be.Equal(t, "/auth/login", rec.Header().Get("Location"))
	be.Equal(t, 1, svc.logoutCalls)
}

// --- fakes & helpers ---

type fakeAuth struct {
	registerErr error
	loginErr    error
	logoutErr   error
	registered  []string
	loginEmail  string
	loginPass   string
	logoutCalls int
}

func (f *fakeAuth) Register(ctx context.Context, email, password string) (core.User, error) {
	f.registered = append(f.registered, email)
	return core.User{ID: testUser.ID, Email: email}, f.registerErr
}

func (f *fakeAuth) Login(w http.ResponseWriter, r *http.Request, email, password string) error {
	f.loginEmail = email
	f.loginPass = password
	return f.loginErr
}

func (f *fakeAuth) Logout(w http.ResponseWriter, r *http.Request) error {
	f.logoutCalls++
	return f.logoutErr
}

func containsBody(rec *httptest.ResponseRecorder, s string) bool {
	return strings.Contains(rec.Body.String(), s)
}
