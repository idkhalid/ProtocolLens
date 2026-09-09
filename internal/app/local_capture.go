package app

import (
	"context"
	"errors"
	"os"

	"protocollens/internal/capture/playwright"
	"protocollens/internal/domain"
)

var (
	ErrCaptureDisabled = errors.New("capture_disabled")
	ErrCaptureBusy     = errors.New("capture_busy")
)

type CaptureRunner interface {
	Capture(context.Context, playwright.Options) (playwright.Result, error)
	Available() error
}

type LocalCapture struct {
	Enabled bool
	Runner  CaptureRunner
	Import  *ImportAnalysis
	sem     chan struct{}
}

func NewLocalCapture(enabled bool, runner CaptureRunner, importAnalysis *ImportAnalysis) *LocalCapture {
	return &LocalCapture{Enabled: enabled, Runner: runner, Import: importAnalysis, sem: make(chan struct{}, 1)}
}

func (uc *LocalCapture) Execute(ctx context.Context, opts playwright.Options) (domain.Analysis, error) {
	if !uc.Enabled {
		return domain.Analysis{}, ErrCaptureDisabled
	}
	if err := playwright.ValidateOptions(ctx, opts); err != nil {
		return domain.Analysis{}, err
	}
	select {
	case uc.sem <- struct{}{}:
		defer func() { <-uc.sem }()
	default:
		return domain.Analysis{}, ErrCaptureBusy
	}
	result, err := uc.Runner.Capture(ctx, opts)
	if err != nil {
		return domain.Analysis{}, err
	}
	defer result.Cleanup()
	file, err := os.Open(result.Path)
	if err != nil {
		return domain.Analysis{}, err
	}
	defer file.Close()
	return uc.Import.Execute(ctx, file)
}

func (uc *LocalCapture) Available() error {
	if !uc.Enabled {
		return ErrCaptureDisabled
	}
	return uc.Runner.Available()
}
