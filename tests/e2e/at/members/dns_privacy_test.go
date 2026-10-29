//go:build e2e

package members

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
	for _, s := range []string{"member", "adguard", "control"} {
		if _, err := l.docker("cp", path, l.name(s)+":/tmp/at-members-probe"); err != nil {
			return false
		}
	}
	return true
}
func (l *live) dns(name, typ, proto string) string {
	raw, err := l.inside("member", "/tmp/at-members-probe", "dns", l.stack.LabPrefix+".53:53", name, typ, proto)
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
	canonical := []any{rule("lab.example.invalid", true, "block"), rule("xn--bcher-kva.example.com", false, "block"), rule("private.example.invalid", true, "allow")}
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
	applied := l.wait(20*time.Second, func() bool {
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
		r.block("After central publication and a 20-second apply window, activated policy is not present in real AdGuard user_rules. Node desired-state publication does not configure this shared lab resolver; no resolver apply credentials/hook are supplied. Responses above are precondition observations, not passing block/cache semantics.")
	}
	r.block("No admitted member DNS/forwarding path and no controlled CNAME/HTTPS/SVCB/rebinding zone fixtures. Apex/label matching, block TTL expiry and allowlist-versus-isolation need an applied policy and those fixtures; API validation alone does not pass AT20.")
}
func (l *live) dnsFailure(r *record) {
	raw, err := l.resolver("/control/dns_info")
	var info struct {
		Upstream  []string `json:"upstream_dns"`
		Fallback  []string `json:"fallback_dns"`
		Bootstrap []string `json:"bootstrap_dns"`
	}
	_ = json.Unmarshal(raw, &info)
	r.check("resolver topology", "record upstream/fallback counts", fmt.Sprintf("API_ok=%t; upstream_count=%d; fallback_count=%d; bootstrap_count=%d; filtering_resolvers=1", err == nil, len(info.Upstream), len(info.Fallback), len(info.Bootstrap)), "AdGuard GET /control/dns_info", err == nil)
	for _, service := range []string{"adguard", "coredns"} {
		_, err = l.docker("pause", l.name(service))
		if err != nil {
			r.block(service + " fault unavailable")
			continue
		}
		// Immediate restoration and Cleanup cover assertions and test interruption.
		l.t.Cleanup(func() { _, _ = l.docker("unpause", l.name(service)) })
		answer := l.dns("failure-"+l.run+"-"+service+".example.invalid", "1", "udp")
		_, restore := l.docker("unpause", l.name(service))
		r.check(service+" failure", "no successful answer while resolver/upstream paused", answer, "docker pause "+l.name(service)+"; unique sentinel query; unpause", restore == nil && (strings.Contains(answer, `"available":false`) || strings.Contains(answer, `"rcode":2`)))
	}
	after := l.dns("lab.example.invalid", "1", "udp")
	r.check("fault restoration", "controlled answer available", after, "probe DNS after both unpause operations", strings.Contains(after, `"available":true`))
	r.block("Running topology contains one filtering resolver and one authoritative CoreDNS upstream, not two independent controlled filtering resolvers. The member entrypoint uses --disable-dns. Faults above prove direct resolver unavailability only; client fallback/public-DNS exclusion and policy-sync failure require devstack follow-up. Android Private DNS/browser DoH scripts and member guidance are in OPERATORS.md.")
}
func (l *live) privacy(r *record) {
	sentinel := "privacy-" + l.run + ".example.invalid"
	response := l.dns(sentinel, "1", "udp")
	r.check("privacy sentinel query", "controlled resolver responds", response, "member probe DNS unique TEST ONLY sentinel", strings.Contains(response, `"available":true`))
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
	r.check("metrics scan", "record endpoint availability and sentinel matches", fmt.Sprintf("read_ok=%t; sentinel_matches=%d", merr == nil, bytes.Count(metrics, []byte(sentinel))), "control loopback /metrics via probe", merr != nil || !bytes.Contains(metrics, []byte(sentinel)))
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
	r.block("No active licensed threat feed exists in the running lab; source/license review and last-good update failure cannot be exercised. Sentinel scan covers DNS only, not a successful member URL flow. Raw usage/rollup purge, file-content scans, retained backup expiry and representative full-qname/URL privacy require the missing fixtures; no complete privacy pass is claimed.")
}
