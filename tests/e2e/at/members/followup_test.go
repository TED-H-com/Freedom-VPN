//go:build e2e

package members

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	if l.pool.QueryRow(ctx, `SELECT id,current_incarnation_id FROM nodes WHERE current_incarnation_id IS NOT NULL ORDER BY created_at ASC LIMIT 1`).Scan(&l.node, &l.inc) != nil {
		t.Fatal("prior run node unavailable")
	}
	ids := []string{"AT02", "AT03", "AT04", "AT08", "AT20", "AT22"}
	for _, id := range ids {
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
			l.revocationReadback(&r)
			devices := l.count(`SELECT count(*) FROM member_devices WHERE status='authorized'`)
			proofs := l.count(`SELECT count(*) FROM netbird_user_bindings n JOIN identity_bindings i ON i.id=n.identity_binding_id WHERE n.external_user_id=i.subject AND n.mapping_proof='shared_issuer_public_subject:'||i.issuer`)
			var desired, applied string
			_ = l.pool.QueryRow(ctx, `SELECT desired_config_generation::text,applied_config_generation::text FROM nodes WHERE id=$1`, l.node).Scan(&desired, &applied)
			observations := l.count(`SELECT count(*) FROM device_authorization_observations WHERE observed_authorized`)
			r.check("post-worker projection readback", "record native proofs, devices and desired/applied separately", fmt.Sprintf("verified_proofs=%d; authorized_devices=%d; observed_authorized=%d; desired_generation=%s; applied_generation=%s", proofs, devices, observations, desired, applied), "read product projections after worker resumes; no projection fixtures", proofs >= 2 && devices >= 4)
		case "AT03", "AT04":
			l.revocationReadback(&r)
		case "AT08":
			// Append observation-only checks; preserve primary failed/blocked/pass state.

			_ = l.pool.QueryRow(ctx, `SELECT id FROM member_groups WHERE code LIKE 'at-a-%' ORDER BY code LIMIT 1`).Scan(&l.groupA)
			_ = l.pool.QueryRow(ctx, `SELECT id FROM member_groups WHERE code LIKE 'at-b-%' ORDER BY code LIMIT 1`).Scan(&l.groupB)
			// Primary suite already exercised mutation/restart. A supplement
			// observes completion; it must not turn a retryable intermediate
			// publication step into a new full acceptance pass.
			var publication []byte
			err := l.pool.QueryRow(ctx, `SELECT coalesce(jsonb_agg(jsonb_build_object('status',status,'steps',steps) ORDER BY created_at),'[]'::jsonb) FROM operations WHERE kind IN ('node.publish','node.disable')`).Scan(&publication)
			r.check("publication completion readback", "record terminal/current step evidence without a new command", string(publication), "read product operation history; no publication mutation", err == nil)
			l.restartReadback(&r)
			l.servingPrecondition(&r)
			l.routeReadback(&r)
		case "AT20":
			for i, b := range r.Blockers {
				r.Blockers[i] = strings.ReplaceAll(b, "No admitted member DNS/forwarding path", "Native member admission is proven in AT01/AT02, but native DNS is disabled and no applied member DNS/forwarding path exists")
			}
			docs := l.count(`SELECT count(*) FROM dns_resolver_documents`)
			r.check("central resolver document readback", "record stored publication versus node delivery", fmt.Sprintf("central_documents=%d", docs), "SELECT dns_resolver_documents", docs > 0)
			l.resolverEvidence(&r)
		case "AT22":
			if os.Getenv("FVPN_AT_MEMBERS_OBSERVE_ONLY") != "1" {
				l.feedLastGood(&r)
			}
		}
		l.write(&r)
		if r.Status == "fail" {
			t.Errorf("%s readback contains a failed subcheck", id)
		}
	}
}
func (l *live) servingPrecondition(r *record) {
	policies := l.count(`SELECT count(*) FROM policy_definitions WHERE type='release_support' AND active_revision_id IS NOT NULL`)
	r.check("shared serving support readback", "nonempty active API policy", fmt.Sprintf("active_release_policies=%d", policies), "read policy_definitions after PrepareServingBaseline", policies > 0)
	l.forwardingGate(r)
	l.resolverEvidence(r)
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
	raw := []byte("# TEST ONLY self-authored feed; no third-party source\nfeed-01.example.invalid\nlab.example.invalid\n")
	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])
	object := "at-members/feed-" + l.run + ".txt"
	upload := func(body []byte) response {
		return l.request(nil, "PUT", "https://releases.example.invalid/e2e/object?name="+object, body, map[string]string{"X-E2E-Secret": l.secret("origin.secret")})
	}
	defer func() { _ = upload(raw) }()
	put := upload(raw)
	r.check("controlled feed input", "204 uploaded to existing controlled origin", detail(put), "PUT lab origin /e2e/object with synthetic feed bytes", put.status == 204)
	if put.status != 204 {
		return
	}
	now := time.Now().UTC()
	feedPayload := map[string]any{"provider": "TEST ONLY self-authored", "feed_code": "at-members-" + l.run, "license_name": "TEST ONLY acceptance fixture", "license_url": "https://releases.example.invalid/at-members/license.txt", "commercial_use_basis": "TEST ONLY self-authored synthetic names; permission for this acceptance fixture; no commercial source approval claimed", "attribution": nil, "source_url": "https://releases.example.invalid/" + object, "format": "domain_list", "categories": []string{"malware"}, "content_sha256": hash, "max_size_bytes": "4096", "valid_from": now.Add(-time.Minute), "valid_until": now.Add(time.Hour), "update_interval_seconds": 300}
	for _, note := range []string{"", " \t "} {
		feedPayload["attribution"] = note
		rejected, _ := l.policy("threat_feed", feedPayload)
		r.check("blank attribution rejected", "422 for blank or whitespace notes", detail(rejected), "POST threat_feed with blank attribution", rejected.status == 422)
	}
	feedPayload["attribution"] = nil
	created, rev := l.policy("threat_feed", feedPayload)
	r.check("feed source and licensing record", "201 with attribution:null, TEST ONLY basis and pin", detail(created), "POST /admin/policies threat_feed", created.status == 201)
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
	beforeDNS := l.dnsLocal("feed-01.example.invalid", "1", "udp")
	r.check("feed positive DNS control", "known controlled upstream name resolves before feed activation", beforeDNS, "node loopback query controlled feed-01.example.invalid before feed profile", strings.Contains(beforeDNS, `"rcode":0`) && strings.Contains(beforeDNS, `"answers":1`))
	profile, pid := l.policy("dns_profile", map[string]any{"profile_code": "at-members-feed", "display_name": map[string]string{"zh-TW": "TEST ONLY feed"}, "block_response": "nxdomain", "blocked_ttl_seconds": 5, "domain_rule_set_revision_ids": []string{}, "threat_feed_revision_ids": []uuid.UUID{rev}})
	r.check("feed profile creation", "record reference validation result", detail(profile), "POST DNS profile referencing approved feed", profile.status == 201)
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
	r.check("real scheduled last-good retention", "empty/invalid-list failure preserves original hash and one content row", fmt.Sprintf("failure_observed=%t; error=%s; hash_preserved=%t; content_rows=%d; failures=%d", observed, reason, good == hash, rows, failures), "wait actual 300-second cadence plus worker sweep; poll product refresh projection", observed && (reason == "zero_rules" || reason == "parse_error") && good == hash && rows == 1 && failures > 0)
	// Replace the earlier empty-inventory finding with this stronger actual run.
	filtered := r.Blockers[:0]
	for _, b := range r.Blockers {
		if !strings.HasPrefix(b, "No active licensed threat feed exists") {
			filtered = append(filtered, b)
		}
	}
	r.Blockers = filtered
	l.resolverEvidence(r)
	answer := l.dnsLocal("feed-01.example.invalid", "1", "udp")
	r.check("node last-good block response", "controlled positive name remains NXDOMAIN after invalid feed update", answer, "node loopback DNS after actual 300-second failed refresh; not a native member DNS claim", strings.Contains(answer, `"rcode":3`))
	r.block("D-054 central feed bootstrap/publication/last-good are exercised using self-authored TEST ONLY data. Complete AT22 still needs member URL traffic and remaining file/raw-usage/backup retention coverage; typed resolver application is recorded separately above. Commercial feed approval remains a business decision.")
}

