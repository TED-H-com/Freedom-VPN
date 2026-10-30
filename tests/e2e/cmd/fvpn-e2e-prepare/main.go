// Command fvpn-e2e-prepare writes one run of dev-stack secrets and config.
// Output stays in the git-ignored state directory. It does not reuse TEST ONLY
// fixture keys (D-039, D-045) and it never prints secret material.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labtls"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	state := flag.String("state", "", "git-ignored state directory")
	reset := flag.Bool("reset", false, "replace an existing state directory")
	prefix := flag.String("prefix", "e2e", "short stack prefix")
	portBase := flag.Int("port-base", 18000, "loopback port base")
	flag.Parse()
	if *state == "" {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-prepare: -state is required")
		os.Exit(2)
	}
	lay, err := parseLayout(*prefix, *portBase)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fvpn-e2e-prepare: %s\n", err.Error())
		os.Exit(2)
	}
	if err := run(*state, *reset, lay); err != nil {
		fmt.Fprintf(os.Stderr, "fvpn-e2e-prepare: failed: %s\n", err.Error())
		os.Exit(1)
	}
	fmt.Println("e2e state ready")
}

func run(state string, reset bool, lay layout) error {
	marker := filepath.Join(state, "ready")
	if !reset {
		if _, err := os.Stat(marker); err == nil {
			return nil
		}
	}
	if err := os.RemoveAll(state); err != nil {
		return err
	}
	if err := os.MkdirAll(state, 0o700); err != nil {
		return err
	}
	now := time.Now().UTC().Truncate(time.Second)
	notBefore := now.Add(-5 * time.Minute)
	notAfter := now.Add(72 * time.Hour)

	ca, err := labtls.NewAuthority("e2e-lab", notBefore, notAfter)
	if err != nil {
		return err
	}
	if err := writeSecret(filepath.Join(state, "ca.pem"), ca.CertPEM); err != nil {
		return err
	}
	if err := writeSecret(filepath.Join(state, "ca.key"), ca.KeyPEM); err != nil {
		return err
	}
	certs := []struct {
		name string
		dns  []string
		ips  []net.IP
	}{
		{"control", []string{"portal.example.invalid", "agent.example.invalid"}, nil},
		{"idp", []string{"idp.example.invalid"}, nil},
		{"payment", []string{"api.stripe.com"}, nil},
		{"origin", []string{"releases.example.invalid"}, []net.IP{net.ParseIP(lay.OriginIP)}},
		{"netbird", []string{"netbird.example.invalid"}, nil},
	}
	for _, item := range certs {
		certPEM, keyPEM, err := labtls.ServerCert(ca, item.dns[0], item.dns, item.ips, notBefore, notAfter)
		if err != nil {
			return err
		}
		dir := filepath.Join(state, "certs")
		if err := writeSecret(filepath.Join(dir, item.name+".crt"), certPEM); err != nil {
			return err
		}
		if err := writeSecret(filepath.Join(dir, item.name+".key"), keyPEM); err != nil {
			return err
		}
		if item.name == "control" {
			// The secrets directory is one read-only mount. A second mount of
			// these files inside that directory cannot create its mountpoint.
			if err := writeSecret(filepath.Join(state, "secrets", "tls.crt"), certPEM); err != nil {
				return err
			}
			if err := writeSecret(filepath.Join(state, "secrets", "tls.key"), keyPEM); err != nil {
				return err
			}
		}
	}

	password, err := randomHex(24)
	if err != nil {
		return err
	}
	if err := writeSecret(filepath.Join(state, "secrets", "postgres.password"), []byte(password)); err != nil {
		return err
	}
	dsn := "postgres://fvpn:" + password + "@postgres:5432/fvpn?sslmode=disable\n"
	if err := writeSecret(filepath.Join(state, "secrets", "database.url"), []byte(dsn)); err != nil {
		return err
	}
	secrets := map[string]int{
		"oidc.secret":    32,
		"stripe.key":     32,
		"webhook.secret": 32,
		"assume.secret":  32,
		"origin.secret":  32,
	}
	for name, n := range secrets {
		raw, err := randomHex(n)
		if err != nil {
			return err
		}
		if err := writeSecret(filepath.Join(state, "secrets", name), []byte(raw+"\n")); err != nil {
			return err
		}
	}
	vaultKey := make([]byte, 32)
	if _, err := rand.Read(vaultKey); err != nil {
		return err
	}
	if err := writeSecret(filepath.Join(state, "secrets", "vault.key"), vaultKey); err != nil {
		return err
	}
	// Non-empty until NetBird bootstrap replaces it. Control refuses an empty file.
	if err := writeSecret(filepath.Join(state, "secrets", "netbird.token"), []byte("pending-netbird-bootstrap\n")); err != nil {
		return err
	}

	ctx := context.Background()
	vaultDir := filepath.Join(state, "vault")
	vault, err := signing.InitializeFileProvider(vaultDir)
	if err != nil {
		return err
	}
	caKeyID, err := vault.Rotate(ctx, signing.NodeCA, 72*time.Hour, time.Hour)
	if err != nil {
		return err
	}
	for _, purpose := range []signing.Purpose{signing.BootstrapSecret, signing.DesiredState, signing.Lease, signing.Recovery, signing.LedgerEpoch} {
		if _, err := vault.Rotate(ctx, purpose, 72*time.Hour, time.Hour); err != nil {
			return err
		}
	}
	der, err := pki.CreateDevCA(ctx, vault, caKeyID, 72*time.Hour)
	if err != nil {
		return err
	}
	agentCA := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := writeSecret(filepath.Join(state, "secrets", "agent-ca.pem"), agentCA); err != nil {
		return err
	}
	releaseKey, err := release.GenerateKey("e2e-release-" + now.Format("20060102T150405Z"))
	if err != nil {
		return err
	}
	if err := releaseKey.Save(filepath.Join(state, "release", "key.json")); err != nil {
		return err
	}
	if err := vault.PinReleaseRoot(ctx, envelope.TrustedKey{
		KeyID:     releaseKey.KeyID,
		PublicKey: releaseKey.Public(),
		Purposes:  []envelope.Purpose{envelope.PurposeRelease},
		NotBefore: notBefore,
		NotAfter:  notAfter,
	}); err != nil {
		return err
	}
	bundle, err := vault.TrustBundle(ctx)
	if err != nil {
		return err
	}
	rawBundle, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return err
	}
	rawBundle = append(rawBundle, '\n')
	if err := os.MkdirAll(filepath.Join(state, "node", "etc"), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(state, "node", "etc", "trust.json"), rawBundle, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(state, "node", "etc", "ca.pem"), ca.CertPEM, 0o644); err != nil {
		return err
	}

	falseValue := false
	zero := 0
	nodes := config.Nodes{
		ProviderDirectory: "/var/lib/fvpn/vault", CACertificateFile: "/secrets/agent-ca.pem", CAKeyID: caKeyID,
		CertificateSeconds: 86400, RenewalOverlapSeconds: 300, BootstrapSeconds: 300,
		InstallerSeconds: 900, PreinstalledSeconds: 604800, AuthorizationSeconds: 300,
		OnlineSeconds: 180, StuckSeconds: 600, ReconcileSeconds: 5, FetchSeconds: 20, DesiredStateSeconds: 600,
		NetBirdURL: "https://netbird.example.invalid/api", NetBirdDeployment: "e2e",
		NetBirdTokenFile: "/secrets/netbird.token",
		DNSResolverIPv4:  []string{lay.AdGuardIP}, DNSResolverIPv6: []string{},
		ReleaseOrigins: []string{"https://releases.example.invalid"},
	}
	if err := nodes.Validate(); err != nil {
		return err
	}
	members := config.Members{
		Provider: "stripe", APIKeyFile: "/secrets/stripe.key", WebhookSecretFiles: []string{"/secrets/webhook.secret"},
		VaultActiveKey: "e2e-v1", VaultKeyFiles: map[string]string{"e2e-v1": "/secrets/vault.key"},
		LiveMode: &falseValue, PriceIDs: []string{"e2e-price"},
		ReturnURLs:       []string{"https://portal.example.invalid:8443/billing/return"},
		FullRefundPolicy: "revoke_original_period", GraceSeconds: &zero, GraceReason: "",
		TrialMaxSeconds: &zero, GrantMaxSeconds: 86400, DeviceLimit: 3,
		PortalTTLSeconds: 60, BillingLeaseSeconds: 120, MaxWebhookBytes: 1 << 20,
		ReconcileSeconds: 300, DeviceSyncSeconds: 15,
		NetBirdURL: "https://netbird.example.invalid/api", NetBirdDeployment: "e2e",
		NetBirdTokenFile: "/secrets/netbird.token", OverlayScope: "e2e-overlay",
	}
	if err := members.Validate(); err != nil {
		return err
	}
	dnsCfg := config.DNS{Upstreams: []string{lay.CoreDNSIP + ":53"}, BootstrapDNS: []string{lay.CoreDNSIP + ":53"}}
	if err := dnsCfg.Validate(); err != nil {
		return err
	}
	central := map[string]any{
		"public_address": ":8443", "agent_address": ":9443", "metrics_address": "127.0.0.1:9091",
		"public_origin":        "https://portal.example.invalid:8443",
		"tls_certificate_file": "/secrets/tls.crt", "tls_key_file": "/secrets/tls.key",
		"agent_ca_file": "/secrets/agent-ca.pem", "database_url_file": "/secrets/database.url",
		"oidc_issuer": "https://netbird.example.invalid/oauth2", "oidc_client_id": "netbird-dashboard",
		"oidc_secret_file": "/secrets/oidc.secret", "admin_claim": "roles", "admin_value": "admin",
		"mfa_amr": []string{"mfa"}, "recent_auth_seconds": 900,
		"session_idle_seconds": 1800, "session_absolute_seconds": 43200,
		"migrate_on_start":    true,
		"members_config_file": "/etc/fvpn/members.json", "nodes_config_file": "/etc/fvpn/nodes.json",
		"dns_config_file":      "/etc/fvpn/dns.json",
		"policies_config_file": "/etc/fvpn/policies.json",
		"usage_config_file":    "/etc/fvpn/usage.json",
	}
	policies := config.PoliciesConfig{FUPSweepSeconds: 60, FUPBatchSize: 100, DNSUpstreamProfile: "lab"}
	if err := policies.Validate(); err != nil {
		return err
	}
	usageCfg := config.UsageConfig{
		SigningConfigFile: "/var/lib/fvpn/vault", RawUsageRetentionDays: 60, RollupRetentionMonths: 24,
		AuditRetentionDays: 365, ProjectionBatchSize: 512,
	}
	if err := usageCfg.Validate(); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(state, "config", "central.json"), central); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(state, "config", "nodes.json"), nodes); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(state, "config", "members.json"), members); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(state, "config", "dns.json"), dnsCfg); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(state, "config", "policies.json"), policies); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(state, "config", "usage.json"), usageCfg); err != nil {
		return err
	}
	if err := writeAgentConfigs(state); err != nil {
		return err
	}
	if err := writeNetBird(state); err != nil {
		return err
	}
	if err := writeAdGuard(state, lay.CoreDNSIP); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(state, "releases"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(state, "node", "var", "e2e"), 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(state, "member", "var", "e2e"), 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(state, "adguard", "work"), 0o755); err != nil {
		return err
	}
	if err := writeFeed(state); err != nil {
		return err
	}
	stack := map[string]any{
		"public_origin":  "https://portal.example.invalid:8443",
		"agent_origin":   "https://agent.example.invalid:9443",
		"idp_issuer":     "https://netbird.example.invalid/oauth2",
		"netbird_api":    "https://netbird.example.invalid/api",
		"release_origin": "https://releases.example.invalid",
		"client_id":      "netbird-dashboard",
		"issuer_layout":  "embedded-dex",
		"price_id":       "e2e-price",
		"return_url":     "https://portal.example.invalid:8443/billing/return",
		"prefix":         "fvpn-" + lay.Prefix,
		"port_base":      lay.PortBase,
		"origin_ip":      lay.OriginIP,
		"lab_prefix":     lay.LabPrefix,
		"host_ports":     lay.Ports,
	}
	if err := writeJSON(filepath.Join(state, "stack.json"), stack); err != nil {
		return err
	}
	if err := writeComposeEnv(state, lay); err != nil {
		return err
	}
	return os.WriteFile(marker, []byte("ready\n"), 0o600)
}

