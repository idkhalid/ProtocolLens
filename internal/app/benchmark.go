package app

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"protocollens/internal/benchmark"
	"protocollens/internal/domain"
	"protocollens/internal/replay"
)

const (
	DefaultBenchmarkRuns = 3
	MinBenchmarkRuns     = 1
	MaxBenchmarkRuns     = 5
)

var (
	ErrBenchmarkInvalidRuns        = errors.New("invalid_benchmark_runs")
	ErrBenchmarkRepeatNeedsConfirm = errors.New("benchmark_repeat_requires_confirmation")
)

type Benchmark struct {
	store  Store
	replay *ExecuteReplay
}

type BenchmarkInput struct {
	AnalysisID                 string
	RequestID                  string
	Request                    replay.Request
	Runs                       int
	AllowRepeatedNonIdempotent bool
}

type BenchmarkResult struct {
	Method     string
	RequestID  string
	Browser    BrowserBenchmark
	HTTP       HTTPBenchmark
	Comparison BenchmarkComparison
}

type BrowserBenchmark struct {
	Available bool
	Duration  time.Duration
}

type HTTPBenchmark struct {
	Runs             []HTTPBenchmarkRun
	Min              time.Duration
	Median           time.Duration
	Mean             time.Duration
	Max              time.Duration
	ConsistentStatus bool
}

type HTTPBenchmarkRun struct {
	Duration  time.Duration
	Status    int
	FinalURL  string
	Truncated bool
}

type BenchmarkComparison struct {
	Available        bool
	MedianSpeedup    float64
	ReductionPercent float64
}

func NewBenchmark(store Store, replay *ExecuteReplay) *Benchmark {
	return &Benchmark{store: store, replay: replay}
}

func (b *Benchmark) Execute(ctx context.Context, in BenchmarkInput) (BenchmarkResult, error) {
	if in.Runs == 0 {
		in.Runs = DefaultBenchmarkRuns
	}
	if in.Runs < MinBenchmarkRuns || in.Runs > MaxBenchmarkRuns {
		return BenchmarkResult{}, ErrBenchmarkInvalidRuns
	}
	method := strings.ToUpper(strings.TrimSpace(in.Request.Method))
	if repeatedNonIdempotent(method, in.Runs) && !in.AllowRepeatedNonIdempotent {
		return BenchmarkResult{}, ErrBenchmarkRepeatNeedsConfirm
	}
	exchange, err := findExchange(ctx, b.store, in.AnalysisID, in.RequestID)
	if err != nil {
		return BenchmarkResult{}, err
	}
	result := BenchmarkResult{Method: method, RequestID: in.RequestID}
	if exchange.Response.Duration > 0 {
		result.Browser = BrowserBenchmark{Available: true, Duration: exchange.Response.Duration}
	}

	durations := make([]time.Duration, 0, in.Runs)
	statuses := make([]int, 0, in.Runs)
	for i := 0; i < in.Runs; i++ {
		if err := ctx.Err(); err != nil {
			return BenchmarkResult{}, err
		}
		replayResult, err := b.replay.Execute(ctx, in.Request)
		if err != nil {
			return BenchmarkResult{}, err
		}
		duration := replayResult.Duration
		if duration <= 0 {
			duration = time.Nanosecond
		}
		durations = append(durations, duration)
		statuses = append(statuses, replayResult.StatusCode)
		result.HTTP.Runs = append(result.HTTP.Runs, HTTPBenchmarkRun{Duration: duration, Status: replayResult.StatusCode, FinalURL: replayResult.FinalURL, Truncated: replayResult.Truncated})
	}
	stats, _ := benchmark.Durations(durations)
	result.HTTP.Min = stats.Min
	result.HTTP.Median = stats.Median
	result.HTTP.Mean = stats.Mean
	result.HTTP.Max = stats.Max
	result.HTTP.ConsistentStatus = consistent(statuses)
	if speedup, reduction, ok := benchmark.Comparison(result.Browser.Duration, result.HTTP.Median); ok {
		result.Comparison = BenchmarkComparison{Available: true, MedianSpeedup: speedup, ReductionPercent: reduction}
	}
	return result, nil
}

func findExchange(ctx context.Context, store Store, analysisID, requestID string) (domain.Exchange, error) {
	if _, err := store.GetAnalysis(ctx, analysisID); err != nil {
		return domain.Exchange{}, err
	}
	exchanges, err := store.ListExchanges(ctx, analysisID)
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

func repeatedNonIdempotent(method string, runs int) bool {
	if runs <= 1 {
		return false
	}
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func consistent(values []int) bool {
	if len(values) == 0 {
		return false
	}
	first := values[0]
	for _, value := range values[1:] {
		if value != first {
			return false
		}
	}
	return true
}
