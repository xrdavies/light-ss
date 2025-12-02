package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/xrdavies/light-ss/internal/config"
	"github.com/xrdavies/light-ss/internal/mgmt"
	"github.com/xrdavies/light-ss/internal/shadowsocks"
)

var (
	// Test command specific flags
	testConfigFile string
	testServer     string
	testPort       int
	testPassword   string
	testMethod     string
	testTimeout    int
	testDuration   int
	testJSON       bool
	testLatencyOnly bool

	// Plugin parameters for test
	testPlugin     string
	testPluginObfs string
	testPluginHost string
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test shadowsocks server connectivity and speed",
	Long: `Test a shadowsocks server without starting the full daemon.
This command tests connection, latency, and optionally download speed.`,
	RunE: runTest,
}

func init() {
	// Config file (optional)
	testCmd.Flags().StringVarP(&testConfigFile, "config", "c", "", "Path to configuration file (optional)")

	// Shadowsocks server flags
	testCmd.Flags().StringVarP(&testServer, "server", "s", "", "Shadowsocks server address")
	testCmd.Flags().IntVarP(&testPort, "port", "p", 0, "Shadowsocks server port")
	testCmd.Flags().StringVar(&testPassword, "password", "", "Shadowsocks password")
	testCmd.Flags().StringVarP(&testMethod, "method", "m", "", "Encryption method (aes-128-gcm, aes-256-gcm, chacha20-poly1305)")
	testCmd.Flags().IntVar(&testTimeout, "timeout", 10, "Connection timeout in seconds")

	// Plugin flags
	testCmd.Flags().StringVar(&testPlugin, "plugin", "", "Plugin name (e.g., simple-obfs)")
	testCmd.Flags().StringVar(&testPluginObfs, "plugin-obfs", "", "Obfuscation mode: http or tls")
	testCmd.Flags().StringVar(&testPluginHost, "plugin-host", "", "Obfuscation host header")

	// Test configuration flags
	testCmd.Flags().IntVar(&testDuration, "duration", 10, "Test duration in seconds")
	testCmd.Flags().BoolVar(&testJSON, "json", false, "Output result as JSON")
	testCmd.Flags().BoolVar(&testLatencyOnly, "latency-only", false, "Only test latency, skip speed test")
}

func runTest(cmd *cobra.Command, args []string) error {
	var ssCfg config.ShadowsocksConfig

	// Load configuration from file if specified
	if testConfigFile != "" {
		cfg, err := config.LoadConfig(testConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		ssCfg = cfg.Shadowsocks
	}

	// Override with command-line flags (flags take precedence)
	if testServer != "" {
		ssCfg.Server = testServer
	}
	if testPort != 0 {
		ssCfg.Port = testPort
	}
	if testPassword != "" {
		ssCfg.Password = testPassword
	}
	if testMethod != "" {
		ssCfg.Method = testMethod
	}
	if testTimeout != 0 {
		ssCfg.Timeout = testTimeout
	}

	// Apply plugin flags
	if testPlugin != "" {
		ssCfg.Plugin = testPlugin
	}
	if testPluginObfs != "" || testPluginHost != "" {
		if ssCfg.PluginOpts == nil {
			ssCfg.PluginOpts = &config.PluginOpts{}
		}
		if testPluginObfs != "" {
			ssCfg.PluginOpts.Obfs = testPluginObfs
		}
		if testPluginHost != "" {
			ssCfg.PluginOpts.ObfsHost = testPluginHost
		}
	}

	// Validate required parameters
	if ssCfg.Server == "" {
		return fmt.Errorf("server address is required (use -s or --server)")
	}
	if ssCfg.Password == "" {
		return fmt.Errorf("password is required (use --password)")
	}
	if ssCfg.Method == "" {
		return fmt.Errorf("encryption method is required (use -m or --method)")
	}

	// Build full server address if port is specified
	if ssCfg.Port != 0 {
		ssCfg.Server = fmt.Sprintf("%s:%d", ssCfg.Server, ssCfg.Port)
	}

	// Set default cipher if not specified
	if ssCfg.Cipher == "" {
		ssCfg.Cipher = ssCfg.Method
	}

	// Create shadowsocks client
	ssClient, err := shadowsocks.NewClient(ssCfg)
	if err != nil {
		return fmt.Errorf("failed to create shadowsocks client: %w", err)
	}

	// Test connectivity
	if !testJSON {
		fmt.Fprintf(os.Stderr, "Testing shadowsocks server %s...\n", ssCfg.Server)
	}

	// Run speed test
	result := &TestResult{
		Server:    ssCfg.Server,
		Cipher:    ssCfg.Cipher,
		Success:   false,
		Timestamp: time.Now(),
	}

	// Run speed test (with or without download test)
	speedTest := mgmt.NewSpeedTest(ssClient)
	testResult, err := speedTest.Run(testDuration, testLatencyOnly)
	if err != nil {
		result.Error = err.Error()
	} else {
		result.Success = true
		result.LatencyMS = testResult.LatencyMS
		result.DownloadSpeedBPS = testResult.DownloadSpeed
		// Convert to Mbps (will be 0 in latency-only mode)
		result.DownloadSpeedMbps = float64(testResult.DownloadSpeed) * 8 / (1024 * 1024)
	}

	// Output result
	if testJSON {
		return outputJSON(result)
	}
	return outputText(result)
}

// TestResult holds the test result
type TestResult struct {
	Server            string    `json:"server"`
	Cipher            string    `json:"cipher"`
	Success           bool      `json:"success"`
	LatencyMS         int64     `json:"latency_ms"`
	DownloadSpeedBPS  int64     `json:"download_speed_bps,omitempty"`
	DownloadSpeedMbps float64   `json:"download_speed_mbps,omitempty"`
	Error             string    `json:"error,omitempty"`
	Timestamp         time.Time `json:"timestamp"`
}

// outputJSON outputs result as JSON
func outputJSON(result *TestResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

// outputText outputs result as human-readable text
func outputText(result *TestResult) error {
	if !result.Success {
		fmt.Fprintf(os.Stderr, "❌ Test failed: %s\n", result.Error)
		return fmt.Errorf("test failed")
	}

	fmt.Printf("✅ Test successful\n")
	fmt.Printf("Server:   %s\n", result.Server)
	fmt.Printf("Cipher:   %s\n", result.Cipher)
	fmt.Printf("Latency:  %dms\n", result.LatencyMS)

	if result.DownloadSpeedMbps > 0 {
		fmt.Printf("Speed:    %.2f Mbps\n", result.DownloadSpeedMbps)
	}

	return nil
}
