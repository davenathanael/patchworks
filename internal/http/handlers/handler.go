package handlers

import (
	"net/http"

	"github.com/davenathanael/patchwork/internal/components"
	"github.com/davenathanael/patchwork/internal/http/middleware"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// New creates a new HTTP handler to be used in a server.
// All routes and its handlers (and its dependencies) are registered here.
func New(comp *components.Components) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RealIP)
	r.Use(middleware.Logger(comp.Config.Environment.IsLocal()))
	r.Use(chimw.Recoverer)
	r.Use(chimw.CleanPath)

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static", http.FileServer(http.Dir("resources/static/"))))

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
