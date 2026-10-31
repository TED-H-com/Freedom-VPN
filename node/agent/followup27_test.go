package agent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// Replay the decision inputs of members run 11e305fe: matching generation 30,
// fresh 60s empty lease, desired bindings, failed probe and retained failed tc
// read. IDs are generated/TEST ONLY; no captured account identifiers are copied.
func TestD126_AT09_CapturedProvisionalSafetyComposition(t *testing.T) {
	in := servingInput()
	in.Now = time.Date(2026, 10, 31, 12, 6, 32, 520681650, time.UTC)
	in.Desired.IssuedAt, in.Desired.ExpiresAt = in.Now.Add(-time.Hour), in.Now.Add(time.Hour)
	in.Desired.ConfigGeneration, in.Lease.ConfigGeneration = 30, 30
	in.Desired.DeviceBindings = []api.DeviceBinding{oneBinding(in.Now)}
	in.Lease.IssuedAt = time.Date(2026, 10, 31, 12, 6, 32, 389206258, time.UTC)
	in.Lease.ExpiresAt = in.Lease.IssuedAt.Add(time.Minute)
	in.Lease.MaxOfflineSeconds = 60
	in.MonoNow = in.LeaseMonoSent + in.Now.Sub(in.Lease.IssuedAt)
	in.ProbeCheck, in.ProbeHealth = api.HealthCheckStatusFailed, api.DataHealthFailed
	in.ApplyError = &api.NodeError{Code: api.ErrorCodeTCFOREIGNFILTER, Message: "TC_FOREIGN_FILTER", Stage: "apply", ObservedAt: in.Now.Add(-36 * time.Millisecond)}
	_, rep := reported(t, in)
	wantReasons := []api.ForwardingBlockReason{api.ForwardingBlockReasonSafetyCheckFailed, api.ForwardingBlockReasonAwaitingMemberGrants}
	if !reflect.DeepEqual(rep.ForwardingBlockReasons, wantReasons) || rep.MemberForwardingEnabled || rep.SafetyStatus != api.SafetyStatusFailed {
		t.Fatalf("mixed report: %+v", rep)
	}
	for _, c := range rep.Checks {
		want := api.ErrorCode("")
		switch c.Name {
		case api.NodeCheckNameLease, api.NodeCheckNameQos, api.NodeCheckNameSourceBinding:
			want = api.ErrorCodeTCFOREIGNFILTER
		case api.NodeCheckNameTunnelProbe:
			want = api.ErrorCodeHEALTHCHECKFAILED
		}
		code, _ := c.ErrorCode.Get()
		if code != want || (want != "" && c.Status != api.HealthCheckStatusFailed) || (want == "" && c.Status != api.HealthCheckStatusPassed) {
			t.Fatalf("check precedence: %+v", c)
		}
	}
	wantErrors := []api.NodeError{
		{Code: api.ErrorCodeAWAITINGMEMBERGRANTS, Message: leaseGrantsPendingDetail, Stage: "lease", ObservedAt: in.Now},
		{Code: api.ErrorCodeHEALTHCHECKFAILED, Message: "tunnel_probe", Stage: "probe", ObservedAt: in.Now},
		*in.ApplyError,
	}
	if !reflect.DeepEqual(rep.Errors, wantErrors) {
		t.Fatalf("stage causes changed: %+v", rep.Errors)
	}
	// The adjacent clean provisional report returns the awaiting codes. The
	// earlier report is a value composed once; another decision cannot alter it.
	before, _ := json.Marshal(rep)
	in.ApplyError = nil
	in.ProbeCheck, in.ProbeHealth = api.HealthCheckStatusPassed, api.DataHealthHealthy
	_, provisional := reported(t, in)
	assertProvisionalReport(t, provisional)
	after, _ := json.Marshal(rep)
	if string(before) != string(after) {
		t.Fatal("later decision changed prior report")
	}
}

