//go:build ignore

package main

// TEST ONLY transparent mTLS receiver. Bodies are retained only for observations
// and lease renewal; authentication headers and credential responses are excluded.
import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func observeProxy() {
	roots := func(path string) *x509.CertPool {
		raw, e := os.ReadFile(path)
		must(e)
		p := x509.NewCertPool()
		if !p.AppendCertsFromPEM(raw) {
			os.Exit(1)
		}
		return p
	}
	cert, e := tls.LoadX509KeyPair("/var/lib/vpn-node/identity/cert.pem", "/var/lib/vpn-node/identity/key.pem")
	must(e)
	target, _ := url.Parse("https://agent.example.invalid:9443")
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots("/etc/vpn-node/ca.pem"), Certificates: []tls.Certificate{cert}}, ResponseHeaderTimeout: 15 * time.Second}
	file, e := os.OpenFile("/var/lib/vpn-node/e2e/isolation-observations.jsonl", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	must(e)
	defer file.Close()
	var mu sync.Mutex
	var queue []map[string]any
	go func() {
		for range time.NewTicker(100 * time.Millisecond).C {
			mu.Lock()
			batch := queue
			queue = nil
			mu.Unlock()
			for _, row := range batch {
				_ = json.NewEncoder(file).Encode(row)
			}
			_ = file.Sync()
		}
	}()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !((r.Method == "POST" && (r.URL.Path == "/api/v1/agent/observations" || strings.Contains(r.URL.Path, "lease"))) || (r.Method == "GET" && strings.Contains(r.URL.Path, "desired-state"))) {
			proxy.ServeHTTP(w, r)
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
		if err != nil {
			http.Error(w, "capture read failed", 500)
			return
		}
		r.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(raw))
		row := map[string]any{"received_at": time.Now().UTC().Format(time.RFC3339Nano), "path": r.URL.Path}
		if json.Valid(raw) {
			row["body"] = json.RawMessage(raw)
		}
		capture := &captureResponse{ResponseWriter: w}
		proxy.ServeHTTP(capture, r)
		row["response_at"] = time.Now().UTC().Format(time.RFC3339Nano)
		row["status"] = capture.status
		if json.Valid(capture.body.Bytes()) {
			row["response"] = json.RawMessage(capture.body.Bytes())
		}
		mu.Lock()
		queue = append(queue, row)
		mu.Unlock()
	})
	srv := &http.Server{Addr: "127.0.0.1:9444", Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12, ClientCAs: roots("/etc/vpn-node/agent-ca.pem"), ClientAuth: tls.RequireAndVerifyClientCert}}
	must(srv.ListenAndServeTLS("/etc/vpn-node/fault-proxy.crt", "/etc/vpn-node/fault-proxy.key"))
}

type captureResponse struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *captureResponse) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }
func (w *captureResponse) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func configureObservation() {
	p := "/etc/vpn-node/agent-run.json"
	raw, e := os.ReadFile(p)
	must(e)
	var cfg map[string]any
	must(json.Unmarshal(raw, &cfg))
	for _, k := range []string{"heartbeat_seconds", "poll_seconds", "observation_seconds"} {
		cfg[k] = 1
	}
	raw, e = json.MarshalIndent(cfg, "", "  ")
	must(e)
	must(os.WriteFile(p, raw, 0600))
}

