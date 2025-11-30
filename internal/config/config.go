package config

import (
	"fmt"
	"strings"
)

// Config is the main configuration structure
type Config struct {
	Shadowsocks ShadowsocksConfig `yaml:"shadowsocks" json:"shadowsocks"`
	Proxies     ProxiesConfig     `yaml:"proxies" json:"proxies"`
	Stats       StatsConfig       `yaml:"stats" json:"stats"`
	Logging     LoggingConfig     `yaml:"logging" json:"logging"`
}

// ShadowsocksConfig contains shadowsocks server configuration
type ShadowsocksConfig struct {
	Server   string       `yaml:"server" json:"server"`     // Server address (can be hostname or IP)
	Port     int          `yaml:"port" json:"port"`         // Server port (optional, can be in Server field)
	Password string       `yaml:"password" json:"password"` // Server password
	Cipher   string       `yaml:"cipher" json:"cipher,omitempty"` // Encryption cipher (method)
	Method   string       `yaml:"method" json:"method,omitempty"` // Alternative name for cipher
	Timeout  int          `yaml:"timeout" json:"timeout,omitempty"` // Connection timeout in seconds
	Plugin   string       `yaml:"plugin" json:"plugin,omitempty"` // Plugin name (e.g., "simple-obfs")
	PluginOpts *PluginOpts `yaml:"plugin_opts" json:"plugin_opts,omitempty"` // Plugin options
}

// PluginOpts contains plugin-specific options
type PluginOpts struct {
	Obfs     string `yaml:"obfs" json:"obfs,omitempty"`           // Obfuscation mode: http, tls
	ObfsHost string `yaml:"obfs-host" json:"obfs-host,omitempty"` // Host header for HTTP obfuscation
}

// ProxiesConfig contains configuration for local proxy servers
type ProxiesConfig struct {
	HTTP   HTTPProxyConfig   `yaml:"http" json:"http"`
	SOCKS5 SOCKS5ProxyConfig `yaml:"socks5" json:"socks5"`
}

// HTTPProxyConfig contains HTTP/HTTPS proxy configuration
type HTTPProxyConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"` // Enable HTTP/HTTPS proxy
	Listen  string `yaml:"listen" json:"listen"`   // Listen address (e.g., "127.0.0.1:8080")
}

// SOCKS5ProxyConfig contains SOCKS5 proxy configuration
type SOCKS5ProxyConfig struct {
	Enabled bool        `yaml:"enabled" json:"enabled"` // Enable SOCKS5 proxy
	Listen  string      `yaml:"listen" json:"listen"`   // Listen address (e.g., "127.0.0.1:1080")
	Auth    *AuthConfig `yaml:"auth" json:"auth,omitempty"` // Optional authentication
}

// AuthConfig contains authentication credentials for proxies
type AuthConfig struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

// StatsConfig contains statistics and monitoring configuration
type StatsConfig struct {
	Enabled  bool `yaml:"enabled" json:"enabled"`   // Enable statistics collection
	Interval int  `yaml:"interval" json:"interval"` // Report interval in seconds
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level" json:"level"`   // Log level: debug, info, warn, error
	Format string `yaml:"format" json:"format"` // Log format: json, text
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Handle server and port
	if c.Shadowsocks.Server == "" {
		return ErrMissingServer
	}

	// If port is specified separately, combine it with server
	if c.Shadowsocks.Port > 0 {
		// Check if server already has a port
		if !strings.Contains(c.Shadowsocks.Server, ":") {
			c.Shadowsocks.Server = fmt.Sprintf("%s:%d", c.Shadowsocks.Server, c.Shadowsocks.Port)
		}
	}

	if c.Shadowsocks.Password == "" {
		return ErrMissingPassword
	}

	// Support "method" as alias for "cipher" (common in SS configs)
	if c.Shadowsocks.Method != "" && c.Shadowsocks.Cipher == "" {
		c.Shadowsocks.Cipher = c.Shadowsocks.Method
	}

	if c.Shadowsocks.Cipher == "" {
		c.Shadowsocks.Cipher = "AEAD_CHACHA20_POLY1305" // Default cipher
	}

	if c.Shadowsocks.Timeout == 0 {
		c.Shadowsocks.Timeout = 300 // Default 5 minutes
	}

	// Enable proxies by default if not specified
	if !c.Proxies.HTTP.Enabled && !c.Proxies.SOCKS5.Enabled {
		// If Proxies section is empty, enable both by default
		c.Proxies.HTTP.Enabled = true
		c.Proxies.SOCKS5.Enabled = true
	}

	if c.Proxies.HTTP.Enabled && c.Proxies.HTTP.Listen == "" {
		c.Proxies.HTTP.Listen = "127.0.0.1:8080" // Default HTTP listen
	}

	if c.Proxies.SOCKS5.Enabled && c.Proxies.SOCKS5.Listen == "" {
		c.Proxies.SOCKS5.Listen = "127.0.0.1:1080" // Default SOCKS5 listen
	}

	// Set defaults for logging
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "text"
	}

	// Set defaults for stats
	if c.Stats.Interval == 0 {
		c.Stats.Interval = 60
	}

	return nil
}
