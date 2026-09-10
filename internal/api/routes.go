package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"protocollens/internal/app"
	"protocollens/internal/domain"
	"protocollens/internal/generator"
	"protocollens/internal/replay"
)

type replayRoutes struct {
	GetTemplate     *app.GetReplayTemplate
	Execute         *app.ExecuteReplay
	GenerateClient  *app.GenerateClient
	MaxRequestBytes int64
}

func NewReplayRoutes(getTemplate *app.GetReplayTemplate, execute *app.ExecuteReplay, generateClient *app.GenerateClient, maxRequestBytes int64) replayRoutes {
	return replayRoutes{GetTemplate: getTemplate, Execute: execute, GenerateClient: generateClient, MaxRequestBytes: maxRequestBytes}
}
func registerRoutes(mux *http.ServeMux, store app.Store, importAnalysis *app.ImportAnalysis, replayUseCases ...replayRoutes) {
	getWorkflow := app.NewGetWorkflowGraph(store)
	var replayUC replayRoutes
	if len(replayUseCases) > 0 {
		replayUC = replayUseCases[0]
	}

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /api/v1/import/har", func(w http.ResponseWriter, r *http.Request) {
		analysis, err := importAnalysis.Execute(r.Context(), r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_har", "HAR file could not be parsed")
			return
		}
		writeJSON(w, http.StatusCreated, toAnalysisResponse(analysis))
	})

	mux.HandleFunc("GET /api/v1/analyses", func(w http.ResponseWriter, r *http.Request) {
		analyses, err := store.ListAnalyses(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "storage_error", "Analyses could not be loaded")
			return
		}
		writeJSON(w, http.StatusOK, toAnalysisResponses(analyses))
	})

	mux.HandleFunc("GET /api/v1/analyses/{id}", func(w http.ResponseWriter, r *http.Request) {
		analysis, err := store.GetAnalysis(r.Context(), r.PathValue("id"))
		if err != nil {
			writeAnalysisLoadError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toAnalysisResponse(analysis))
	})

	mux.HandleFunc("GET /api/v1/analyses/{id}/endpoints", func(w http.ResponseWriter, r *http.Request) {
		analysisID := r.PathValue("id")
		if _, err := store.GetAnalysis(r.Context(), analysisID); err != nil {
			writeAnalysisLoadError(w, err)
			return
		}
		endpoints, err := store.ListEndpoints(r.Context(), analysisID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "storage_error", "Endpoints could not be loaded")
			return
		}
		writeJSON(w, http.StatusOK, toEndpointResponses(endpoints))
	})

	mux.HandleFunc("GET /api/v1/analyses/{id}/sessions", func(w http.ResponseWriter, r *http.Request) {
		analysisID := r.PathValue("id")
		if _, err := store.GetAnalysis(r.Context(), analysisID); err != nil {
			writeAnalysisLoadError(w, err)
			return
		}
		artifacts, err := store.ListSessionArtifacts(r.Context(), analysisID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "storage_error", "Session artifacts could not be loaded")
			return
		}
		writeJSON(w, http.StatusOK, toSessionArtifactsResponse(artifacts))
	})

	mux.HandleFunc("GET /api/v1/analyses/{id}/dependencies", func(w http.ResponseWriter, r *http.Request) {
		analysisID := r.PathValue("id")
		if _, err := store.GetAnalysis(r.Context(), analysisID); err != nil {
			writeAnalysisLoadError(w, err)
			return
		}
		dependencies, err := store.ListDependencies(r.Context(), analysisID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "storage_error", "Dependencies could not be loaded")
			return
		}
		writeJSON(w, http.StatusOK, toDependenciesResponse(dependencies))
	})

	mux.HandleFunc("GET /api/v1/analyses/{id}/workflow", func(w http.ResponseWriter, r *http.Request) {
		workflow, err := getWorkflow.Execute(r.Context(), r.PathValue("id"))
		if err != nil {
			writeAnalysisLoadError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toWorkflowResponse(workflow))
	})

	mux.HandleFunc("GET /api/v1/analyses/{id}/requests/{requestID}/replay-template", func(w http.ResponseWriter, r *http.Request) {
		if replayUC.GetTemplate == nil {
			writeError(w, http.StatusForbidden, "replay_disabled", "HTTP replay is disabled on this server")
			return
		}
		template, err := replayUC.GetTemplate.Execute(r.Context(), r.PathValue("id"), r.PathValue("requestID"))
		if err != nil {
			writeAnalysisLoadError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, template)
	})

	mux.HandleFunc("POST /api/v1/replay", func(w http.ResponseWriter, r *http.Request) {
		if replayUC.Execute == nil {
			writeError(w, http.StatusForbidden, "replay_disabled", "HTTP replay is disabled on this server")
			return
		}
		limit := replayUC.MaxRequestBytes
		if limit <= 0 {
			limit = 1 << 20
		}
		var input replayRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit+4096)).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_replay_request", "Replay request is invalid")
			return
		}
		if int64(len(input.Body)) > limit {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "Replay request body is too large")
			return
		}
		result, err := replayUC.Execute.Execute(r.Context(), app.ReplayRequestFrom(input.Method, input.URL, input.Headers, input.Body, input.FollowRedirects))
		if err != nil {
			writeReplayError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toReplayResponse(result))
	})
	mux.HandleFunc("POST /api/v1/benchmark", func(w http.ResponseWriter, r *http.Request) {
		if replayUC.Execute == nil {
			writeError(w, http.StatusForbidden, "replay_disabled", "HTTP replay is disabled on this server")
			return
		}
		limit := replayUC.MaxRequestBytes
		if limit <= 0 {
			limit = 1 << 20
		}
		var input benchmarkRequest
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit+4096))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_benchmark_request", "Benchmark request is invalid")
			return
		}
		if err := dec.Decode(&struct{}{}); err != io.EOF {
			writeError(w, http.StatusBadRequest, "invalid_benchmark_request", "Benchmark request is invalid")
			return
		}
		if int64(len(input.Body)) > limit {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "Benchmark request body is too large")
			return
		}
		result, err := app.NewBenchmark(store, replayUC.Execute).Execute(r.Context(), app.BenchmarkInput{
			AnalysisID:                 input.AnalysisID,
			RequestID:                  input.RequestID,
			Request:                    app.ReplayRequestFrom(input.Method, input.URL, input.Headers, input.Body, input.FollowRedirects),
			Runs:                       input.Runs,
			AllowRepeatedNonIdempotent: input.AllowRepeatedNonIdempotent,
		})
		if err != nil {
			writeBenchmarkError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toBenchmarkResponse(result))
	})
	mux.HandleFunc("POST /api/v1/generate", func(w http.ResponseWriter, r *http.Request) {
		if replayUC.GenerateClient == nil {
			writeError(w, http.StatusForbidden, "replay_disabled", "HTTP replay is disabled on this server")
			return
		}
		var input generateRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
			return
		}
		output, err := replayUC.GenerateClient.Execute(r.Context(), input.AnalysisID, input.RequestID, generator.Target(input.Target))
		if err != nil {
			if errors.Is(err, generator.ErrUnsupportedTarget) {
				writeError(w, http.StatusBadRequest, "unsupported_target", "Unsupported generator target")
				return
			}
			writeAnalysisLoadError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, output)
	})
}

