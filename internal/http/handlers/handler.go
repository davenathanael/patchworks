package handlers

import (
	"net/http"

	"github.com/davenathanael/patchwork/internal/components"
	"github.com/davenathanael/patchwork/internal/http/views"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

// New creates a new HTTP handler to be used in a server.
// All routes and its handlers (and its dependencies) are registered here.
func New(components *components.Components) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	// FIXME: modify routes
	r.Get("/", handleGetHome(components))

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
