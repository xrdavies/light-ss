package proxy

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"github.com/elazarl/goproxy"
	"github.com/xrdavies/light-ss/internal/config"
	"github.com/xrdavies/light-ss/internal/shadowsocks"
	"github.com/xrdavies/light-ss/internal/stats"
)

// HTTPServer wraps an HTTP/HTTPS proxy server
type HTTPServer struct {
	server     *http.Server
	proxy      *goproxy.ProxyHttpServer
	listenAddr string
	ssClient   *shadowsocks.Client
	collector  *stats.Collector
}

// NewHTTPServer creates a new HTTP/HTTPS proxy server
func NewHTTPServer(cfg config.HTTPProxyConfig, ssClient *shadowsocks.Client, collector *stats.Collector) (*HTTPServer, error) {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false

	// Create custom transport that uses shadowsocks
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := ssClient.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}

			// Wrap connection with stats tracking if collector is enabled
			if collector != nil {
				conn = stats.NewTrackedConn(conn, collector, "http", addr)
			}

			return conn, nil
		},
	}

	proxy.Tr = transport

	// Handle HTTPS CONNECT requests
	proxy.OnRequest().HandleConnect(goproxy.FuncHttpsHandler(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		return goproxy.OkConnect, host
	}))

	server := &http.Server{
		Addr:    cfg.Listen,
		Handler: proxy,
	}

	return &HTTPServer{
		server:     server,
		proxy:      proxy,
		listenAddr: cfg.Listen,
		ssClient:   ssClient,
		collector:  collector,
	}, nil
}

// Start starts the HTTP/HTTPS proxy server
func (h *HTTPServer) Start() error {
	slog.Info("HTTP/HTTPS proxy started", "listen", h.listenAddr)

	go func() {
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

// Stop stops the HTTP/HTTPS proxy server
func (h *HTTPServer) Stop(ctx context.Context) error {
	slog.Info("Stopping HTTP/HTTPS proxy")
	return h.server.Shutdown(ctx)
}
