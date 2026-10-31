// Package streamstats records TEST ONLY endpoint counts and TCP_INFO, never
// application bytes, addresses, device IDs, or error strings containing them.
package streamstats

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

type Sample struct {
	At         time.Time         `json:"at"`
	LocalPort  int               `json:"local_port"`
	RemotePort int               `json:"remote_port"`
	ReadBytes  int64             `json:"read_bytes"`
	WriteBytes int64             `json:"write_bytes"`
	Lines      int               `json:"complete_lines"`
	Partial    int               `json:"partial_line_bytes"`
	Error      string            `json:"error"`
	TCP        map[string]uint64 `json:"tcp_info"`
}

type Conn struct {
	net.Conn
	mu     sync.Mutex
	sample Sample
}

func Wrap(c net.Conn) *Conn { return &Conn{Conn: c, sample: Sample{Error: "none"}} }
func Category(e error) string {
	if e == nil {
		return "none"
	}
	if errors.Is(e, io.EOF) {
		return "eof"
	}
	var n net.Error
	if errors.As(e, &n) && n.Timeout() {
		return "timeout"
	}
	return "io_error"
}
func (c *Conn) Read(b []byte) (int, error) {
	n, e := c.Conn.Read(b)
	c.mu.Lock()
	c.sample.ReadBytes += int64(n)
	if e != nil {
		c.sample.Error = Category(e)
	}
	c.mu.Unlock()
	return n, e
}
func (c *Conn) Write(b []byte) (int, error) {
	n, e := c.Conn.Write(b)
	c.mu.Lock()
	c.sample.WriteBytes += int64(n)
	if e != nil {
		c.sample.Error = Category(e)
	}
	c.mu.Unlock()
	return n, e
}
func (c *Conn) Line(length int, complete bool, e error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if complete {
		c.sample.Lines++
	} else {
		c.sample.Partial = length
		if e == nil {
			c.sample.Error = "line_mismatch"
		}
	}
	if e != nil {
		c.sample.Error = Category(e)
	}
}
func (c *Conn) Snapshot() Sample {
	c.mu.Lock()
	s := c.sample
	c.mu.Unlock()
	s.At = time.Now().UTC()
	if a, ok := c.LocalAddr().(*net.TCPAddr); ok {
		s.LocalPort = a.Port
	}
	if a, ok := c.RemoteAddr().(*net.TCPAddr); ok {
		s.RemotePort = a.Port
	}
	s.TCP = tcpInfo(c.Conn)
	return s
}
