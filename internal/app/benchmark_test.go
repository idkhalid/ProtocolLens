package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"protocollens/internal/domain"
	"protocollens/internal/replay"
)

type benchmarkStore struct {
	analysis  domain.Analysis
	exchanges []domain.Exchange
}

func (s benchmarkStore) SaveAnalysis(context.Context, domain.Analysis, []domain.Exchange, []domain.EndpointSummary, []domain.SessionArtifact, []domain.Dependency) error {
	return nil
}
func (s benchmarkStore) GetAnalysis(_ context.Context, id string) (domain.Analysis, error) {
	if id != s.analysis.ID {
		return domain.Analysis{}, domain.ErrNotFound
	}
	return s.analysis, nil
}
func (s benchmarkStore) ListAnalyses(context.Context) ([]domain.Analysis, error) {
	return []domain.Analysis{s.analysis}, nil
}
func (s benchmarkStore) ListEndpoints(context.Context, string) ([]domain.EndpointSummary, error) {
	return nil, nil
}
func (s benchmarkStore) ListExchanges(context.Context, string) ([]domain.Exchange, error) {
	return s.exchanges, nil
}
func (s benchmarkStore) ListSessionArtifacts(context.Context, string) ([]domain.SessionArtifact, error) {
	return nil, nil
}
func (s benchmarkStore) ListDependencies(context.Context, string) ([]domain.Dependency, error) {
	return nil, nil
}

type benchmarkRoundTrip func(*http.Request) (*http.Response, error)

func (rt benchmarkRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return rt(req) }

func TestBenchmarkSequentialRunsAndStats(t *testing.T) {
	var mu sync.Mutex
	active := 0
	maxActive := 0
	calls := 0
	bm := testBenchmark(t, benchmarkRoundTrip(func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		calls++
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()
		time.Sleep(time.Millisecond)
		mu.Lock()
		active--
		mu.Unlock()
		return benchmarkResponse(req, http.StatusOK), nil
	}), 5, 200*time.Millisecond)
	result, err := bm.Execute(context.Background(), BenchmarkInput{AnalysisID: "a1", RequestID: "req-000001", Runs: 3, Request: replay.Request{Method: "GET", URL: "http://1.1.1.1/"}})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 3 || maxActive != 1 || len(result.HTTP.Runs) != 3 || !result.Browser.Available || !result.Comparison.Available || !result.HTTP.ConsistentStatus {
		t.Fatalf("calls=%d maxActive=%d result=%#v", calls, maxActive, result)
	}
}

func TestBenchmarkRunsBoundsAndDefault(t *testing.T) {
	calls := 0
	bm := testBenchmark(t, benchmarkRoundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		return benchmarkResponse(req, http.StatusOK), nil
	}), 5, 200*time.Millisecond)
	base := BenchmarkInput{AnalysisID: "a1", RequestID: "req-000001", Request: replay.Request{Method: "GET", URL: "http://1.1.1.1/"}}
	for _, runs := range []int{-1, 6} {
		in := base
		in.Runs = runs
		if _, err := bm.Execute(context.Background(), in); !errors.Is(err, ErrBenchmarkInvalidRuns) {
			t.Fatalf("runs %d err=%v", runs, err)
		}
	}
	if _, err := bm.Execute(context.Background(), base); err != nil {
		t.Fatal(err)
	}
	if calls != DefaultBenchmarkRuns {
		t.Fatalf("default runs calls=%d", calls)
	}
}

func TestBenchmarkNonIdempotentAckPolicy(t *testing.T) {
	calls := 0
	bm := testBenchmark(t, benchmarkRoundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		return benchmarkResponse(req, http.StatusOK), nil
	}), 5, 200*time.Millisecond)
	for _, method := range []string{"GET", "HEAD", "OPTIONS"} {
		if _, err := bm.Execute(context.Background(), BenchmarkInput{AnalysisID: "a1", RequestID: "req-000001", Runs: 3, Request: replay.Request{Method: method, URL: "http://1.1.1.1/"}}); err != nil {
			t.Fatalf("%s err=%v", method, err)
		}
	}
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE", "PURGE"} {
		before := calls
		_, err := bm.Execute(context.Background(), BenchmarkInput{AnalysisID: "a1", RequestID: "req-000001", Runs: 3, Request: replay.Request{Method: method, URL: "http://1.1.1.1/"}})
		if !errors.Is(err, ErrBenchmarkRepeatNeedsConfirm) || calls != before {
			t.Fatalf("%s err=%v calls before=%d after=%d", method, err, before, calls)
		}
	}
	if _, err := bm.Execute(context.Background(), BenchmarkInput{AnalysisID: "a1", RequestID: "req-000001", Runs: 1, Request: replay.Request{Method: "POST", URL: "http://1.1.1.1/"}}); err != nil {
		t.Fatalf("POST x1 err=%v", err)
	}
	if _, err := bm.Execute(context.Background(), BenchmarkInput{AnalysisID: "a1", RequestID: "req-000001", Runs: 3, AllowRepeatedNonIdempotent: true, Request: replay.Request{Method: "POST", URL: "http://1.1.1.1/"}}); err != nil {
		t.Fatalf("POST x3 ack err=%v", err)
	}
}

