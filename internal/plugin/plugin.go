package plugin

import (
	"context"
	"net"

	"github.com/xrdavies/light-ss/internal/config"
)

// Plugin is an interface for shadowsocks plugins
type Plugin interface {
	// Name returns the plugin name
	Name() string

	// WrapConn wraps a connection with plugin functionality
	WrapConn(conn net.Conn) (net.Conn, error)

	// DialContext creates a connection through the plugin
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

// NewPlugin creates a plugin based on configuration
func NewPlugin(cfg config.ShadowsocksConfig) (Plugin, error) {
	if cfg.Plugin == "" {
		return nil, nil // No plugin configured
	}

	switch cfg.Plugin {
	case "simple-obfs", "obfs-local":
		return NewSimpleObfs(cfg.PluginOpts)
	default:
		return nil, nil // Unknown plugin, proceed without it
	}
}