// F2-05: restart must resume through the unchanged entrypoint, without a token.
func (l *live) restartReadback(r *record) {
	var stage string
	resumed := l.wait(45*time.Second, func() bool {
		raw, _ := l.inside("node", "cat", "/var/lib/vpn-node/e2e/status.json")
		var state struct {
			Stage string `json:"stage"`
		}
		_ = json.Unmarshal(raw, &state)
		stage = state.Stage
		return stage == "enrolled" && (l.restartAfter.IsZero() || l.count(`SELECT count(*) FROM node_observations WHERE node_id=$1 AND received_at>$2`, l.node, l.restartAfter) > 0)
	})
	_, tokenErr := l.inside("node", "test", "-e", "/var/lib/vpn-node/e2e/enrollment.token")
	_, nftErr := l.inside("node", "nft", "-j", "list", "table", "inet", "fvpn")
	var current uuid.UUID
	err := l.pool.QueryRow(l.ctx, `SELECT current_incarnation_id FROM nodes WHERE id=$1`, l.node).Scan(&current)
	r.check("durable restart without token", "enrolled, same incarnation, nft readable, token absent", fmt.Sprintf("resumed=%t; stage=%s; same_incarnation=%t; nft_readable=%t; token_absent=%t", resumed, stage, current == l.inc, nftErr == nil, tokenErr != nil), "docker restart node; poll entrypoint and product incarnation; no Service workaround", resumed && err == nil && current == l.inc && nftErr == nil && tokenErr != nil)
}
