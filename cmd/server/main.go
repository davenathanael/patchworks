package main

import (
	"log/slog"

	"github.com/davenathanael/patchwork/internal/http/server"
)

func main() {
	slog.Info("Starting Patchworks...")

	server.Run()
}
