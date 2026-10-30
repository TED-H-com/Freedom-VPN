//go:build e2e

package members

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// This optional readback extends the just-finished run on the same stack. It
// never creates entitlement, identity, device, subscription or feed projections.
// A separate command permits incremental evidence even when full ATs are blocked.
func TestATMembers_FollowupReadback(t *testing.T) {
	if os.Getenv("FVPN_AT_MEMBERS_READBACK") != "1" {
		t.Skip("run at-members-readback after the full live suite")
	}
	if os.Getenv("FVPN_E2E_PREFIX") != "w4-members" || os.Getenv("FVPN_E2E_PORT_BASE") != "24000" {
		t.Fatal("wrong assigned instance")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 12*time.Minute)
	defer cancel()
	l := &live{t: t, ctx: ctx, root: repository(t), run: uuid.NewString()[:8]}
	l.state = filepath.Join(l.root, ".agent/e2e-state-w4-members")
	l.open()
	l.admin = l.login("e2e-admin", "admin")
	if l.pool.QueryRow(ctx, `SELECT id,current_incarnation_id FROM nodes WHERE current_incarnation_id IS NOT NULL ORDER BY created_at DESC LIMIT 1`).Scan(&l.node, &l.inc) != nil {
		t.Fatal("prior run node unavailable")
	}
	for _, id := range []string{"AT02", "AT08", "AT20", "AT22"} {
		var r record
		raw, err := os.ReadFile(filepath.Join(l.root, "docs/e2e/at/members", id+".json"))
		if err != nil || json.Unmarshal(raw, &r) != nil {
			t.Fatal("prior acceptance record unavailable")
		}
		// Refuse to mix evidence from another stack build/incarnation.
		if r.NodeIncarnation != l.inc.String() {
			t.Fatal("prior record belongs to another incarnation")
		}
		for name, digest := range r.Build.Images {
			if l.build.Images[name] != digest {
				t.Fatal("runtime image changed since primary run")
			}
		}
		r.FollowupBuild = &l.build
		switch id {
		case "AT02":
			devices := l.count(`SELECT count(*) FROM member_devices WHERE status='authorized'`)
			proofs := l.count(`SELECT count(*) FROM netbird_user_bindings n JOIN identity_bindings i ON i.id=n.identity_binding_id WHERE n.external_user_id=i.subject AND n.mapping_proof='shared_issuer_public_subject:'||i.issuer`)
			var desired, applied string
			_ = l.pool.QueryRow(ctx, `SELECT desired_config_generation::text,applied_config_generation::text FROM nodes WHERE id=$1`, l.node).Scan(&desired, &applied)
			observations := l.count(`SELECT count(*) FROM device_authorization_observations WHERE observed_authorized`)
			r.check("post-worker projection readback", "record native proofs, devices and desired/applied separately", fmt.Sprintf("verified_proofs=%d; authorized_devices=%d; observed_authorized=%d; desired_generation=%s; applied_generation=%s", proofs, devices, observations, desired, applied), "read product projections after worker resumes; no projection fixtures", proofs >= 2 && devices >= 4)
		case "AT08":
			l.servingPrecondition(&r)
		case "AT20":
			for i, b := range r.Blockers {
				r.Blockers[i] = strings.ReplaceAll(b, "No admitted member DNS/forwarding path", "Native member admission is proven in AT01/AT02, but native DNS is disabled and no applied member DNS/forwarding path exists")
			}
			docs := l.count(`SELECT count(*) FROM dns_resolver_documents`)
			r.check("central resolver document readback", "record stored publication versus node delivery", fmt.Sprintf("central_documents=%d", docs), "SELECT dns_resolver_documents", docs > 0)
			r.block("Node delivery of the resolver document remains absent: w3-contracts-r5, then the nodes/agent follow-ups. Central compilation is not resolver application.")
		case "AT22":
			l.feedLastGood(&r)
		}
		l.write(&r)
		if r.Status == "fail" {
			t.Errorf("%s readback contains a failed subcheck", id)
		}
	}
}
func (l *live) servingPrecondition(r *record) {
	var provisioning, safety, desired, applied string
	var hasPeer bool
	_ = l.pool.QueryRow(l.ctx, `SELECT n.provisioning_state,n.safety_status,n.desired_config_generation::text,n.applied_config_generation::text,i.peer_id IS NOT NULL FROM nodes n JOIN node_incarnations i ON i.id=n.current_incarnation_id WHERE n.id=$1`, l.node).Scan(&provisioning, &safety, &desired, &applied, &hasPeer)
	policies := l.count(`SELECT count(*) FROM policy_definitions WHERE type='release_support' AND active_revision_id IS NOT NULL`)
	r.check("serving precondition diagnosis", "read actual readiness and active policy", fmt.Sprintf("provisioning=%s; safety=%s; desired=%s; applied=%s; node_peer_present=%t; active_release_policies=%d", provisioning, safety, desired, applied, hasPeer, policies), "SELECT node/incarnation and active release_support policy", provisioning != "")
	cert, e1 := l.inside("node", "cat", "/var/lib/vpn-node/identity/cert.pem")
	key, e2 := l.inside("node", "cat", "/var/lib/vpn-node/identity/key.pem")
	if e1 != nil || e2 != nil {
		r.block("node identity unavailable for desired-state readback")
		return
	}
	pair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		r.block("node TLS identity invalid")
		return
	}
	transport := l.transport.Clone()
	defer transport.CloseIdleConnections()
	transport.TLSClientConfig.Certificates = []tls.Certificate{pair}
	got := l.request(&browser{client: &http.Client{Transport: transport, Timeout: 20 * time.Second}}, "GET", "https://agent.example.invalid:9443/api/v1/agent/desired-state", nil, nil)
	// Match a known safe error, never emit arbitrary server error details.
	releaseRequired := strings.Contains(string(got.body), "Release support policy required")
	r.check("real node desired-state gate", "record mTLS response and precise missing precondition", fmt.Sprintf("%s; release_support_required=%t", detail(got), releaseRequired), "GET /agent/desired-state using actual node mTLS", got.status != 0)
	if releaseRequired {
		r.block("Observed mTLS desired-state 503 DEPENDENCY_UNAVAILABLE: Release support policy required. The fresh e2e-up hook supplies no active release-support policy; tests/e2e/harness.go:insertReleaseSupport uses non-contract {}. Request a reusable valid signed boot-release/support-policy setup hook from devstack. No readiness or projection rows are fabricated.")
	}
	if !hasPeer {
		r.block("Node incarnation peer_id is still null; D-055 owner w3-wire-nodes follow-up 10 must supply real management peer observation. This run cannot isolate its effect until the missing release-support baseline is supplied.")
	}
}

