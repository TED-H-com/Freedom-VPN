package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

// dnsProbe reads only configuration counts and a synthetic answer. Credentials
// and query names never enter evidence or logs.
func dnsProbe() error {
	raw, err := os.ReadFile("/etc/vpn-node/resolver-credentials.json")
	if err != nil {
		return err
	}
	var cred struct{ Username, Password string }
	if err = json.Unmarshal(raw, &cred); err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	read := func(path string, out any) error {
		req, err := http.NewRequest("GET", "http://127.0.0.1:8080/control/"+path, nil)
		if err != nil {
			return err
		}
		req.SetBasicAuth(cred.Username, cred.Password)
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("resolver status %d", resp.StatusCode)
		}
		return json.NewDecoder(resp.Body).Decode(out)
	}
	var filter struct {
		Rules []string `json:"user_rules"`
	}
	var queries struct {
		Enabled     bool `json:"enabled"`
		FileEnabled bool `json:"file_enabled"`
	}
	var stats struct {
		Enabled bool `json:"enabled"`
	}
	if err = read("filtering/status", &filter); err != nil {
		return err
	}
	if err = read("querylog/config", &queries); err != nil {
		return err
	}
	if err = read("stats/config", &stats); err != nil {
		return err
	}
	rules, _ := json.Marshal(filter.Rules)
	hash := sha256.Sum256(rules)
	resolver := &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "udp", "127.0.0.1:53")
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ips, dnsErr := resolver.LookupHost(ctx, "feed-01.example.invalid")
	blocked := false
	if e, ok := dnsErr.(*net.DNSError); ok {
		blocked = e.IsNotFound
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"admin_http": 200, "rule_count": len(filter.Rules), "rules_sha256": hex.EncodeToString(hash[:]), "query_log_enabled": queries.Enabled, "query_log_file_enabled": queries.FileEnabled, "statistics_enabled": stats.Enabled, "synthetic_blocked": blocked, "synthetic_answer_count": len(ips)})
}