func TestD087_AT10_FixedGenerationProvisionalToGranted(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	r.srv.SetMaxOfflineSeconds(60)
	r.eng.cfg.RenewLeadSeconds = 30
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	r.eng.cfg.HeartbeatSeconds = 1
	r.eng.cfg.ObservationSeconds = 3600 // transitions must bypass periodic cadence
	gen := r.eng.appliedGen
	bindings := append([]api.DeviceBinding(nil), r.eng.desired.DeviceBindings...)
	oldLease := r.eng.lease.LeaseId
	r.srv.SetMemberGrantsReady(false)
	// An ordinary timed renewal, with the current lease still in force. No
	// generation bump, leaseFresh mutation, unpublish, restart or freeze.
	window := r.eng.lease.ExpiresAt.Sub(r.eng.lease.IssuedAt)
	r.clock.Advance(window - r.eng.renewLead(window) + r.eng.renewLead(window)/10 + time.Nanosecond)
	stepOK(t, r)
	provisional, ok := r.srv.LastObservation()
	if !ok {
		t.Fatal("missing delivered provisional report")
	}
	assertD087TransitionReport(t, provisional)
	if r.eng.appliedGen != gen || !reflect.DeepEqual(r.eng.desired.DeviceBindings, bindings) || r.eng.lease.LeaseId == oldLease || len(r.plane.last.Bindings) != 0 {
		t.Fatal("transition changed generation or admitted a provisional binding")
	}
	before := r.srv.LeaseRenewals()
	r.srv.SetMemberGrantsReady(true)
	r.clock.Advance(r.eng.provisionalRetryAt - r.clock.Mono() - time.Nanosecond)
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before || r.eng.serving {
		t.Fatal("served or renewed before grant slot")
	}
	r.clock.Advance(time.Nanosecond)
	stepOK(t, r)
	granted, ok := r.srv.LastObservation()
	if !ok || !granted.MemberForwardingEnabled || granted.ReportId == provisional.ReportId || len(granted.ForwardingBlockReasons) != 0 || r.eng.appliedGen != gen || !reflect.DeepEqual(r.eng.desired.DeviceBindings, bindings) || len(r.plane.last.Bindings) != 1 || r.srv.LeaseRenewals() != before+1 {
		t.Fatalf("granted transition missing: %+v", granted)
	}
	t.Logf("fixed generation=%d: provisional report delivered; scheduled grant request after 1s; new serving report delivered before 3600s observation interval", gen)
}

func TestD104_TimingMeasurementRetainsFailedOperation(t *testing.T) {
	r := newRig(t)
	path := filepath.Join(r.dir, "timing.jsonl")
	r.eng.cfg.TestTimingPath = path
	// No external executables: invalid model exits before any kernel call,
	// but the enforcement producer still supplies its failed-operation timing.
	res, err := enforce.Apply(t.Context(), nil, enforce.Model{}, nil)
	if err == nil {
		t.Fatal("invalid model accepted")
	}
	r.eng.measureTiming(res.Evidence, "apply")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("enabled outside test_mode")
	}
	r.eng.cfg.TestMode = true
	r.eng.measureTiming(res.Evidence, "apply")
	r.eng.measureTiming(res.Evidence, "apply")
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var rows [2]Measurement
	for i := range rows {
		if err := dec.Decode(&rows[i]); err != nil {
			t.Fatal(err)
		}
		m := rows[i]
		if m.Sequence != decstr.U64(i+1) || m.BootID != r.eng.bootID || m.Timing == nil || m.Timing.Succeeded || m.Timing.Operation != "apply" || m.Timing.Start <= 0 || m.Timing.End < m.Timing.Start || m.Anchor.MonoBefore < m.Timing.End || m.Anchor.MonoAfter < m.Anchor.MonoBefore {
			t.Fatalf("incomplete failed-operation evidence: %+v", m)
		}
	}
	if rows[0].RunID != rows[1].RunID {
		t.Fatal("run identity changed")
	}
}

