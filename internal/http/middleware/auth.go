package middleware

import (
	"context"
	"net/http"

	"github.com/davenathanael/patchwork/internal/core"
)

type contextKey struct{}

var userContextKey contextKey

// SessionReader is the interface that auth services must satisfy for middleware.
type SessionReader interface {
	GetUserFromCookie(r *http.Request) (core.User, bool)
}

// Auth returns a middleware that reads the session cookie, validates it, and injects
// the authenticated user into the request context.
// Unauthenticated requests are redirected to /auth/login.
func Auth(svc SessionReader) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, found := svc.GetUserFromCookie(r)
			if !found {
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext retrieves the authenticated user from the request context.
// Returns (zero User, false) if no user is authenticated.
func UserFromContext(ctx context.Context) (core.User, bool) {
	user, ok := ctx.Value(userContextKey).(core.User)
	return user, ok
}
