package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"protocollens/internal/app"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(addr string, logger *slog.Logger, store app.Store, importAnalysis *app.ImportAnalysis, replayUseCases ...replayRoutes) *Server {
	return NewServerWithLocalCapture(addr, logger, store, importAnalysis, LocalCaptureRoutes{Addr: addr}, replayUseCases...)
}

func NewServerWithLocalCapture(addr string, logger *slog.Logger, store app.Store, importAnalysis *app.ImportAnalysis, captureRoutes LocalCaptureRoutes, replayUseCases ...replayRoutes) *Server {
	mux := http.NewServeMux()
	registerRoutes(mux, store, importAnalysis, replayUseCases...)
	registerLocalCaptureRoutes(mux, captureRoutes)

	handler := cors(logRequests(logger, mux))
	return &Server{
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       90 * time.Second,
			WriteTimeout:      90 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
}

func (s *Server) Run(ctx context.Context) error {
	errc := make(chan error, 1)
	go func() {
		err := s.httpServer.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			errc <- err
			return
		}
		errc <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	case err := <-errc:
		return err
	}
}
