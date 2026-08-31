package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/ajg/form"
	"github.com/davenathanael/patchwork/internal/auth"
	"github.com/davenathanael/patchwork/internal/core"
	"github.com/davenathanael/patchwork/internal/http/views"
)

// AuthRegistrar registers new users.
type AuthRegistrar interface {
	Register(ctx context.Context, email, password string) (core.User, error)
}

// AuthLoginHandler logs users in.
type AuthLoginHandler interface {
	Login(w http.ResponseWriter, r *http.Request, email, password string) error
}

// AuthLogoutHandler logs users out.
type AuthLogoutHandler interface {
	Logout(w http.ResponseWriter, r *http.Request) error
}

type credentialsForm struct {
	Email    string `form:"email"`
	Password string `form:"password"`
}

// handleGetLogin renders the login page.
func handleGetLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := views.LoginPage().Render(w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// handlePostLogin authenticates the user and redirects home.
func handlePostLogin(svc AuthLoginHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var f credentialsForm
		if err := form.NewDecoder(r.Body).Decode(&f); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if f.Email == "" || f.Password == "" {
			http.Error(w, "email and password are required", http.StatusBadRequest)
			return
		}

		err := svc.Login(w, r, f.Email, f.Password)
		if errors.Is(err, auth.ErrInvalidCredentials) {
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// handleGetRegister renders the registration page.
func handleGetRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := views.RegisterPage().Render(w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// handlePostRegister creates a new user and redirects to login.
func handlePostRegister(svc AuthRegistrar) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var f credentialsForm
		if err := form.NewDecoder(r.Body).Decode(&f); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if f.Email == "" || f.Password == "" {
			http.Error(w, "email and password are required", http.StatusBadRequest)
			return
		}

		if _, err := svc.Register(r.Context(), f.Email, f.Password); err != nil {
			if errors.Is(err, core.ErrEmailTaken) {
				http.Error(w, core.ErrEmailTaken.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/auth/login", http.StatusFound)
	}
}

// handleGetLogout logs the user out.
func handleGetLogout(svc AuthLogoutHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := svc.Logout(w, r); err != nil {
			http.Error(w, "logout failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/auth/login", http.StatusFound)
	}
}