func TestBenchmarkStopsAfterFailure(t *testing.T) {
	calls := 0
	bm := testBenchmark(t, benchmarkRoundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 2 {
			return nil, errors.New("boom")
		}
		return benchmarkResponse(req, http.StatusOK), nil
	}), 5, 200*time.Millisecond)
	_, err := bm.Execute(context.Background(), BenchmarkInput{AnalysisID: "a1", RequestID: "req-000001", Runs: 3, Request: replay.Request{Method: "GET", URL: "http://1.1.1.1/"}})
	if err == nil || calls != 2 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestBenchmarkCancellationStopsRemainingRuns(t *testing.T) {
	started := make(chan struct{})
	calls := 0
	bm := testBenchmark(t, benchmarkRoundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		close(started)
		<-req.Context().Done()
		return nil, req.Context().Err()
	}), 5, 200*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := bm.Execute(ctx, BenchmarkInput{AnalysisID: "a1", RequestID: "req-000001", Runs: 5, Request: replay.Request{Method: "GET", URL: "http://1.1.1.1/"}})
		done <- err
	}()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestBenchmarkNon2XXAndMixedStatus(t *testing.T) {
	statuses := []int{http.StatusOK, http.StatusForbidden, http.StatusOK}
	calls := 0
	bm := testBenchmark(t, benchmarkRoundTrip(func(req *http.Request) (*http.Response, error) {
		status := statuses[calls]
		calls++
		return benchmarkResponse(req, status), nil
	}), 5, 200*time.Millisecond)
	result, err := bm.Execute(context.Background(), BenchmarkInput{AnalysisID: "a1", RequestID: "req-000001", Runs: 3, Request: replay.Request{Method: "GET", URL: "http://1.1.1.1/"}})
	if err != nil || len(result.HTTP.Runs) != 3 || result.HTTP.ConsistentStatus {
		t.Fatalf("err=%v result=%#v", err, result)
	}
}

func TestBenchmarkUnavailableBaselineDisablesComparison(t *testing.T) {
	bm := testBenchmark(t, benchmarkRoundTrip(func(req *http.Request) (*http.Response, error) { return benchmarkResponse(req, http.StatusOK), nil }), 1, 0)
	result, err := bm.Execute(context.Background(), BenchmarkInput{AnalysisID: "a1", RequestID: "req-000001", Runs: 1, Request: replay.Request{Method: "GET", URL: "http://1.1.1.1/"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Browser.Available || result.Comparison.Available {
		t.Fatalf("result=%#v", result)
	}
}

func benchmarkResponse(req *http.Request, status int) *http.Response {
	return &http.Response{StatusCode: status, Request: req, Header: http.Header{"Content-Type": {"text/plain"}}, Body: io.NopCloser(strings.NewReader("")), ContentLength: 0}
}

func testBenchmark(t *testing.T, rt http.RoundTripper, maxConcurrent int, browserDuration time.Duration) *Benchmark {
	t.Helper()
	store := benchmarkStore{analysis: domain.Analysis{ID: "a1"}, exchanges: []domain.Exchange{{Request: domain.Request{ID: "req-000001"}, Response: domain.Response{Duration: browserDuration}}}}
	executor := &replay.Executor{Policy: replay.NewDestinationPolicy([]uint16{80, 443}), Client: &http.Client{Transport: rt}, Timeout: time.Second, MaxRequestSize: 1024, MaxResponseSize: 1024}
	return NewBenchmark(store, NewExecuteReplay(true, executor, maxConcurrent))
}