func assertD087TransitionReport(t *testing.T, rep api.ObservationReport) {
	t.Helper()
	if rep.MemberForwardingEnabled || rep.ProvisioningState != api.ProvisioningStateReady || rep.SafetyStatus != api.SafetyStatusPassed || rep.DataHealth != api.DataHealthHealthy || !reflect.DeepEqual(rep.ForwardingBlockReasons, []api.ForwardingBlockReason{api.ForwardingBlockReasonAwaitingMemberGrants}) || len(rep.Errors) != 1 || rep.Errors[0].Stage != "lease" || rep.Errors[0].Code != api.ErrorCodeAWAITINGMEMBERGRANTS {
		t.Fatalf("not a qualifying D087 report: %+v", rep)
	}
	for _, name := range []api.NodeCheckName{api.NodeCheckNameLease, api.NodeCheckNameQos, api.NodeCheckNameSourceBinding} {
		c, ok := checkBy(rep.Checks, name)
		code, _ := c.ErrorCode.Get()
		if !ok || c.Status != api.HealthCheckStatusFailed || code != api.ErrorCodeAWAITINGMEMBERGRANTS {
			t.Fatalf("provisional check: %+v", c)
		}
	}
}

func TestD104_TimingMeasurementLiveAndBlockedSnapshots(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	m := kernelClockModel(*r.eng.appliedModel)
	x := &renderedExecutor{model: m}
	plane := SystemPlane{Executor: x}
	res, err := plane.Apply(t.Context(), m, nil)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(r.dir, "operations.jsonl")
	r.eng.cfg.TestMode, r.eng.cfg.TestTimingPath = true, path
	r.eng.measureTiming(res.Evidence, "apply")
	ev, err := plane.VerifyServing(t.Context(), m, res.Plan.Assignments)
	if err != nil {
		t.Fatal(err)
	}
	p := &evidencePlane{memPlane: r.plane, ev: ev}
	r.eng.plane = p
	if !r.eng.verifyServingModel(t.Context(), m, res.Plan.Assignments) {
		t.Fatal("clean live read failed")
	}
	x.foreignTC = true
	ev, err = plane.VerifyServing(t.Context(), m, res.Plan.Assignments)
	if err == nil {
		t.Fatal("foreign filter passed")
	}
	p.ev, p.err = ev, err
	if r.eng.verifyServingModel(t.Context(), m, res.Plan.Assignments) {
		t.Fatal("failed live read passed")
	}
	r.eng.auditBlocked(t.Context())
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rows []Measurement
	for _, line := range bytes.Split(bytes.TrimSpace(raw), []byte("\n")) {
		var row Measurement
		if err := json.Unmarshal(line, &row); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if len(rows) != 4 {
		t.Fatalf("snapshots=%d want Apply + 2 live + blocked", len(rows))
	}
	for i, row := range rows {
		if uint64(row.Sequence) != uint64(i+1) || row.BootID != r.eng.bootID || row.RunID != rows[0].RunID || row.Timing == nil || len(row.Timing.Verifications) != 1 || row.TC != row.Timing.Verifications[0].SHA256 {
			t.Fatalf("snapshot lost provenance: %+v", row)
		}
		if row.Timing.Succeeded != (i < 2) {
			t.Fatalf("failure relabeled: %+v", row)
		}
	}
	if rows[3].Source != "blocked_audit" || rows[0].Timing.Operation != "apply" || rows[1].Timing.Operation != "verify" {
		t.Fatal("operation/source conflated")
	}
	if rows[0].Timing.Start == rows[1].Timing.Start {
		t.Fatal("Apply relabeled as live")
	}
	// This diagnostic stays separate from the HTTP observation contract.
	rep := mustReport(t, r, t.Context())
	wire, _ := json.Marshal(rep)
	if bytes.Contains(wire, []byte("start_ns")) || bytes.Contains(wire, []byte("run_id")) {
		t.Fatal("local timing leaked to observation")
	}
}
