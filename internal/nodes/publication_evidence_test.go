package nodes

import (
	"encoding/json"
	"net/netip"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/agent"
	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func TestSECOPS11_RecordedBaselineDoesNotProveServing(t *testing.T) {
	// testdata/secops11_recorded_baseline.json is the SECOPS-11 recorded baseline
	// captured by the security-ops harness before #109 and is frozen here on purpose:
	// the harness's docs/e2e/at/secops/live.json is rewritten by every acceptance run
	// and must not be read by unit tests.
	raw, err := os.ReadFile("testdata/secops11_recorded_baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		PublicationOperation struct {
			Status string `json:"status"`
		} `json:"publication_operation"`
		WatchdogState struct {
			Serving      string `json:"serving"`
			Reason       string `json:"reason"`
			PolicyExpiry string `json:"policy_expires_mono_ns"`
		} `json:"watchdog_state"`
		NodeAfterPublication struct {
			DataHealth       api.DataHealth       `json:"data_health"`
			ConfigGeneration api.ConfigGeneration `json:"config_generation"`
		} `json:"node_after_publication"`
		FlowProbe struct {
			CountedFlow bool `json:"counted_flow"`
			NodeServing bool `json:"node_serving"`
		} `json:"flow_probe"`
	}
	if err = json.Unmarshal(raw, &record); err != nil {
		t.Fatal(err)
	}
	if record.PublicationOperation.Status != "succeeded" || record.WatchdogState.Serving != "0" || record.WatchdogState.Reason != "recovery_required" || record.WatchdogState.PolicyExpiry != "0" || record.FlowProbe.CountedFlow || record.FlowProbe.NodeServing {
		t.Fatal("SECOPS-11 fixture shape changed")
	}
	now := time.Now().UTC()
	report := api.ObservationReport{ProvisioningState: api.ProvisioningStateReady, DataHealth: record.NodeAfterPublication.DataHealth, SafetyStatus: api.SafetyStatusPassed, MemberForwardingEnabled: record.FlowProbe.NodeServing}
	report.AppliedConfigGeneration = record.NodeAfterPublication.ConfigGeneration.Applied
	for _, name := range []api.NodeCheckName{api.NodeCheckNameInternetOnly, api.NodeCheckNameMeterPath, api.NodeCheckNameTunnelProbe, api.NodeCheckNameLease, api.NodeCheckNameQos, api.NodeCheckNameSourceBinding} {
		report.Checks = append(report.Checks, api.NodeCheck{Name: name, Status: api.HealthCheckStatusPassed})
	}
	if PublicationForwardingEvidence(report, now) {
		t.Fatal("baseline claimed member serving")
	}
	report.MemberForwardingEnabled = true
	if PublicationForwardingEvidence(report, now) {
		t.Fatal("missing lease claimed member serving")
	}
	report.Lease.LeaseId.Set(uuid.New())
	report.Lease.ExpiresAt.Set(now.Add(time.Minute))
	if !PublicationForwardingEvidence(report, now) {
		t.Fatal("armed policy rejected")
	}
	report.AppliedConfigGeneration.Set(0)
	if PublicationForwardingEvidence(report, now) {
		t.Fatal("generation zero claimed serving")
	}
	report.AppliedConfigGeneration.Set(1)
	report.Lease.ExpiresAt.Set(now)
	if PublicationForwardingEvidence(report, now) {
		t.Fatal("expired lease claimed serving")
	}
}