// §12.3 / D-054: all content, state and publication rows below are produced by
// the running worker. The only mutations are a TEST ONLY config parameter,
// self-authored origin bytes and the normal policy API. Real clock, no SQL clock.
func (l *live) feedLastGood(r *record) {
	path := filepath.Join(l.state, "config/dns.json")
	original, err := os.ReadFile(path)
	var cfg map[string]any
	if err != nil || json.Unmarshal(original, &cfg) != nil {
		r.block("lab DNS settings unavailable for bounded feed cadence")
		return
	}
	cfg["feed_update_cadence_seconds"] = 300
	changed, _ := json.Marshal(cfg)
	if os.WriteFile(path, changed, 0600) != nil {
		r.block("TEST ONLY feed cadence configuration unavailable")
		return
	}
	defer func() { _ = os.WriteFile(path, original, 0600); _, _ = l.docker("restart", l.name("worker")) }()
	if _, err = l.docker("restart", l.name("worker")); err != nil {
		r.block("worker restart for feed cadence unavailable")
		return
	}
	r.check("TEST ONLY feed cadence", "300 seconds through normal settings; restored afterwards", "worker restarted; cadence=300", "set generated DNS config feed_update_cadence_seconds=300; restart own worker", true)
	raw := []byte("# TEST ONLY self-authored feed; no third-party source\nfeed-one.example.invalid\nfeed-two.example.invalid\n")
	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])
	object := "at-members/feed-" + l.run + ".txt"
	upload := func(body []byte) response {
		return l.request(nil, "PUT", "https://releases.example.invalid/e2e/object?name="+object, body, map[string]string{"X-E2E-Secret": l.secret("origin.secret")})
	}
	put := upload(raw)
	r.check("controlled feed input", "204 uploaded to existing controlled origin", detail(put), "PUT lab origin /e2e/object with synthetic feed bytes", put.status == 204)
	if put.status != 204 {
		return
	}
	now := time.Now().UTC()
	created, rev := l.policy("threat_feed", map[string]any{"provider": "TEST ONLY self-authored", "feed_code": "at-members-" + l.run, "license_name": "TEST ONLY acceptance fixture", "license_url": "https://releases.example.invalid/at-members/license.txt", "commercial_use_basis": "TEST ONLY self-authored synthetic names; permission for this acceptance fixture; no commercial source approval claimed", "attribution": nil, "source_url": "https://releases.example.invalid/" + object, "format": "domain_list", "categories": []string{"malware"}, "content_sha256": hash, "max_size_bytes": "4096", "valid_from": now.Add(-time.Minute), "valid_until": now.Add(time.Hour), "update_interval_seconds": 300})
	r.check("feed source and licensing record", "201 with explicit TEST ONLY basis and pin", detail(created), "POST /admin/policies threat_feed", created.status == 201)
	if rev == uuid.Nil {
		return
	}
	activated := l.activate(rev)
	r.check("feed approval activation", "202, normal worker fetch", detail(activated), "POST policy activate with If-Match", activated.status == 202)
	var good, reason string
	var failures int
	read := func() bool {
		err = l.pool.QueryRow(l.ctx, `SELECT coalesce(last_good_content,''),coalesce(last_error_code,''),consecutive_failures FROM dns_feed_refresh_state WHERE feed_revision_id=$1`, rev).Scan(&good, &reason, &failures)
		return err == nil && (good != "" || reason != "")
	}
	_ = l.wait(40*time.Second, read)
	r.check("production feed bootstrap", "record verified content or exact fetch precondition", fmt.Sprintf("hash_matches_pin=%t; last_error=%s; failures=%d", good == hash, reason, failures), "poll dns_feed_refresh_state written by running worker", err == nil)
	l.write(r)
	if good != hash {
		r.block("D-054 worker could not bootstrap the controlled feed: " + reason + ". Last-good cannot be tested without a verified initial snapshot.")
		return
	}
	profile, pid := l.policy("dns_profile", map[string]any{"profile_code": "at-members-feed", "display_name": map[string]string{"zh-TW": "TEST ONLY feed"}, "block_response": "nxdomain", "blocked_ttl_seconds": 5, "domain_rule_set_revision_ids": []string{}, "threat_feed_revision_ids": []uuid.UUID{rev}})
	if profile.status == 201 && pid != uuid.Nil {
		_ = l.activate(pid)
	}
	published := l.wait(30*time.Second, func() bool {
		return l.count(`SELECT count(*) FROM dns_resolver_documents WHERE profile_revision_id=$1`, pid) > 0
	})
	r.check("feed document publication", "worker compiles approved last-good into resolver document", fmt.Sprintf("published=%t", published), "activate DNS profile referencing feed; poll dns_resolver_documents", published)
	// Comment-only update parses to zero valid names: it must not replace last-good.
	bad := upload([]byte("# TEST ONLY intentionally empty update\n"))
	r.check("controlled bad update", "204; no product projection modified", detail(bad), "PUT same origin object with comment-only content", bad.status == 204)
	l.write(r)
	observed := l.wait(390*time.Second, func() bool { _ = read(); return reason != "" })
	rows := l.count(`SELECT count(*) FROM dns_feed_contents WHERE feed_revision_id=$1`, rev)
	r.check("real scheduled last-good retention", "zero_rules failure preserves original hash and one content row", fmt.Sprintf("failure_observed=%t; error=%s; hash_preserved=%t; content_rows=%d; failures=%d", observed, reason, good == hash, rows, failures), "wait actual 300-second cadence plus worker sweep; poll product refresh projection", observed && reason == "zero_rules" && good == hash && rows == 1 && failures > 0)
	// Replace the earlier empty-inventory finding with this stronger actual run.
	filtered := r.Blockers[:0]
	for _, b := range r.Blockers {
		if !strings.HasPrefix(b, "No active licensed threat feed exists") {
			filtered = append(filtered, b)
		}
	}
	r.Blockers = filtered
	r.block("D-054 central feed bootstrap/publication/last-good are exercised using self-authored TEST ONLY data. AT22 remains blocked on resolver document delivery (w3-contracts-r5, then nodes/agent), metrics listener, real member URL flow and remaining file/raw-usage/backup retention coverage. Commercial feed approval remains a business decision.")
}
