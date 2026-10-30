// Command proxyprobe runs bounded AT28 requests inside an owned lab namespace.
// Credentials arrive over protected stdin; only response metadata is emitted.
package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

func main() {
	var in struct {
		Dial, Method, Path string
		Body               json.RawMessage
		Headers            map[string]string
		Count              int
		Forge              bool
	}
	if json.NewDecoder(io.LimitReader(os.Stdin, 1<<20)).Decode(&in) != nil || in.Count < 1 || in.Count > 150 {
		os.Exit(2)
	}
	ca, err := os.ReadFile("/etc/vpn-node/ca.pem")
	if err != nil {
		os.Exit(2)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca) {
		os.Exit(2)
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, "tcp", in.Dial)
	}}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
	rows := []map[string]any{}
	for i := 0; i < in.Count; i++ {
		req, err := http.NewRequest(in.Method, "https://portal.example.invalid"+in.Path, bytes.NewReader(in.Body))
		if err != nil {
			os.Exit(2)
		}
		req.Header.Set("Content-Type", "application/json")
		for k, v := range in.Headers {
			req.Header.Set(k, v)
		}
		if in.Forge {
			req.Header.Set("X-Forwarded-For", fmt.Sprintf("192.0.2.%d", i%200+1))
		}
		started := time.Now()
		resp, err := client.Do(req)
		row := map[string]any{"index": i + 1, "utc": started.UTC(), "elapsed_ms": float64(time.Since(started).Microseconds()) / 1000, "status": 0}
		if err == nil {
			raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			_ = resp.Body.Close()
			var e struct {
				Error struct {
					Code    string         `json:"code"`
					Details map[string]any `json:"details"`
				} `json:"error"`
			}
			_ = json.Unmarshal(raw, &e)
			row["status"] = resp.StatusCode
			row["code"] = e.Error.Code
			row["retry_after"] = resp.Header.Get("Retry-After")
			row["limiter"] = e.Error.Details["limiter"]
		}
		rows = append(rows, row)
	}
	_ = json.NewEncoder(os.Stdout).Encode(rows)
}
