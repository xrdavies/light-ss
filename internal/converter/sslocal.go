package converter

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/xrdavies/light-ss/internal/config"
)

// SSLocalConfig represents shadowsocks-libev configuration format
type SSLocalConfig struct {
	Server       string `json:"server"`
	ServerPort   int    `json:"server_port"`
	LocalAddress string `json:"local_address"`
	LocalPort    int    `json:"local_port"`
	Password     string `json:"password"`
	Method       string `json:"method"`
	Timeout      int    `json:"timeout"`
	Plugin       string `json:"plugin,omitempty"`
	PluginOpts   string `json:"plugin_opts,omitempty"`
}

// FromSSLocal converts ss-local config to our format
func FromSSLocal(inputPath string) (*config.Config, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var ssConfig SSLocalConfig
	if err := json.Unmarshal(data, &ssConfig); err != nil {
		return nil, fmt.Errorf("failed to parse ss-local config: %w", err)
	}

	// Build our config
	cfg := &config.Config{
		Shadowsocks: config.ShadowsocksConfig{
			Server:   ssConfig.Server,
			Port:     ssConfig.ServerPort,
			Password: ssConfig.Password,
			Method:   ssConfig.Method,
			Timeout:  ssConfig.Timeout,
		},
		Proxies: config.ProxiesConfig{
			HTTP: config.HTTPProxyConfig{
				Enabled: false, // ss-local doesn't have HTTP proxy
			},
			SOCKS5: config.SOCKS5ProxyConfig{
				Enabled: true,
				Listen:  fmt.Sprintf("%s:%d", ssConfig.LocalAddress, ssConfig.LocalPort),
			},
		},
		Stats: config.StatsConfig{
			Enabled:  false,
			Interval: 60,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}

	// Handle plugin
	if ssConfig.Plugin != "" {
		cfg.Shadowsocks.Plugin = normalizePluginName(ssConfig.Plugin)

		// Parse plugin_opts string format: "obfs=http;obfs-host=example.com"
		if ssConfig.PluginOpts != "" {
			opts, err := parsePluginOptsString(ssConfig.PluginOpts)
			if err != nil {
				return nil, fmt.Errorf("failed to parse plugin_opts: %w", err)
			}
			cfg.Shadowsocks.PluginOpts = opts
		}
	}

	return cfg, nil
}

// parsePluginOptsString parses ss-local style plugin options
// Format: "obfs=http;obfs-host=example.com"
func parsePluginOptsString(opts string) (*config.PluginOpts, error) {
	result := &config.PluginOpts{}

	parts := strings.Split(opts, ";")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch key {
		case "obfs":
			result.Obfs = value
		case "obfs-host":
			result.ObfsHost = value
		}
	}

	return result, nil
}

// normalizePluginName converts various plugin names to our standard names
func normalizePluginName(name string) string {
	switch name {
	case "obfs-local", "obfs":
		return "simple-obfs"
	default:
		return name
	}
}
