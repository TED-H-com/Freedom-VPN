package members

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/agent"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func readF20JSON(t *testing.T, name string, out any) {
	t.Helper()
	f, err := os.Open("../../../../docs/e2e/at/members/history/follow-up-18-run2/" + name)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	if len(name) > 3 && name[len(name)-3:] == ".gz" {
		z, e := gzip.NewReader(f)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { _ = z.Close() })
		err = json.NewDecoder(z).Decode(out)
	} else {
		err = json.NewDecoder(f).Decode(out)
	}
	if err != nil {
		t.Fatal(err)
	}
}
func f20Restart(t *testing.T) (time.Time, []restartLease, []restartObservation) {
	t.Helper()
	var doc struct {
		OriginalSteps []struct {
			Actual string `json:"actual"`
		} `json:"original_steps"`
	}
	readF20JSON(t, "F18-AT09-restart.json", &doc)
	var cap struct {
		Capture struct {
			Started      time.Time            `json:"started_at"`
			Observations []restartObservation `json:"observations"`
		} `json:"capture"`
	}
	var timeline struct {
		Leases []restartLease `json:"leases"`
	}
	if json.Unmarshal([]byte(doc.OriginalSteps[4].Actual), &cap) != nil || json.Unmarshal([]byte(doc.OriginalSteps[6].Actual), &timeline) != nil {
		t.Fatal("restart record")
	}
	// The old timeline omitted lease IDs. Join its unique expiry/generation to
	// the report's exact lease ID (Postgres stored microseconds, report has ns).
	// This adapter is only for immutable old evidence; live SQL now includes IDs.
	for i := range timeline.Leases {
		l := &timeline.Leases[i]
		for _, o := range cap.Capture.Observations {
			expiry, e := o.Report.Lease.ExpiresAt.Get()
			gen, _ := o.Report.AppliedConfigGeneration.Get()
			if e == nil && expiry.Truncate(time.Microsecond).Equal(l.ExpiresAt.Truncate(time.Microsecond)) && gen.String() == l.Generation {
				id, _ := o.Report.Lease.LeaseId.Get()
				if l.ID != uuid.Nil && l.ID != id {
					t.Fatal("ambiguous archived lease")
				}
				l.ID = id
			}
		}
	}
	return cap.Capture.Started, timeline.Leases, cap.Capture.Observations
}
func TestF20_AT09RecordedRestartAndForeignFilter(t *testing.T) {
	at, leases, obs := f20Restart(t)
	got := assessRestart(at, "66", leases, obs)
	if !got.SequenceOK || got.D067Status != "blocked" || got.GrantlessLeases != 0 || got.PopulatedObservations != 2 || got.Sequence != "populated from first renewal" {
		t.Fatalf("offline M-2 proof: %+v", got)
	}
	if got.NoForeignFilter {
		t.Fatal("H-1 false foreign-filter report passed")
	}
	t.Log("offline proof only: D-067 blocked/not exercised; restart lease sequence accepted; separate no-TC_FOREIGN_FILTER subcheck FAILS; historical run remains failed")
}
func TestF20_AT09LeaseJoinAndSafetyFailures(t *testing.T) {
	at, leases, obs := f20Restart(t)
	// First archived report has only a real tunnel-probe safety failure.
	clean := []restartObservation{obs[0]}
	if got := assessRestart(at, "66", leases, clean); !got.SequenceOK || !got.NoForeignFilter {
		t.Fatal("clean populated safety report", got)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*api.ObservationReport)
	}{
		{"unknown held lease", func(r *api.ObservationReport) { r.Lease.LeaseId.Set(uuid.New()) }},
		{"probe cannot override lease", func(r *api.ObservationReport) {
			for i := range r.Checks {
				if r.Checks[i].Name == api.NodeCheckNameLease {
					r.Checks[i].Status = api.HealthCheckStatusFailed
					r.Checks[i].ErrorCode.Set(api.ErrorCodeHEALTHCHECKFAILED)
				}
			}
		}},
		{"unjustified safety check", func(r *api.ObservationReport) { r.Errors = nil }},
		{"invented lease error", func(r *api.ObservationReport) {
			r.Errors = append(r.Errors, api.NodeError{Stage: "lease", Code: api.ErrorCodeAWAITINGMEMBERGRANTS})
		}},
		{"awaiting under populated", func(r *api.ObservationReport) {
			r.ForwardingBlockReasons = append(r.ForwardingBlockReasons, api.ForwardingBlockReasonAwaitingMemberGrants)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, _ := json.Marshal(clean)
			var copy []restartObservation
			_ = json.Unmarshal(raw, &copy)
			tc.mutate(&copy[0].Report)
			if assessRestart(at, "66", leases, copy).SequenceOK {
				t.Fatal("invalid populated report passed")
			}
		})
	}
	// A newer central renewal at receipt time must not change the held lease.
	old := leases[len(leases)-1]
	old.ID = uuid.New()
	old.IssuedAt = obs[0].Report.ObservedAt.Add(time.Millisecond)
	old.MemberGrants = 0
	got := assessRestart(at, "66", append(leases, old), clean)
	if got.PopulatedObservations != 1 || got.ProvisionalObservations != 0 {
		t.Fatal("joined latest lease instead of held lease", got)
	}
	// Pre-restart reports/leases cannot satisfy D-067 or poison the new slice.
	pre := obs[1]
	pre.ReceivedAt = at.Add(-time.Second)
	pre.Report.ObservedAt = pre.ReceivedAt
	got = assessRestart(at, "66", leases, append(clean, pre))
	if !got.NoForeignFilter || got.D067Status != "blocked" {
		t.Fatal("pre-restart evidence leaked", got)
	}
}
func f20Recovery(t *testing.T) ([]json.RawMessage, json.RawMessage, json.RawMessage) {
	t.Helper()
	var trace struct {
		Records []json.RawMessage `json:"records"`
	}
	var cap struct {
		Observation json.RawMessage `json:"observation"`
		Sender      json.RawMessage `json:"sender"`
	}
	readF20JSON(t, "AT02-enforcement-timing.json", &trace)
	readF20JSON(t, "F18-AT02-recovery.json.gz", &cap)
	return trace.Records, cap.Observation, cap.Sender
}
func TestF20_AT02RecordedBracketAccounting(t *testing.T) {
	trace, obs, sent := f20Recovery(t)
	out, ok, failed := siblingRecovery(trace, obs, sent)
	if !ok || !failed || out["measurement_valid"] != true || out["bracket_accounting_ok"] != true || out["sibling_recovery_ok"] != true || out["failed_apply_bracketed"] != 3 || out["lost_admitted_packets"] != 1242 || out["not_observed_at_node_ingress_interface_down"] != 983 || out["losses_outside_baseline_or_completed_guard"] != 0 {
		t.Fatalf("offline proof: ok=%v failed=%v valid=%v accounting=%v recovery=%v failed linked=%v losses=%v interface down=%v outside=%v unplaced=%v", ok, failed, out["measurement_valid"], out["bracket_accounting_ok"], out["sibling_recovery_ok"], out["failed_apply_bracketed"], out["lost_admitted_packets"], out["not_observed_at_node_ingress_interface_down"], out["losses_outside_baseline_or_completed_guard"], out["unplaced_packets"])
	}
	total := 0
	for _, b := range out["brackets"].([]recoveryBracket) {
		total += b.Losses
		t.Logf("bracket baseline=%s recovery=%s duration=%s losses=%d forwarding=%t", b.Baseline, b.Recovery, b.Duration, b.Losses, b.Recovered)
	}
	if total != 1242 {
		t.Fatalf("bracket total %d duplicates or excludes losses", total)
	}
	found := false
	for _, s := range out["closed_recovery_steps"].([]recoveryStep) {
		if s.Sequence == 584 {
			found = true
			if s.Until <= s.End || s.Losses != 249 || s.InstallLosses != 10 || s.DenyAllLosses != 239 {
				t.Fatalf("deny-all interval duration=%s losses=%d install losses=%d hold losses=%d", s.Duration, s.Losses, s.InstallLosses, s.DenyAllLosses)
			}
			t.Logf("Apply 584 install=%s (%d losses), deny-all hold=%s (%d losses), total=%s (%d losses); M-1 remains reported", s.InstallDuration, s.InstallLosses, s.DenyAllDuration, s.DenyAllLosses, s.Duration, s.Losses)
		}
	}
	if !found {
		t.Fatal("closedRecovery Apply 584 omitted")
	}
	t.Log("offline proof only: 190 bracket accounting and 191 sibling recovery pass the new predicate; all 1,242 losses counted; no historical verdict changed")
}
func TestF20_AT02IncompleteOrUnrecoveredCannotPass(t *testing.T) {
	trace, obs, sent := f20Recovery(t)
	for _, mode := range []string{"capture drops", "unplaced send", "loss outside bracket", "missing post-close egress", "unverified serving Apply", "missing heartbeat", "missing populated lease"} {
		t.Run(mode, func(t *testing.T) {
			rows := append([]json.RawMessage{}, trace...)
			var cap siblingObservation
			_ = json.Unmarshal(obs, &cap)
			var sender struct {
				Boot     string        `json:"boot_id"`
				Clock    string        `json:"clock"`
				Attempts int           `json:"attempts"`
				Errors   int           `json:"send_errors"`
				Sends    []sendAttempt `json:"sends"`
			}
			_ = json.Unmarshal(sent, &sender)
			switch mode {
			case "capture drops":
				cap.Stats.Drops = 1
			case "unplaced send":
				sender.Sends[4984].Before = cap.Start - 1
			case "loss outside bracket":
				for i, p := range cap.Packets {
					if p.Sequence == 0 && p.Interface == "fvup0" {
						cap.Packets = append(cap.Packets[:i], cap.Packets[i+1:]...)
						break
					}
				}
			case "missing post-close egress":
				// Remove all sibling traffic following the last close, including sends:
				// measurement remains complete, but recovery must independently fail.
				cutoff := time.Duration(63887650000000)
				packets := []siblingPacket{}
				for _, p := range cap.Packets {
					if p.At < cutoff {
						packets = append(packets, p)
					}
				}
				cap.Packets = packets
				sends := []sendAttempt{}
				for _, a := range sender.Sends {
					if a.After < cutoff {
						sends = append(sends, a)
					}
				}
				sender.Sends = sends
				sender.Attempts = len(sends)
				cap.End = cutoff
			default:
				for i, raw := range rows {
					var m agent.Measurement
					_ = json.Unmarshal(raw, &m)
					if m.Heartbeat != nil && mode == "missing heartbeat" {
						m.Heartbeat.Written = false
					}
					if m.Lease != nil && mode == "missing populated lease" {
						m.Lease.Members = 0
					}
					if m.Timing != nil && m.Timing.Operation == "apply" && mode == "unverified serving Apply" {
						m.Timing.Verified = false
					}
					rows[i], _ = json.Marshal(m)
				}
			}
			ob, _ := json.Marshal(cap)
			se, _ := json.Marshal(sender)
			out, ok, _ := siblingRecovery(rows, ob, se)
			if ok {
				t.Fatal("incomplete/unsafe result passed")
			}
			if mode == "loss outside bracket" && (out["measurement_valid"] != true || out["bracket_accounting_ok"] != false || out["sibling_recovery_ok"] != true) {
				t.Fatal("190 and 191 no longer independent")
			}
		})
	}
}
