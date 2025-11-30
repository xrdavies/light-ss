package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/xrdavies/light-ss/internal/config"
)

// Response structures
type HealthResponse struct {
	Status string `json:"status"`
	Uptime string `json:"uptime"`
}

type VersionResponse struct {
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	BuildTime string `json:"build_time,omitempty"`
}

type StatsResponse struct {
	TotalConnections  int64  `json:"total_connections"`
	ActiveConnections int64  `json:"active_connections"`
	HTTPConnections   int64  `json:"http_connections"`
	SOCKS5Connections int64  `json:"socks5_connections"`
	BytesSent         int64  `json:"bytes_sent"`
	BytesReceived     int64  `json:"bytes_received"`
	UploadSpeed       int64  `json:"upload_speed"`       // bytes/sec
	DownloadSpeed     int64  `json:"download_speed"`     // bytes/sec
	Uptime            string `json:"uptime"`
}

type SpeedTestResponse struct {
	DownloadSpeed    int64   `json:"download_speed"`    // bytes/sec
	LatencyMS        int64   `json:"latency_ms"`
	TestDurationSec  int     `json:"test_duration_sec"`
}

type ConfigResponse struct {
	Server    string            `json:"server"`
	Cipher    string            `json:"cipher"`
	Plugin    string            `json:"plugin,omitempty"`
	PluginOpts map[string]string `json:"plugin_opts,omitempty"`
	Proxies   string            `json:"proxies,omitempty"`
	HTTP      string            `json:"http,omitempty"`
	SOCKS5    string            `json:"socks5,omitempty"`
}

type ReloadRequest struct {
	Server      string                `json:"server"`
	Password    string                `json:"password"`
	Cipher      string                `json:"cipher,omitempty"`
	Plugin      string                `json:"plugin,omitempty"`
	PluginOpts  *config.PluginOpts    `json:"plugin_opts,omitempty"`
}

type SuccessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// handleHealth returns health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var uptime string
	if s.collector != nil {
		stats := s.collector.GetStats()
		uptime = stats.Uptime.Round(time.Second).String()
	}

	writeJSON(w, http.StatusOK, HealthResponse{
		Status: "ok",
		Uptime: uptime,
	})
}

// handleVersion returns version information
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// TODO: Get actual version from build flags
	writeJSON(w, http.StatusOK, VersionResponse{
		Version:   "1.0.0",
		Commit:    "dev",
		BuildTime: time.Now().Format(time.RFC3339),
	})
}

// handleStats returns current statistics
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.collector == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "statistics not enabled")
		return
	}

	stats := s.collector.GetStats()
	writeJSON(w, http.StatusOK, StatsResponse{
		TotalConnections:  stats.TotalConnections,
		ActiveConnections: stats.ActiveConnections,
		HTTPConnections:   stats.HTTPConnections,
		SOCKS5Connections: stats.SOCKS5Connections,
		BytesSent:         stats.BytesSent,
		BytesReceived:     stats.BytesReceived,
		UploadSpeed:       stats.UploadSpeed,
		DownloadSpeed:     stats.DownloadSpeed,
		Uptime:            stats.Uptime.Round(time.Second).String(),
	})
}

// handleSpeedTest runs an active speed test
func (s *Server) handleSpeedTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.speedTest == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "speed test not available")
		return
	}

	// Parse duration parameter (default 10 seconds)
	duration := 10
	if durationStr := r.URL.Query().Get("duration"); durationStr != "" {
		if d, err := strconv.Atoi(durationStr); err == nil && d > 0 && d <= 300 {
			duration = d
		}
	}

	// Run speed test
	result, err := s.speedTest.Run(duration)
	if err != nil {
		slog.Error("Speed test failed", "error", err)
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("speed test failed: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, SpeedTestResponse{
		DownloadSpeed:   result.DownloadSpeed,
		LatencyMS:       result.LatencyMS,
		TestDurationSec: duration,
	})
}

// handleConfig returns current configuration (sanitized)
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.manager == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "manager not available")
		return
	}

	cfg := s.manager.GetConfig()
	response := ConfigResponse{
		Server: cfg.Shadowsocks.Server,
		Cipher: cfg.Shadowsocks.Cipher,
		Plugin: cfg.Shadowsocks.Plugin,
	}

	if cfg.Shadowsocks.PluginOpts != nil {
		response.PluginOpts = map[string]string{
			"obfs":      cfg.Shadowsocks.PluginOpts.Obfs,
			"obfs-host": cfg.Shadowsocks.PluginOpts.ObfsHost,
		}
	}

	if cfg.Proxies.Unified != "" {
		response.Proxies = cfg.Proxies.Unified
	} else {
		response.HTTP = cfg.Proxies.HTTPListen
		response.SOCKS5 = cfg.Proxies.SOCKS5Listen
	}

	writeJSON(w, http.StatusOK, response)
}

// handleReload hot-reloads shadowsocks configuration
func (s *Server) handleReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.manager == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "manager not available")
		return
	}

	var req ReloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Validate required fields
	if req.Server == "" || req.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "server and password are required")
		return
	}

	// Build new config
	newConfig := config.ShadowsocksConfig{
		Server:     req.Server,
		Password:   req.Password,
		Cipher:     req.Cipher,
		Plugin:     req.Plugin,
		PluginOpts: req.PluginOpts,
		Timeout:    300, // Use default timeout
	}

	// Set default cipher if not provided
	if newConfig.Cipher == "" {
		newConfig.Cipher = "AEAD_CHACHA20_POLY1305"
	}

	// Reload configuration
	if err := s.manager.ReloadConfig(newConfig); err != nil {
		slog.Error("Configuration reload failed", "error", err)
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("reload failed: %v", err))
		return
	}

	slog.Info("Configuration reloaded successfully")
	writeJSON(w, http.StatusOK, SuccessResponse{
		Status:  "ok",
		Message: "Configuration reloaded successfully",
	})
}

// handleStop initiates graceful shutdown
func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{
		Status:  "ok",
		Message: "Shutdown initiated",
	})

	// Trigger shutdown asynchronously
	go func() {
		time.Sleep(100 * time.Millisecond) // Give time for response to be sent
		slog.Info("Shutdown requested via API")
		// TODO: Implement actual shutdown trigger
	}()
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}
