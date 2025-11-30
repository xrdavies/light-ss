package shadowsocks

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/shadowsocks/go-shadowsocks2/core"
	"github.com/shadowsocks/go-shadowsocks2/socks"
	"github.com/xrdavies/light-ss/internal/config"
	"github.com/xrdavies/light-ss/internal/plugin"
)

// Client wraps a shadowsocks connection and provides dialing capabilities
type Client struct {
	serverAddr string
	cipher     core.Cipher
	timeout    time.Duration
	plugin     plugin.Plugin
}

// NewClient creates a new shadowsocks client from configuration
func NewClient(cfg config.ShadowsocksConfig) (*Client, error) {
	// Create cipher based on config
	cipher, err := core.PickCipher(cfg.Cipher, nil, cfg.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher %s: %w", cfg.Cipher, err)
	}

	// Create plugin if configured
	plug, err := plugin.NewPlugin(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin: %w", err)
	}

	pluginInfo := "none"
	if plug != nil {
		pluginInfo = plug.Name()
	}

	slog.Info("Shadowsocks client created",
		"server", cfg.Server,
		"cipher", cfg.Cipher,
		"timeout", cfg.Timeout,
		"plugin", pluginInfo)

	return &Client{
		serverAddr: cfg.Server,
		cipher:     cipher,
		timeout:    time.Duration(cfg.Timeout) * time.Second,
		plugin:     plug,
	}, nil
}

// Dial connects to the target address through the shadowsocks server
func (c *Client) Dial(network, addr string) (net.Conn, error) {
	return c.DialContext(context.Background(), network, addr)
}

// DialContext connects to the target address through the shadowsocks server with context
func (c *Client) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	slog.Debug("Dialing through shadowsocks",
		"network", network,
		"target", addr,
		"server", c.serverAddr)

	// Parse target address for shadowsocks protocol
	tgt := socks.ParseAddr(addr)
	if tgt == nil {
		return nil, fmt.Errorf("failed to parse target address: %s", addr)
	}

	// Create a dialer with timeout
	dialer := &net.Dialer{
		Timeout: c.timeout,
	}

	// Dial to shadowsocks server
	rc, err := dialer.DialContext(ctx, "tcp", c.serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to shadowsocks server: %w", err)
	}

	// Apply plugin if configured (wrap before cipher)
	if c.plugin != nil {
		slog.Debug("Applying plugin", "plugin", c.plugin.Name())
		rc, err = c.plugin.WrapConn(rc)
		if err != nil {
			rc.Close()
			return nil, fmt.Errorf("failed to apply plugin: %w", err)
		}
	}

	// Wrap connection with cipher
	rc = c.cipher.StreamConn(rc)

	// Send target address through shadowsocks protocol
	if _, err := rc.Write(tgt); err != nil {
		rc.Close()
		return nil, fmt.Errorf("failed to send target address: %w", err)
	}

	slog.Debug("Connected to target through shadowsocks",
		"target", addr)

	return rc, nil
}
