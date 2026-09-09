package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"protocollens/internal/app"
	"protocollens/internal/capture/har"
	"protocollens/internal/capture/playwright"
	"protocollens/internal/config"
	"protocollens/internal/storage/sqlite"
)

var execCommand = func(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}

func runCapture(cfg config.Config, args []string) error {
	var url string
	var headed bool
	duration := playwright.DefaultDuration

	for i := 0; i < len(args); i++ {
		if args[i] == "--headed" {
			headed = true
		} else if args[i] == "--duration" && i+1 < len(args) {
			parsed, err := time.ParseDuration(args[i+1])
			if err != nil {
				parsedSec, errSec := strconv.Atoi(args[i+1])
				if errSec != nil {
					return fmt.Errorf("invalid duration: %v", err)
				}
				parsed = time.Duration(parsedSec) * time.Second
			}
			duration = parsed
			i++
		} else if !strings.HasPrefix(args[i], "--") {
			url = args[i]
		}
	}
	if url == "" {
		return fmt.Errorf("missing url")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	runner := playwright.Runner{ExecCommand: execCommand, Stdout: os.Stdout, Stderr: os.Stderr}
	result, err := runner.Capture(ctx, playwright.Options{URL: url, Duration: duration, Headed: headed, AllowedPorts: cfg.ReplayAllowedPorts})
	if err != nil {
		return err
	}
	defer result.Cleanup()

	store, err := sqlite.Open(ctx, cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("database open failed: %v", err)
	}
	defer store.Close()

	file, err := os.Open(result.Path)
	if err != nil {
		return fmt.Errorf("failed to open capture har: %v", err)
	}
	defer file.Close()

	analysis, err := app.NewImportAnalysis(har.NewImporter(cfg.MaxUploadBytes), store).Execute(ctx, file)
	if err != nil {
		return fmt.Errorf("import failed: %v", err)
	}
	printAnalysis(analysis)
	return nil
}
