package agent

import (
	"errors"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func writeAgentConfig(t *testing.T, extra string) string {
	t.Helper()
	dir := t.TempDir()
	body := `{
		"control_url": "https://control.example.invalid",
		"server_ca_file": "/etc/vpn-node/ca.pem",
		"trust_bundle": "/etc/vpn-node/trust.json",
		"datapath": {"member_ingress": "fv-fh", "uplink": "fv-ih", "snat": "192.0.2.37"},
		"agent_release": "0.0.0-test",
		"netbird_backend": "kernel",
		"host_reserve_bps": 1` + extra + `
	}`
	path := filepath.Join(dir, "agent.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDNSConfigAbsent(t *testing.T) {
	cfg, err := LoadConfig(writeAgentConfig(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DNS != nil {
		t.Fatalf("dns %+v", cfg.DNS)
	}
}

func TestDNSConfigHostLocal(t *testing.T) {
	cred := filepath.Join(t.TempDir(), "creds.json")
	ok := `, "dns": {"resolvers": [{"id": "local", "admin_url": "http://127.0.0.1:8080", "dns_addr": "127.0.0.1:53"}], "resolver_credentials_file": "` + cred + `"}`
	cfg, err := LoadConfig(writeAgentConfig(t, ok))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DNS == nil || len(cfg.DNS.Resolvers) != 1 || cfg.DNS.Resolvers[0].DNSAddr != "127.0.0.1:53" {
		t.Fatalf("%+v", cfg.DNS)
	}
	v6 := `, "dns": {"resolvers": [{"admin_url": "https://[::1]:8443/", "dns_addr": "[::1]:53"}], "resolver_credentials_file": "` + cred + `"}`
	if _, err := LoadConfig(writeAgentConfig(t, v6)); err != nil {
		t.Fatal(err)
	}

	secretURL := "http://user:pw-d061@203.0.113.50:8080/admin?x=1#f"
	bad := `, "dns": {"resolvers": [{"admin_url": "` + secretURL + `", "dns_addr": "203.0.113.50:53"}], "resolver_credentials_file": "` + cred + `"}`
	_, err = LoadConfig(writeAgentConfig(t, bad))
	if !errors.Is(err, errDNSHost) {
		t.Fatalf("public: %v", err)
	}
	if strings.Contains(err.Error(), "203.0.113.50") || strings.Contains(err.Error(), "pw-d061") {
		t.Fatalf("error leaked the target: %v", err)
	}

	cases := []struct {
		name  string
		extra string
		want  error
	}{
		{"hostname", `, "dns": {"resolvers": [{"admin_url": "http://localhost:8080", "dns_addr": "127.0.0.1:53"}], "resolver_credentials_file": "` + cred + `"}`, errDNSHost},
		{"no port", `, "dns": {"resolvers": [{"admin_url": "http://127.0.0.1", "dns_addr": "127.0.0.1:53"}], "resolver_credentials_file": "` + cred + `"}`, errDNSHost},
		{"path", `, "dns": {"resolvers": [{"admin_url": "http://127.0.0.1:8080/admin", "dns_addr": "127.0.0.1:53"}], "resolver_credentials_file": "` + cred + `"}`, errDNSHost},
		{"docs", `, "dns": {"resolvers": [{"admin_url": "http://192.0.2.1:8080", "dns_addr": "192.0.2.1:53"}], "resolver_credentials_file": "` + cred + `"}`, errDNSHost},
		{"link local", `, "dns": {"resolvers": [{"admin_url": "http://169.254.1.1:8080", "dns_addr": "169.254.1.1:53"}], "resolver_credentials_file": "` + cred + `"}`, errDNSHost},
		{"mapped", `, "dns": {"resolvers": [{"admin_url": "http://[::ffff:127.0.0.1]:8080", "dns_addr": "127.0.0.1:53"}], "resolver_credentials_file": "` + cred + `"}`, errDNSHost},
		{"id", `, "dns": {"resolvers": [{"id": "http://x", "admin_url": "http://127.0.0.1:8080", "dns_addr": "127.0.0.1:53"}], "resolver_credentials_file": "` + cred + `"}`, errDNSID},
		{"dup", `, "dns": {"resolvers": [{"id": "a", "admin_url": "http://127.0.0.1:8080", "dns_addr": "127.0.0.1:53"}, {"id": "a", "admin_url": "http://127.0.0.1:8081", "dns_addr": "127.0.0.1:53"}], "resolver_credentials_file": "` + cred + `"}`, errDNSID},
		{"only file", `, "dns": {"resolvers": [], "resolver_credentials_file": "` + cred + `"}`, errDNSBoth},
		{"only resolvers", `, "dns": {"resolvers": [{"admin_url": "http://127.0.0.1:8080", "dns_addr": "127.0.0.1:53"}]}`, errDNSBoth},
		{"relative", `, "dns": {"resolvers": [{"admin_url": "http://127.0.0.1:8080", "dns_addr": "127.0.0.1:53"}], "resolver_credentials_file": "creds.json"}`, errDNSPath},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadConfig(writeAgentConfig(t, tc.extra))
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v", err)
			}
			if strings.Contains(err.Error(), "127.0.0.1") || strings.Contains(err.Error(), "8080") {
				t.Fatalf("error leaked an address: %v", err)
			}
		})
	}

	var many strings.Builder
	many.WriteString(`, "dns": {"resolvers": [`)
	for i := 0; i < maxDNSResolvers+1; i++ {
		if i > 0 {
			many.WriteString(",")
		}
		many.WriteString(`{"admin_url": "http://127.0.0.1:8080", "dns_addr": "127.0.0.1:53"}`)
	}
	many.WriteString(`], "resolver_credentials_file": "` + cred + `"}`)
	if _, err := LoadConfig(writeAgentConfig(t, many.String())); !errors.Is(err, errDNSMany) {
		t.Fatalf("count: %v", err)
	}

	missing := absentPrivate(t)
	extra := `, "dns": {"resolvers": [{"admin_url": "http://` + missing.String() + `:8080", "dns_addr": "` + net.JoinHostPort(missing.String(), "53") + `"}], "resolver_credentials_file": "` + cred + `"}`
	_, err = LoadConfig(writeAgentConfig(t, extra))
	if !errors.Is(err, errDNSHost) || strings.Contains(err.Error(), missing.String()) {
		t.Fatalf("absent private: %v", err)
	}

	if ip, ok := assignedPrivate(); ok {
		host := ip.String()
		admin := "http://" + host + ":8080"
		if ip.Is6() {
			admin = "http://[" + host + "]:8080"
		}
		extra := `, "dns": {"resolvers": [{"admin_url": "` + admin + `", "dns_addr": "` + net.JoinHostPort(host, "53") + `"}], "resolver_credentials_file": "` + cred + `"}`
		if _, err := LoadConfig(writeAgentConfig(t, extra)); err != nil {
			t.Fatalf("assigned %s: %v", host, err)
		}
	}
}

func absentPrivate(t *testing.T) netip.Addr {
	t.Helper()
	have := interfaceSet(t)
	for _, c := range []string{"10.255.255.1", "192.168.255.1", "172.31.255.1", "10.254.254.1"} {
		ip := netip.MustParseAddr(c)
		if !have[ip] {
			return ip
		}
	}
	t.Fatal("every candidate private address is assigned")
	return netip.Addr{}
}

func assignedPrivate() (netip.Addr, bool) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return netip.Addr{}, false
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP == nil {
			continue
		}
		ip, ok := netip.AddrFromSlice(ipnet.IP)
		if !ok {
			continue
		}
		ip = ip.Unmap()
		if ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			return ip, true
		}
	}
	return netip.Addr{}, false
}

