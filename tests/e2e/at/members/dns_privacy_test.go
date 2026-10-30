//go:build e2e

package members

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

func (l *live) installProbe() bool {
	path := filepath.Join(l.root, ".agent/at-members/probe")
	cmd := exec.CommandContext(l.ctx, "go", "build", "-o", path, "./tests/e2e/at/members/cmd/probe")
	cmd.Dir = l.root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
	if _, err := cmd.CombinedOutput(); err != nil {
		return false
	}
	services := []string{"member", "node", "adguard", "control", "origin"}
	if l.second.NodeID != uuid.Nil {
		services = append(services, "node2", "node2-adguard")
	}
	for _, s := range services {
		if _, err := l.docker("cp", path, l.name(s)+":/tmp/at-members-probe"); err != nil {
			return false
		}
	}
	return true
}
func (l *live) dns(name, typ, proto string) string {
	raw, err := l.inside("member", "/tmp/at-members-probe", "dns", l.stack.LabPrefix+".40:53", name, typ, proto)
	if err != nil {
		return "probe unavailable"
	}
	return strings.TrimSpace(string(raw))
}

// Host-local resolver observation is explicitly separate from member forwarding.
func (l *live) dnsLocal(name, typ, proto string) string {
	raw, err := l.inside(l.nodeContainer(), "/tmp/at-members-probe", "dns", "127.0.0.1:53", name, typ, proto)
	if err != nil {
		return "probe unavailable"
	}
	return strings.TrimSpace(string(raw))
}
func (l *live) resolver(path string) ([]byte, error) {
	cmd := exec.CommandContext(l.ctx, "docker", "exec", "-i", l.name("adguard"), "/tmp/at-members-probe", "http", path)
	cmd.Stdin = strings.NewReader(l.secret("adguard.password"))
	return cmd.CombinedOutput()
}
func rule(domain string, sub bool, action string) map[string]any {
	return map[string]any{"rule_id": uuid.New(), "domain": domain, "include_subdomains": sub, "action": action, "profile_id": nil, "reason": "admin_rule", "revision": "1"}
}
func (l *live) dnsRules(r *record) {
	if !l.installProbe() {
		r.block("lab DNS probe could not be installed")
		return
	}
	// HTTP authority requires canonical ASCII. Normalization is a UI/input
	// concern; accepting mixed-case/U-label policy bytes would violate contract.
	for _, domain := range []string{"EXAMPLE.COM", "example.com.", "bücher.example.com", "https://example.com/path"} {
		got, _ := l.policy("domain_rule_set", map[string]any{"rules": []any{rule(domain, true, "block")}})
		r.check("reject noncanonical "+domain, "422 (contract canonical ASCII)", detail(got), "POST /admin/policies domain_rule_set", got.status == 422)
	}
	canonical := []any{rule("lab.example.invalid", true, "block"), rule("xn--bcher-kva.example.com", false, "block"), rule("feed-01.example.invalid", false, "allow")}
	good, id := l.policy("domain_rule_set", map[string]any{"rules": canonical})
	l.dnsRuleRev = id
	r.check("canonical rules", "201, immutable canonical payload", detail(good), "POST canonical A-label, include_subdomains and allow rules", good.status == 201)
	duplicate, _ := l.policy("domain_rule_set", map[string]any{"rules": []any{rule("example.com", true, "block"), rule("example.com", true, "block")}})
	conflict, _ := l.policy("domain_rule_set", map[string]any{"rules": []any{rule("example.com", true, "block"), rule("example.com", true, "allow")}})
	r.check("duplicate and conflict rejection", "both 409 conflict", fmt.Sprintf("duplicate=%d; conflict=%d", duplicate.status, conflict.status), "POST duplicate/conflicting rules", duplicate.status == 409 && conflict.status == 409)
	if id != uuid.Nil {
		profile, pid := l.policy("dns_profile", map[string]any{"profile_code": "at-members", "display_name": map[string]string{"zh-TW": "TEST ONLY"}, "block_response": "nxdomain", "blocked_ttl_seconds": 5, "domain_rule_set_revision_ids": []uuid.UUID{id}, "threat_feed_revision_ids": []string{}})
		r.check("DNS profile", "201 with TEST ONLY 5-second TTL", detail(profile), "POST /admin/policies dns_profile", profile.status == 201)
		if pid != uuid.Nil {
			act := l.activate(pid)
			r.check("DNS activation", "202 pending external apply", detail(act), "POST /admin/policies/{id}/activate", act.status == 202)
		}
	}
	// Worker publication and resolver application are asynchronous. Observe a
	// bounded convergence window before declaring a missing application path.
	_ = l.wait(30*time.Second, func() bool { return l.count(`SELECT count(*) FROM dns_resolver_documents`) > 0 })
	documents := l.count(`SELECT count(*) FROM dns_resolver_documents`)
	r.check("compiled DNS documents", "observe central publication separately from resolver apply", fmt.Sprintf("stored_documents=%d", documents), "SELECT count dns_resolver_documents", true)
	var status struct {
		UserRules []string `json:"user_rules"`
	}
	var err error
	applied := l.wait(45*time.Second, func() bool {
		var raw []byte
		raw, err = l.resolver("/control/filtering/status")
		_ = json.Unmarshal(raw, &status)
		for _, line := range status.UserRules {
			if strings.Contains(line, "lab.example.invalid") {
				return true
			}
		}
		return false
	})
	var compileStatus string
	_ = l.pool.QueryRow(l.ctx, `SELECT status FROM operations WHERE kind='policy.activate' ORDER BY created_at DESC LIMIT 1`).Scan(&compileStatus)
	r.check("policy operation versus resolver", "record compilation operation separately from application", "latest_policy_operation="+compileStatus, "SELECT operations.status for policy.activate", compileStatus != "")
	r.check("actual resolver readback", "record applied-rule presence", fmt.Sprintf("API_ok=%t; user_rules=%d; requested_block_present=%t", err == nil, len(status.UserRules), applied), "AdGuard GET /control/filtering/status", err == nil)
	for _, q := range []struct{ name, typ string }{{"lab.example.invalid", "1"}, {"LAB.example.invalid.", "1"}, {"sub.lab.example.invalid", "1"}, {"notexample.com", "1"}, {"lab.example.invalid", "5"}, {"lab.example.invalid", "28"}, {"lab.example.invalid", "65"}, {"lab.example.invalid", "64"}} {
		actual := l.dns(q.name, q.typ, "udp")
		r.check("wire DNS "+q.name+" type "+q.typ, "record actual controlled resolver answer (management path)", actual, "docker exec member probe dns controlled-resolver", actual != "probe unavailable")
	}
	if !applied {
		r.block("Requested block rule was absent after the bounded resolver application wait; typed evidence below identifies the stage.")
	}
	l.resolverEvidence(r)
	r.block("Native member admission is tested in AT01/AT02, but the entrypoint disables native DNS and there is no applied member DNS/forwarding path or controlled CNAME/HTTPS/SVCB/rebinding zone fixture. Apex/label matching, block TTL expiry and allowlist-versus-isolation need an applied policy and those fixtures; API validation alone does not pass AT20.")
}
func (l *live) dnsFailure(r *record) {
	if !l.secondExit(r) || !l.installProbe() {
		r.block("F4-02: second resolver prerequisite unavailable")
		return
	}
	// Two independent filtering resolvers, not one filter plus its authority.
	for _, item := range []struct{ node, resolver string }{{"node", "adguard"}, {"node2", "node2-adguard"}} {
		other := "node2"
		if item.node == "node2" {
			other = "node"
		}
		before, _ := l.inside(item.node, "/tmp/at-members-probe", "dns", "127.0.0.1:53", "lab.example.invalid", "1", "udp")
		_, pause := l.docker("pause", l.name(item.resolver))
		if pause != nil {
			r.block("Filtering resolver fault unavailable: " + item.resolver)
			continue
		}
		l.t.Cleanup(func() { l.cleanupDocker("unpause", l.name(item.resolver)) })
		unavailable, _ := l.inside(item.node, "/tmp/at-members-probe", "dns", "127.0.0.1:53", "lab.example.invalid", "1", "udp")
		fallback, _ := l.inside(other, "/tmp/at-members-probe", "dns", "127.0.0.1:53", "lab.example.invalid", "1", "udp")
		_, resume := l.docker("unpause", l.name(item.resolver))
		r.check(item.resolver+" isolated failure", "paused filter unavailable; other filter still blocks synthetic domain", fmt.Sprintf("before=%s; paused=%s; other=%s", before, unavailable, fallback), "pause exactly one filtering resolver; query both node loopbacks; unpause (explicit probes, not native client failover)", resume == nil && strings.Contains(string(unavailable), `"available":false`) && strings.Contains(string(fallback), `"rcode":3`))
	}
	l.resolverEvidence(r)
	restore := l.useSecond()
	l.resolverEvidence(r)
	restore()
	l.dnsSyncFault(r)
	r.block("F4-05: two independent filtering resolvers survive alternating faults, but the native member entrypoint still uses --disable-dns. No product-distributed two-resolver client configuration/fallback is available; loopback probes do not prove automatic member failover or public-DNS exclusion. Owner: w3-devstack-e2e member-entrypoint.sh and native DNS fixture. Android Private DNS/browser DoH remain needs operator run; guidance in OPERATORS.md.")
}

