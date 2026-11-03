package isolation

import "testing"

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
