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
	r.Use(chimw.RedirectSlashes)
	r.Use(chimw.RequestID)

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static", http.FileServer(http.Dir("resources/static/"))))

	// Auth routes (public)
	r.Get("/auth/login", handleGetLogin(comp.AuthService))
	r.Get("/auth/callback", handleGetCallback(comp.AuthService))
	r.Get("/auth/logout", handleGetLogout(comp.AuthService))

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(comp.AuthService))

		r.Get("/", handleGetHome(comp))
		r.Post("/bookmarks", handlePostBookmarks(comp))

		r.Get("/collections", handleGetCollections(comp))
		r.Get("/collections/new", handleGetCollectionCreation(comp))
		r.Post("/collections", handlePostCollection(comp))
		r.Get("/collections/{id}", handleGetCollectionById(comp))
		r.Get("/collections/{id}/edit", handleGetCollectionEdit(comp))
		r.Post("/collections/{id}/edit", handlePutCollectionById(comp))
		r.Delete("/collections/{id}", handleDeleteCollectionById(comp))
		r.Post("/collections/{id}/delete", handleDeleteCollectionById(comp))
		r.Post("/collections/{id}/members", handlePostCollectionMember(comp))
		r.Delete("/collections/{collectionId}/members/{userId}", handleDeleteCollectionMember(comp))
		r.Post("/collections/{collectionId}/members/{userId}/delete", handleDeleteCollectionMember(comp))
	})

	return r
}
