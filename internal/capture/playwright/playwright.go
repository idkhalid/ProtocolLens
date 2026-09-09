package playwright

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"

	"protocollens/internal/replay"
)

const (
	DefaultDuration = 5 * time.Second
	MinDuration     = time.Second
	MaxDuration     = 60 * time.Second
	DefaultJSPath   = "browser/dist/capture.js"
)

var (
	ErrAdapterMissing = errors.New("capture_adapter_missing")
	ErrNodeMissing    = errors.New("node_missing")
	ErrInvalidURL     = errors.New("invalid_capture_url")
)

type Options struct {
	URL          string
	Duration     time.Duration
	Headed       bool
	AllowedPorts []uint16
}

type Result struct {
	Path    string
	Cleanup func()
}

type Runner struct {
	JSPath      string
	ExecCommand func(string, ...string) *exec.Cmd
	Stdout      io.Writer
	Stderr      io.Writer
}

func ValidateOptions(ctx context.Context, opts Options) error {
	if opts.Duration < MinDuration || opts.Duration > MaxDuration {
		return fmt.Errorf("duration must be between 1s and 60s")
	}
	if _, err := replay.NewDestinationPolicy(opts.AllowedPorts).ValidateURL(ctx, opts.URL); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}
	return nil
}

func (r Runner) Capture(ctx context.Context, opts Options) (Result, error) {
	if err := ValidateOptions(ctx, opts); err != nil {
		return Result{}, err
	}
	if err := r.available(); err != nil {
		return Result{}, err
	}

	tmpFile, err := os.CreateTemp("", "protocollens-*.har")
	if err != nil {
		return Result{}, fmt.Errorf("failed to create temp file: %v", err)
	}
	_ = tmpFile.Close()
	cleanup := func() { _ = os.Remove(tmpFile.Name()) }

	nodeArgs := []string{r.jsPath(), "--", opts.URL, "--har", tmpFile.Name(), "--duration", strconv.Itoa(int(opts.Duration.Milliseconds()))}
	if opts.Headed {
		nodeArgs = append(nodeArgs, "--headed")
	}
	cmd := r.execCommand()("node", nodeArgs...)
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	configureProcess(cmd)
	if err := cmd.Start(); err != nil {
		cleanup()
		return Result{}, fmt.Errorf("%w: %v", ErrNodeMissing, err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	captureCtx, cancel := context.WithTimeout(ctx, opts.Duration+40*time.Second)
	defer cancel()

	var captureErr error
	select {
	case err := <-done:
		captureErr = err
	case <-captureCtx.Done():
		_ = requestStop(cmd)
		select {
		case err := <-done:
			captureErr = err
		case <-time.After(3 * time.Second):
			forceKillTree(cmd)
			<-done
			captureErr = fmt.Errorf("capture cancelled or timed out")
		}
	}
	if captureErr != nil {
		cleanup()
		return Result{}, fmt.Errorf("capture process failed: %v", captureErr)
	}
	stat, err := os.Stat(tmpFile.Name())
	if err != nil || stat.Size() == 0 {
		cleanup()
		return Result{}, fmt.Errorf("capture artifact is missing or empty")
	}
	return Result{Path: tmpFile.Name(), Cleanup: cleanup}, nil
}

func (r Runner) Available() error {
	if err := r.available(); err != nil {
		return err
	}
	if _, err := exec.LookPath("node"); err != nil {
		return ErrNodeMissing
	}
	return nil
}

func (r Runner) available() error {
	if _, err := os.Stat(r.jsPath()); err != nil {
		if os.IsNotExist(err) {
			return ErrAdapterMissing
		}
		return err
	}
	return nil
}

func (r Runner) jsPath() string {
	if r.JSPath == "" {
		return DefaultJSPath
	}
	return r.JSPath
}

func (r Runner) execCommand() func(string, ...string) *exec.Cmd {
	if r.ExecCommand != nil {
		return r.ExecCommand
	}
	return exec.Command
}
