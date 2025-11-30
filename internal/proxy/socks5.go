package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/armon/go-socks5"
	"github.com/xrdavies/light-ss/internal/config"
	"github.com/xrdavies/light-ss/internal/shadowsocks"
	"github.com/xrdavies/light-ss/internal/stats"
)

// SOCKS5Server wraps a SOCKS5 proxy server
type SOCKS5Server struct {
	listener   net.Listener
	server     *socks5.Server
	listenAddr string
	ssClient   *shadowsocks.Client
	collector  *stats.Collector
}

// NewSOCKS5Server creates a new SOCKS5 proxy server
func NewSOCKS5Server(listen string, auth *config.AuthConfig, ssClient *shadowsocks.Client, collector *stats.Collector) (*SOCKS5Server, error) {
	conf := &socks5.Config{
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := ssClient.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}

			// Wrap connection with stats tracking if collector is enabled
			if collector != nil {
				conn = stats.NewTrackedConn(conn, collector, "socks5", addr)
			}

			return conn, nil
		},
	}

	// Add authentication if configured
	if auth != nil {
		credentials := socks5.StaticCredentials{
			auth.Username: auth.Password,
		}
		authenticator := socks5.UserPassAuthenticator{Credentials: credentials}
		conf.AuthMethods = []socks5.Authenticator{authenticator}
		slog.Info("SOCKS5 authentication enabled", "username", auth.Username)
	}

	server, err := socks5.New(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to create SOCKS5 server: %w", err)
	}

	return &SOCKS5Server{
		server:     server,
		listenAddr: listen,
		ssClient:   ssClient,
		collector:  collector,
	}, nil
}

// Start starts the SOCKS5 proxy server
func (s *SOCKS5Server) Start() error {
	listener, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.listenAddr, err)
	}

	s.listener = listener
	slog.Info("SOCKS5 proxy started", "listen", s.listenAddr)

	go func() {
		if err := s.server.Serve(listener); err != nil {
			slog.Error("SOCKS5 server error", "error", err)
		}
	}()

	return nil
}

// Stop stops the SOCKS5 proxy server
func (s *SOCKS5Server) Stop() error {
	if s.listener != nil {
		slog.Info("Stopping SOCKS5 proxy")
		return s.listener.Close()
	}
	return nil
}
