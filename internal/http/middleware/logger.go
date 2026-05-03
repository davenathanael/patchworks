package middleware

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/httplog/v3"
)

// Logger returns a middleware that logs HTTP requests using slog.
func Logger(isLocal bool) func(http.Handler) http.Handler {
	// TODO: use consistent format with normal slog.Info etc
	// TODO: explore possibilities of using this as the canonical log line middleware
	logFormat := httplog.SchemaECS.Concise(isLocal)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: logFormat.ReplaceAttr,
	})).With(
		slog.String("app", "patchworks"),
		// slog.String("version", "git hash"), // TODO: get from config
		// slog.String("env", "dev"), // TODO: get from config
	)

	return httplog.RequestLogger(logger, &httplog.Options{
		Level:         slog.LevelInfo,
		Schema:        httplog.SchemaECS,
		RecoverPanics: true,
	})
}