func writeComposeEnv(state string, lay layout) error {
	body := fmt.Sprintf(`FVPN_E2E_PREFIX=%s
FVPN_E2E_PORT_BASE=%d
FVPN_E2E_LAB_PREFIX=%s
FVPN_E2E_ORIGIN_PREFIX=%s
FVPN_E2E_PORT_CONTROL=%d
FVPN_E2E_PORT_AGENT=%d
FVPN_E2E_PORT_IDP=%d
FVPN_E2E_PORT_ORIGIN=%d
FVPN_E2E_PORT_PAYMENT=%d
FVPN_E2E_PORT_GATEWAY=%d
FVPN_E2E_PORT_STUN=%d
FVPN_E2E_PORT_POSTGRES=%d
`, lay.Prefix, lay.PortBase, lay.LabPrefix, lay.OriginPrefix,
		lay.Ports["control"], lay.Ports["agent"], lay.Ports["idp"], lay.Ports["origin"],
		lay.Ports["payment"], lay.Ports["gateway"], lay.Ports["stun"], lay.Ports["postgres"])
	return writeSecret(filepath.Join(state, "render", "compose.env"), []byte(body))
}

func writeAgentConfigs(state string) error {
	base := map[string]any{
		"state_dir": "/var/lib/vpn-node/state", "identity_dir": "/var/lib/vpn-node/identity",
		"journal_dir": "/var/lib/vpn-node/update-journal", "lock_path": "/var/lib/vpn-node/update.lock",
		"server_ca_file": "/etc/vpn-node/ca.pem", "trust_bundle": "/etc/vpn-node/trust.json",
		"datapath":      map[string]string{"member_ingress": "fvmb0", "uplink": "fvup0", "snat": "192.0.2.20"},
		"agent_release": "0.1.0",
		// TEST ONLY rate. The agent requires the field and has no product default (D-034).
		"host_reserve_bps": 1000000, "netbird_backend": "kernel",
		"heartbeat_seconds": 5, "poll_seconds": 5, "watchdog_stale_seconds": 30, "watchdog_interval_seconds": 5,
		"boot_id_file": "/etc/vpn-node/boot-id", "test_mode": true,
		"updater_recover": "/usr/local/bin/fvpn-updater", "updater_config": "/var/lib/vpn-node/e2e/updater.json",
	}
	enroll := cloneMap(base)
	enroll["control_url"] = "https://portal.example.invalid:8443"
	runCfg := cloneMap(base)
	runCfg["control_url"] = "https://agent.example.invalid:9443"
	dir := filepath.Join(state, "node", "etc")
	if err := writeJSON(filepath.Join(dir, "agent-enroll.json"), enroll); err != nil {
		return err
	}
	return writeJSON(filepath.Join(dir, "agent-run.json"), runCfg)
}