// All timestamps use the host CLOCK_MONOTONIC shared by these containers.
func monoNS() int64 {
	var ts unix.Timespec
	must(unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts))
	return ts.Nano()
}
func clockSample() map[string]string {
	return map[string]string{"mono_ns": strconv.FormatInt(monoNS(), 10), "wall": time.Now().UTC().Format(time.RFC3339Nano)}
}
func deadlineProbes() {
	deadline, e := strconv.ParseInt(os.Args[3], 10, 64)
	must(e)
	f, e := os.Create(os.Args[4])
	must(e)
	defer f.Close()
	// The main cadence is phased at D-(100ms+1ms). A second cadence has
	// an arbitrary 37ms offset and shares the explicit D-101ms boundary sample.
	type scheduled struct {
		at    int64
		phase string
	}
	var schedule []scheduled
	firstN, lastN := int64(-80), int64(20)
	if os.Args[1] == "renewal-probes" {
		deadline = monoNS() + 70000000000
		firstN, lastN = -698, -1
	}
	for n := firstN; n <= lastN; n++ {
		schedule = append(schedule, scheduled{deadline - 101000000 + n*100000000, "phased"})
		schedule = append(schedule, scheduled{deadline - 64000000 + n*100000000, "arbitrary_37ms"})
	}
	sort.Slice(schedule, func(i, j int) bool { return schedule[i].at < schedule[j].at })
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i, s := range schedule {
		if delay := s.at - monoNS(); delay > 0 {
			time.Sleep(time.Duration(delay))
		}
		wg.Add(1)
		go func(id int, s scheduled) {
			defer wg.Done()
			port := 42000 + id
			dst := &net.UDPAddr{IP: net.ParseIP(os.Args[2]), Port: 8080}
			c, e := net.DialUDP("udp4", &net.UDPAddr{Port: port}, dst)
			row := map[string]any{"packet_id": id, "source_port": port, "phase": s.phase, "scheduled_mono_ns": fmt.Sprint(s.at), "deadline_mono_ns": fmt.Sprint(deadline), "send": clockSample()}
			result := "socket_error"
			if e == nil {
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(80 * time.Millisecond))
				_, e = c.Write([]byte("TEST ONLY"))
				if e == nil {
					b := make([]byte, 9)
					n, err := c.Read(b)
					result = "timeout"
					if err == nil && n == 9 && string(b) == "TEST ONLY" {
						result = "ok"
					}
				} else {
					result = "send_error"
				}
			}
			row["read_completion"] = clockSample()
			row["result"] = result
			mu.Lock()
			_ = json.NewEncoder(f).Encode(row)
			mu.Unlock()
		}(i+1, s)
	}
	wg.Wait()
}
func packetClock() {
	seconds, e := strconv.Atoi(os.Args[2])
	must(e)
	f, e := os.Create(os.Args[3])
	must(e)
	defer f.Close()
	// SOCK_DGRAM strips link headers on both WireGuard and Ethernet. Retain
	// only UDP headers for TEST ONLY source-port packet IDs, never payload.
	fd, e := unix.Socket(unix.AF_PACKET, unix.SOCK_DGRAM|unix.SOCK_NONBLOCK, int(0x0300))
	must(e)
	defer unix.Close(fd)
	ifaces, e := net.Interfaces()
	must(e)
	names := map[int]string{}
	for _, it := range ifaces {
		names[it.Index] = it.Name
	}
	_ = json.NewEncoder(f).Encode(map[string]any{"event": "ready", "clock": clockSample()})
	end := monoNS() + int64(seconds)*1000000000
	b := make([]byte, 65536)
	for monoNS() < end {
		n, sa, e := unix.Recvfrom(fd, b, 0)
		stamp := clockSample()
		if e == unix.EAGAIN {
			time.Sleep(200 * time.Microsecond)
			continue
		}
		if e != nil {
			must(e)
		}
		link, ok := sa.(*unix.SockaddrLinklayer)
		if !ok || n < 28 || b[0]>>4 != 4 || b[9] != 17 {
			continue
		}
		ihl := int(b[0]&15) * 4
		if ihl < 20 || n < ihl+8 {
			continue
		}
		src, dst := int(binary.BigEndian.Uint16(b[ihl:])), int(binary.BigEndian.Uint16(b[ihl+2:]))
		id := src - 42000
		if dst != 8080 || id < 1 || id > 2000 {
			continue
		}
		name := names[link.Ifindex]
		if name != "fvmb0" && name != "fvup0" {
			continue
		}
		_ = json.NewEncoder(f).Encode(map[string]any{"packet_id": id, "interface": name, "packet_type": link.Pkttype, "capture_read": stamp, "source_port": src, "destination_port": dst})
	}
	stats, e := unix.GetsockoptTpacketStats(fd, unix.SOL_PACKET, unix.PACKET_STATISTICS)
	row := map[string]any{"event": "complete", "clock": clockSample(), "statistics_available": e == nil}
	if e == nil {
		row["packets"] = stats.Packets
		row["drops"] = stats.Drops
	}
	_ = json.NewEncoder(f).Encode(row)
}

// Passive netlink monitor: timestamps are userspace receipt bounds, never
// mislabelled as renderer start or command completion.
func nftClock() {
	f, e := os.OpenFile("/var/lib/vpn-node/e2e/isolation-nft-events.jsonl", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	must(e)
	defer f.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "nft", "-j", "monitor")
	pipe, e := cmd.StdoutPipe()
	must(e)
	must(cmd.Start())
	scanner := bufio.NewScanner(pipe)
	scanner.Buffer(make([]byte, 65536), 8<<20)
	for scanner.Scan() {
		at := clockSample()
		line := scanner.Bytes()
		var doc map[string]any
		if json.Unmarshal(line, &doc) != nil {
			continue
		}
		entries, _ := doc["nftables"].([]any)
		if len(entries) == 0 {
			entries = []any{doc}
		}
		for _, entry := range entries {
			actions, _ := entry.(map[string]any)
			for action, obj := range actions {
				kinds, _ := obj.(map[string]any)
				for kind, body := range kinds {
					b, _ := body.(map[string]any)
					if b["table"] != "fvpn" {
						continue
					}
					name, _ := b["name"].(string)
					chain, _ := b["chain"].(string)
					if name != "fv_deadline_guard" && name != "allowed_v4" && kind != "rule" {
						continue
					}
					row := map[string]any{"receipt": at, "action": action, "kind": kind, "name": name, "chain": chain}
					if comment, ok := b["comment"].(string); ok && strings.HasPrefix(comment, "fvpn_generation=") {
						row["generation_comment"] = comment
					}
					_ = json.NewEncoder(f).Encode(row)
				}
			}
		}
	}
	_ = cmd.Wait()
}
