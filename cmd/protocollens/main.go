package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"protocollens/internal/app"
	"protocollens/internal/capture/har"
	"protocollens/internal/config"
	"protocollens/internal/domain"
	"protocollens/internal/storage/sqlite"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	cfg := config.Load()

	switch os.Args[1] {
	case "analyze":
		if len(os.Args) != 3 {
			usage()
		}
		runAnalyze(cfg, os.Args[2])
	case "capture":
		if err := runCapture(cfg, os.Args[2:]); err != nil {
			log.Fatalf("capture error: %v", err)
		}
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  protocollens analyze <file.har>")
	fmt.Fprintln(os.Stderr, "  protocollens capture [--headed] [--duration <seconds>] <url>")
	os.Exit(2)
}

func runAnalyze(cfg config.Config, filepath string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store, err := sqlite.Open(ctx, cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	file, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	analysis, err := app.NewImportAnalysis(har.NewImporter(cfg.MaxUploadBytes), store).Execute(ctx, file)
	if err != nil {
		log.Fatal(err)
	}

	printAnalysis(analysis)
}

func printAnalysis(analysis domain.Analysis) {
	fmt.Printf("analysis_id: %s\nrequests: %d\nendpoints: %d\nsession_artifacts: %d\ndependencies: %d\nduration_ms: %d\n",
		analysis.ID,
		analysis.RequestCount,
		analysis.EndpointCount,
		analysis.SessionCount,
		analysis.DependencyCount,
		analysis.ImportDuration,
	)
}
