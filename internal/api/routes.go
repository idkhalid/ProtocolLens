package api

import (
	"errors"
	"net/http"

	"protocollens/internal/app"
	"protocollens/internal/domain"
)

func registerRoutes(mux *http.ServeMux, store app.Store, importAnalysis *app.ImportAnalysis) {
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
}

func writeAnalysisLoadError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "Analysis was not found")
		return
	}
	writeError(w, http.StatusInternalServerError, "storage_error", "Analysis could not be loaded")
}
