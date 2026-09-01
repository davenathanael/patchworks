package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/davenathanael/patchwork/internal/core"
	"github.com/davenathanael/patchwork/internal/http/views"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

// ctxKey is a private context key type for the error ID.
type ctxKey string

const errorIDKey ctxKey = "error-id"

// ErrorIDFromContext returns the error ID assigned by HandleError, or "".
func ErrorIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(errorIDKey).(string)
	return id
}

// HandleError renders the user-facing response for an unhandled error:
// logs with a fresh, non-enumerable error ID, emits X-Error-Id, then renders
// a toast (htmx) or a full error page. Only *unexpected* errors should reach it.
func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	status, userMsg := classify(err)
	errorID := uuid.New().String()
	ctx := context.WithValue(r.Context(), errorIDKey, errorID)

	slog.ErrorContext(ctx, "unexpected request error",
		"error_id", errorID,
		"request_id", chimw.GetReqID(r.Context()),
		"method", r.Method, "path", r.URL.Path,
		"error", err)

	w.Header().Set("X-Error-Id", errorID)

	if views.IsHtmx(r) {
		w.Header().Set("HX-Retarget", "#toast-container")
		w.Header().Set("HX-Reswap", "innerHTML")
		w.WriteHeader(status)
		_ = views.Toast("error", userMsg, errorID).Render(w)
		return
	}
	w.WriteHeader(status)
	_ = views.ErrorPage(userMsg, errorID).Render(w)
}

// classify maps sentinel errors to HTTP status + user message. This is the
// ONLY place core sentinels become HTTP statuses (core stays framework-free).
func classify(err error) (int, string) {
	switch {
	case errors.Is(err, core.ErrInvalidCredentials):
		return http.StatusUnauthorized, "Invalid email or password"
	case errors.Is(err, core.ErrEmailTaken):
		return http.StatusConflict, "That email is already registered"
	case errors.Is(err, core.ErrNotFound):
		return http.StatusNotFound, "That page or resource could not be found"
	default:
		return http.StatusInternalServerError, "Something went wrong"
	}
}
