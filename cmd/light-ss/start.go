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
	configFile string
	logLevel   string
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the shadowsocks client and local proxies",
	Long:  `Start the shadowsocks client and run local HTTP/HTTPS and SOCKS5 proxy servers.`,
	RunE:  runStart,
}

func init() {
	startCmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Path to configuration file")
	startCmd.Flags().StringVar(&logLevel, "log-level", "", "Log level (debug, info, warn, error)")
}

func runStart(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override log level if specified via flag
	if logLevel != "" {
		cfg.Logging.Level = logLevel
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
