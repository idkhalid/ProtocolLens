package app

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"protocollens/internal/domain"
	"protocollens/internal/replay"
)

var (
	ErrReplayDisabled = errors.New("replay_disabled")
	ErrReplayBusy     = errors.New("replay_busy")
)

type GetReplayTemplate struct {
	store Store
}

func NewGetReplayTemplate(store Store) *GetReplayTemplate {
	return &GetReplayTemplate{store: store}
}

func (uc *GetReplayTemplate) Execute(ctx context.Context, analysisID, requestID string) (replay.Template, error) {
	if _, err := uc.store.GetAnalysis(ctx, analysisID); err != nil {
		return replay.Template{}, err
	}
	exchange, err := uc.exchange(ctx, analysisID, requestID)
	if err != nil {
		return replay.Template{}, err
	}
	return replay.BuildTemplate(analysisID, exchange.Request)
}

func (uc *GetReplayTemplate) exchange(ctx context.Context, analysisID, requestID string) (domain.Exchange, error) {
	exchanges, err := uc.store.ListExchanges(ctx, analysisID)
	if err != nil {
		return domain.Exchange{}, err
	}
	for _, exchange := range exchanges {
		if exchange.Request.ID == requestID {
			return exchange, nil
		}
	}
	return domain.Exchange{}, domain.ErrNotFound
}

type ExecuteReplay struct {
	Enabled  bool
	Executor *replay.Executor
	sem      chan struct{}
}

func NewExecuteReplay(enabled bool, executor *replay.Executor, maxConcurrent int) *ExecuteReplay {
	if maxConcurrent <= 0 {
		maxConcurrent = 4
	}
	return &ExecuteReplay{Enabled: enabled, Executor: executor, sem: make(chan struct{}, maxConcurrent)}
}

func (uc *ExecuteReplay) Execute(ctx context.Context, request replay.Request) (replay.Result, error) {
	if !uc.Enabled {
		return replay.Result{}, ErrReplayDisabled
	}
	select {
	case uc.sem <- struct{}{}:
		defer func() { <-uc.sem }()
	default:
		return replay.Result{}, ErrReplayBusy
	}
	return uc.Executor.Execute(ctx, request)
}

func ReplayRequestFrom(method, rawURL string, headers http.Header, body string, followRedirects bool) replay.Request {
	clean := http.Header{}
	for name, values := range headers {
		for _, value := range values {
			if strings.TrimSpace(value) != replay.Redacted {
				clean.Add(name, value)
			}
		}
	}
	return replay.Request{Method: method, URL: rawURL, Headers: clean, Body: []byte(body), FollowRedirects: followRedirects}
}