func interfaceSet(t *testing.T) map[netip.Addr]bool {
	t.Helper()
	out := map[netip.Addr]bool{}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip, ok := netip.AddrFromSlice(ipnet.IP)
		if ok {
			out[ip.Unmap()] = true
		}
	}
	return out
}

func TestResolverCredentialParse(t *testing.T) {
	user, pass, err := parseResolverCredentials([]byte(`{"username":"e2e","password":"pw-d061"}`))
	if err != nil || user != "e2e" || pass != "pw-d061" {
		t.Fatal("parse")
	}
	for _, raw := range []string{
		`{"username":"e2e"}`,
		`{"username":"e2e","password":"","extra":1}`,
		`{"username":"e2e","password":"pw","token":"x"}`,
		`{"username":"e2e","password":"has space"}` + "\n{}",
		`{"username":"bad` + "\n" + `","password":"pw"}`,
		`{"username":"` + strings.Repeat("u", maxResolverSecret+1) + `","password":"pw"}`,
	} {
		if _, _, err := parseResolverCredentials([]byte(raw)); !errors.Is(err, errResolverCred) {
			t.Fatalf("%q: %v", raw, err)
		}
	}
}

func TestResolverCredentialFileRefused(t *testing.T) {
	r := newRig(t)
	const secret = "pw-d061-do-not-leak"
	path := filepath.Join(t.TempDir(), "creds.json")
	if err := os.WriteFile(path, []byte(`{"username":"e2e","password":"`+secret+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	r.eng.cfg.DNS = &DNSConfig{
		Resolvers:               []DNSResolver{{ID: "lab", AdminURL: "http://127.0.0.1:8080", DNSAddr: "127.0.0.1:53"}},
		ResolverCredentialsFile: path,
	}
	r.eng.reloadResolvers()
	if r.eng.dnsApply != nil || r.eng.dnsCredDetail != resolverCredDetail {
		t.Fatalf("mode apply=%v detail=%q", r.eng.dnsApply != nil, r.eng.dnsCredDetail)
	}
	if strings.Contains(r.log.String(), secret) || strings.Contains(r.log.String(), path) {
		t.Fatal("log leaked the credential file")
	}

	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	r.eng.reloadResolvers()
	if os.Getuid() == 0 {
		if r.eng.dnsApply == nil || r.eng.dnsCredDetail != "" {
			t.Fatal("root 0600 file was refused")
		}
		if strings.Contains(r.log.String(), secret) {
			t.Fatal("log leaked the password")
		}
		return
	}
	if r.eng.dnsApply != nil || r.eng.dnsCredDetail != resolverCredDetail {
		t.Fatalf("owner apply=%v detail=%q", r.eng.dnsApply != nil, r.eng.dnsCredDetail)
	}
	if strings.Contains(r.log.String(), secret) {
		t.Fatal("log leaked the password")
	}

	keep := &scriptDNS{}
	r.eng.dnsApply = keep
	r.eng.dnsPinned = true
	r.eng.reloadResolvers()
	if r.eng.dnsApply != keep {
		t.Fatal("pinned applier was replaced")
	}
}

func TestResolverCredentialRefusalCheck(t *testing.T) {
	r := newRig(t)
	const secret = "pw-d061-check"
	path := filepath.Join(t.TempDir(), "creds.json")
	if err := os.WriteFile(path, []byte(`{"username":"e2e","password":"`+secret+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	r.eng.cfg.DNS = &DNSConfig{
		Resolvers:               []DNSResolver{{AdminURL: "http://127.0.0.1:8080", DNSAddr: "127.0.0.1:53"}},
		ResolverCredentialsFile: path,
	}
	r.eng.reloadResolvers()
	body := []byte(exampleResolverBody)
	ref := exampleRef(body)
	r.srv.PutResolverBody(ref.DocumentId, body)
	r.srv.SetResolverRef(&ref)
	enrollBoot(t, r)
	stepOK(t, r)
	c := mustDNS(t, r.eng.decision.Checks)
	if c.Status != api.HealthCheckStatusFailed || c.Detail == nil || *c.Detail != resolverCredDetail {
		t.Fatalf("detail %v status %s", c.Detail, c.Status)
	}
	if codeOf(t, c) != api.ErrorCodeFORBIDDEN {
		t.Fatalf("code %s", codeOf(t, c))
	}
	if strings.Contains(*c.Detail, secret) || strings.Contains(*c.Detail, "127.0.0.1") || strings.Contains(*c.Detail, path) {
		t.Fatalf("detail leaked: %s", *c.Detail)
	}
	ev := mustEvidence(t, r)
	if ev.Outcome != api.DnsPolicyEvidenceOutcomeFailed {
		t.Fatalf("outcome %s", ev.Outcome)
	}
	if strings.Contains(r.log.String(), secret) {
		t.Fatal("log leaked the password")
	}
}

func TestResolverCredentialUnusedWithoutDocument(t *testing.T) {
	r := newRig(t)
	path := filepath.Join(t.TempDir(), "creds.json")
	if err := os.WriteFile(path, []byte(`{"username":"e2e","password":"pw-d061-unused"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	r.eng.cfg.DNS = &DNSConfig{
		Resolvers:               []DNSResolver{{AdminURL: "http://127.0.0.1:8080", DNSAddr: "127.0.0.1:53"}},
		ResolverCredentialsFile: path,
	}
	r.eng.reloadResolvers()
	enrollBoot(t, r)
	stepOK(t, r)
	c := mustDNS(t, r.eng.decision.Checks)
	if c.Status != api.HealthCheckStatusPassed || c.Detail != nil {
		t.Fatalf("dns %+v", c)
	}
	rep, ok := r.srv.LastObservation()
	if !ok {
		t.Fatal("no observation")
	}
	if _, err := rep.DnsPolicyEvidence.Get(); err == nil {
		t.Fatal("evidence without a resolver document")
	}
}

func TestResolverReloadClearsBadApplier(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("a 0600 file is usable for uid 0")
	}
	r := newRig(t)
	path := filepath.Join(t.TempDir(), "creds.json")
	if err := os.WriteFile(path, []byte(`{"username":"e2e","password":"pw"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	r.eng.cfg.DNS = &DNSConfig{
		Resolvers:               []DNSResolver{{AdminURL: "http://127.0.0.1:8080", DNSAddr: "127.0.0.1:53"}},
		ResolverCredentialsFile: path,
	}
	r.eng.dnsApply = &scriptDNS{}
	r.eng.reloadResolvers()
	if r.eng.dnsApply != nil || r.eng.dnsCredDetail != resolverCredDetail {
		t.Fatal("reload kept an applier for a non-root credential file")
	}
}
