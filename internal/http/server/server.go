package server

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/davenathanael/patchwork/internal/components"
	"github.com/davenathanael/patchwork/internal/config"
	"github.com/davenathanael/patchwork/internal/http/handlers"
	"github.com/davenathanael/patchwork/internal/logging"
)

// Run starts the HTTP server.
func Run() {
	ctx := context.Background()

	comp, err := components.New(ctx)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := comp.Close(ctx); err != nil {
			panic(err)
		}
	}()

	if comp.Config.Environment.IsLocal() {
		logging.ConfigureLocalLogger()
	}

	handler := handlers.New(comp)
	server := newServer(comp.Config.HTTPServer, handler)

	slog.Info("Starting Patchwork server...")

	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}

func newServer(cfg config.HTTPServer, handler http.Handler) http.Server {
	return http.Server{
		Addr:         net.JoinHostPort(cfg.Host, cfg.Port),
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
		Handler:      handler,
	}
}
