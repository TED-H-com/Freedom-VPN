package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"time"
)

// A persistent HTTPS connection to the existing controlled origin. No new
// service, payload logging or UDP endpoint is introduced by this test probe.
func monitor() {
	raw, err := os.ReadFile("/etc/vpn-node/ca.pem")
	if err != nil {
		return
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(raw) {
		return
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{Timeout: time.Second}).DialContext(ctx, "tcp4", net.JoinHostPort(controlledOrigin, "443"))
	}}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, Timeout: 2 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	end := time.Now().Add(85 * time.Second)
	positive := false
	for time.Now().Before(end) {
		reused := false
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{GotConn: func(info httptrace.GotConnInfo) { reused = info.Reused }})
		req, _ := http.NewRequestWithContext(ctx, "GET", "https://releases.example.invalid/healthz", nil)
		res, err := client.Do(req)
		ok := false
		if err == nil {
			body, e := io.ReadAll(io.LimitReader(res.Body, 4096))
			_ = res.Body.Close()
			ok = e == nil && res.StatusCode == 200 && string(body) == "ok\n"
		}
		cancel()
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"at": time.Now().UTC().Format(time.RFC3339Nano), "origin_receipt": ok, "reused_tcp": reused})
		if positive && !ok {
			return
		}
		positive = positive || ok
		time.Sleep(500 * time.Millisecond)
	}
}
