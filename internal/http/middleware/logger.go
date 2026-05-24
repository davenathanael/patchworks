package middleware

import (
	"log/slog"
	"net/http"

	loggerpkg "github.com/davenathanael/patchwork/pkg/logger"
	"github.com/go-chi/httplog/v3"
)

// Logger returns a middleware that logs HTTP requests using slog.
func Logger(isLocal bool) func(http.Handler) http.Handler {
	// TODO: use consistent format with normal slog.Info etc
	// TODO: explore possibilities of using this as the canonical log line middleware
	logFormat := httplog.SchemaECS.Concise(isLocal)

	var logger *slog.Logger
	if isLocal {
		logger = loggerpkg.NewLocalLogger()
	} else {
		logger = loggerpkg.NewJSONLogger()
	}

	return httplog.RequestLogger(logger, &httplog.Options{
		Level:         slog.LevelInfo,
		Schema:        logFormat,
		RecoverPanics: true,
	})
}