func TestAT08_PublicationBindingAwareEvidence(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct {
		name  string
		empty bool
		edit  func(*api.ObservationReport)
		want  string
	}{
		{"empty_not_run", true, nil, ""},
		{"bound_not_run", false, nil, "serving_checks_unconfirmed"},
		{"empty_failed", true, func(r *api.ObservationReport) { r.Checks[3].Status = api.HealthCheckStatusFailed }, "check_failed"},
		{"bound_failed", false, func(r *api.ObservationReport) { r.Checks[3].Status = api.HealthCheckStatusFailed }, "check_failed"},
		{"failed_then_passed_duplicate", true, func(r *api.ObservationReport) {
			c := r.Checks[0]
			c.Status = api.HealthCheckStatusFailed
			r.Checks = append([]api.NodeCheck{c}, r.Checks...)
		}, "check_failed"},
		{"forwarding_off", true, func(r *api.ObservationReport) { r.MemberForwardingEnabled = false }, "forwarding_disabled"},
		{"tunnel_not_run", true, func(r *api.ObservationReport) { r.Checks[2].Status = api.HealthCheckStatusNotRun }, "tunnel_not_passed"},
		{"isolation_absent", true, func(r *api.ObservationReport) { r.Checks = r.Checks[1:] }, "isolation_unconfirmed"},
		{"serving_check_absent", true, func(r *api.ObservationReport) { r.Checks = r.Checks[:3] }, "serving_checks_unconfirmed"},
		{"degraded", true, func(r *api.ObservationReport) { r.DataHealth = api.DataHealthDegraded }, "readiness_unconfirmed"},
		{"isolation_apply_error_pending", true, func(r *api.ObservationReport) {
			r.Errors = []api.NodeError{{
				Code:       api.ErrorCodeISOLATIONPOLICYINVALID,
				Stage:      "apply",
				Message:    "conntrack delete pending; serving ruleset verified",
				ObservedAt: now,
			}}
		}, "isolation_unconfirmed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := api.ObservationReport{ProvisioningState: api.ProvisioningStateReady, SafetyStatus: api.SafetyStatusPassed, DataHealth: api.DataHealthHealthy, MemberForwardingEnabled: true}
			r.AppliedConfigGeneration.Set(2)
			for i, name := range []api.NodeCheckName{api.NodeCheckNameInternetOnly, api.NodeCheckNameMeterPath, api.NodeCheckNameTunnelProbe, api.NodeCheckNameLease, api.NodeCheckNameQos, api.NodeCheckNameSourceBinding} {
				status := api.HealthCheckStatusNotRun
				if i < 3 {
					status = api.HealthCheckStatusPassed
				}
				r.Checks = append(r.Checks, api.NodeCheck{Name: name, Status: status})
			}
			if tc.edit != nil {
				tc.edit(&r)
			}
			if cause := publicationForwardingFailure(r, now, tc.empty); cause != tc.want || len(cause) > 64 {
				t.Fatal("unexpected publication gate", cause)
			}
		})
	}
}

func buildAgentReportForTest(now time.Time, revocationPending, conntrackBlocked bool, conntrackDetail string) api.ObservationReport {
	relID := uuid.MustParse("019a0000-0000-7000-8000-0000000000c1")
	memberID := uuid.MustParse("019a0000-0000-7000-8000-00000000000a")
	deviceID := uuid.MustParse("019a0000-0000-7000-8000-0000000000a1")
	nodeID := uuid.MustParse("019a0000-0000-7000-8000-0000000000e1")
	incID := uuid.MustParse("019a0000-0000-7000-8000-0000000000e2")
	leaseID := uuid.MustParse("019a0000-0000-7000-8000-0000000000e4")

	ds := &api.DesiredState{
		PublicationIntent: api.PublicationIntentPublished,
		ServingReleaseIds: []uuid.UUID{relID},
		ConfigGeneration:  3,
		BindingRevision:   2,
		NodeCapacity: presence.NewNullableWithValue(api.NodeCapacity{
			UpBps: 200_000_000, DownBps: 200_000_000,
		}),
		Dns: api.DnsServiceConfig{ResolverIpv4: []string{}, ResolverIpv6: []string{}},
		DeviceBindings: []api.DeviceBinding{
			{
				AllowedIpv4:     []string{"100.64.1.2/32"},
				AllowedIpv6:     []string{},
				BindingRevision: 2,
				DeviceId:        deviceID,
				MemberId:        memberID,
				PeerId:          "peer-a1",
				ValidFrom:       now.Add(-time.Minute),
				ValidUntil:      now.Add(time.Hour),
			},
		},
	}
	lease := &api.ServingLease{
		LeaseId:           leaseID,
		NodeId:            nodeID,
		NodeIncarnationId: incID,
		ConfigGeneration:  3,
		IssuedAt:          now,
		ExpiresAt:         now.Add(time.Hour),
		MaxOfflineSeconds: 86400,
		Members: []api.MemberLease{
			{
				MemberId:            memberID,
				LeaseEnd:            now.Add(time.Hour),
				EffectiveServiceEnd: now.Add(time.Hour),
			},
		},
	}
	in := agent.Input{
		Now:               now,
		MonoNow:           2 * time.Second,
		Datapath:          enforce.Datapath{MemberIngress: "fv-fh", Uplink: "fv-ih", SNAT: netip.MustParseAddr("192.0.2.37")},
		Desired:           ds,
		DesiredOK:         true,
		Lease:             lease,
		LeaseFresh:        true,
		LeaseMonoSent:     time.Second,
		AnchorOK:          true,
		Backend:           "kernel",
		ReleaseID:         relID,
		Enrolled:          true,
		BaselineOK:        true,
		StorageOK:         true,
		HostReserveBPS:    1_000_000,
		ProbeRan:          true,
		ProbeCheck:        api.HealthCheckStatusPassed,
		ProbeHealth:       api.DataHealthHealthy,
		RevocationPending: revocationPending,
		ConntrackBlocked:  conntrackBlocked,
		ConntrackDetail:   conntrackDetail,
	}
	d := agent.Decide(in)
	rep := api.ObservationReport{
		ReportId:                uuid.New(),
		NodeId:                  nodeID,
		NodeIncarnationId:       incID,
		BootId:                  uuid.MustParse("019a0000-0000-7000-8000-0000000000e3"),
		AgentVersion:            "0.0.0-test",
		ObservedAt:              now,
		ProvisioningState:       d.Provisioning,
		SafetyStatus:            d.Safety,
		DataHealth:              d.DataHealth,
		MemberForwardingEnabled: d.Serve,
		ForwardingBlockReasons:  d.Reasons,
		Checks:                  d.Checks,
		Errors:                  d.Errors,
	}
	if d.Applied > 0 {
		rep.AppliedConfigGeneration.Set(decstr.U64(d.Applied))
	}
	rep.Lease.LeaseId.Set(leaseID)
	rep.Lease.ExpiresAt.Set(lease.ExpiresAt)
	rep.Netbird.Backend = api.ObservationReportNetbirdBackendKernel
	return rep
}

