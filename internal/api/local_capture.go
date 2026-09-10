package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"protocollens/internal/app"
	"protocollens/internal/capture/playwright"
	"protocollens/internal/config"
	"protocollens/internal/replay"
)

const captureBodyLimit = 8 << 10

func NewLocalCaptureRoutes(addr string, allowedPorts []uint16, useCase *app.LocalCapture) LocalCaptureRoutes {
	return LocalCaptureRoutes{Addr: addr, AllowedPorts: allowedPorts, UseCase: useCase}
}

type LocalCaptureRoutes struct {
	Addr         string
	AllowedPorts []uint16
	UseCase      *app.LocalCapture
}

func registerLocalCaptureRoutes(mux *http.ServeMux, routes LocalCaptureRoutes) {
	mux.HandleFunc("GET /api/v1/capabilities", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, captureCapabilities(routes))
	})
	mux.HandleFunc("POST /api/v1/local/capture", func(w http.ResponseWriter, r *http.Request) {
		if routes.UseCase == nil || !routes.UseCase.Enabled {
			writeError(w, http.StatusForbidden, "capture_disabled", "Local capture is disabled on this server")
			return
		}
		if !config.LocalCaptureBindAllowed(routes.Addr) || !localWorkbenchRequest(r) {
			writeError(w, http.StatusForbidden, "local_capture_forbidden", "Local capture requires a loopback workbench request")
			return
		}
		var input localCaptureRequest
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, captureBodyLimit))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_capture_request", "Capture request is invalid")
			return
		}
		if err := dec.Decode(&struct{}{}); err != io.EOF {
			writeError(w, http.StatusBadRequest, "invalid_capture_request", "Capture request is invalid")
			return
		}
		analysis, err := routes.UseCase.Execute(r.Context(), playwright.Options{URL: input.URL, Duration: time.Duration(input.DurationSeconds) * time.Second, Headed: input.Headed, AllowedPorts: routes.AllowedPorts})
		if err != nil {
			writeCaptureError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toAnalysisResponse(analysis))
	})
}

type localCaptureRequest struct {
	URL             string `json:"url"`
	DurationSeconds int    `json:"duration_seconds"`
	Headed          bool   `json:"headed"`
}

func captureCapabilities(routes LocalCaptureRoutes) map[string]any {
	out := map[string]any{"local_capture": map[string]any{"enabled": false, "adapter_built": false, "node_available": false}}
	if routes.UseCase == nil || !routes.UseCase.Enabled || !config.LocalCaptureBindAllowed(routes.Addr) {
		return out
	}
	lc := out["local_capture"].(map[string]any)
	lc["enabled"] = true
	err := routes.UseCase.Available()
	lc["adapter_built"] = !errors.Is(err, playwright.ErrAdapterMissing)
	lc["node_available"] = !errors.Is(err, playwright.ErrNodeMissing) && !errors.Is(err, playwright.ErrAdapterMissing)
	return out
}

func localWorkbenchRequest(r *http.Request) bool {
	if !localHost(r.Host) {
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" && !allowedLocalOrigin(origin) {
		return false
	}
	return r.RemoteAddr == "" || localHost(r.RemoteAddr)
}

func localHost(value string) bool {
	host := value
	if h, _, err := net.SplitHostPort(value); err == nil {
		host = h
	} else if strings.Contains(value, ":") && !strings.HasPrefix(value, "[") {
		return false
	}
	if strings.HasPrefix(host, "[") || strings.HasSuffix(host, "]") {
		if !(strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]")) {
			return false
		}
		host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	}
	if host == "localhost" {
		return true
	}
	addr, err := netip.ParseAddr(host)
	return err == nil && addr.IsLoopback()
}

func writeCaptureError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, app.ErrCaptureDisabled):
		writeError(w, http.StatusForbidden, "capture_disabled", "Local capture is disabled on this server")
	case errors.Is(err, app.ErrCaptureBusy):
		writeError(w, http.StatusConflict, "capture_busy", "A local capture is already running")
	case errors.Is(err, playwright.ErrAdapterMissing):
		writeError(w, http.StatusServiceUnavailable, "capture_adapter_missing", "Capture adapter is not built")
	case errors.Is(err, playwright.ErrNodeMissing):
		writeError(w, http.StatusServiceUnavailable, "node_unavailable", "Node is unavailable")
	case errors.Is(err, playwright.ErrInvalidURL), errors.Is(err, replay.ErrUnsupportedScheme), errors.Is(err, replay.ErrUserinfo), errors.Is(err, replay.ErrDestinationBlocked), errors.Is(err, replay.ErrLocalHostname), errors.Is(err, replay.ErrPortBlocked):
		writeError(w, http.StatusBadRequest, "destination_blocked", "Target URL was rejected")
	case strings.Contains(err.Error(), "duration must be"):
		writeError(w, http.StatusBadRequest, "invalid_duration", "Capture duration must be between 1 and 60 seconds")
	case errors.Is(err, context.Canceled):
		writeError(w, http.StatusBadRequest, "capture_cancelled", "Capture was cancelled")
	default:
		writeError(w, http.StatusBadGateway, "capture_failed", "Capture failed")
	}
}
