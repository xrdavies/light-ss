package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/xrdavies/light-ss/internal/config"
	"github.com/xrdavies/light-ss/internal/proxy"
	"github.com/xrdavies/light-ss/internal/shadowsocks"
	"github.com/xrdavies/light-ss/internal/stats"
)

// Manager manages all proxy servers and their lifecycle
type Manager struct {
	httpServer   *proxy.HTTPServer
	socks5Server *proxy.SOCKS5Server
	ssClient     *shadowsocks.Client
	collector    *stats.Collector
	reporter     *stats.Reporter
	config       *config.Config
}

// NewManager creates a new server manager
func NewManager(cfg *config.Config) (*Manager, error) {
	// Create shadowsocks client
	ssClient, err := shadowsocks.NewClient(cfg.Shadowsocks)
	if err != nil {
		return nil, fmt.Errorf("failed to create shadowsocks client: %w", err)
	}

	// Create stats collector if enabled
	var collector *stats.Collector
	var reporter *stats.Reporter
	if cfg.Stats.Enabled {
		collector = stats.NewCollector()
		reporter = stats.NewReporter(collector, cfg.Stats.Interval)
		slog.Info("Statistics collection enabled", "interval", cfg.Stats.Interval)
	}

	mgr := &Manager{
		ssClient:  ssClient,
		collector: collector,
		reporter:  reporter,
		config:    cfg,
	}

	// Create HTTP proxy if enabled
	if cfg.Proxies.HTTP.Enabled {
		httpServer, err := proxy.NewHTTPServer(cfg.Proxies.HTTP, ssClient, collector)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP server: %w", err)
		}
		mgr.httpServer = httpServer
	}

	// Create SOCKS5 proxy if enabled
	if cfg.Proxies.SOCKS5.Enabled {
		socks5Server, err := proxy.NewSOCKS5Server(cfg.Proxies.SOCKS5, ssClient, collector)
		if err != nil {
			return nil, fmt.Errorf("failed to create SOCKS5 server: %w", err)
		}
		mgr.socks5Server = socks5Server
	}

	return mgr, nil
}

// Start starts all enabled proxy servers
func (m *Manager) Start() error {
	// Start HTTP proxy if enabled
	if m.httpServer != nil {
		if err := m.httpServer.Start(); err != nil {
			return fmt.Errorf("failed to start HTTP server: %w", err)
		}
	}

	// Start SOCKS5 proxy if enabled
	if m.socks5Server != nil {
		if err := m.socks5Server.Start(); err != nil {
			return fmt.Errorf("failed to start SOCKS5 server: %w", err)
		}
	}

	// Start stats reporter if enabled
	if m.reporter != nil {
		m.reporter.Start()
		slog.Info("Statistics reporter started")
	}

	return nil
}

// Shutdown gracefully shuts down all servers
func (m *Manager) Shutdown(ctx context.Context) error {
	slog.Info("Initiating graceful shutdown")

	// Stop stats reporter first
	if m.reporter != nil {
		m.reporter.Stop()
		slog.Info("Statistics reporter stopped")
	}

	// Log final stats if collector exists
	if m.collector != nil {
		finalStats := m.collector.GetStats()
		slog.Info("Final statistics",
			"total_connections", finalStats.TotalConnections,
			"active_connections", finalStats.ActiveConnections,
			"http_connections", finalStats.HTTPConnections,
			"socks5_connections", finalStats.SOCKS5Connections,
			"bytes_sent", finalStats.BytesSent,
			"bytes_received", finalStats.BytesReceived,
			"uptime", finalStats.Uptime.String(),
		)
	}

	// Stop HTTP proxy
	if m.httpServer != nil {
		if err := m.httpServer.Stop(ctx); err != nil {
			slog.Error("Error stopping HTTP server", "error", err)
		} else {
			slog.Info("HTTP server stopped")
		}
	}

	// Stop SOCKS5 proxy
	if m.socks5Server != nil {
		if err := m.socks5Server.Stop(); err != nil {
			slog.Error("Error stopping SOCKS5 server", "error", err)
		} else {
			slog.Info("SOCKS5 server stopped")
		}
	}

	return nil
}
