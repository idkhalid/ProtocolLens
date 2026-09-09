package app

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExecuteReplayDisabledAndBusy(t *testing.T) {
	disabled := NewExecuteReplay(false, nil, 1)
	if _, err := disabled.Execute(context.Background(), ReplayRequestFrom("GET", "https://example.com", nil, "", false)); !errors.Is(err, ErrReplayDisabled) {
		t.Fatalf("disabled err = %v", err)
	}

	busy := NewExecuteReplay(true, nil, 1)
	busy.sem <- struct{}{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := busy.Execute(ctx, ReplayRequestFrom("GET", "https://example.com", nil, "", false)); !errors.Is(err, ErrReplayBusy) {
		t.Fatalf("busy err = %v", err)
	}
}
