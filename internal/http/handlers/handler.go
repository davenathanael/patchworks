package handlers

import (
	"net/http"

	"github.com/davenathanael/patchwork/internal/components"
	"github.com/davenathanael/patchwork/internal/http/middleware"
	"github.com/davenathanael/patchwork/internal/http/views"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// New creates a new HTTP handler to be used in a server.
// All routes and its handlers (and its dependencies) are registered here.
func New(comp *components.Components) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.CleanPath)

	// Auth routes (public)
	r.Get("/auth/login", handleGetLogin(comp.AuthService))
	r.Get("/auth/callback", handleGetCallback(comp.AuthService))
	r.Get("/auth/logout", handleGetLogout(comp.AuthService))

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(comp.AuthService, authToCoreUser))

		r.Get("/", handleGetHome(comp))
	})

	return r
}

func handleGetHome(components *components.Components) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := views.HomePage().Render(w)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}
}
