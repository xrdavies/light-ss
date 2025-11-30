package converter

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"github.com/xrdavies/light-ss/internal/config"
)

// ClashProxy represents a single Clash proxy configuration
type ClashProxy struct {
	Name       string                 `yaml:"name"`
	Type       string                 `yaml:"type"`
	Server     string                 `yaml:"server"`
	Port       int                    `yaml:"port"`
	Cipher     string                 `yaml:"cipher"`
	Password   string                 `yaml:"password"`
	UDP        bool                   `yaml:"udp,omitempty"`
	Plugin     string                 `yaml:"plugin,omitempty"`
	PluginOpts map[string]interface{} `yaml:"plugin-opts,omitempty"`
}

// ClashConfig represents Clash configuration structure
type ClashConfig struct {
	Proxies []ClashProxy `yaml:"proxies"`
}

// FromClash converts Clash config to our format
// If multiple proxies exist, converts the first shadowsocks proxy
func FromClash(inputPath string) (*config.Config, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var clashConfig ClashConfig
	if err := yaml.Unmarshal(data, &clashConfig); err != nil {
		return nil, fmt.Errorf("failed to parse Clash config: %w", err)
	}

	// Find first shadowsocks proxy
	var ssProxy *ClashProxy
	for i := range clashConfig.Proxies {
		if clashConfig.Proxies[i].Type == "ss" {
			ssProxy = &clashConfig.Proxies[i]
			break
		}
	}

	if ssProxy == nil {
		return nil, fmt.Errorf("no shadowsocks proxy found in Clash config")
	}

	// Build our config
	cfg := &config.Config{
		Shadowsocks: config.ShadowsocksConfig{
			Server:   ssProxy.Server,
			Port:     ssProxy.Port,
			Password: ssProxy.Password,
			Cipher:   ssProxy.Cipher,
			Timeout:  300, // Default timeout
		},
		Proxies: config.ProxiesConfig{
			HTTP: config.HTTPProxyConfig{
				Enabled: true,
				Listen:  "127.0.0.1:8080",
			},
			SOCKS5: config.SOCKS5ProxyConfig{
				Enabled: true,
				Listen:  "127.0.0.1:1080",
			},
		},
		Stats: config.StatsConfig{
			Enabled:  true,
			Interval: 60,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}

	// Handle plugin
	if ssProxy.Plugin != "" {
		cfg.Shadowsocks.Plugin = normalizePluginName(ssProxy.Plugin)

		// Parse Clash plugin-opts format
		if ssProxy.PluginOpts != nil {
			opts := parseClashPluginOpts(ssProxy.PluginOpts)
			cfg.Shadowsocks.PluginOpts = opts
		}
	}

	return cfg, nil
}

// parseClashPluginOpts converts Clash plugin options to our format
func parseClashPluginOpts(opts map[string]interface{}) *config.PluginOpts {
	result := &config.PluginOpts{}

	if mode, ok := opts["mode"].(string); ok {
		result.Obfs = mode
	}
	if host, ok := opts["host"].(string); ok {
		result.ObfsHost = host
	}

	// Clash uses "mode" but we use "obfs"
	// Both http and tls are supported
	return result
}
