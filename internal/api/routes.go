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
			status := http.StatusInternalServerError
			code := "storage_error"
			message := "Analysis could not be loaded"
			if errors.Is(err, domain.ErrNotFound) {
				status = http.StatusNotFound
				code = "not_found"
				message = "Analysis was not found"
			}
			writeError(w, status, code, message)
			return
		}
		writeJSON(w, http.StatusOK, toAnalysisResponse(analysis))
	})

	mux.HandleFunc("GET /api/v1/analyses/{id}/endpoints", func(w http.ResponseWriter, r *http.Request) {
		analysisID := r.PathValue("id")
		if _, err := store.GetAnalysis(r.Context(), analysisID); err != nil {
			status := http.StatusInternalServerError
			code := "storage_error"
			message := "Analysis could not be loaded"
			if errors.Is(err, domain.ErrNotFound) {
				status = http.StatusNotFound
				code = "not_found"
				message = "Analysis was not found"
			}
			writeError(w, status, code, message)
			return
		}

		endpoints, err := store.ListEndpoints(r.Context(), analysisID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "storage_error", "Endpoints could not be loaded")
			return
		}
		writeJSON(w, http.StatusOK, toEndpointResponses(endpoints))
	})
}
