package isolation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestD144_ScheduledExpectedBlocksDoNotHideFailures(t *testing.T) {
	p := Positive{DeviceID: "TEST ONLY", Counted: true, Before: "1", After: "2"}
	entries := ExpectedBlocks()
	if len(entries) != 2 {
		t.Fatal("expected two evidenced operator rows")
	}
	for _, e := range entries {
		r := Record{ID: e.Case, Status: StatusBlocked, Checks: []Check{{Name: e.Row, Status: StatusBlocked}}}
		if !ScheduledCaseAcceptable(r) || r.Status != StatusBlocked || r.Checks[0].Status != StatusBlocked {
			t.Fatal("mapping changed acceptance")
		}
		r.Checks[0].Status = StatusFail
		if ScheduledCaseAcceptable(r) {
			t.Fatal("listed failure hidden")
		}
		r.Checks[0].Status = StatusPass
		r.Checks[0].Positive = p
		r.Status = StatusPass
		if !ScheduledCaseAcceptable(r) {
			t.Fatal("unblocked row rejected")
		}
		r.Checks = append(r.Checks, Check{Name: "unexpected", Status: StatusBlocked})
		if ScheduledCaseAcceptable(r) {
			t.Fatal("unknown block hidden")
		}
	}
	if ScheduledCaseAcceptable(Record{ID: "AT13", Status: StatusBlocked, Checks: []Check{{Name: "short whole-node lease expiry", Status: StatusBlocked}}}) {
		t.Fatal("unexplained runner block allowed")
	}
}

func TestAT13_AT10_Nightly41640196805ControlRestorationSequence(t *testing.T) {
	const runID = "775bc692-e6ef-4490-8056-ba6cfa818b69"
	mustTime := func(raw string) time.Time {
		t.Helper()
		ts, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			t.Fatal(err)
		}
		return ts
	}
	// Recorded timestamps from scheduled run 41640196805:
	// - deadline() returned and spawned `docker start` at 13:19:22.5917021Z.
	// - unawaited wholeNodeExpiry() failed lab.login at 13:19:22.9982753Z and
	//   received nginx 502 Bad Gateway on fleetAudits while fvpn-control booted.
	// - fvpn-control served its first request at 13:19:30.47655856Z, issued gen
	//   33 at 13:19:33.160249011Z, and drained 6 queued observations through
	//   13:19:40.000917Z.
	// - report 01a1a190-bebb-7497-989f-36bfe8b78a8d (observed 13:19:40.218635399Z,
	//   accepted 13:19:40.222691621Z) was the first post-restoration current
	//   observation with a drained retry queue, 17.21s before foreignRedirect().
	controlStart := mustTime("2026-11-03T13:19:22.5917021Z")
	samples := []struct {
		name   string
		sample ControlReadinessSample
		want   bool
	}{
		{
			name: "nginx_502_at_unawaited_whole_node_expiry_entry",
			sample: ControlReadinessSample{
				LoginOK:           false,
				FleetConfigStatus: 502,
				QueueAvailable:    true,
				QueuePending:      3,
				ObservedAt:        mustTime("2026-11-03T13:18:50.193055318Z"),
				AppliedGeneration: "32",
			},
			want: false,
		},
		{
			name: "queued_outage_report_01a1a190_0d14_before_control_start",
			sample: ControlReadinessSample{
				LoginOK:                 true,
				FleetConfigStatus:       200,
				QueueAvailable:          true,
				QueuePending:            5,
				ObservedAt:              mustTime("2026-11-03T13:18:54.740587562Z"),
				AppliedGeneration:       "32",
				MemberForwardingEnabled: true,
			},
			want: false,
		},
		{
			name: "post_start_report_01a1a190_b7b7_while_retry_queue_not_drained",
			sample: ControlReadinessSample{
				LoginOK:                 true,
				FleetConfigStatus:       200,
				QueueAvailable:          true,
				QueuePending:            1,
				ObservedAt:              mustTime("2026-11-03T13:19:38.423577389Z"),
				AppliedGeneration:       "33",
				MemberForwardingEnabled: true,
			},
			want: false,
		},
		{
			name: "first_current_drained_report_01a1a190_bebb_at_13_19_40",
			sample: ControlReadinessSample{
				LoginOK:                 true,
				FleetConfigStatus:       200,
				QueueAvailable:          true,
				QueuePending:            0,
				ObservedAt:              mustTime("2026-11-03T13:19:40.218635399Z"),
				AppliedGeneration:       "33",
				MemberForwardingEnabled: true,
			},
			want: true,
		},
	}
	for _, tc := range samples {
		if got := SampleControlReadiness(controlStart, "33", tc.sample); got != tc.want {
			t.Fatalf("%s: got %t, want %t", tc.name, got, tc.want)
		}
	}

	dir := t.TempDir()
	// Pre-fix behavior: when deadline() did not call waitControl() and
	// wholeNodeExpiry() returned before stopped=true, control-observation-readiness.json
	// was absent and a non-waiting check (wait == nil) blocked AT10.
	if ControlObservationReady(dir, runID, controlStart, nil) {
		t.Fatal("missing control-observation-readiness.json treated as ready without wait")
	}

	writeReadiness := func(v map[string]any) {
		t.Helper()
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if err := WriteExtra(dir, "control-observation-readiness.json", string(raw)); err != nil {
			t.Fatal(err)
		}
	}

	// Inherited readiness from a different run_id (as in 41456092596) or from an
	// earlier control restart before controlStart must not satisfy readiness without wait.
	writeReadiness(map[string]any{
		"run_id":     "83b457ef-ea16-490b-9923-1fea0df86b36",
		"started_at": controlStart.Add(time.Second),
		"ready":      true,
	})
	if ControlObservationReady(dir, runID, controlStart, nil) {
		t.Fatal("foreign run_id readiness accepted without wait")
	}
	writeReadiness(map[string]any{
		"run_id":     runID,
		"started_at": controlStart.Add(-time.Second),
		"ready":      true,
	})
	if ControlObservationReady(dir, runID, controlStart, nil) {
		t.Fatal("pre-restart readiness accepted without wait")
	}

	// Fixed behavior: waiting establishes and persists current-run readiness at
	// or after controlStart, and subsequent checks reuse that record without
	// re-polling until another control restart occurs.
	waited := 0
	waitFn := func() bool {
		waited++
		for _, tc := range samples {
			if SampleControlReadiness(controlStart, "33", tc.sample) {
				writeReadiness(map[string]any{
					"run_id":       runID,
					"started_at":   controlStart,
					"completed_at": mustTime("2026-11-03T13:19:40.227146Z"),
					"ready":        true,
				})
				return true
			}
		}
		return false
	}
	if !ControlObservationReady(dir, runID, controlStart, waitFn) || waited != 1 {
		t.Fatalf("expected wait to establish readiness once, waited=%d", waited)
	}
	if !ControlObservationReady(dir, runID, controlStart, func() bool {
		t.Fatal("unexpected second wait when current restart is already ready")
		return false
	}) {
		t.Fatal("persisted current readiness not reused")
	}
	if _, err := os.Stat(filepath.Join(dir, "control-observation-readiness.json")); err != nil {
		t.Fatal(err)
	}
}
