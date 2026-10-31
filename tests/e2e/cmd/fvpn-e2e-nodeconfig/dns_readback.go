package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// TEST ONLY read-only, allowlisted settings; credentials and upstream addresses
// never leave the container. This is independent of the published policy body.
func dnsReadback() error {
	raw, err := os.ReadFile("/etc/vpn-node/resolver-credentials.json")
	if err != nil {
		return err
	}
	var credentials struct{ Username, Password string }
	if err = json.Unmarshal(raw, &credentials); err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:8080/control/dns_info", nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(credentials.Username, credentials.Password)
	response, err := (&http.Client{Timeout: 4 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return fmt.Errorf("resolver readback status %d", response.StatusCode)
	}
	var info map[string]json.RawMessage
	if err = json.NewDecoder(response.Body).Decode(&info); err != nil {
		return err
	}
	out := map[string]any{"read_at": time.Now().UTC(), "source": "GET /control/dns_info"}
	for _, key := range []string{"cache_size", "cache_ttl_min", "cache_ttl_max", "cache_optimistic", "upstream_mode", "upstream_timeout", "blocking_mode", "blocked_response_ttl"} {
		value, ok := info[key]
		if !ok {
			return fmt.Errorf("missing setting %s", key)
		}
		out[key] = value
	}
	var fallback []string
	if err = json.Unmarshal(info["fallback_dns"], &fallback); err != nil {
		return err
	}
	out["fallback_dns_count"] = len(fallback)
	return json.NewEncoder(os.Stdout).Encode(out)
}
