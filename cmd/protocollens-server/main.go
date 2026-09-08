package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"protocollens/internal/api"
	"protocollens/internal/app"
	"protocollens/internal/capture/har"
	"protocollens/internal/config"
	"protocollens/internal/storage/sqlite"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	store, err := sqlite.Open(ctx, cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	importAnalysis := app.NewImportAnalysis(har.NewImporter(cfg.MaxUploadBytes), store)
	server := api.NewServer(cfg.Addr, logger, store, importAnalysis)
	logger.Info("starting server", "addr", cfg.Addr)
	if err := server.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
