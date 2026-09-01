package handlers

import (
	"net/http"

	"github.com/davenathanael/patchwork/internal/http/middleware"
	"github.com/davenathanael/patchwork/internal/http/views"
)

// Handler is an error-returning HTTP handler. Expected errors are classified
// in the handler (re-render / redirect); anything else is returned and
// rendered once by middleware.HandleError.
type Handler func(w http.ResponseWriter, r *http.Request) error

// ServeHTTP runs the handler, routing unhandled errors through the single
// HandleError wrapper. Handler is thus itself an http.Handler and can be
// registered directly with chi's r.Method(...) — no per-route wrapper needed.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		middleware.HandleError(w, r, err)
	}
}

// validateCredentials returns inline field errors for an incomplete
// credentials form; an empty FormErrors means the input is valid.
func validateCredentials(f views.CredentialsForm) views.FormErrors {
	errs := views.FormErrors{}
	if f.Email == "" {
		errs["email"] = "email is required"
	}
	if f.Password == "" {
		errs["password"] = "password is required"
	}
	return errs
}

// validateCollection returns inline field errors for a collection form; an
// empty FormErrors means the input is valid.
func validateCollection(f views.CollectionForm) views.FormErrors {
	errs := views.FormErrors{}
	if f.Name == "" {
		errs["name"] = "name is required"
	}
	return errs
}
