package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/xrdavies/light-ss/internal/config"
	"github.com/xrdavies/light-ss/internal/server"
	"github.com/xrdavies/light-ss/internal/stats"
)

// Server is the management API HTTP server
type Server struct {
	listen    string
	token     string
	manager   *server.Manager
	collector *stats.Collector
	speedTest *SpeedTest
	httpServer *http.Server
	router    *http.ServeMux
}

// NewServer creates a new API server
func NewServer(cfg config.APIConfig, mgr *server.Manager, collector *stats.Collector, speedTest *SpeedTest) *Server {
	s := &Server{
		listen:    cfg.Listen,
		token:     cfg.Token,
		manager:   mgr,
		collector: collector,
		speedTest: speedTest,
		router:    http.NewServeMux(),
	}

	// Register routes
	s.registerRoutes()

	return s
}

// registerRoutes sets up all API endpoints
func (s *Server) registerRoutes() {
	// Wrap handlers with middleware
	s.router.HandleFunc("/health", s.withLogging(s.handleHealth))
	s.router.HandleFunc("/version", s.withLogging(s.handleVersion))
	s.router.HandleFunc("/stats", s.withLogging(s.withAuth(s.handleStats)))
	s.router.HandleFunc("/speedtest", s.withLogging(s.withAuth(s.handleSpeedTest)))
	s.router.HandleFunc("/config", s.withLogging(s.withAuth(s.handleConfig)))
	s.router.HandleFunc("/reload", s.withLogging(s.withAuth(s.handleReload)))
	s.router.HandleFunc("/stop", s.withLogging(s.withAuth(s.handleStop)))
}

// Start starts the API server
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:    s.listen,
		Handler: s.router,
	}

	slog.Info("Starting management API server", "address", s.listen)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("API server error: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the API server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		slog.Info("Shutting down API server")
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
