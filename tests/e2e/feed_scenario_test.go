//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	"github.com/google/uuid"
)

// threatFeed changes policy inputs only. Feed state, audit and publication rows
// are observations of the worker, never fixtures (D-050, D-054).
func (r *runner) threatFeed(ctx context.Context, admin *session) {
	rec := Record{ID: "threat-feed-central", Title: "Approved feed bootstrap, wrong pin and resolver publication", Spec: []string{"§12.3", "§12.5", "§19 AT22"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	r.record(Record{ID: "threat-feed-node", Title: "Feed delivery and enforcement on the node", Spec: []string{"§12.5", "§19 AT22"}, Status: StatusBlocked, Observations: []Observation{{Name: "owner", Detail: "D-056: contracts revision 5 owner, followed by node delivery/application (w2-agent-core). Central resolver publication alone does not prove on-node enforcement."}}})
	note := recPathNote(&rec)
	var fixture struct {
		SHA   string `json:"content_sha256"`
		URL   string `json:"source_url"`
		Count int    `json:"rule_count"`
	}
	raw, err := os.ReadFile(filepath.Join(r.lab.state, "feed.json"))
	if err != nil || json.Unmarshal(raw, &fixture) != nil {
		note("fixture", "missing prepare-time digest")
		return
	}
	note("prepared_content_sha256", fixture.SHA)
	conn, err := r.lab.db(ctx)
	if err != nil {
		note("database", "unavailable")
		return
	}
	defer conn.Close(ctx)
	create := func(kind, code string, def uuid.UUID, payload any) (uuid.UUID, uuid.UUID) {
		body := map[string]any{"type": kind, "schema_version": 1, "reason": "TEST ONLY lab feed acceptance", "payload": payload}
		if def == uuid.Nil {
			body["code"] = code
		} else {
			body["definition_id"] = def
		}
		resp, err := admin.call(ctx, "POST", r.lab.public+"/api/v1/admin/policies", body, map[string]string{"Idempotency-Key": uuid.NewString()})
		note("create_"+code, httpDetail(resp, err))
		if resp.Status >= 400 {
			note("create_error", apiErrorText(resp.Body))
		}
		var v struct {
			ID         uuid.UUID `json:"revision_id"`
			Definition uuid.UUID `json:"definition_id"`
		}
		if err != nil || resp.Status != 201 || json.Unmarshal(resp.Body, &v) != nil {
			return uuid.Nil, uuid.Nil
		}
		return v.ID, v.Definition
	}
	activate := func(rev, def uuid.UUID, label string) bool {
		var version string
		if conn.QueryRow(ctx, `SELECT resource_version::text FROM policy_definitions WHERE id=$1`, def).Scan(&version) != nil {
			return false
		}
		resp, err := admin.call(ctx, "POST", r.lab.public+"/api/v1/admin/policies/"+def.String()+"/activate", map[string]any{"revision_id": rev, "reason": "TEST ONLY approval"}, map[string]string{"Idempotency-Key": uuid.NewString(), "If-Match": `"` + version + `"`})
		note("activate_"+label, httpDetail(resp, err))
		return err == nil && resp.Status == 202
	}
	poll := func(fn func() bool) bool {
		wait, cancel := context.WithTimeout(ctx, 90*time.Second)
		defer cancel()
		return waitUntil(wait, time.Second, fn)
	}
	now := time.Now().UTC()
	payload := map[string]any{
		"provider": "TEST ONLY lab", "feed_code": "e2e-feed", "license_name": "TEST ONLY synthetic fixture", "license_url": "https://example.invalid/license",
		"commercial_use_basis": "Synthetic TEST ONLY fixture authored for acceptance", "attribution": "TEST ONLY synthetic acceptance fixture",
		"source_url": fixture.URL, "format": "hosts", "categories": []string{"malware"}, "content_sha256": fixture.SHA,
		"max_size_bytes": "4096", "valid_from": now.Add(-time.Minute), "valid_until": now.Add(2 * time.Hour), "update_interval_seconds": 300,
	}
	good, feedDef := create("threat_feed", "e2e-feed", uuid.Nil, payload)
	if good == uuid.Nil || !activate(good, feedDef, "good_feed") {
		return
	}
	var lastGood, kind string
	var count, successes int
	bootstrap := poll(func() bool {
		return conn.QueryRow(ctx, `SELECT s.last_good_content,c.rule_count,c.kind FROM dns_feed_refresh_state s JOIN dns_feed_contents c ON c.feed_revision_id=s.feed_revision_id AND c.content_sha256=s.last_good_content WHERE s.feed_revision_id=$1`, good).Scan(&lastGood, &count, &kind) == nil && lastGood == fixture.SHA && count == fixture.Count && kind == "approved_snapshot"
	})
	note("bootstrap", boolText(bootstrap))
	note("last_good_content", lastGood)
	note("normalized_rule_count", itoa(count))
	note("content_kind", kind)
	if !bootstrap {
		return
	}
	if conn.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='dns.feed.refreshed' AND resource='policy_revision:' || $1 AND result='succeeded'`, good.String()).Scan(&successes) != nil {
		return
	}
	note("bootstrap_audit_count", itoa(successes))
	beforeMetrics := r.feedMetrics()
	note("metrics_after_bootstrap", beforeMetrics)
	profile, profileDef := create("dns_profile", "e2e-feed-profile", uuid.Nil, map[string]any{
		"profile_code": "e2e-feed-profile", "display_name": map[string]string{"zh-TW": "TEST ONLY", "en": "TEST ONLY"},
		"domain_rule_set_revision_ids": []string{}, "threat_feed_revision_ids": []uuid.UUID{good}, "block_response": "nxdomain", "blocked_ttl_seconds": 60,
	})
	if profile == uuid.Nil || !activate(profile, profileDef, "profile") {
		return
	}
	r.profileID = profile
	var generation, docHash string
	var docRaw []byte
	published := poll(func() bool {
		return conn.QueryRow(ctx, `SELECT d.policy_generation::text,d.document_sha256,d.document FROM dns_resolver_documents d JOIN dns_publication_heads h ON h.profile_revision_id=d.profile_revision_id AND h.policy_generation=d.policy_generation WHERE d.profile_revision_id=$1 AND h.status='active'`, profile).Scan(&generation, &docHash, &docRaw) == nil
	})
	note("resolver_document_exists", boolText(published))
	note("policy_generation", generation)
	note("resolver_document_sha256", docHash)
	if !published {
		return
	}
	var doc dnspublish.Document
	normalized := json.Unmarshal(docRaw, &doc) == nil && dnspublish.Verify(doc) == nil && len(doc.Feeds) == 1
	if normalized {
		feed := doc.Feeds[0]
		normalized = feed.ContentSHA256 == fixture.SHA && feed.RuleCount == fixture.Count && len(feed.Entries) == fixture.Count
		for i, e := range feed.Entries {
			normalized = normalized && e.Domain == fmt.Sprintf("feed-%02d.example.invalid", i+1)
		}
		hash, _, err := dnspublish.Hash(doc)
		normalized = normalized && err == nil && hash == docHash
	}
	note("document_embeds_normalized_rules_and_hash_matches", boolText(normalized))
	// A new revision has its own bootstrap pin; failure must not destroy the
	// previous revision's content or the existing resolver publication.
	payload["content_sha256"] = strings.Repeat("0", 64)
	bad, _ := create("threat_feed", "e2e-feed-bad-pin", feedDef, payload)
	if bad == uuid.Nil || !activate(bad, feedDef, "bad_pin") {
		return
	}
	var badGood, errorCode string
	var failures int
	rejected := poll(func() bool {
		return conn.QueryRow(ctx, `SELECT COALESCE(last_good_content,'null'),COALESCE(last_error_code,''),consecutive_failures FROM dns_feed_refresh_state WHERE feed_revision_id=$1`, bad).Scan(&badGood, &errorCode, &failures) == nil && errorCode == "content_mismatch"
	})
	note("wrong_pin_rejected", boolText(rejected))
	note("wrong_pin_last_good_content", badGood)
	note("wrong_pin_last_error_code", errorCode)
	note("wrong_pin_consecutive_failures", itoa(failures))
	var audits int
	if conn.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='dns.feed.refresh_failed' AND resource='policy_revision:' || $1 AND result='failed' AND after_diff->>'reason_code'='content_mismatch'`, bad.String()).Scan(&audits) != nil {
		return
	}
	note("refresh_failed_audit_count", itoa(audits))
	var retained, headHash, headGeneration string
	if conn.QueryRow(ctx, `SELECT s.last_good_content,h.document_sha256,h.policy_generation::text FROM dns_feed_refresh_state s CROSS JOIN dns_publication_heads h WHERE s.feed_revision_id=$1 AND h.definition_id=$2`, good, profileDef).Scan(&retained, &headHash, &headGeneration) != nil {
		return
	}
	kept := retained == fixture.SHA && headHash == docHash && headGeneration == generation
	note("previous_revision_last_good_and_publication_retained", boolText(kept))
	afterMetrics := r.feedMetrics()
	note("metrics_after_bad_pin", afterMetrics)
	note("scope", "Last-good is per revision: the rejected new revision stays null; the prior approved content and resolver generation remain intact. No on-node claim.")
	if rejected && badGood == "null" && failures > 0 && audits > 0 && successes > 0 && normalized && kept && strings.Contains(beforeMetrics, `result="success"} 1`) && strings.Contains(beforeMetrics, "fvpn_dns_feed_last_good_age_seconds{") && strings.Contains(afterMetrics, `result="content_mismatch"} 1`) {
		rec.Status = StatusPass
	}
}

func (r *runner) feedMetrics() string {
	raw, err := r.lab.docker(r.lab.prefix+"-worker", "wget", "-qO-", "http://127.0.0.1:9092/metrics")
	if err != nil {
		return "unavailable"
	}
	var lines []string
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "fvpn_dns_feed_") && strings.Contains(line, `feed_code="e2e-feed"`) {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "; ")
}
