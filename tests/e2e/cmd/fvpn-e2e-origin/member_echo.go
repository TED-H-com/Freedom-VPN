package main

import (
	"bufio"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth"
	"github.com/google/uuid"
)

// memberEcho is the TEST ONLY persistent member session. TCP :9446 registers
// a device and echoes lines. UDP :9445 answers probeauth only for a device
// that TCP registered. It is not the updater probe on UDP :4443.
type memberEcho struct {
	mu      sync.Mutex
	devices map[string]int
}

func newMemberEcho() *memberEcho {
	return &memberEcho{devices: map[string]int{}}
}

func (m *memberEcho) serveTCP(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go m.handleTCP(conn)
	}
}

func (m *memberEcho) handleTCP(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(25 * time.Minute))
	reader := bufio.NewReaderSize(conn, 2048)
	line, err := readLimitedLine(reader, 80)
	id, ok := memberHello(line)
	if err != nil || !ok {
		return
	}
	m.mu.Lock()
	m.devices[id.String()]++
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.devices[id.String()]--
		if m.devices[id.String()] == 0 {
			delete(m.devices, id.String())
		}
	}()
	if _, err = conn.Write([]byte("OK\n")); err != nil {
		return
	}
	for {
		line, err = readLimitedLine(reader, 1024)
		if err != nil {
			return
		}
		if _, err = io.WriteString(conn, "ECHO "+line+"\n"); err != nil {
			return
		}
	}
}

func (m *memberEcho) registered(id uuid.UUID) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.devices[id.String()]
	return ok
}

func (m *memberEcho) serveUDP(conn net.PacketConn, key []byte) {
	buf := make([]byte, 2048)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			return
		}
		if n != probeauth.Size {
			continue
		}
		pkt := append([]byte(nil), buf[:n]...)
		id, ok := probeauth.Open(key, pkt, false)
		if !ok || !m.registered(id) {
			continue
		}
		_, _ = conn.WriteTo(probeauth.Seal(key, id, true), addr)
	}
}

func memberHello(line string) (uuid.UUID, bool) {
	const prefix = "MEMBER "
	if !strings.HasPrefix(line, prefix) {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(strings.TrimSpace(line[len(prefix):]))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, false
	}
	return id, true
}

func readLimitedLine(r *bufio.Reader, max int) (string, error) {
	var buf []byte
	for len(buf) <= max {
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		if b == '\n' {
			return strings.TrimRight(string(buf), "\r"), nil
		}
		buf = append(buf, b)
	}
	return "", io.ErrUnexpectedEOF
}
