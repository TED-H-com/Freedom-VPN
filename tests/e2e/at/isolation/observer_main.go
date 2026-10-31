//go:build ignore

package main

// TEST ONLY transparent mTLS receiver. Bodies are retained only for observations
// and lease renewal; authentication headers and credential responses are excluded.
import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
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
