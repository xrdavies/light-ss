package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the main configuration structure
type Config struct {
	Name        string            `yaml:"name" json:"name,omitempty"`           // Optional instance name
	Shadowsocks ShadowsocksConfig `yaml:"shadowsocks" json:"shadowsocks"`
	Proxies     ProxiesConfig     `yaml:"proxies" json:"proxies"`
	Stats       StatsConfig       `yaml:"stats" json:"stats"`
	Logging     LoggingConfig     `yaml:"logging" json:"logging"`
	API         APIConfig         `yaml:"api" json:"api"`
}

// ProxiesConfig can be either a string (unified mode) or an object (separate mode)
type ProxiesConfig struct {
	// Internal parsed values
	Unified      string
	HTTPListen   string
	SOCKS5Listen string
	SOCKS5Auth   *AuthConfig
}

// UnmarshalJSON handles both string and object formats for proxies
func (p *ProxiesConfig) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first (unified mode)
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		p.Unified = str
		return nil
	}

	// Otherwise, unmarshal as object (separate mode)
	var obj struct {
		HTTP   string `json:"http"`
		SOCKS5 string `json:"socks5"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	p.HTTPListen = obj.HTTP
	p.SOCKS5Listen = obj.SOCKS5

	// Parse SOCKS5 auth if present (user:pass@host:port)
	if p.SOCKS5Listen != "" {
		p.SOCKS5Auth = parseAuth(&p.SOCKS5Listen)
	}

	return nil
}

// UnmarshalYAML handles both string and object formats for proxies
func (p *ProxiesConfig) UnmarshalYAML(value *yaml.Node) error {
	// Try to unmarshal as string first (unified mode)
	var str string
	if err := value.Decode(&str); err == nil {
		p.Unified = str
		return nil
	}

	// Otherwise, unmarshal as object (separate mode)
	var obj struct {
		HTTP   string `yaml:"http"`
		SOCKS5 string `yaml:"socks5"`
	}
	if err := value.Decode(&obj); err != nil {
		return err
	}

	p.HTTPListen = obj.HTTP
	p.SOCKS5Listen = obj.SOCKS5

	// Parse SOCKS5 auth if present (user:pass@host:port)
	if p.SOCKS5Listen != "" {
		p.SOCKS5Auth = parseAuth(&p.SOCKS5Listen)
	}

	return nil
}

// MarshalJSON handles marshaling back to JSON
func (p ProxiesConfig) MarshalJSON() ([]byte, error) {
	if p.Unified != "" {
		return json.Marshal(p.Unified)
	}

	obj := map[string]string{}
	if p.HTTPListen != "" {
		obj["http"] = p.HTTPListen
	}
	if p.SOCKS5Listen != "" {
		// Add auth back if present
		if p.SOCKS5Auth != nil {
			obj["socks5"] = fmt.Sprintf("%s:%s@%s", p.SOCKS5Auth.Username, p.SOCKS5Auth.Password, p.SOCKS5Listen)
		} else {
			obj["socks5"] = p.SOCKS5Listen
		}
	}

	return json.Marshal(obj)
}

// MarshalYAML handles marshaling back to YAML
func (p ProxiesConfig) MarshalYAML() (interface{}, error) {
	if p.Unified != "" {
		return p.Unified, nil
	}

	obj := map[string]string{}
	if p.HTTPListen != "" {
		obj["http"] = p.HTTPListen
	}
	if p.SOCKS5Listen != "" {
		// Add auth back if present
		if p.SOCKS5Auth != nil {
			obj["socks5"] = fmt.Sprintf("%s:%s@%s", p.SOCKS5Auth.Username, p.SOCKS5Auth.Password, p.SOCKS5Listen)
		} else {
			obj["socks5"] = p.SOCKS5Listen
		}
	}

	return obj, nil
}

// ParseAuth extracts username:password from user:pass@host:port format
// Modifies addr to remove the auth part
func ParseAuth(addr *string) *AuthConfig {
	return parseAuth(addr)
}

// parseAuth extracts username:password from user:pass@host:port format
// Modifies addr to remove the auth part
func parseAuth(addr *string) *AuthConfig {
	if addr == nil || *addr == "" {
		return nil
	}

	// Look for @ symbol
	atIndex := strings.LastIndex(*addr, "@")
	if atIndex == -1 {
		return nil
	}

	// Extract auth part (before @)
	authPart := (*addr)[:atIndex]
	hostPart := (*addr)[atIndex+1:]

	// Split auth into username:password
	colonIndex := strings.Index(authPart, ":")
	if colonIndex == -1 {
		return nil
	}

	username := authPart[:colonIndex]
	password := authPart[colonIndex+1:]

	// Update addr to only contain host:port
	*addr = hostPart

	return &AuthConfig{
		Username: username,
		Password: password,
	}
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

// APIConfig contains management API configuration
type APIConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`             // Enable management API
	Listen  string `yaml:"listen" json:"listen"`               // Listen address (e.g., "127.0.0.1:8090")
	Token   string `yaml:"token" json:"token,omitempty"`       // Optional bearer token for authentication
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

	// Set defaults for proxies if not specified
	if c.Proxies.Unified == "" && c.Proxies.HTTPListen == "" && c.Proxies.SOCKS5Listen == "" {
		// If no proxy configuration specified, enable unified mode by default
		c.Proxies.Unified = "127.0.0.1:1080"
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

	// Set defaults for API
	if c.API.Listen == "" && c.API.Enabled {
		c.API.Listen = "127.0.0.1:8090"
	}

	return nil
}
