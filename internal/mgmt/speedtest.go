package mgmt

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/xrdavies/light-ss/internal/shadowsocks"
)

// SpeedTest performs active speed tests through shadowsocks connection
type SpeedTest struct {
	ssClient *shadowsocks.Client
}

// SpeedTestResult holds the results of a speed test
type SpeedTestResult struct {
	DownloadSpeed int64 // bytes per second
	LatencyMS     int64 // latency in milliseconds
}

// NewSpeedTest creates a new speed test instance
func NewSpeedTest(ssClient *shadowsocks.Client) *SpeedTest {
	return &SpeedTest{
		ssClient: ssClient,
	}
}

// Run executes a speed test for the specified duration
// If latencyOnly is true, only measures latency without downloading test data
func (st *SpeedTest) Run(durationSec int, latencyOnly bool) (*SpeedTestResult, error) {
	var latency int64
	var err error

	if latencyOnly {
		// For latency-only mode, use google.com for faster and more reliable testing
		latencyStart := time.Now()
		conn, err := st.ssClient.Dial("tcp", "www.google.com:80")
		if err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
		defer conn.Close()
		latency = time.Since(latencyStart).Milliseconds()

		return &SpeedTestResult{
			DownloadSpeed: 0, // No download test performed
			LatencyMS:     latency,
		}, nil
	}

	// For full speed test, measure latency to cloudflare
	latencyStart := time.Now()
	conn, err := st.ssClient.Dial("tcp", "speed.cloudflare.com:443")
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()
	latency = time.Since(latencyStart).Milliseconds()

	// Perform download speed test
	testURL := "https://speed.cloudflare.com/__down?bytes=10000000"
	start := time.Now()

	// Create HTTP request
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Use a custom HTTP client that uses shadowsocks connection
	client := &http.Client{
		Transport: &http.Transport{
			Dial: st.ssClient.Dial,
		},
		Timeout: time.Duration(durationSec+5) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Download for specified duration or until complete
	deadline := time.Now().Add(time.Duration(durationSec) * time.Second)
	bytesRead := int64(0)
	buf := make([]byte, 32*1024) // 32KB buffer

	for time.Now().Before(deadline) {
		n, err := resp.Body.Read(buf)
		bytesRead += int64(n)
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}

	elapsed := time.Since(start).Seconds()
	if elapsed == 0 {
		elapsed = 0.001 // Prevent division by zero
	}

	speed := int64(float64(bytesRead) / elapsed)

	return &SpeedTestResult{
		DownloadSpeed: speed,
		LatencyMS:     latency,
	}, nil
}
