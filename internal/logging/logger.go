package logging

import (
	"log/slog"
	"os"
	"time"

	"github.com/go-chi/httplog/v3"
	"github.com/lmittmann/tint"
)

func ConfigureLocalLogger() {
	logger := NewLocalLogger()

	slog.SetDefault(logger)
}

func NewLocalLogger() *slog.Logger {
	return slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		AddSource:   true,
		Level:       slog.LevelDebug,
		ReplaceAttr: httplog.SchemaECS.Concise(true).ReplaceAttr,
		TimeFormat:  time.Kitchen,
	}))
}

func NewJSONLogger() *slog.Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: httplog.SchemaECS.ReplaceAttr,
	})).With(
		slog.String("app", "patchworks"),
		// slog.String("version", "git hash"), // TODO: get from config
		// slog.String("env", "dev"), // TODO: get from config
	)

	return logger
}
