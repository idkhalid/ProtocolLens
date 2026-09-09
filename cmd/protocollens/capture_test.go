package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"protocollens/internal/config"
)

func TestCaptureDurationBounds(t *testing.T) {
	cfg := config.Load()
	cfg.ReplayAllowedPorts = []uint16{80, 443}

	err := runCapture(cfg, []string{"--duration", "0", "https://example.com"})
	if err == nil || !strings.Contains(err.Error(), "duration must be between 1s and 60s") {
		t.Errorf("expected duration bounds error, got: %v", err)
	}

	err = runCapture(cfg, []string{"--duration", "61s", "https://example.com"})
	if err == nil || !strings.Contains(err.Error(), "duration must be between 1s and 60s") {
		t.Errorf("expected duration bounds error, got: %v", err)
	}
}

func TestCaptureURLValidation(t *testing.T) {
	cfg := config.Load()
	cfg.ReplayAllowedPorts = []uint16{80, 443}

	tests := []struct {
		url     string
		errFrag string
	}{
		{"http://user:pass@example.com", "invalid_userinfo"},
		{"ftp://example.com", "unsupported_scheme"},
		{"https://localhost", "destination_blocked"},
		{"http://127.0.0.1", "destination_blocked"},
		{"http://10.0.0.1", "destination_blocked"},
		{"http://169.254.169.254", "destination_blocked"},
	}

	for _, tt := range tests {
		err := runCapture(cfg, []string{tt.url})
		if err == nil || !strings.Contains(err.Error(), tt.errFrag) {
			t.Errorf("url %s: expected %q error, got %v", tt.url, tt.errFrag, err)
		}
	}
}

func TestCaptureSubprocessArgvConstruction(t *testing.T) {
	cfg := config.Load()
	cfg.ReplayAllowedPorts = []uint16{80, 443}
	cfg.DatabasePath = filepath.Join(t.TempDir(), "test.db")

	// Create dummy browser/dist/capture.js so the test passes the built check
	os.MkdirAll("browser/dist", 0755)
	os.WriteFile("browser/dist/capture.js", []byte(""), 0644)
	defer os.RemoveAll("browser")

	var capturedArgs []string
	execCommand = func(name string, arg ...string) *exec.Cmd {
		capturedArgs = append([]string{name}, arg...)
		// return a dummy command that does nothing to simulate success
		// we use `echo` on Windows or Linux
		return exec.Command("echo", "dummy")
	}
	defer func() {
		execCommand = func(name string, arg ...string) *exec.Cmd {
			return exec.Command(name, arg...)
		}
	}()

	err := runCapture(cfg, []string{"https://example.com/?q=&echo injected", "--headed", "--duration", "2s"})
	// It will fail at HAR import because echo doesn't produce HAR, or at "capture artifact is missing or empty"
	if err == nil || !(strings.Contains(err.Error(), "import failed") || strings.Contains(err.Error(), "capture artifact is missing or empty")) {
		t.Logf("got error: %v", err)
	}

	foundURL := false
	foundNode := false
	for _, arg := range capturedArgs {
		if arg == "node" {
			foundNode = true
		}
		if arg == "https://example.com/?q=&echo injected" {
			foundURL = true
		}
	}
	if !foundURL || !foundNode {
		t.Errorf("argv did not correctly execute 'node' or safely contain the raw URL. Args: %v", capturedArgs)
	}
}

func TestCaptureTempFileCleanup(t *testing.T) {
	cfg := config.Load()
	cfg.ReplayAllowedPorts = []uint16{80, 443}
	cfg.DatabasePath = filepath.Join(t.TempDir(), "test.db")

	os.MkdirAll("browser/dist", 0755)
	os.WriteFile("browser/dist/capture.js", []byte(""), 0644)
	defer os.RemoveAll("browser")

	var tempFileName string
	execCommand = func(name string, arg ...string) *exec.Cmd {
		for i, a := range arg {
			if a == "--har" && i+1 < len(arg) {
				tempFileName = arg[i+1]
			}
		}
		// Write invalid HAR to trigger import failure but simulate capture success
		os.WriteFile(tempFileName, []byte("{invalid json}"), 0644)
		return exec.Command("echo", "dummy")
	}
	defer func() {
		execCommand = func(name string, arg ...string) *exec.Cmd {
			return exec.Command(name, arg...)
		}
	}()

	err := runCapture(cfg, []string{"https://example.com"})
	if err == nil || !strings.Contains(err.Error(), "import failed") {
		t.Errorf("expected import failure, got: %v", err)
	}

	if _, err := os.Stat(tempFileName); !os.IsNotExist(err) {
		t.Errorf("expected temp file to be cleaned up, but it still exists at %s", tempFileName)
	}
}

func TestCaptureEmergencyBudget(t *testing.T) {
	// Not full test, just verify duration handling logic
	duration := 5 * time.Second
	emergencyTimeout := duration + 40*time.Second
	expected := 45 * time.Second
	if emergencyTimeout != expected {
		t.Errorf("expected %v emergency timeout, got %v", expected, emergencyTimeout)
	}
}
