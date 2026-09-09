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
	"protocollens/internal/config"
	"protocollens/internal/replay"
	"protocollens/internal/storage/sqlite"
)

var execCommand = func(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}

func runCapture(cfg config.Config, args []string) error {
	var url string
	var headed bool
	duration := 5 * time.Second

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

	if duration < time.Second || duration > 60*time.Second {
		return fmt.Errorf("duration must be between 1s and 60s")
	}

	policy := replay.NewDestinationPolicy(cfg.ReplayAllowedPorts)
	if _, err := policy.ValidateURL(context.Background(), url); err != nil {
		return fmt.Errorf("invalid capture url: %v", err)
	}

	jsEntry := "browser/dist/capture.js"
	if _, err := os.Stat(jsEntry); os.IsNotExist(err) {
		return fmt.Errorf("browser capture adapter is not built; run:\ncd browser\nnpm install\nnpm run build")
	}

	// Calculate emergency timeout
	// Browser startup allowance: 10s
	// Navigation max: 15s
	// Capture duration: duration
	// Cleanup allowance: 10s
	// Emergency margin: 5s
	// Total: duration + 40s
	emergencyTimeout := duration + 40*time.Second
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	ctx, cancelTimeout := context.WithTimeout(ctx, emergencyTimeout)
	defer cancelTimeout()

	tmpFile, err := os.CreateTemp("", "protocollens-*.har")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	tmpFile.Close() // close it so playwright can write to it
	defer os.Remove(tmpFile.Name())

	nodeArgs := []string{jsEntry, "--", url, "--har", tmpFile.Name(), "--duration", strconv.Itoa(int(duration.Milliseconds()))}
	if headed {
		nodeArgs = append(nodeArgs, "--headed")
	}

	cmd := execCommand("node", nodeArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	configureCaptureProcess(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("capture failed to start: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var captureErr error
	select {
	case err := <-done:
		captureErr = err
	case <-ctx.Done():
		// graceful shutdown
		requestCaptureStop(cmd)

		select {
		case err := <-done:
			captureErr = err
		case <-time.After(3 * time.Second):
			// force termination
			forceKillCaptureTree(cmd)
			<-done
			captureErr = fmt.Errorf("capture cancelled or timed out (forced termination)")
		}
	}

	if captureErr != nil {
		return fmt.Errorf("capture process failed: %v", captureErr)
	}

	// Artifact Validation
	stat, err := os.Stat(tmpFile.Name())
	if err != nil || stat.Size() == 0 {
		return fmt.Errorf("capture artifact is missing or empty")
	}

	store, err := sqlite.Open(ctx, cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("database open failed: %v", err)
	}
	defer store.Close()

	file, err := os.Open(tmpFile.Name())
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
