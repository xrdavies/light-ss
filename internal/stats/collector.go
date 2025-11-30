package stats

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// SpeedSample represents bandwidth measurement at a point in time
type SpeedSample struct {
	timestamp time.Time
	sent      int64
	received  int64
}

// SpeedTracker tracks bandwidth speed using a sliding window
type SpeedTracker struct {
	samples    []SpeedSample
	windowSize time.Duration
	mu         sync.RWMutex
}

// NewSpeedTracker creates a new speed tracker
func NewSpeedTracker(windowSize time.Duration) *SpeedTracker {
	return &SpeedTracker{
		samples:    make([]SpeedSample, 0, 100), // Pre-allocate for efficiency
		windowSize: windowSize,
	}
}

// AddSample records a new speed sample
func (st *SpeedTracker) AddSample(sent, received int64) {
	st.mu.Lock()
	defer st.mu.Unlock()

	now := time.Now()
	st.samples = append(st.samples, SpeedSample{
		timestamp: now,
		sent:      sent,
		received:  received,
	})

	// Remove samples outside the window
	cutoff := now.Add(-st.windowSize)
	validIdx := 0
	for i, sample := range st.samples {
		if sample.timestamp.After(cutoff) {
			validIdx = i
			break
		}
	}
	if validIdx > 0 {
		st.samples = st.samples[validIdx:]
	}
}

// GetCurrentSpeed calculates current speed in bytes/sec
func (st *SpeedTracker) GetCurrentSpeed() (uploadSpeed, downloadSpeed int64) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	if len(st.samples) < 2 {
		return 0, 0
	}

	// Get oldest and newest samples
	oldest := st.samples[0]
	newest := st.samples[len(st.samples)-1]

	duration := newest.timestamp.Sub(oldest.timestamp).Seconds()
	if duration == 0 {
		return 0, 0
	}

	sentDiff := newest.sent - oldest.sent
	receivedDiff := newest.received - oldest.received

	uploadSpeed = int64(float64(sentDiff) / duration)
	downloadSpeed = int64(float64(receivedDiff) / duration)

	return uploadSpeed, downloadSpeed
}

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

	// Speed tracker
	speedTracker *SpeedTracker

	// Start time
	startTime time.Time

	// Background ticker for speed sampling
	ticker *time.Ticker
	done   chan struct{}
}

// NewCollector creates a new stats collector
func NewCollector() *Collector {
	c := &Collector{
		startTime:    time.Now(),
		speedTracker: NewSpeedTracker(10 * time.Second), // 10-second window
		done:         make(chan struct{}),
	}

	// Start background speed sampling (every second)
	c.ticker = time.NewTicker(1 * time.Second)
	go c.sampleSpeed()

	return c
}

// sampleSpeed records periodic speed samples
func (c *Collector) sampleSpeed() {
	for {
		select {
		case <-c.ticker.C:
			sent := c.bytesSent.Load()
			received := c.bytesReceived.Load()
			c.speedTracker.AddSample(sent, received)
		case <-c.done:
			return
		}
	}
}

// Stop stops the speed sampling
func (c *Collector) Stop() {
	if c.ticker != nil {
		c.ticker.Stop()
	}
	close(c.done)
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
	uploadSpeed, downloadSpeed := c.speedTracker.GetCurrentSpeed()

	return Stats{
		TotalConnections:   c.totalConnections.Load(),
		ActiveConnections:  c.activeConnections.Load(),
		HTTPConnections:    c.httpConnections.Load(),
		SOCKS5Connections:  c.socks5Connections.Load(),
		BytesSent:          c.bytesSent.Load(),
		BytesReceived:      c.bytesReceived.Load(),
		UploadSpeed:        uploadSpeed,
		DownloadSpeed:      downloadSpeed,
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
	UploadSpeed        int64 // bytes/sec
	DownloadSpeed      int64 // bytes/sec
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