func TestEvaluateObservationEvidence_IsolationApplyError(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	// 1. Agent's real pending-under-the-bound shape (D-070):
	// code=ISOLATION_POLICY_INVALID, stage=apply, message="conntrack delete pending; serving ruleset verified",
	// member_forwarding_enabled=true, safety_status=passed, every check passed.
	report := buildAgentReportForTest(now, true, false, "")
	if !report.MemberForwardingEnabled || report.SafetyStatus != api.SafetyStatusPassed {
		t.Fatalf("agent pending report = (forwarding=%v, safety=%s), want (true, passed)", report.MemberForwardingEnabled, report.SafetyStatus)
	}
	for _, ch := range report.Checks {
		if ch.Status != api.HealthCheckStatusPassed {
			t.Fatalf("agent pending check %s = %s, want passed", ch.Name, ch.Status)
		}
	}
	if len(report.Errors) != 1 || report.Errors[0].Code != api.ErrorCodeISOLATIONPOLICYINVALID || report.Errors[0].Stage != "apply" || report.Errors[0].Message != "conntrack delete pending; serving ruleset verified" {
		t.Fatalf("agent pending errors = %+v, want single ISOLATION_POLICY_INVALID apply error", report.Errors)
	}

	safety, health, forwarding, _ := evaluateObservationEvidence(report)
	if safety != api.SafetyStatusPassed {
		t.Fatalf("safety = %q, want passed while conntrack delete retry is pending", safety)
	}
	if health != api.DataHealthDegraded {
		t.Fatalf("health = %q, want degraded while ISOLATION_POLICY_INVALID apply error is present", health)
	}
	if !forwarding {
		t.Fatalf("forwarding = %v, want true", forwarding)
	}
	if _, ok := isPendingIsolationApplyError(report, forwarding); !ok {
		t.Fatal("expected isPendingIsolationApplyError to be true")
	}
	if isFailClosedIsolationApplyError(report, safety, forwarding) {
		t.Fatal("expected isFailClosedIsolationApplyError to be false while pending under bound")
	}

	// Keying on code + stage (D-070): a non-"apply" stage does not match hasIsolationApplyError.
	nonApply := report
	nonApply.Errors = []api.NodeError{{
		Code:       api.ErrorCodeISOLATIONPOLICYINVALID,
		Stage:      "preflight",
		Message:    "conntrack delete pending; serving ruleset verified",
		ObservedAt: now,
	}}
	if _, ok := hasIsolationApplyError(nonApply); ok {
		t.Fatal("expected hasIsolationApplyError to reject stage != apply")
	}

	// 2. Agent's real fail-closed shapes (D-070):
	// member_forwarding_enabled=false, safety_status=failed, forwarding_block_reasons contains safety_check_failed,
	// check internet_only failed with ISOLATION_POLICY_INVALID, and ISOLATION_POLICY_INVALID apply error in errors.
	for _, detail := range []string{
		"Conntrack delete exceeded the retry bound.",
		"Conntrack is not installed.",
	} {
		t.Run(detail, func(t *testing.T) {
			failClosed := buildAgentReportForTest(now, false, true, detail)
			if failClosed.MemberForwardingEnabled || failClosed.SafetyStatus != api.SafetyStatusFailed {
				t.Fatalf("agent fail-closed report = (forwarding=%v, safety=%s), want (false, failed)", failClosed.MemberForwardingEnabled, failClosed.SafetyStatus)
			}
			if !hasForwardingBlockReason(failClosed, api.ForwardingBlockReasonSafetyCheckFailed) {
				t.Fatalf("agent fail-closed ForwardingBlockReasons = %v, want safety_check_failed", failClosed.ForwardingBlockReasons)
			}
			var foundInternetOnly bool
			for _, ch := range failClosed.Checks {
				if ch.Name == api.NodeCheckNameInternetOnly {
					foundInternetOnly = true
					code, err := ch.ErrorCode.Get()
					var gotDetail string
					if ch.Detail != nil {
						gotDetail = *ch.Detail
					}
					if ch.Status != api.HealthCheckStatusFailed || err != nil || code != api.ErrorCodeISOLATIONPOLICYINVALID || gotDetail != detail {
						t.Fatalf("agent fail-closed internet_only check = %+v, want failed with ISOLATION_POLICY_INVALID and detail %q", ch, detail)
					}
				}
			}
			if !foundInternetOnly {
				t.Fatal("missing internet_only check on agent fail-closed report")
			}

			safetyFC, healthFC, forwardingFC, _ := evaluateObservationEvidence(failClosed)
			if safetyFC != api.SafetyStatusFailed {
				t.Fatalf("fail-closed safety = %q, want failed", safetyFC)
			}
			if healthFC != api.DataHealthDegraded {
				t.Fatalf("fail-closed health = %q, want degraded", healthFC)
			}
			if forwardingFC {
				t.Fatalf("fail-closed forwarding = %v, want false", forwardingFC)
			}
			if _, ok := isPendingIsolationApplyError(failClosed, forwardingFC); ok {
				t.Fatal("expected isPendingIsolationApplyError to be false once node failed closed")
			}
			if !isFailClosedIsolationApplyError(failClosed, safetyFC, forwardingFC) {
				t.Fatal("expected isFailClosedIsolationApplyError to be true on real agent fail-closed shape")
			}
		})
	}
}