func writeNetBird(state string) error {
	authSecret, err := randomHex(32)
	if err != nil {
		return err
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return err
	}
	cfg := map[string]any{"server": map[string]any{
		"listenAddress": ":8080", "exposedAddress": "https://netbird.example.invalid:443",
		"stunPorts": []int{3478}, "metricsPort": 9090, "healthcheckAddress": ":9000",
		"logLevel": "warn", "logFile": "console", "authSecret": authSecret, "dataDir": "/var/lib/netbird",
		"disableAnonymousMetrics": true, "disableGeoliteUpdate": true,
		"auth": map[string]any{
			"issuer": "https://netbird.example.invalid/oauth2",
			// The all-in-one server copies these onto the static dashboard
			// client. A third OAuth client cannot be registered. The portal
			// therefore uses netbird-dashboard.
			"dashboardRedirectURIs": []string{
				"https://netbird.example.invalid/nb-auth",
				"https://netbird.example.invalid/nb-silent-auth",
				"https://portal.example.invalid:8443/auth/callback",
			},
			"cliRedirectURIs": []string{"http://localhost:53000/"},
		},
		"store": map[string]any{"engine": "sqlite", "encryptionKey": base64.StdEncoding.EncodeToString(key)},
	}}
	return writeJSON(filepath.Join(state, "netbird", "config.yaml"), cfg)
}

