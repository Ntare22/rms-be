package app

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Server wraps the HTTP server used by the API process.
type Server struct {
	httpServer *http.Server
}

// NewServer constructs an HTTP server for the RMS API.
func NewServer(deps *Dependencies) *Server {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", deps.Config.Port),
		Handler:      NewRouter(deps),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return &Server{httpServer: srv}
}

// ListenAndServe starts accepting incoming connections.
func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
