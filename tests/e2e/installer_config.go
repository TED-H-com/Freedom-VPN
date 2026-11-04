package e2e

import "encoding/json"

// installerServingConfig is the existing TEST ONLY runtime drop-in. Keeping
// the transformation separate lets offline tests pin its unchanged cadence.
func installerServingConfig(raw []byte) ([]byte, error) {
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	cfg["dns"] = map[string]any{"resolvers": []any{map[string]string{"id": "lab", "admin_url": "http://127.0.0.1:8080", "dns_addr": "127.0.0.1:53"}}, "resolver_credentials_file": "/etc/vpn-node/resolver-credentials.json"}
	cfg["test_mode"] = true
	cfg["heartbeat_seconds"], cfg["poll_seconds"], cfg["observation_seconds"] = 5, 5, 5
	return json.MarshalIndent(cfg, "", "  ")
}
