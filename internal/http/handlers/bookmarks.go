package handlers

import (
	"net/http"

	"github.com/davenathanael/patchwork/internal/components"
	"github.com/davenathanael/patchwork/internal/http/middleware"
	"github.com/davenathanael/patchwork/internal/http/views"
)

func handleGetHome(components *components.Components) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.UserFromContext(r.Context())
		if !ok {
			http.Error(w, "user not found in context", 500)
			return
		}

		err := views.HomePage(user).Render(w)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}
}