func TestD067_AwaitingMemberGrantsObservationAndPublicationEvidence(t *testing.T) {
	raw, err := os.ReadFile("../../contracts/examples/valid/ObservationReport.awaiting-member-grants.json")
	if err != nil {
		t.Fatalf("read example: %v", err)
	}
	var report api.ObservationReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("unmarshal example: %v", err)
	}
	if report.MemberForwardingEnabled {
		t.Fatal("expected member_forwarding_enabled=false on awaiting_member_grants example")
	}
	if len(report.ForwardingBlockReasons) != 1 || report.ForwardingBlockReasons[0] != api.ForwardingBlockReasonAwaitingMemberGrants {
		t.Fatalf("forwarding_block_reasons = %v, want [awaiting_member_grants]", report.ForwardingBlockReasons)
	}
	if !report.ForwardingBlockReasons[0].Valid() {
		t.Fatal("expected awaiting_member_grants to be a valid ForwardingBlockReason")
	}

	safety, health, forwarding, errs := evaluateObservationEvidence(report)
	if safety != api.SafetyStatusPassed {
		t.Fatalf("safety = %q, want passed (provisional node lease, not expired)", safety)
	}
	if health != api.DataHealthHealthy {
		t.Fatalf("health = %q, want healthy", health)
	}
	if forwarding {
		t.Fatalf("forwarding = %v, want false", forwarding)
	}
	if len(errs) != 1 || errs[0].Code != api.ErrorCodeAWAITINGMEMBERGRANTS || errs[0].Stage != "lease" || errs[0].Message != "Serving lease has no member grants yet." {
		t.Fatalf("errors = %+v, want one AWAITING_MEMBER_GRANTS lease error", errs)
	}

	now := report.ObservedAt.UTC()
	// A failed lease/qos/source_binding check returns check_failed before the forwarding_disabled branch.
	if cause := publicationForwardingFailure(report, now, false); cause != "check_failed" {
		t.Fatalf("publicationForwardingFailure = %q, want check_failed", cause)
	}
	if PublicationForwardingEvidence(report, now) {
		t.Fatal("PublicationForwardingEvidence must be false while awaiting_member_grants has member_forwarding_enabled=false")
	}
}
