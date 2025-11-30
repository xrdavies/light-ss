package stats

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Reporter periodically reports statistics
type Reporter struct {
	collector *Collector
	interval  time.Duration
	cancel    context.CancelFunc
}

// NewReporter creates a new stats reporter
func NewReporter(collector *Collector, interval int) *Reporter {
	return &Reporter{
		collector: collector,
		interval:  time.Duration(interval) * time.Second,
	}
}

// Start starts periodic stats reporting
func (r *Reporter) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	ticker := time.NewTicker(r.interval)
	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				r.report()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop stops the stats reporter
func (r *Reporter) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
}

// report logs current statistics
func (r *Reporter) report() {
	stats := r.collector.GetStats()

	slog.Info("Statistics",
		"total_connections", stats.TotalConnections,
		"active_connections", stats.ActiveConnections,
		"http_connections", stats.HTTPConnections,
		"socks5_connections", stats.SOCKS5Connections,
		"bytes_sent", formatBytes(stats.BytesSent),
		"bytes_received", formatBytes(stats.BytesReceived),
		"upload_speed", formatSpeed(stats.UploadSpeed),
		"download_speed", formatSpeed(stats.DownloadSpeed),
		"uptime", stats.Uptime.Round(time.Second).String(),
	)
}

// formatSpeed formats speed into a human-readable string
func formatSpeed(bytesPerSec int64) string {
	const unit = 1024
	if bytesPerSec < unit {
		return fmt.Sprintf("%d B/s", bytesPerSec)
	}
	div, exp := int64(unit), 0
	for n := bytesPerSec / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB/s", float64(bytesPerSec)/float64(div), "KMGTPE"[exp])
}

// formatBytes formats bytes into a human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
