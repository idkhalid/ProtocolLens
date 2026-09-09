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
	"protocollens/internal/capture/playwright"
	"protocollens/internal/config"
	"protocollens/internal/replay"
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

	if cfg.LocalCaptureEnabled && !config.LocalCaptureBindAllowed(cfg.Addr) {
		log.Fatalf("local capture requires PROTOCOLLENS_ADDR to be loopback, got %q", cfg.Addr)
	}

	importAnalysis := app.NewImportAnalysis(har.NewImporter(cfg.MaxUploadBytes), store)
	policy := replay.NewDestinationPolicy(cfg.ReplayAllowedPorts)
	executor := replay.NewExecutor(policy, cfg.ReplayTimeout, cfg.ReplayMaxRequestBytes, cfg.ReplayMaxResponseBytes)
	getTemplate := app.NewGetReplayTemplate(store)
	generateClient := app.NewGenerateClient(getTemplate)
	replayRoutes := api.NewReplayRoutes(getTemplate, app.NewExecuteReplay(cfg.ReplayEnabled, executor, cfg.ReplayMaxConcurrent), generateClient, cfg.ReplayMaxRequestBytes)
	localCapture := app.NewLocalCapture(cfg.LocalCaptureEnabled, playwright.Runner{Stdout: os.Stdout, Stderr: os.Stderr}, importAnalysis)
	server := api.NewServerWithLocalCapture(cfg.Addr, logger, store, importAnalysis, api.NewLocalCaptureRoutes(cfg.Addr, cfg.ReplayAllowedPorts, localCapture), replayRoutes)
	logger.Info("starting server", "addr", cfg.Addr)
	if err := server.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
