package proxy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"

	"github.com/armon/go-socks5"
	"github.com/elazarl/goproxy"
	"github.com/xrdavies/light-ss/internal/shadowsocks"
	"github.com/xrdavies/light-ss/internal/stats"
)

// UnifiedProxy serves both HTTP/HTTPS and SOCKS5 on a single port
type UnifiedProxy struct {
	listen     string
	getClient  func() *shadowsocks.Client // Function to get current client (for hot-reload)
	collector  *stats.Collector
	listener   net.Listener
	httpProxy  *goproxy.ProxyHttpServer
	socks5Conf *socks5.Config
}

// NewUnifiedProxy creates a unified proxy that handles both protocols
func NewUnifiedProxy(listen string, getClient func() *shadowsocks.Client, collector *stats.Collector) (*UnifiedProxy, error) {
	u := &UnifiedProxy{
		listen:    listen,
		getClient: getClient,
		collector: collector,
	}

	// Setup HTTP proxy
	httpProxy := goproxy.NewProxyHttpServer()
	httpProxy.Verbose = false
	httpProxy.Tr = &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := u.getClient().DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			if collector != nil {
				conn = stats.NewTrackedConn(conn, collector, "http", addr)
			}
			return conn, nil
		},
	}
	httpProxy.ConnectDial = httpProxy.Tr.Dial

	// Setup SOCKS5 config
	socks5Conf := &socks5.Config{
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := u.getClient().DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			if collector != nil {
				conn = stats.NewTrackedConn(conn, collector, "socks5", addr)
			}
			return conn, nil
		},
	}

	u.httpProxy = httpProxy
	u.socks5Conf = socks5Conf

	return u, nil
}

// Start begins listening and serving both protocols
func (u *UnifiedProxy) Start(ctx context.Context) error {
	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", u.listen)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", u.listen, err)
	}
	u.listener = listener

	slog.Info("unified proxy started", "address", u.listen, "protocols", "HTTP/HTTPS/SOCKS5")

	// Create SOCKS5 server
	socks5Server, err := socks5.New(u.socks5Conf)
	if err != nil {
		return fmt.Errorf("failed to create SOCKS5 server: %w", err)
	}

	go func() {
		<-ctx.Done()
		u.listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				slog.Error("failed to accept connection", "error", err)
				continue
			}
		}

		go u.handleConnection(conn, socks5Server)
	}
}

// handleConnection detects protocol and routes to appropriate handler
func (u *UnifiedProxy) handleConnection(conn net.Conn, socks5Server *socks5.Server) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in connection handler", "error", r)
			conn.Close()
		}
	}()

	// Create buffered reader to peek at first byte
	reader := bufio.NewReader(conn)
	firstByte, err := reader.Peek(1)
	if err != nil {
		// Connection closed before sending data during protocol detection
		slog.Debug("failed to peek first byte for protocol detection (connection closed by client)", "error", err)
		conn.Close()
		return
	}

	// Wrap connection with buffered reader
	bufferedConn := &bufferConn{
		Conn:   conn,
		reader: reader,
	}

	// SOCKS5 version byte is 0x05
	if firstByte[0] == 0x05 {
		slog.Debug("detected SOCKS5 protocol")
		if err := socks5Server.ServeConn(bufferedConn); err != nil {
			slog.Error("SOCKS5 connection failed", "error", err)
		}
	} else {
		slog.Debug("detected HTTP protocol")
		// Handle as HTTP/HTTPS
		u.handleHTTP(bufferedConn, reader)
	}
}

// handleHTTP processes HTTP/HTTPS requests
func (u *UnifiedProxy) handleHTTP(conn net.Conn, reader *bufio.Reader) {
	// Parse HTTP request
	req, err := http.ReadRequest(reader)
	if err != nil {
		if err != io.EOF {
			slog.Error("failed to read HTTP request", "error", err)
		}
		conn.Close()
		return
	}

	// Handle CONNECT method (HTTPS tunneling)
	if req.Method == http.MethodConnect {
		u.handleConnect(conn, req)
		return
	}

	// Handle regular HTTP request
	req.URL.Scheme = "http"
	req.URL.Host = req.Host

	// Create response writer
	writer := newConnResponseWriter(conn)

	// Serve with goproxy
	u.httpProxy.ServeHTTP(writer, req)
}

// handleConnect handles HTTPS CONNECT tunneling
func (u *UnifiedProxy) handleConnect(clientConn net.Conn, req *http.Request) {
	defer clientConn.Close()

	// Connect to target through shadowsocks
	targetConn, err := u.getClient().DialContext(context.Background(), "tcp", req.Host)
	if err != nil {
		slog.Error("failed to connect to target", "host", req.Host, "error", err)
		fmt.Fprintf(clientConn, "HTTP/1.1 502 Bad Gateway\r\n\r\n")
		return
	}
	defer targetConn.Close()

	if u.collector != nil {
		targetConn = stats.NewTrackedConn(targetConn, u.collector, "http", req.Host)
	}

	// Send success response
	fmt.Fprintf(clientConn, "HTTP/1.1 200 Connection established\r\n\r\n")

	// Bidirectional copy
	errCh := make(chan error, 2)
	go func() {
		_, err := io.Copy(targetConn, clientConn)
		errCh <- err
	}()
	go func() {
		_, err := io.Copy(clientConn, targetConn)
		errCh <- err
	}()

	// Wait for either direction to complete
	<-errCh
}

// Shutdown gracefully stops the proxy
func (u *UnifiedProxy) Shutdown(ctx context.Context) error {
	if u.listener != nil {
		return u.listener.Close()
	}
	return nil
}

// bufferConn wraps a connection with a buffered reader
type bufferConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferConn) Read(b []byte) (int, error) {
	return c.reader.Read(b)
}

// connResponseWriter implements http.ResponseWriter for raw connections
type connResponseWriter struct {
	conn       net.Conn
	header     http.Header
	statusCode int
	written    bool
}

func newConnResponseWriter(conn net.Conn) *connResponseWriter {
	return &connResponseWriter{
		conn:   conn,
		header: make(http.Header),
	}
}

func (w *connResponseWriter) Header() http.Header {
	return w.header
}

func (w *connResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.conn.Write(b)
}

func (w *connResponseWriter) WriteHeader(statusCode int) {
	if w.written {
		return
	}
	w.written = true
	w.statusCode = statusCode

	// Write status line
	fmt.Fprintf(w.conn, "HTTP/1.1 %d %s\r\n", statusCode, http.StatusText(statusCode))

	// Write headers
	for key, values := range w.header {
		for _, value := range values {
			fmt.Fprintf(w.conn, "%s: %s\r\n", key, value)
		}
	}
	fmt.Fprintf(w.conn, "\r\n")
}
