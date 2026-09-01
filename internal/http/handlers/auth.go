package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/ajg/form"
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

// renderLogin renders the login page, preserving values and errors.
func renderLogin(w http.ResponseWriter, f views.CredentialsForm) error {
	if err := views.LoginPage(f).Render(w); err != nil {
		return fmt.Errorf("render login page: %w", err)
	}
	return nil
}

// renderRegister renders the registration page, preserving values and errors.
func renderRegister(w http.ResponseWriter, f views.CredentialsForm) error {
	if err := views.RegisterPage(f).Render(w); err != nil {
		return fmt.Errorf("render register page: %w", err)
	}
	return nil
}

// handleGetLogin renders the login page.
func handleGetLogin() Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return renderLogin(w, views.CredentialsForm{})
	}
}

// handlePostLogin authenticates the user and redirects home.
func handlePostLogin(svc AuthLoginHandler) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var f views.CredentialsForm
		if err := form.NewDecoder(r.Body).Decode(&f); err != nil {
			f.Errors = views.FormErrors{"form": "invalid form data"}
			return renderLogin(w, f)
		}

		if errs := validateCredentials(f); len(errs) > 0 {
			f.Errors = errs
			return renderLogin(w, f)
		}

		err := svc.Login(w, r, f.Email, f.Password)
		if errors.Is(err, core.ErrInvalidCredentials) {
			f.Errors = views.FormErrors{"form": "invalid email or password"}
			return renderLogin(w, f)
		}
		if err != nil {
			return fmt.Errorf("login: %w", err)
		}

		http.Redirect(w, r, "/", http.StatusFound)
		return nil
	}
}

// handleGetRegister renders the registration page.
func handleGetRegister() Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return renderRegister(w, views.CredentialsForm{})
	}
}

// handlePostRegister creates a new user and redirects to login.
func handlePostRegister(svc AuthRegistrar) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var f views.CredentialsForm
		if err := form.NewDecoder(r.Body).Decode(&f); err != nil {
			f.Errors = views.FormErrors{"form": "invalid form data"}
			return renderRegister(w, f)
		}

		if errs := validateCredentials(f); len(errs) > 0 {
			f.Errors = errs
			return renderRegister(w, f)
		}

		_, err := svc.Register(r.Context(), f.Email, f.Password)
		if errors.Is(err, core.ErrEmailTaken) {
			f.Errors = views.FormErrors{"form": core.ErrEmailTaken.Error()}
			return renderRegister(w, f)
		}
		if err != nil {
			return fmt.Errorf("register: %w", err)
		}

		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return nil
	}
}

// handleGetLogout logs the user out.
func handleGetLogout(svc AuthLogoutHandler) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := svc.Logout(w, r); err != nil {
			return fmt.Errorf("logout: %w", err)
		}
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return nil
	}
}