func (l *live) dnsSyncFault(r *record) {
	marker := filepath.Join(l.state, "node/var/e2e/corrupt-resolver")
	if err := os.WriteFile(marker, []byte("*"), 0600); err != nil {
		r.block("Shared resolver transport marker unavailable")
		return
	}
	defer func() { _ = os.Remove(marker) }()
	profile, rev := l.policy("dns_profile", map[string]any{"profile_code": "at-sync-fault", "display_name": map[string]string{"zh-TW": "TEST ONLY"}, "block_response": "nxdomain", "blocked_ttl_seconds": 5, "domain_rule_set_revision_ids": []uuid.UUID{l.dnsRuleRev}, "threat_feed_revision_ids": []string{}})
	if profile.status != 201 {
		r.check("sync fault policy input", "201", detail(profile), "POST new DNS profile revision", false)
		return
	}
	act := l.activate(rev)
	r.check("sync fault activation", "202", detail(act), "activate normal profile while shared byte-corruption hook armed", act.status == 202)
	var evidence api.DnsPolicyEvidence
	failed := l.wait(90*time.Second, func() bool {
		got := l.api(l.admin, "GET", "/api/v1/admin/nodes/"+l.node.String(), nil, nil)
		var node api.NodeDetail
		if json.Unmarshal(got.body, &node) != nil {
			return false
		}
		var e error
		evidence, e = node.DnsPolicyEvidence.Get()
		if e != nil {
			return false
		}
		code, _ := evidence.ErrorCode.Get()
		return evidence.Outcome == api.DnsPolicyEvidenceOutcomeFailed && code == api.ErrorCodeHASHMISMATCH
	})
	code, _ := evidence.ErrorCode.Get()
	r.check("resolver synchronization corruption", "failed HASH_MISMATCH; last-good synthetic rule still blocks", fmt.Sprintf("observed=%t; outcome=%s; error=%s; last_good=%s", failed, evidence.Outcome, code, l.dnsLocal("lab.example.invalid", "1", "udp")), "shared node loopback mTLS corruption hook; typed NodeDetail evidence and DNS query", failed && strings.Contains(l.dnsLocal("lab.example.invalid", "1", "udp"), `"rcode":3`))
	_ = os.Remove(marker)
	l.resolverEvidence(r)
}
func (l *live) privacy(r *record) {
	sentinel := "privacy-" + l.run + ".example.invalid"
	response := l.dnsLocal(sentinel, "1", "udp")
	r.check("privacy sentinel query", "controlled resolver responds", response, "node loopback probe DNS unique TEST ONLY sentinel (resolver privacy only)", strings.Contains(response, `"available":true`))
	for _, path := range []string{"/control/querylog/config", "/control/stats/config"} {
		raw, err := l.resolver(path)
		var v struct {
			Enabled     bool `json:"enabled"`
			FileEnabled bool `json:"file_enabled"`
		}
		decode := json.Unmarshal(raw, &v)
		r.check(path, "disabled", fmt.Sprintf("read_ok=%t; enabled=%t; file_enabled=%t", err == nil && decode == nil, v.Enabled, v.FileEnabled), "AdGuard GET "+path, err == nil && decode == nil && !v.Enabled && !v.FileEnabled)
	}
	for _, path := range []string{"/control/querylog", "/control/stats"} {
		raw, err := l.resolver(path)
		n := bytes.Count(raw, []byte(sentinel))
		r.check(path+" sentinel scan", "no sentinel retained", fmt.Sprintf("read_ok=%t; sentinel_matches=%d", err == nil, n), "scan actual resolver API response in memory", err == nil && n == 0)
	}
	raw, err := l.inside("adguard", "cat", "/opt/adguardhome/conf/AdGuardHome.yaml")
	var config struct {
		Querylog struct {
			Enabled     bool
			FileEnabled bool `yaml:"file_enabled"`
		}
		Statistics struct{ Enabled bool }
		UserRules  []string `yaml:"user_rules"`
	}
	decode := yaml.Unmarshal(raw, &config)
	r.check("on-disk privacy config", "query history and statistics disabled", fmt.Sprintf("read_ok=%t; querylog=%t; querylog_file=%t; statistics=%t", err == nil && decode == nil, config.Querylog.Enabled, config.Querylog.FileEnabled, config.Statistics.Enabled), "read assigned AdGuardHome.yaml; emit only booleans", err == nil && decode == nil && !config.Querylog.Enabled && !config.Querylog.FileEnabled && !config.Statistics.Enabled)
	matches := 0
	scanned := 0
	for _, s := range []string{"control", "worker", "node", "member", "adguard", "coredns", "netbird", "gateway", "payment", "origin", "idp", "postgres"} {
		raw, err := l.docker("logs", l.name(s))
		if err == nil {
			scanned++
			matches += bytes.Count(raw, []byte(sentinel))
		}
	}
	r.check("service log scan", "12 services; no sentinel", fmt.Sprintf("services_scanned=%d; sentinel_matches=%d", scanned, matches), "docker logs assigned containers; count sentinel in memory", scanned == 12 && matches == 0)
	metrics, merr := l.inside("control", "/tmp/at-members-probe", "http", "metrics")
	r.check("metrics scan", "record endpoint availability and sentinel matches", fmt.Sprintf("read_ok=%t; sentinel_matches=%d", merr == nil, bytes.Count(metrics, []byte(sentinel))), "control loopback /metrics via probe", merr == nil && !bytes.Contains(metrics, []byte(sentinel)))
	if merr != nil {
		r.block("Configured control metrics listener 127.0.0.1:9091 is unreachable from the control container; metrics privacy cannot be inspected. Config.MetricsAddress is not wired by apps/control.")
	}
	dump, derr := l.inside("postgres", "pg_dump", "-U", "fvpn", "-d", "fvpn", "--data-only")
	r.check("database backup scan", "dump succeeds; no query sentinel", fmt.Sprintf("dump_ok=%t; sentinel_matches=%d", derr == nil, bytes.Count(dump, []byte(sentinel))), "pg_dump own lab DB into memory only; scan sentinel", derr == nil && !bytes.Contains(dump, []byte(sentinel)))
	audit := l.count(`SELECT count(*) FROM audit_events a WHERE to_jsonb(a)::text LIKE $1`, "%"+sentinel+"%")
	r.check("audit scan", "no query sentinel", fmt.Sprintf("sentinel_rows=%d", audit), "SELECT count audit JSON containing sentinel", audit == 0)
	// §16.4: insert only synthetic audit rows, then use the guarded purge; never
	// delete append-only data directly or alter production retention settings.
	old, young := uuid.New(), uuid.New()
	_, err = l.pool.Exec(l.ctx, `INSERT INTO audit_events(id,actor,actor_type,action,resource,result,occurred_at) VALUES($1,'00000000-0000-0000-0000-000000000003','system','retention.test','fleet_config:00000000-0000-0000-0000-000000000001','succeeded',now()-interval '200 days'),($2,'00000000-0000-0000-0000-000000000003','system','retention.test','fleet_config:00000000-0000-0000-0000-000000000001','succeeded',now())`, old, young)
	if err != nil {
		r.block("synthetic retention rows rejected")
	} else {
		_, early := l.pool.Exec(l.ctx, `SELECT purge_retained('audit_events',now(),'TEST ONLY too early')`)
		var n int
		purge := l.pool.QueryRow(l.ctx, `SELECT purge_retained('audit_events',now()-interval '181 days','TEST ONLY AT22')`).Scan(&n)
		left := l.count(`SELECT count(*) FROM audit_events WHERE id=$1`, young)
		oldLeft := l.count(`SELECT count(*) FROM audit_events WHERE id=$1`, old)
		r.check("bounded audit retention", "reject early cutoff; delete only expired synthetic row", fmt.Sprintf("early_rejected=%t; purge_ok=%t; deleted=%d; old_remaining=%d; young_remaining=%d", early != nil, purge == nil, n, oldLeft, left), "SELECT purge_retained audit_events; inspect old/young IDs", early != nil && purge == nil && n == 1 && oldLeft == 0 && left == 1)
	}
	feeds := l.count(`SELECT count(*) FROM policy_definitions WHERE type='threat_feed' AND active_revision_id IS NOT NULL`)
	r.check("feed inventory", "record actual active feed count", fmt.Sprintf("active_feed_definitions=%d", feeds), "SELECT active threat_feed policies", true)
	r.block("No active licensed threat feed exists before the supplemental at-members-readback experiment; it supplies a self-authored licensed input and polls the real 300-second failure cadence. Sentinel scan covers DNS; forwarding uses a fixed controlled health path without a URL sentinel. F4-06: raw usage/rollup purge, file-content scans, retained backup expiry and representative full-qname/URL privacy require the missing fixtures; no complete privacy pass is claimed.")
}

