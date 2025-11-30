package plugin

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"

	"github.com/xrdavies/light-ss/internal/config"
)

// SimpleObfs implements the simple-obfs plugin
type SimpleObfs struct {
	mode     string // "http" or "tls"
	obfsHost string // Host header for HTTP obfuscation
}

// NewSimpleObfs creates a new simple-obfs plugin
func NewSimpleObfs(opts *config.PluginOpts) (*SimpleObfs, error) {
	if opts == nil {
		return nil, fmt.Errorf("simple-obfs requires plugin options")
	}

	mode := opts.Obfs
	if mode == "" {
		mode = "http" // Default to HTTP
	}

	obfsHost := opts.ObfsHost
	if obfsHost == "" && mode == "http" {
		obfsHost = "www.bing.com" // Default host
	}

	slog.Info("Simple-obfs plugin initialized", "mode", mode, "obfs-host", obfsHost)

	return &SimpleObfs{
		mode:     mode,
		obfsHost: obfsHost,
	}, nil
}

// Name returns the plugin name
func (p *SimpleObfs) Name() string {
	return "simple-obfs"
}

// WrapConn wraps a connection with obfuscation
func (p *SimpleObfs) WrapConn(conn net.Conn) (net.Conn, error) {
	switch p.mode {
	case "http":
		return p.wrapHTTP(conn)
	case "tls":
		return p.wrapTLS(conn)
	default:
		return nil, fmt.Errorf("unsupported obfs mode: %s", p.mode)
	}
}

// DialContext is not used for simple-obfs as it wraps existing connections
func (p *SimpleObfs) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return nil, fmt.Errorf("DialContext not supported for simple-obfs, use WrapConn instead")
}

// wrapHTTP wraps a connection with HTTP obfuscation
func (p *SimpleObfs) wrapHTTP(conn net.Conn) (net.Conn, error) {
	slog.Debug("Wrapping connection with HTTP obfs", "host", p.obfsHost)

	// Return HTTP-wrapped connection
	// The actual HTTP GET request will be sent when first data is written
	return &obfsHTTPConn{
		Conn:       conn,
		host:       p.obfsHost,
		firstWrite: true,
		firstRead:  true,
		reader:     bufio.NewReader(conn),
	}, nil
}

// wrapTLS wraps a connection with TLS obfuscation
func (p *SimpleObfs) wrapTLS(conn net.Conn) (net.Conn, error) {
	// For TLS obfuscation, we wrap data in TLS record format
	slog.Debug("TLS obfs mode - wrapping connection")

	// Return wrapped connection that adds TLS record framing
	return &obfsTLSConn{
		Conn: conn,
	}, nil
}

// obfsHTTPConn wraps a net.Conn to send/receive HTTP-obfuscated data
type obfsHTTPConn struct {
	net.Conn
	host        string
	firstWrite  bool
	firstRead   bool
	reader      *bufio.Reader
}

// Write sends data with HTTP obfuscation
func (c *obfsHTTPConn) Write(b []byte) (int, error) {
	if c.firstWrite {
		// On first write, prepend HTTP GET request
		req := fmt.Sprintf("GET / HTTP/1.1\r\n"+
			"Host: %s\r\n"+
			"User-Agent: curl/7.68.0\r\n"+
			"Upgrade: websocket\r\n"+
			"Connection: Upgrade\r\n"+
			"Content-Length: %d\r\n"+
			"\r\n", c.host, len(b))

		slog.Debug("Sending HTTP obfs header with data", "host", c.host, "dataLen", len(b))

		// Send HTTP header + data together
		combined := append([]byte(req), b...)
		n, err := c.Conn.Write(combined)
		c.firstWrite = false

		if err != nil {
			return 0, err
		}
		// Return the length of actual data written (not including HTTP header)
		if n > len(req) {
			return n - len(req), nil
		}
		return 0, nil
	}

	// Subsequent writes go directly
	return c.Conn.Write(b)
}

// Read reads data and strips HTTP response wrapping if present
func (c *obfsHTTPConn) Read(b []byte) (int, error) {
	if c.firstRead {
		c.firstRead = false

		// Try to read HTTP response header and skip it
		// Simple-obfs server sends: HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n
		slog.Debug("Reading first response, checking for HTTP header")

		// Peek to see if there's an HTTP response
		peek, err := c.reader.Peek(12)
		if err == nil && len(peek) >= 12 {
			if string(peek[:4]) == "HTTP" {
				// Read and discard HTTP response headers
				for {
					line, err := c.reader.ReadString('\n')
					if err != nil {
						break
					}
					slog.Debug("Skipping HTTP header line", "line", line)
					// Empty line marks end of headers
					if line == "\r\n" || line == "\n" {
						break
					}
				}
				slog.Debug("HTTP response headers skipped")
			}
		}
	}

	// Read actual data
	return c.reader.Read(b)
}

// obfsTLSConn wraps a net.Conn with TLS obfuscation
type obfsTLSConn struct {
	net.Conn
}

// Write writes to the obfuscated connection
func (c *obfsTLSConn) Write(b []byte) (int, error) {
	// Wrap data in TLS record format
	// Type (1 byte): 0x17 (Application Data)
	// Version (2 bytes): 0x03 0x03 (TLS 1.2)
	// Length (2 bytes): len(b)
	// Data: b

	const maxRecordSize = 16384
	written := 0

	for written < len(b) {
		chunkSize := len(b) - written
		if chunkSize > maxRecordSize {
			chunkSize = maxRecordSize
		}

		chunk := b[written : written+chunkSize]

		// Create TLS record
		record := make([]byte, 5+len(chunk))
		record[0] = 0x17       // Application Data
		record[1] = 0x03       // TLS 1.2
		record[2] = 0x03
		binary.BigEndian.PutUint16(record[3:5], uint16(len(chunk)))
		copy(record[5:], chunk)

		// Write record
		n, err := c.Conn.Write(record)
		if err != nil {
			return written, err
		}

		if n < 5 {
			return written, fmt.Errorf("failed to write complete TLS record header")
		}

		written += n - 5 // Subtract header bytes from count
	}

	return written, nil
}
