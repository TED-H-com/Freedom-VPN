//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestE2E_Followup20Grants(t *testing.T) {
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
	current, e := s.Call(ctx, b, "GET", "/api/v1/admin/fleet-config", nil, nil)
	must(e)
	changed, e := s.Call(ctx, b, "PUT", "/api/v1/admin/fleet-config", map[string]any{"max_offline_lease_seconds": 60, "reason": "TEST ONLY bounded D-087 before enrollment"}, map[string]string{"If-Match": current.Header.Get("ETag"), "Idempotency-Key": uuid.NewString()})
	must(e)
	if changed.Status != 200 {
		t.Fatalf("fleet config HTTP %d", changed.Status)
	}
	must(s.PrepareNativeUnderlay(ctx))
	must(s.InstallFixtureProbe(ctx))
	must(s.StartFixtureTargets(ctx))
	rev, _, err := s.ActivatePolicy(ctx, b, "domain_rule_set", "grants20-rules", map[string]any{"rules": []any{map[string]any{"rule_id": uuid.New(), "domain": "feed-01.example.invalid", "include_subdomains": true, "action": "block", "profile_id": nil, "reason": "admin_rule", "revision": "1"}}})
	must(err)
	r.profileID, _, err = s.ActivatePolicy(ctx, b, "dns_profile", "grants20-profile", map[string]any{"profile_code": "grants20", "display_name": map[string]string{"zh-TW": "TEST ONLY native18"}, "block_response": "nxdomain", "blocked_ttl_seconds": 5, "domain_rule_set_revision_ids": []uuid.UUID{rev}, "threat_feed_revision_ids": []string{}})
	must(err)
	r.catalogue(ctx, admin)
	_, err = PrepareServingBaseline(ctx, admin.http, admin.csrf, false)
	must(err)
	if r.enroll(ctx, admin) == uuid.Nil {
		return
	}
	r.desired(ctx)
	r.publish(ctx, admin, r.nodeID)
	r.fixtureServingMember(ctx, admin)
	if r.flowCounted {
		r.memberGrants20(ctx, s)
	}
}

