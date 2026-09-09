package app

import (
	"context"
	"testing"

	"protocollens/internal/capture/playwright"
)

type cancelRunner struct{}

func (cancelRunner) Capture(ctx context.Context, opts playwright.Options) (playwright.Result, error) {
	<-ctx.Done()
	return playwright.Result{}, ctx.Err()
}

func (cancelRunner) Available() error { return nil }

func TestLocalCapturePropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	uc := NewLocalCapture(true, cancelRunner{}, nil)
	done := make(chan error, 1)
	go func() {
		_, err := uc.Execute(ctx, playwright.Options{URL: "http://1.1.1.1", Duration: playwright.DefaultDuration, AllowedPorts: []uint16{80, 443}})
		done <- err
	}()
	cancel()
	if err := <-done; err != context.Canceled {
		t.Fatalf("err = %v", err)
	}
}
