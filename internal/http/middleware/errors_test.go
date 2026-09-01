package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/carlmjohnson/be"
	"github.com/davenathanael/patchwork/internal/core"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{name: "invalid credentials", err: core.ErrInvalidCredentials, wantStatus: http.StatusUnauthorized, wantMsg: "Invalid email or password"},
		{name: "email taken", err: core.ErrEmailTaken, wantStatus: http.StatusConflict, wantMsg: "That email is already registered"},
		{name: "not found", err: core.ErrNotFound, wantStatus: http.StatusNotFound, wantMsg: "That page or resource could not be found"},
		{name: "wrapped invalid credentials", err: fmt.Errorf("wrap: %w", core.ErrInvalidCredentials), wantStatus: http.StatusUnauthorized, wantMsg: "Invalid email or password"},
		{name: "wrapped email taken", err: fmt.Errorf("wrap: %w", core.ErrEmailTaken), wantStatus: http.StatusConflict, wantMsg: "That email is already registered"},
		{name: "wrapped not found", err: fmt.Errorf("wrap: %w", core.ErrNotFound), wantStatus: http.StatusNotFound, wantMsg: "That page or resource could not be found"},
		{name: "unknown", err: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantMsg: "Something went wrong"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, msg := classify(tt.err)
			be.Equal(t, tt.wantStatus, status)
			be.Equal(t, tt.wantMsg, msg)
		})
	}
}

func TestHandleErrorFullPage(t *testing.T) {
	rec := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/bookmarks", nil)

	HandleError(rec, r, errors.New("boom"))

	be.Equal(t, http.StatusInternalServerError, rec.Code)
	errorID := rec.Header().Get("X-Error-Id")
	be.True(t, errorID != "")
	body := rec.Body.String()
	be.True(t, strings.Contains(body, "Something went wrong"))
	be.True(t, strings.Contains(body, errorID))
}

func TestHandleErrorHtmx(t *testing.T) {
	rec := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/bookmarks", nil)
	r.Header.Set("HX-Request", "true")

	HandleError(rec, r, errors.New("boom"))

	be.Equal(t, http.StatusInternalServerError, rec.Code)
	be.Equal(t, "#toast-container", rec.Header().Get("HX-Retarget"))
	be.Equal(t, "innerHTML", rec.Header().Get("HX-Reswap"))
	errorID := rec.Header().Get("X-Error-Id")
	body := rec.Body.String()
	be.True(t, strings.Contains(body, "Something went wrong"))
	be.True(t, strings.Contains(body, "Ref "+errorID))
}

func TestErrorIDFromContextEmpty(t *testing.T) {
	be.Equal(t, "", ErrorIDFromContext(context.Background()))
}
