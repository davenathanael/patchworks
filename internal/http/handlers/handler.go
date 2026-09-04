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
	r.Method("GET", "/auth/login", handleGetLogin())
	r.Method("POST", "/auth/login", handlePostLogin(comp.AuthService))
	r.Method("GET", "/auth/register", handleGetRegister())
	r.Method("POST", "/auth/register", handlePostRegister(comp.AuthService))
	r.Method("GET", "/auth/logout", handleGetLogout(comp.AuthService))

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(comp.AuthService))

		r.Method("GET", "/", handleGetHome(comp))
		r.Method("POST", "/bookmarks", handlePostBookmarks(comp))
		r.Method("GET", "/bookmarks/{id}", handleGetBookmarkById(comp))
		r.Method("GET", "/bookmarks/{id}/edit", handleGetBookmarkEdit(comp))
		r.Method("POST", "/bookmarks/{id}/edit", handlePostBookmarkEdit(comp))

		r.Method("GET", "/collections", handleGetCollections(comp))
		r.Method("GET", "/collections/new", handleGetCollectionCreation(comp))
		r.Method("POST", "/collections", handlePostCollection(comp))
		r.Method("GET", "/collections/{id}", handleGetCollectionById(comp))
		r.Method("GET", "/collections/{id}/edit", handleGetCollectionEdit(comp))
		r.Method("POST", "/collections/{id}/edit", handlePutCollectionById(comp))
		r.Method("DELETE", "/collections/{id}", handleDeleteCollectionById(comp))
		r.Method("POST", "/collections/{id}/delete", handleDeleteCollectionById(comp))
		r.Method("POST", "/collections/{id}/members", handlePostCollectionMember(comp))
		r.Method("DELETE", "/collections/{collectionId}/members/{userId}", handleDeleteCollectionMember(comp))
		r.Method("POST", "/collections/{collectionId}/members/{userId}/delete", handleDeleteCollectionMember(comp))
	})

	return r
}
