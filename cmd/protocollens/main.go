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
	"protocollens/internal/storage/sqlite"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] != "analyze" {
		fmt.Fprintln(os.Stderr, "usage: protocollens analyze file.har")
		os.Exit(2)
	}

	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store, err := sqlite.Open(ctx, cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	file, err := os.Open(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	analysis, err := app.NewImportAnalysis(har.NewImporter(cfg.MaxUploadBytes), store).Execute(ctx, file)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("analysis_id: %s\nrequests: %d\nendpoints: %d\nduration_ms: %d\n",
		analysis.ID,
		analysis.RequestCount,
		analysis.EndpointCount,
		analysis.ImportDuration,
	)
}