func writeAdGuard(state, coreDNS string) error {
	password, err := randomHex(16)
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return err
	}
	if err := writeSecret(filepath.Join(state, "secrets", "adguard.password"), []byte(password+"\n")); err != nil {
		return err
	}
	// schema_version 29 is what v0.107.59 writes. A lower version runs the
	// upgrader, which nested bootstrap_dns into a sequence of sequences and
	// then refused to parse it. Query log stays off (SEC02).
	body := fmt.Sprintf(`# Generated per run. schema_version 29; query log is off (SEC02).
auth_attempts: 5
block_auth_min: 15
clients:
  persistent: []
  runtime_sources:
    arp: true
    dhcp: true
    hosts: true
    rdns: false
    whois: true
dhcp:
  dhcpv4: {}
  enabled: false
dns:
  aaaa_disabled: true
  allowed_clients:
    - 127.0.0.0/8
    - 172.16.0.0/12
    - 10.0.0.0/8
    - 192.168.0.0/16
    - 100.64.0.0/10
  anonymize_client_ip: true
  bind_hosts:
    - 0.0.0.0
  bootstrap_dns:
    - 127.0.0.11:53
  cache_size: 4096
  cache_ttl_max: 0
  cache_ttl_min: 0
  edns_client_subnet:
    custom_ip: ""
    enabled: false
    use_custom: false
  enable_dnssec: false
  hostsfile_enabled: false
  port: 53
  ratelimit: 0
  refuse_any: true
  serve_plain_dns: true
  upstream_dns:
    - %s:53
  upstream_mode: load_balance
filtering:
  blocked_services:
    schedule:
      time_zone: Local
  safe_fs_patterns:
    - /opt/adguardhome/work/data/userfilters/*
  safe_search:
    bing: false
    duckduckgo: false
    enabled: false
    google: false
    pixabay: false
    yandex: false
    youtube: false
filters: []
http:
  address: 127.0.0.1:8080
  pprof:
    enabled: false
    port: 6060
  session_ttl: 720h
os:
  group: ""
  rlimit_nofile: 0
  user: ""
querylog:
  enabled: false
  file_enabled: false
  ignored: []
  interval: 2160h
  size_memory: 1000
schema_version: 29
statistics:
  enabled: false
  ignored: []
  interval: 24h
user_rules: []
users:
  - name: e2e
    password: %q
whitelist_filters: []
`, coreDNS, string(hash))
	dir := filepath.Join(state, "adguard", "conf")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "AdGuardHome.yaml"), []byte(body), 0o600)
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in)+1)
	for k, v := range in {
		out[k] = v
	}
	return out
}

func writeJSON(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

func writeSecret(path string, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o600)
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
