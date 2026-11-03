//go:build e2e

package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// One invocation is one fresh-stack attempt. Run once with the TEST ONLY stale
// ETag switch, then once on another fresh stack without it; retain both IDs.
func TestE2E_Followup32Publisher(t *testing.T) {
	if os.Getenv("FVPN_E2E_FOLLOWUP32") != "1" {
		t.Skip("set FVPN_E2E_FOLLOWUP32=1 on a fresh owned stack")
	}
	l, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	defer l.transport.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Minute)
	defer cancel()
	head, err := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	runID := uuid.NewString()
	r := &runner{t: t, lab: l, runID: runID, sourceCommit: strings.TrimSpace(string(head)), summary: true,
		dir: filepath.Join(l.root, "docs/e2e/runs/2026-11-03/followup32", runID)}
	defer r.finish()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	admin := r.login(ctx, "e2e-admin", "admin")
	if admin == nil {
		return
	}
	s := &Stack{lab: l}
	b := BrowserSession{Client: admin.http, CSRF: admin.csrf}
	must(s.PrepareNativeUnderlay(ctx))
	must(s.InstallFixtureProbe(ctx))
	must(s.StartFixtureTargets(ctx))
	rev, _, err := s.ActivatePolicy(ctx, b, "domain_rule_set", "publisher32-rules", map[string]any{"rules": []any{map[string]any{"rule_id": uuid.New(), "domain": "feed-01.example.invalid", "include_subdomains": true, "action": "block", "profile_id": nil, "reason": "admin_rule", "revision": "1"}}})
	must(err)
	r.profileID, _, err = s.ActivatePolicy(ctx, b, "dns_profile", "publisher32-profile", map[string]any{"profile_code": "publisher32", "display_name": map[string]string{"zh-TW": "TEST ONLY publisher32"}, "block_response": "nxdomain", "blocked_ttl_seconds": 5, "domain_rule_set_revision_ids": []uuid.UUID{rev}, "threat_feed_revision_ids": []string{}})
	must(err)
	r.catalogue(ctx, admin)
	_, err = PrepareServingBaseline(ctx, admin.http, admin.csrf, false)
	must(err)
	if r.enroll(ctx, admin) == uuid.Nil {
		return
	}
	r.desired(ctx)
	r.publish(ctx, admin, r.nodeID)
	r.publisherReceipts32(ctx)
	if r.bad {
		return
	}
	r.fixtureServingMember(ctx, admin)
	r.servingGate26(ctx, "publisher-serving")
}

func (r *runner) publisherReceipts32(ctx context.Context) {
	rec := Record{ID: "publisher-retry-proof", Title: "Real publisher conflict attempts and durable command keys", Spec: []string{"§14.5", "API01"}, Status: StatusBlocked}
	defer func() { r.record(rec) }()
	forced := os.Getenv("FVPN_E2E_PUBLISH_STALE_ETAG") == "1"
	noteJSON(&rec, "test_only_stale_etag", forced)
	raw, err := os.ReadFile(filepath.Join(r.dir, "publish.json"))
	if err != nil {
		return
	}
	var published Record
	if json.Unmarshal(raw, &published) != nil {
		return
	}
	conn, err := r.lab.db(ctx)
	if err != nil {
		return
	}
	defer conn.Close(ctx)
	commands, conflicts := 0, 0
	valid := published.Status == StatusPass
	for _, observation := range published.Observations {
		if observation.Name != "node_write" {
			continue
		}
		var trace nodeWriteTrace
		if json.Unmarshal([]byte(observation.Detail), &trace) != nil || len(trace.Attempts) == 0 {
			return
		}
		commands++
		key := trace.Attempts[0].IdempotencyKey
		traceConflicts, traceValid := publisherTrace32(trace, forced)
		conflicts += traceConflicts
		valid = valid && traceValid
		// The same-key retry reached 202, so the 412 was not replay-cached.
		// Read back the sole durable success reference for that key as well.
		hash := sha256.Sum256([]byte(key))
		var count, successes int
		err = conn.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE state='completed' AND response_status=202 AND operation_id IS NOT NULL)
			FROM idempotency_requests WHERE key_hash=$1`, hash[:]).Scan(&count, &successes)
		noteJSON(&rec, "durable_command", map[string]any{"method": trace.Attempts[0].Method, "key": key, "rows": count, "accepted_operation_rows": successes, "read_ok": err == nil, "at": time.Now().UTC()})
		valid = valid && err == nil && count == 1 && successes == 1
	}
	noteJSON(&rec, "commands", commands)
	noteJSON(&rec, "real_conflicts", conflicts)
	valid = valid && commands == 2 && (!forced || conflicts >= 2)
	if valid {
		rec.Status = StatusPass
	}
}