func qualifyingGrantReport(e GrantWireEvent, node uuid.UUID, after time.Time, provisional bool) (api.ObservationReport, bool) {
	var report api.ObservationReport
	if e.Kind != "observation" || e.Status != 202 || e.Started.Before(after) || json.Unmarshal(e.Request, &report) != nil || report.NodeId != node || report.ProvisioningState != "ready" || report.SafetyStatus != "passed" {
		return report, false
	}
	lease, err := report.Lease.LeaseId.Get()
	if err != nil || lease == uuid.Nil {
		return report, false
	}
	if !provisional {
		return report, report.MemberForwardingEnabled && len(report.ForwardingBlockReasons) == 0
	}
	if report.MemberForwardingEnabled || len(report.ForwardingBlockReasons) != 1 || report.ForwardingBlockReasons[0] != "awaiting_member_grants" || len(report.Errors) != 1 || report.Errors[0].Code != "AWAITING_MEMBER_GRANTS" || report.Errors[0].Stage != "lease" {
		return report, false
	}
	checks := map[string]bool{}
	for _, c := range report.Checks {
		if c.Name == "lease" || c.Name == "qos" || c.Name == "source_binding" {
			code, e := c.ErrorCode.Get()
			checks[string(c.Name)] = e == nil && code == "AWAITING_MEMBER_GRANTS"
		}
	}
	return report, checks["lease"] && checks["qos"] && checks["source_binding"]
}
func (r *runner) memberGrants20(ctx context.Context, s *Stack) {
	rec := Record{ID: "member-grants-fixed-generation", Title: "D-087 natural empty lease, accepted provisional report, restored grants and counted serving", Spec: []string{"AT10", "ISO01", "ISO02", "§16.3"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	baseline, err := s.MemberGrantInvariants(ctx, r.nodeID)
	noteJSON(&rec, "before", baseline)
	if err != nil {
		note("error", err.Error())
		return
	}
	// Refuse a dirty starting state, including a generation that has not applied.
	var inv struct {
		Node        []string          `json:"node"`
		Publication []string          `json:"publication"`
		Frozen      bool              `json:"frozen"`
		Bindings    []json.RawMessage `json:"bindings"`
	}
	if json.Unmarshal(baseline, &inv) != nil || len(inv.Node) != 5 || inv.Node[1] != "ready" || inv.Node[2] != "passed" || inv.Node[3] != inv.Node[4] || inv.Frozen || len(inv.Bindings) == 0 || len(inv.Publication) < 1 || inv.Publication[0] != "published" {
		note("error", "clean fixed-generation baseline absent")
		return
	}
	falseFlip, err := s.SetMemberGrantsReady(ctx, false)
	noteJSON(&rec, "false_flip", falseFlip)
	if err != nil {
		note("error", err.Error())
		return
	}
	defer func() { _, _ = s.SetMemberGrantsReady(context.WithoutCancel(ctx), true) }()
	var provisional, serving api.ObservationReport
	var provisionalEvent, servingEvent GrantWireEvent
	await := func(after time.Time, want bool) bool {
		wait, cancel := context.WithTimeout(ctx, 3*time.Minute)
		defer cancel()
		return waitUntil(wait, time.Second, func() bool {
			events, e := s.MemberGrantEvidence(wait)
			if e != nil {
				return false
			}
			for _, v := range events {
				p, ok := qualifyingGrantReport(v, r.nodeID, after, want)
				if ok {
					if want {
						provisional, provisionalEvent = p, v
					} else {
						serving, servingEvent = p, v
					}
					return true
				}
			}
			return false
		})
	}
	provisionalOK := await(falseFlip.Completed, true)
	middle, middleErr := s.MemberGrantInvariants(ctx, r.nodeID)
	noteJSON(&rec, "provisional_invariants", middle)
	noteJSON(&rec, "provisional_receipt", provisionalEvent)
	if !provisionalOK {
		events, _ := s.MemberGrantEvidence(ctx)
		noteJSON(&rec, "original_wire", events)
		note("error", "no qualifying accepted provisional report; grants released only by cleanup")
		return
	}
	trueFlip, err := s.SetMemberGrantsReady(ctx, true)
	noteJSON(&rec, "true_flip", trueFlip)
	if err != nil {
		return
	}
	servingOK := await(trueFlip.Completed, false)
	noteJSON(&rec, "serving_receipt", servingEvent)
	after, afterErr := s.MemberGrantInvariants(ctx, r.nodeID)
	noteJSON(&rec, "after", after)
	beforeCounter, be := s.DeviceByteCounters(ctx, r.usageDevice)
	transfer, te := s.fixtureProbe(ctx, r.lab.member, "-mode", "bulk", "-addr", r.lab.originIP+":9446", "-device", r.usageDevice.String())
	afterCounter, ae := s.DeviceByteCounters(ctx, r.usageDevice)
	noteJSON(&rec, "counted_forwarding", map[string]any{"before": beforeCounter, "after": afterCounter, "transfer": transfer})
	events, wireErr := s.MemberGrantEvidence(ctx)
	noteJSON(&rec, "original_wire", events)
	// Join report lease IDs to genuine signed response bodies, never inferred slots.
	empty, populated := false, false
	for _, v := range events {
		if v.Kind != "lease" || v.Status != 200 {
			continue
		}
		var lease struct {
			Signed struct {
				ID         uuid.UUID         `json:"lease_id"`
				Generation string            `json:"config_generation"`
				Grants     []json.RawMessage `json:"member_grants"`
			} `json:"signed"`
		}
		if json.Unmarshal(v.Response, &lease) != nil {
			continue
		}
		p, _ := provisional.Lease.LeaseId.Get()
		f, _ := serving.Lease.LeaseId.Get()
		empty = empty || lease.Signed.ID == p && len(lease.Signed.Grants) == 0 && v.Started.After(falseFlip.Completed) && lease.Signed.Generation == inv.Node[3]
		populated = populated || lease.Signed.ID == f && len(lease.Signed.Grants) > 0 && v.Started.After(trueFlip.Completed) && lease.Signed.Generation == inv.Node[3]
	}
	same := middleErr == nil && afterErr == nil && bytes.Equal(baseline, middle) && bytes.Equal(baseline, after)
	noteJSON(&rec, "proof", map[string]any{"invariants_byte_equal": same, "empty_lease_report_join": empty, "populated_lease_report_join": populated, "distinct_reports": provisional.ReportId != serving.ReportId, "true_to_serving_receipt_ms": servingEvent.Completed.Sub(trueFlip.Completed).Milliseconds()})
	if same && empty && populated && servingOK && wireErr == nil && be == nil && te == nil && ae == nil && transfer.Available && transfer.Bytes == 64064 && afterCounter.Upload > beforeCounter.Upload && afterCounter.Download > beforeCounter.Download {
		rec.Status = StatusPass
	}
}
