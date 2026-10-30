package main

import (
	"net"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth"
)

func (p *trafficProbe) datagram(key, raw []byte, source string) []byte {
	id, ok := probeauth.Open(key, raw, false)
	if !ok {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.id == "" || p.id != id.String() {
		return nil
	}
	p.count++
	p.source = source
	return probeauth.Seal(key, id, true)
}
func (p *trafficProbe) serveUDP(conn net.PacketConn, key []byte) {
	var buf [256]byte
	for {
		n, addr, err := conn.ReadFrom(buf[:])
		if err != nil {
			return
		}
		host, _, err := net.SplitHostPort(addr.String())
		if err != nil {
			continue
		}
		reply := p.datagram(key, buf[:n], host)
		if len(reply) == 0 {
			continue
		}
		_, _ = conn.WriteTo(reply, addr)
	}
}
