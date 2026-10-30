package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"time"
)

// TEST ONLY authenticated GET burst. Cookies arrive on protected stdin and
// never appear in arguments, files or output. Only fixed response metadata is
// returned, with no response body, device ID, cookie or request header values.
func httpBurst() error {
	var input struct {
		Dial   string `json:"dial"`
		Cookie string `json:"cookie"`
		Count  int    `json:"count"`
		Forge  bool   `json:"forge"`
	}
	if json.NewDecoder(io.LimitReader(os.Stdin, 32768)).Decode(&input) != nil {
		return fmt.Errorf("invalid burst input")
	}
	addr, err := netip.ParseAddrPort(input.Dial)
	if err != nil || !netip.MustParsePrefix("172.28.0.0/16").Contains(addr.Addr()) || addr.Port() != 8443 || input.Count < 1 || input.Count > 200 || input.Cookie == "" {
		return fmt.Errorf("burst must target a bounded lab public listener")
	}
	ca, err := os.ReadFile("/etc/vpn-node/ca.pem")
	if err != nil {
		return err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return fmt.Errorf("lab trust unavailable")
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}, DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, network, input.Dial)
	}}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	out := struct {
		Counts          map[int]int `json:"status_counts"`
		RateLimited     int         `json:"rate_limited_codes"`
		RetryAfter      int         `json:"retry_after_present"`
		AddressRefusals int         `json:"address_refusals"`
		SubjectRefusals int         `json:"subject_refusals"`
		Seconds         float64     `json:"elapsed_seconds"`
	}{Counts: map[int]int{}}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	start := time.Now()
	for i := 0; i < input.Count; i++ {
		req, err := http.NewRequestWithContext(ctx, "GET", "https://portal.example.invalid:8443/api/v1/member/devices", nil)
		if err != nil {
			return err
		}
		req.Header.Set("Cookie", input.Cookie)
		if input.Forge {
			req.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", 1+i%200))
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("bounded burst transport failed")
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("burst response unavailable")
		}
		out.Counts[resp.StatusCode]++
		if resp.StatusCode == 429 {
			var apiError struct {
				Error struct{ Code, Message string }
			}
			if json.Unmarshal(body, &apiError) == nil && apiError.Error.Code == "RATE_LIMITED" {
				out.RateLimited++
			}
			if resp.Header.Get("Retry-After") != "" {
				out.RetryAfter++
			}
			switch apiError.Error.Message {
			case "Authentication request rate exceeded.":
				out.AddressRefusals++
			case "Request rate exceeded.":
				out.SubjectRefusals++
			}
		}
	}
	out.Seconds = time.Since(start).Seconds()
	return json.NewEncoder(os.Stdout).Encode(out)
}
