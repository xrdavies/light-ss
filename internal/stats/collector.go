package stats

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Collector collects statistics about connections and bandwidth
type Collector struct {
	mu sync.RWMutex

	// Connection counters
	totalConnections   atomic.Int64
	activeConnections  atomic.Int64
	httpConnections    atomic.Int64
	socks5Connections  atomic.Int64

	// Bandwidth counters
	bytesSent     atomic.Int64
	bytesReceived atomic.Int64

	// Start time
	startTime time.Time
}

// NewCollector creates a new stats collector
func NewCollector() *Collector {
	return &Collector{
		startTime: time.Now(),
	}
}

// RecordConnection records a new connection
func (c *Collector) RecordConnection(proxyType string) {
	c.totalConnections.Add(1)
	c.activeConnections.Add(1)

	switch proxyType {
	case "http":
		c.httpConnections.Add(1)
	case "socks5":
		c.socks5Connections.Add(1)
	}
}

// RecordDisconnection records a connection closure
func (c *Collector) RecordDisconnection() {
	c.activeConnections.Add(-1)
}

// RecordBytesSent records bytes sent
func (c *Collector) RecordBytesSent(n int64) {
	c.bytesSent.Add(n)
}

// RecordBytesReceived records bytes received
func (c *Collector) RecordBytesReceived(n int64) {
	c.bytesReceived.Add(n)
}

// GetStats returns current statistics
func (c *Collector) GetStats() Stats {
	return Stats{
		TotalConnections:   c.totalConnections.Load(),
		ActiveConnections:  c.activeConnections.Load(),
		HTTPConnections:    c.httpConnections.Load(),
		SOCKS5Connections:  c.socks5Connections.Load(),
		BytesSent:          c.bytesSent.Load(),
		BytesReceived:      c.bytesReceived.Load(),
		Uptime:             time.Since(c.startTime),
	}
}

// Stats holds statistics data
type Stats struct {
	TotalConnections   int64
	ActiveConnections  int64
	HTTPConnections    int64
	SOCKS5Connections  int64
	BytesSent          int64
	BytesReceived      int64
	Uptime             time.Duration
}

// TrackedConn wraps a net.Conn to track bandwidth
type TrackedConn struct {
	net.Conn
	collector *Collector
	proxyType string
	target    string
	closed    bool
	mu        sync.Mutex
}

// NewTrackedConn creates a new tracked connection
func NewTrackedConn(conn net.Conn, collector *Collector, proxyType, target string) *TrackedConn {
	collector.RecordConnection(proxyType)

	return &TrackedConn{
		Conn:      conn,
		collector: collector,
		proxyType: proxyType,
		target:    target,
	}
}

// Read reads from the connection and tracks bytes
func (t *TrackedConn) Read(b []byte) (int, error) {
	n, err := t.Conn.Read(b)
	if n > 0 {
		t.collector.RecordBytesReceived(int64(n))
	}
	return n, err
}

// Write writes to the connection and tracks bytes
func (t *TrackedConn) Write(b []byte) (int, error) {
	n, err := t.Conn.Write(b)
	if n > 0 {
		t.collector.RecordBytesSent(int64(n))
	}
	return n, err
}

// Close closes the connection and records disconnection
func (t *TrackedConn) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.closed {
		t.closed = true
		t.collector.RecordDisconnection()
	}

	return t.Conn.Close()
}

var _ net.Conn = (*TrackedConn)(nil)
var _ io.ReadWriteCloser = (*TrackedConn)(nil)
