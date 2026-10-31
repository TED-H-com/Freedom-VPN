//go:build e2e

package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestE2E_Followup20Regular(t *testing.T) {
	if os.Getenv("FVPN_E2E_FOLLOWUP20") != "1" {
		t.Skip("set FVPN_E2E_FOLLOWUP20=1 with a fresh owned stack")
	}
	l, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	defer l.transport.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Minute)
	defer cancel()
	head, _ := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
	id := uuid.NewString()
	r := &runner{t: t, lab: l, runID: id, sourceCommit: strings.TrimSpace(string(head)), dir: filepath.Join(l.root, "docs/e2e/runs", time.Now().UTC().Format("2006-01-02"), "followup20", id)}
	defer r.finish()
	s := &Stack{lab: l}
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	admin := r.login(ctx, "e2e-admin", "admin")
	if admin == nil {
		return
	}
	b := BrowserSession{Client: admin.http, CSRF: admin.csrf}
	must(s.PrepareNativeUnderlay(ctx))
	must(s.InstallFixtureProbe(ctx))
	must(s.StartFixtureTargets(ctx))
	rev, _, err := s.ActivatePolicy(ctx, b, "domain_rule_set", "grants20-rules", map[string]any{"rules": []any{map[string]any{"rule_id": uuid.New(), "domain": "feed-01.example.invalid", "include_subdomains": true, "action": "block", "profile_id": nil, "reason": "admin_rule", "revision": "1"}}})
	must(err)
	r.profileID, _, err = s.ActivatePolicy(ctx, b, "dns_profile", "grants20-profile", map[string]any{"profile_code": "grants20", "display_name": map[string]string{"zh-TW": "TEST ONLY native18"}, "block_response": "nxdomain", "blocked_ttl_seconds": 5, "domain_rule_set_revision_ids": []uuid.UUID{rev}, "threat_feed_revision_ids": []string{}})
	must(err)
	r.catalogue(ctx, admin)
	good, bad := r.releases(ctx, admin)
	if r.enroll(ctx, admin) == uuid.Nil {
		return
	}
	r.desired(ctx)
	r.publish(ctx, admin, r.nodeID)
	r.fixtureServingMember(ctx, admin)
	if r.flowCounted {
		r.agentSupervisor(ctx)
		r.updater(ctx, good, bad)
		r.restartNode(ctx)
	}
}