// D-056: a central document or HTTP 200 alone never proves node application.
func (l *live) resolverEvidence(r *record) {
	// A committed document precedes the next heartbeat. Wait for typed evidence
	// of this profile before recording a genuine issued-document mismatch.
	_ = l.wait(45*time.Second, func() bool {
		got := l.api(l.admin, "GET", "/api/v1/admin/nodes/"+l.node.String(), nil, nil)
		var node api.NodeDetail
		if json.Unmarshal(got.body, &node) != nil {
			return false
		}
		ev, err := node.DnsPolicyEvidence.Get()
		if err != nil || node.DnsPolicyEvidence.IsNull() {
			return false
		}
		return ev.Outcome == api.DnsPolicyEvidenceOutcomeApplied && l.count(`SELECT count(*) FROM dns_resolver_documents WHERE id=$1 AND profile_revision_id=(SELECT active_revision_id FROM policy_definitions WHERE type='dns_profile' ORDER BY id LIMIT 1)`, ev.DocumentId) > 0
	})

	got := l.api(l.admin, "GET", "/api/v1/admin/nodes/"+l.node.String(), nil, nil)
	var node api.NodeDetail
	decode := json.Unmarshal(got.body, &node)
	evidence, getErr := node.DnsPolicyEvidence.Get()
	present := getErr == nil && node.DnsPolicyEvidence.IsSpecified() && !node.DnsPolicyEvidence.IsNull()
	var document bool
	if present {
		document = l.count(`SELECT count(*) FROM dns_resolver_documents WHERE id=$1 AND profile_revision_id=(SELECT active_revision_id FROM policy_definitions WHERE type='dns_profile' ORDER BY id LIMIT 1)`, evidence.DocumentId) > 0
	}
	code, _ := evidence.ErrorCode.Get()
	r.check("typed DNS evidence", "record exact issued-document application, or explicit null", fmt.Sprintf("HTTP=%d; decoded=%t; specified=%t; null=%t; present=%t; outcome=%s; error_code=%s; issued_document_matches=%t", got.status, decode == nil, node.DnsPolicyEvidence.IsSpecified(), node.DnsPolicyEvidence.IsNull(), present, evidence.Outcome, code, document), "GET NodeDetail.dns_policy_evidence using generated 1.2.0 type; read issued document", got.status == 200 && decode == nil)
	if !present {
		r.block("D-056/D-061: typed resolver evidence is unavailable on this live node; owner w3-wire-nodes / w2-agent-core; inspect issued document and accepted observation.")
		return
	}
	// This run can supersede the historical missing-reference observation.
	kept := r.Blockers[:0]
	for _, b := range r.Blockers {
		if !strings.HasPrefix(b, "D-056/D-061: typed resolver evidence is unavailable") && !(document && evidence.Outcome == api.DnsPolicyEvidenceOutcomeApplied && strings.HasPrefix(b, "F2-03:")) {
			kept = append(kept, b)
		}
	}
	r.Blockers = kept
	if evidence.Outcome != api.DnsPolicyEvidenceOutcomeApplied || !document {
		r.block("F2-03: this run has an issued resolver document and non-null typed DNS evidence, but outcome=" + string(evidence.Outcome) + " error=" + string(code) + ". node/agent/resolver.go:noteDNSApply reports resolver application failure; node/dns/applier.go:Apply owns convergence/readback. The earlier missing-reference result is historical, not this run's diagnosis. See requests.md.")
	}
	cert, e1 := l.inside(l.nodeContainer(), "cat", "/var/lib/vpn-node/identity/cert.pem")
	key, e2 := l.inside(l.nodeContainer(), "cat", "/var/lib/vpn-node/identity/key.pem")
	if e1 != nil || e2 != nil {
		r.block("node identity unavailable for resolver HTTP readback")
		return
	}
	pair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		r.block("node mTLS pair unavailable")
		return
	}
	transport := l.transport.Clone()
	defer transport.CloseIdleConnections()
	transport.TLSClientConfig.Certificates = []tls.Certificate{pair}
	client := &browser{client: &http.Client{Transport: transport, Timeout: 20 * time.Second}}
	fetched := l.request(client, "GET", "https://agent.example.invalid:9443/api/v1/agent/resolver-documents/"+evidence.DocumentId.String(), nil, nil)
	hash, hashErr := canonjson.HashHex(fetched.body)
	r.check("issued resolver document mTLS fetch", "200 canonical bytes match node evidence digest", fmt.Sprintf("HTTP=%d; hash_matches=%t; bytes=%d", fetched.status, hashErr == nil && hash == evidence.ContentSha256, len(fetched.body)), "GET /agent/resolver-documents/{issued_id} using actual node mTLS; hash in memory", fetched.status == 200 && hashErr == nil && hash == evidence.ContentSha256)
	desired := l.request(client, "GET", "https://agent.example.invalid:9443/api/v1/agent/desired-state", nil, nil)
	ambiguous := strings.Contains(string(desired.body), "Node DNS profile selection is ambiguous")
	r.check("current desired-state DNS selection", "record response independently of previously issued document", fmt.Sprintf("HTTP=%d; code=%s; ambiguous_active_profiles=%t", desired.status, desired.code, ambiguous), "GET /agent/desired-state via actual mTLS; retain only allowlisted error", desired.status != 0)
	if ambiguous {
		r.block("F2-04: multiple API-activated DNS profiles make current desired-state GET return 503 DEPENDENCY_UNAVAILABLE (Node DNS profile selection is ambiguous). ResolveDocumentInputs has no node-to-profile selection; plan profile references do not resolve it. See requests.md.")
	}
}