func writeAnalysisLoadError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "Analysis was not found")
		return
	}
	writeError(w, http.StatusInternalServerError, "storage_error", "Analysis could not be loaded")
}

func writeBenchmarkError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, app.ErrBenchmarkInvalidRuns):
		writeError(w, http.StatusBadRequest, "invalid_runs", "Benchmark runs must be between 1 and 5")
	case errors.Is(err, app.ErrBenchmarkRepeatNeedsConfirm):
		writeError(w, http.StatusBadRequest, "repeat_requires_confirmation", "Repeated execution requires confirmation for this HTTP method")
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Analysis or request was not found")
	case errors.Is(err, context.Canceled):
		writeError(w, http.StatusBadRequest, "benchmark_cancelled", "Benchmark was cancelled")
	default:
		writeReplayError(w, err)
	}
}
func writeReplayError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, app.ErrReplayDisabled):
		writeError(w, http.StatusForbidden, "replay_disabled", "HTTP replay is disabled on this server")
	case errors.Is(err, app.ErrReplayBusy):
		writeError(w, http.StatusTooManyRequests, "replay_busy", "Replay executor is busy")
	case errors.Is(err, replay.ErrRequestTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "Replay request body is too large")
	case errors.Is(err, replay.ErrUnsupportedScheme):
		writeError(w, http.StatusBadRequest, "unsupported_scheme", "Replay URL scheme is not supported")
	case errors.Is(err, replay.ErrPortBlocked):
		writeError(w, http.StatusBadRequest, "port_blocked", "Replay destination port is not allowed")
	case errors.Is(err, replay.ErrDestinationBlocked), errors.Is(err, replay.ErrLocalHostname), errors.Is(err, replay.ErrUserinfo):
		writeError(w, http.StatusBadRequest, "destination_blocked", "Replay destination is blocked")
	case errors.Is(err, replay.ErrTimeout):
		writeError(w, http.StatusGatewayTimeout, "replay_timeout", "Replay request timed out")
	case errors.Is(err, replay.ErrInvalidRequest):
		writeError(w, http.StatusBadRequest, "invalid_replay_request", "Replay request is invalid")
	default:
		writeError(w, http.StatusBadGateway, "upstream_error", "Replay request failed")
	}
}
