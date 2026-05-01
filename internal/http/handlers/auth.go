package handlers

import (
	"net/http"

	"github.com/davenathanael/patchwork/internal/core"
	"github.com/davenathanael/patchwork/pkg/auth"
	"github.com/google/uuid"
)

// AuthLoginInitiator is the interface for initiating login.
type AuthLoginInitiator interface {
	InitiateLogin(w http.ResponseWriter, r *http.Request)
}

// AuthCallbackHandler is the interface for handling OAuth callbacks.
type AuthCallbackHandler interface {
	HandleCallback(w http.ResponseWriter, r *http.Request) (auth.User, auth.Session, error)
}

// AuthLogoutHandler is the interface for logout.
type AuthLogoutHandler interface {
	Logout(w http.ResponseWriter, r *http.Request) error
}

// handleGetLogin initiates the login flow by redirecting to the OIDC provider.
func handleGetLogin(svc AuthLoginInitiator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.InitiateLogin(w, r)
	}
}

// handleGetCallback handles the OAuth callback, validates the state, and exchanges the code.
func handleGetCallback(svc AuthCallbackHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _, err := svc.HandleCallback(w, r)
		if err != nil {
			http.Error(w, "authentication failed: "+err.Error(), http.StatusUnauthorized)
			return
		}
		// If no error, HandleCallback has already set the session cookie and we should redirect
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// handleGetLogout logs out the user and clears the session.
func handleGetLogout(svc AuthLogoutHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := svc.Logout(w, r); err != nil {
			http.Error(w, "logout failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/auth/login", http.StatusFound)
	}
}

func authToCoreUser(authUser auth.User) core.User {
	id, _ := uuid.Parse(authUser.ID)
	return core.User{
		ID:    id,
		Email: authUser.Email,
	}
}
