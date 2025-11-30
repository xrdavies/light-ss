package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/xrdavies/light-ss/internal/config"
	"github.com/xrdavies/light-ss/internal/server"
)

var (
	// Config file
	configFile string

	// Shadowsocks server parameters
	ssServer   string
	ssPort     int
	ssPassword string
	ssMethod   string
	ssTimeout  int

	// Plugin parameters
	ssPlugin     string
	pluginObfs string
	pluginHost string

	// Proxy parameters
	proxies      string
	httpProxy    string
	socks5Proxy  string

	// Logging
	logLevel string
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the shadowsocks client and local proxies",
	Long:  `Start the shadowsocks client and run local HTTP/HTTPS and SOCKS5 proxy servers.`,
	RunE:  runStart,
}

func init() {
	// Config file (optional)
	startCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to configuration file (optional)")

	// Shadowsocks server flags
	startCmd.Flags().StringVarP(&ssServer, "server", "s", "", "Shadowsocks server address")
	startCmd.Flags().IntVarP(&ssPort, "port", "p", 0, "Shadowsocks server port")
	startCmd.Flags().StringVar(&ssPassword, "password", "", "Shadowsocks password")
	startCmd.Flags().StringVarP(&ssMethod, "method", "m", "", "Encryption method (aes-128-gcm, aes-256-gcm, chacha20-poly1305)")
	startCmd.Flags().IntVar(&ssTimeout, "timeout", 0, "Connection timeout in seconds")

	// Plugin flags
	startCmd.Flags().StringVar(&ssPlugin, "plugin", "", "Plugin name (e.g., simple-obfs)")
	startCmd.Flags().StringVar(&pluginObfs, "plugin-obfs", "", "Obfuscation mode: http or tls")
	startCmd.Flags().StringVar(&pluginHost, "plugin-host", "", "Obfuscation host header")

	// Proxy flags
	startCmd.Flags().StringVar(&proxies, "proxies", "", "Unified proxy listen address (e.g., 127.0.0.1:1080)")
	startCmd.Flags().StringVar(&httpProxy, "http-proxy", "", "HTTP/HTTPS proxy listen address")
	startCmd.Flags().StringVar(&socks5Proxy, "socks5-proxy", "", "SOCKS5 proxy listen address (supports user:pass@host:port)")

	// Logging flags
	startCmd.Flags().StringVar(&logLevel, "log-level", "", "Log level (debug, info, warn, error)")
}

func runStart(cmd *cobra.Command, args []string) error {
	var cfg *config.Config
	var err error

	// Load configuration from file if specified
	if configFile != "" {
		cfg, err = config.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	} else {
		// No config file, create default config
		cfg = &config.Config{
			Shadowsocks: config.ShadowsocksConfig{},
			Proxies:     config.ProxiesConfig{},
			Stats:       config.StatsConfig{Enabled: false, Interval: 60},
			Logging:     config.LoggingConfig{Level: "info", Format: "text"},
		}
	}

	// Override with command-line flags (flags take precedence)
	applyFlags(cfg)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Set up logging
	if err := setupLogging(cfg.Logging); err != nil {
		return fmt.Errorf("failed to setup logging: %w", err)
	}

	slog.Info("Starting light-ss", "version", rootCmd.Version)

	// Create and start server manager
	mgr, err := server.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server manager: %w", err)
	}

	if err := mgr.Start(); err != nil {
		return fmt.Errorf("failed to start servers: %w", err)
	}

	slog.Info("All servers started successfully")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	slog.Info("Received shutdown signal", "signal", sig.String())

	// Graceful shutdown with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := mgr.Shutdown(ctx); err != nil {
		slog.Error("Error during shutdown", "error", err)
		return err
	}

	slog.Info("Shutdown complete")
	return nil
}

// applyFlags applies command-line flags to the configuration
func applyFlags(cfg *config.Config) {
	// Shadowsocks server flags
	if ssServer != "" {
		cfg.Shadowsocks.Server = ssServer
	}
	if ssPort != 0 {
		cfg.Shadowsocks.Port = ssPort
	}
	if ssPassword != "" {
		cfg.Shadowsocks.Password = ssPassword
	}
	if ssMethod != "" {
		cfg.Shadowsocks.Method = ssMethod
	}
	if ssTimeout != 0 {
		cfg.Shadowsocks.Timeout = ssTimeout
	}

	// Plugin flags
	if ssPlugin != "" {
		cfg.Shadowsocks.Plugin = ssPlugin
	}
	if pluginObfs != "" || pluginHost != "" {
		if cfg.Shadowsocks.PluginOpts == nil {
			cfg.Shadowsocks.PluginOpts = &config.PluginOpts{}
		}
		if pluginObfs != "" {
			cfg.Shadowsocks.PluginOpts.Obfs = pluginObfs
		}
		if pluginHost != "" {
			cfg.Shadowsocks.PluginOpts.ObfsHost = pluginHost
		}
	}

	// Proxy flags
	if proxies != "" {
		cfg.Proxies.Unified = proxies
		// Clear separate mode if unified is specified
		cfg.Proxies.HTTPListen = ""
		cfg.Proxies.SOCKS5Listen = ""
	}
	if httpProxy != "" {
		cfg.Proxies.HTTPListen = httpProxy
		// Clear unified mode if separate mode is specified
		cfg.Proxies.Unified = ""
	}
	if socks5Proxy != "" {
		cfg.Proxies.SOCKS5Listen = socks5Proxy
		// Parse auth if present
		cfg.Proxies.SOCKS5Auth = config.ParseAuth(&cfg.Proxies.SOCKS5Listen)
		// Clear unified mode if separate mode is specified
		cfg.Proxies.Unified = ""
	}

	// Logging flags
	if logLevel != "" {
		cfg.Logging.Level = logLevel
	}
}

func setupLogging(cfg config.LoggingConfig) error {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return nil
}
