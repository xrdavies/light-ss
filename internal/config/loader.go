package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	ErrMissingServer   = errors.New("shadowsocks server address is required")
	ErrMissingPassword = errors.New("shadowsocks password is required")
	ErrNoProxyEnabled  = errors.New("at least one proxy type must be enabled")
)

// LoadConfig loads configuration from a YAML or JSON file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config

	// Detect format based on file extension
	ext := strings.ToLower(filepath.Ext(path))

	if ext == ".json" {
		// Parse as JSON
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config file: %w", err)
		}
	} else if ext == ".yaml" || ext == ".yml" {
		// Parse as YAML
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config file: %w", err)
		}
	} else {
		// Try JSON first, then YAML
		if err := json.Unmarshal(data, &cfg); err != nil {
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return nil, fmt.Errorf("failed to parse config file as JSON or YAML: %w", err)
			}
		}
	}

	// Apply environment variable overrides
	applyEnvOverrides(&cfg)

	// Validate configuration first (this handles method->cipher conversion)
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Normalize cipher names (support both formats)
	normalizeCipherName(&cfg)

	return &cfg, nil
}

// applyEnvOverrides applies environment variable overrides to the configuration
func applyEnvOverrides(cfg *Config) {
	if server := os.Getenv("LIGHT_SS_SERVER"); server != "" {
		cfg.Shadowsocks.Server = server
	}
	if password := os.Getenv("LIGHT_SS_PASSWORD"); password != "" {
		cfg.Shadowsocks.Password = password
	}
	if cipher := os.Getenv("LIGHT_SS_CIPHER"); cipher != "" {
		cfg.Shadowsocks.Cipher = cipher
	}
	if httpListen := os.Getenv("LIGHT_SS_HTTP_LISTEN"); httpListen != "" {
		cfg.Proxies.HTTPListen = httpListen
	}
	if socks5Listen := os.Getenv("LIGHT_SS_SOCKS5_LISTEN"); socks5Listen != "" {
		cfg.Proxies.SOCKS5Listen = socks5Listen
	}
	if logLevel := os.Getenv("LIGHT_SS_LOG_LEVEL"); logLevel != "" {
		cfg.Logging.Level = logLevel
	}
}

// normalizeCipherName converts cipher names to the format expected by go-shadowsocks2
// Supports both formats: "aes-128-gcm" and "AEAD_AES_128_GCM"
func normalizeCipherName(cfg *Config) {
	cipher := strings.ToUpper(cfg.Shadowsocks.Cipher)

	// Map of common cipher names to go-shadowsocks2 format
	cipherMap := map[string]string{
		"AES-128-GCM":         "AEAD_AES_128_GCM",
		"AES-192-GCM":         "AEAD_AES_192_GCM",
		"AES-256-GCM":         "AEAD_AES_256_GCM",
		"CHACHA20-POLY1305":   "AEAD_CHACHA20_POLY1305",
		"CHACHA20-IETF-POLY1305": "AEAD_CHACHA20_POLY1305",
		"XCHACHA20-POLY1305":  "AEAD_XCHACHA20_POLY1305",
	}

	// Convert dashes to underscores and check map
	normalized := strings.ReplaceAll(cipher, "-", "_")

	// Check if already in correct format
	if strings.HasPrefix(normalized, "AEAD_") {
		cfg.Shadowsocks.Cipher = normalized
		return
	}

	// Check cipher map
	if mapped, ok := cipherMap[cipher]; ok {
		cfg.Shadowsocks.Cipher = mapped
		return
	}

	// Try adding AEAD_ prefix
	if !strings.HasPrefix(normalized, "AEAD_") {
		cfg.Shadowsocks.Cipher = "AEAD_" + normalized
	}
}
