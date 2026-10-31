//go:build linux

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/sys/unix"
)

// Only this assigned isolated origin is reachable. No payload, address, URL,
// certificate or packet is retained: observers emit counts per interface.
const controlledOrigin = "192.0.4.10"
const probePort = 38443

func probeSourcePort() uint16 {
	if len(os.Args) == 3 {
		return probePort
	}
	if len(os.Args) != 4 {
		os.Exit(2)
	}
	port, err := strconv.ParseUint(os.Args[3], 10, 16)
	if err != nil || port < 1024 {
		os.Exit(2)
	}
	return uint16(port)
}

func transfer() {
	port := probeSourcePort()
	result := map[string]any{"http_status": 0, "body_bytes": 0, "origin_receipt": false}
	defer func() { _ = json.NewEncoder(os.Stdout).Encode(result) }()
	raw, err := os.ReadFile("/etc/vpn-node/ca.pem")
	if err != nil {
		return
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(raw) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	tr := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}, DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		if addr != "releases.example.invalid:443" {
			return nil, fmt.Errorf("outside lab")
		}
		return (&net.Dialer{Timeout: 5 * time.Second, LocalAddr: &net.TCPAddr{Port: int(port)}}).DialContext(ctx, "tcp4", net.JoinHostPort(controlledOrigin, "443"))
	}}
	defer tr.CloseIdleConnections()
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://releases.example.invalid/healthz", nil)
	req.Header.Set("X-E2E-Probe", os.Args[2])
	res, err := (&http.Client{Transport: tr, Timeout: 6 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 4096))
	result["http_status"] = res.StatusCode
	result["body_bytes"] = len(body)
	result["origin_receipt"] = err == nil && res.StatusCode == 200 && string(body) == "ok\n"
}

func observe() {
	port := probeSourcePort()
	counts := map[string]map[string]int{}
	result := map[string]any{"available": false, "interfaces": counts, "duration_seconds": 9, "filter": fmt.Sprintf("controlled-origin TCP/443, client port %d", port)}
	defer func() { _ = json.NewEncoder(os.Stdout).Encode(result) }()
	// SOCK_DGRAM removes the link header on both Ethernet and WireGuard.
	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_DGRAM|unix.SOCK_NONBLOCK, int(htons(unix.ETH_P_ALL)))
	if err != nil {
		return
	}
	defer unix.Close(fd)
	result["available"] = true
	buf := make([]byte, 65536)
	started := time.Now()
	end := started.Add(9 * time.Second)
	defer func() { result["elapsed_ms"] = time.Since(started).Milliseconds() }()
	for time.Now().Before(end) {
		_, err := unix.Poll([]unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}, 100)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			result["poll_error"] = true
			return
		}
		n, addr, err := unix.Recvfrom(fd, buf, 0)
		if err != nil {
			continue
		}
		direction := packetDirection(buf[:n], port)
		if direction == "" {
			continue
		}
		link, ok := addr.(*unix.SockaddrLinklayer)
		if !ok {
			continue
		}
		iface, err := net.InterfaceByIndex(link.Ifindex)
		if err != nil {
			continue
		}
		if counts[iface.Name] == nil {
			counts[iface.Name] = map[string]int{}
		}
		counts[iface.Name][direction]++
	}
}
func htons(v uint16) uint16 { return v<<8 | v>>8 }
func packetDirection(p []byte, port uint16) string {
	if len(p) < 20 || p[0]>>4 != 4 || p[9] != 6 {
		return ""
	}
	offset := int(p[0]&15) * 4
	if offset < 20 || len(p) < offset+20 {
		return ""
	}
	src, dst := net.IP(p[12:16]).String(), net.IP(p[16:20]).String()
	sport, dport := binary.BigEndian.Uint16(p[offset:]), binary.BigEndian.Uint16(p[offset+2:])
	if dst == controlledOrigin && dport == 443 && sport == port {
		return "to_origin"
	}
	if src == controlledOrigin && sport == 443 && dport == port {
		return "from_origin"
	}
	return ""
}
