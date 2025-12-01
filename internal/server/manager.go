package server

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/xrdavies/light-ss/internal/config"
	"github.com/xrdavies/light-ss/internal/proxy"
	"github.com/xrdavies/light-ss/internal/shadowsocks"
	"github.com/xrdavies/light-ss/internal/stats"
)

// Manager manages all proxy servers and their lifecycle
type Manager struct {
	unifiedProxy *proxy.UnifiedProxy
	httpServer   *proxy.HTTPServer
	socks5Server *proxy.SOCKS5Server
	ssClient     *shadowsocks.Client
	collector    *stats.Collector
	reporter     *stats.Reporter
	config       *config.Config
	apiServer    interface{} // Will be *api.Server, using interface{} to avoid circular dependency

	// For hot-reload support
	ssClientMu sync.RWMutex
	oldClients []*shadowsocks.Client

	// For graceful shutdown
	ctx        context.Context
	cancelFunc context.CancelFunc
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
		reporter = stats.NewReporter(collector, cfg.Stats.Interval, cfg.Name)
		slog.Info("Statistics collection enabled", "interval", cfg.Stats.Interval)
	}

	// Create cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	mgr := &Manager{
		ssClient:   ssClient,
		collector:  collector,
		reporter:   reporter,
		config:     cfg,
		ctx:        ctx,
		cancelFunc: cancel,
	}

	// Check if unified mode is enabled
	if cfg.Proxies.Unified != "" {
		// Create unified proxy for both HTTP/HTTPS and SOCKS5
		unifiedProxy, err := proxy.NewUnifiedProxy(cfg.Proxies.Unified, mgr.GetSSClient, collector)
		if err != nil {
			return nil, fmt.Errorf("failed to create unified proxy: %w", err)
		}
		mgr.unifiedProxy = unifiedProxy
		slog.Info("Using unified proxy mode", "address", cfg.Proxies.Unified)
	} else {
		// Separate mode: create HTTP and SOCKS5 proxies separately
		// Create HTTP proxy if enabled
		if cfg.Proxies.HTTPListen != "" {
			httpServer, err := proxy.NewHTTPServer(cfg.Proxies.HTTPListen, ssClient, collector)
			if err != nil {
				return nil, fmt.Errorf("failed to create HTTP server: %w", err)
			}
			mgr.httpServer = httpServer
			slog.Info("HTTP/HTTPS proxy enabled", "address", cfg.Proxies.HTTPListen)
		}

		// Create SOCKS5 proxy if enabled
		if cfg.Proxies.SOCKS5Listen != "" {
			socks5Server, err := proxy.NewSOCKS5Server(cfg.Proxies.SOCKS5Listen, cfg.Proxies.SOCKS5Auth, ssClient, collector)
			if err != nil {
				return nil, fmt.Errorf("failed to create SOCKS5 server: %w", err)
			}
			mgr.socks5Server = socks5Server
			slog.Info("SOCKS5 proxy enabled", "address", cfg.Proxies.SOCKS5Listen)
		}
	}

	// Create API server if enabled (imported locally to avoid circular dependency)
	if cfg.API.Enabled {
		// Import api package inline to avoid circular dependency
		// This will be handled through interface{} type and late binding
		slog.Info("API server will be initialized during startup", "address", cfg.API.Listen)
	}

	return mgr, nil
}

// Start starts all enabled proxy servers
func (m *Manager) Start() error {
	// Start stats reporter if enabled
	if m.reporter != nil {
		m.reporter.Start()
		slog.Info("Statistics reporter started")
	}

	// Start unified proxy if enabled
	if m.unifiedProxy != nil {
		go func() {
			if err := m.unifiedProxy.Start(m.ctx); err != nil {
				slog.Error("Unified proxy error", "error", err)
			}
		}()
		return nil
	}

	// Otherwise start HTTP and SOCKS5 proxies separately
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

	return nil
}

// Shutdown gracefully shuts down all servers
func (m *Manager) Shutdown(ctx context.Context) error {
	slog.Info("Initiating graceful shutdown")

	// Cancel context to signal all goroutines to stop
	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	// Stop stats reporter first
	if m.reporter != nil {
		m.reporter.Stop()
		slog.Info("Statistics reporter stopped")
	}

	// Stop stats collector
	if m.collector != nil {
		m.collector.Stop()
		slog.Info("Statistics collector stopped")
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

	// Stop unified proxy if enabled
	if m.unifiedProxy != nil {
		if err := m.unifiedProxy.Shutdown(ctx); err != nil {
			slog.Error("Error stopping unified proxy", "error", err)
		} else {
			slog.Info("Unified proxy stopped")
		}
		return nil
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

// GetConfig returns the current configuration
func (m *Manager) GetConfig() *config.Config {
	m.ssClientMu.RLock()
	defer m.ssClientMu.RUnlock()
	return m.config
}

// GetSSClient returns the shadowsocks client (thread-safe)
func (m *Manager) GetSSClient() *shadowsocks.Client {
	m.ssClientMu.RLock()
	defer m.ssClientMu.RUnlock()
	return m.ssClient
}

// GetCollector returns the stats collector
func (m *Manager) GetCollector() *stats.Collector {
	return m.collector
}

// ReloadConfig hot-reloads the shadowsocks configuration
func (m *Manager) ReloadConfig(newConfig config.ShadowsocksConfig) error {
	slog.Info("Reloading shadowsocks configuration", "server", newConfig.Server)

	// Create new shadowsocks client
	newClient, err := shadowsocks.NewClient(newConfig)
	if err != nil {
		return fmt.Errorf("failed to create new SS client: %w", err)
	}

	// Acquire write lock
	m.ssClientMu.Lock()
	defer m.ssClientMu.Unlock()

	// Save old client for graceful shutdown
	if m.ssClient != nil {
		m.oldClients = append(m.oldClients, m.ssClient)
	}

	// Swap to new client
	oldClient := m.ssClient
	m.ssClient = newClient

	// Update configuration
	m.config.Shadowsocks = newConfig

	// Note: Proxy servers will use the new client for new connections
	// Existing connections will continue using the old client until they close

	slog.Info("Configuration reloaded successfully",
		"old_server", func() string {
			if oldClient != nil {
				return "previous"
			}
			return "none"
		}(),
		"new_server", newConfig.Server,
	)

	return nil
}

