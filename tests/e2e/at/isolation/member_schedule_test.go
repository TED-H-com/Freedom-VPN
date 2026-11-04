package isolation

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

type renewalSample struct {
	At    time.Time `json:"at"`
	Lease struct {
		Issued   time.Time `json:"issued_at"`
		Expires  time.Time `json:"expires_at"`
		Accepted bool      `json:"accepted_serving"`
	} `json:"lease"`
}

// This replays timing only, not grants or signed documents. The live driver
// still requires the original matching member/service deadlines and serving ACK.
func replayMemberWindow(end time.Time, samples []renewalSample) (string, string) {
	for _, s := range samples {
		if s.Lease.Accepted && s.Lease.Expires.After(end.Add(25*time.Second)) && s.At.Before(end.Add(-5*time.Second)) {
			return StatusPass, s.Lease.Issued.Format(time.RFC3339Nano)
		}
	}
	return StatusBlocked, "finite-member-renewal-wait.json: no accepted node lease extends more than 25s beyond the member deadline before the D-5s cutoff"
}

func TestAT13_MemberScheduleFollowup21RenewalReplay(t *testing.T) {
	const path = "../../../../docs/e2e/at/isolation/history/follow-up-21-run/finite-member-renewal-wait.json"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rec struct {
		RunID    string          `json:"run_id"`
		Deadline time.Time       `json:"member_deadline"`
		Samples  []renewalSample `json:"samples"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatal(err)
	}
	if rec.RunID != "7599576e-339c-49c9-8bf5-0f710818cbda" || len(rec.Samples) != 152 {
		t.Fatal("wrong retained run")
	}
	if status, _ := replayMemberWindow(rec.Deadline, rec.Samples); status != StatusBlocked {
		t.Fatal("original schedule must remain blocked")
	}
	for i, s := range rec.Samples {
		p, ok := PlanMemberExpiry(s.Lease.Issued, s.Lease.Expires, s.At, s.Lease.Accepted)
		if !ok {
			continue
		}
		status, evidence := replayMemberWindow(p.Deadline, rec.Samples[i+1:])
		if status != StatusPass || p.Deadline.Format(time.RFC3339) != "2026-11-04T03:08:37Z" || evidence != "2026-11-04T03:08:03.470797Z" {
			t.Fatalf("schedule=%+v status=%s evidence=%s", p, status, evidence)
		}
		t.Logf("original=blocked; timing replay=pass; anchor=%s D=%s qualifying issue=%s (includes the actual 10.460446s early renewal); no signed record edited", p.Issued, p.Deadline, evidence)
		return
	}
	t.Fatal("no fresh anchor in original samples")
}

func TestAT13_MemberScheduleJitterAndEarlyRenewal(t *testing.T) {
	for _, fraction := range []time.Duration{0, 999 * time.Millisecond} {
		anchor := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC).Add(fraction)
		plan, ok := PlanMemberExpiry(anchor, anchor.Add(time.Minute), anchor.Add(time.Second), true)
		if !ok {
			t.Fatal("valid anchor rejected")
		}
		// Entire claimed 10..34s range, including fractional boundaries; setup
		// may postpone the first sample until A+34s, acceptance by issue+2s.
		for offset := 10 * time.Second; offset <= 34*time.Second; offset += time.Millisecond {
			s := renewalSample{At: anchor.Add(max(34*time.Second, offset+2*time.Second))}
			s.Lease.Issued, s.Lease.Expires, s.Lease.Accepted = anchor.Add(offset), anchor.Add(offset+time.Minute), true
			if status, detail := replayMemberWindow(plan.Deadline, []renewalSample{s}); status != StatusPass {
				t.Fatal(fraction, offset, status, detail)
			}
		}
	}
}

func TestAT13_MemberScheduleMissingWindowStillBlocks(t *testing.T) {
	a := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	p, _ := PlanMemberExpiry(a, a.Add(time.Minute), a, true)
	var samples []renewalSample
	for _, offset := range []time.Duration{0, 9 * time.Second, 40 * time.Second} {
		s := renewalSample{At: a.Add(offset)}
		s.Lease.Issued, s.Lease.Expires, s.Lease.Accepted = a.Add(offset), a.Add(offset+time.Minute), true
		samples = append(samples, s)
	}
	if status, detail := replayMemberWindow(p.Deadline, samples); status != StatusBlocked || !strings.Contains(detail, "finite-member-renewal-wait.json") {
		t.Fatal(status, detail)
	} else {
		t.Log(status, detail)
	}
	for name, v := range map[string]struct {
		issued, expires, observed time.Time
		accepted                  bool
	}{
		"stale":        {a, a.Add(time.Minute), a.Add(6 * time.Second), true},
		"future":       {a, a.Add(time.Minute), a.Add(-time.Second), true},
		"wrong window": {a, a.Add(59 * time.Second), a, true},
		"unaccepted":   {a, a.Add(time.Minute), a, false},
	} {
		if _, ok := PlanMemberExpiry(v.issued, v.expires, v.observed, v.accepted); ok {
			t.Fatal(name, "anchor accepted")
		}
	}
}

func TestAT13_MemberScheduleUTCReceipt(t *testing.T) {
	// Database timestamps can carry the session location; receipts use UTC.
	a := time.Date(2026, 11, 4, 1, 18, 26, 165788000, time.FixedZone("TEST ONLY", -5*60*60))
	p, ok := PlanMemberExpiry(a, a.Add(time.Minute), a.Add(time.Second), true)
	if !ok || !p.Issued.Equal(a) || p.Deadline.Format(time.RFC3339) != "2026-11-04T06:19:10Z" {
		t.Fatalf("instant or schedule changed: %+v", p)
	}
	for _, stamp := range []time.Time{p.Issued, p.Expires, p.Observed, p.NextExpected, p.Deadline} {
		if stamp.Location() != time.UTC {
			t.Fatal("receipt is not UTC", stamp)
		}
	}
}

func TestAT13_MemberScheduleRunnerAndFollowup23Replay(t *testing.T) {
	for _, tc := range []struct{ dir, run, oldStatus, deadline, issue string }{
		{"follow-up-24-nightly-41836274140", "8ee5129e-6ae1-41f4-8486-f1d92bec8b34", StatusBlocked, "2026-11-04T13:34:05Z", "2026-11-04T13:33:40.265996Z"},
		{"follow-up-23", "328ab9e0-7ff4-495a-81c1-410ae8a7d876", StatusPass, "2026-11-04T10:45:36Z", "2026-11-04T10:45:10.111815Z"},
	} {
		t.Run(tc.dir, func(t *testing.T) {
			read := func(name string, dst any) {
				raw, err := os.ReadFile("../../../../docs/e2e/at/isolation/history/" + tc.dir + "/" + name)
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(raw, dst); err != nil {
					t.Fatal(err)
				}
			}
			var anchor struct {
				RunID    string               `json:"run_id"`
				Schedule MemberExpirySchedule `json:"schedule"`
				Anchored bool                 `json:"anchored"`
			}
			var rec struct {
				RunID    string          `json:"run_id"`
				Deadline time.Time       `json:"member_deadline"`
				Samples  []renewalSample `json:"samples"`
			}
			read("finite-member-schedule.json", &anchor)
			read("finite-member-renewal-wait.json", &rec)
			var signed struct {
				Samples []struct {
					Lease struct {
						Members []struct {
							End        time.Time `json:"lease_end"`
							ServiceEnd time.Time `json:"effective_service_end"`
						} `json:"members"`
					} `json:"lease"`
				} `json:"samples"`
			}
			read("finite-member-renewal-wait.json", &signed)
			for _, s := range signed.Samples {
				matches := 0
				for _, m := range s.Lease.Members {
					if m.End.Equal(rec.Deadline) && m.ServiceEnd.Equal(rec.Deadline) {
						matches++
					}
				}
				if matches != 1 {
					t.Fatal("original signed member/service deadlines must both equal original D")
				}
			}
			if anchor.RunID != tc.run || rec.RunID != tc.run {
				t.Fatal("different retained run")
			}
			if status, _ := replayMemberWindow(rec.Deadline, rec.Samples); status != tc.oldStatus {
				t.Fatal("historical result changed", status)
			}
			p, ok := PlanMemberExpiry(anchor.Schedule.Issued, anchor.Schedule.Expires, anchor.Schedule.Observed, anchor.Anchored)
			if !ok || p.Deadline.Format(time.RFC3339) != tc.deadline {
				t.Fatal(p, ok)
			}
			// Counterfactual timing only: the new grant would be created with
			// the planned D. Original signed documents and verdicts stay intact.
			status, issue := replayMemberWindow(p.Deadline, rec.Samples)
			if status != StatusPass || issue != tc.issue {
				t.Fatal(status, issue)
			}
			t.Logf("original=%s; new timing=pass; A=%s D=%s first sample=%s qualifying issue=%s", tc.oldStatus, p.Issued, p.Deadline, rec.Samples[0].At, issue)
		})
	}
}

func TestAT13_MemberScheduleStrictWaitBoundaries(t *testing.T) {
	a := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	p, _ := PlanMemberExpiry(a, a.Add(time.Minute), a, true)
	s := renewalSample{At: a.Add(34 * time.Second)}
	s.Lease.Issued, s.Lease.Expires, s.Lease.Accepted = a.Add(10*time.Second), a.Add(70*time.Second), true
	for _, kind := range []string{"unaccepted", "extension equality", "cutoff equality", "late setup"} {
		v := s
		switch kind {
		case "unaccepted":
			v.Lease.Accepted = false
		case "extension equality":
			v.Lease.Expires = p.Deadline.Add(25 * time.Second)
		case "cutoff equality":
			v.At = p.Deadline.Add(-5 * time.Second)
		case "late setup":
			v.At = p.Deadline
		}
		if status, _ := replayMemberWindow(p.Deadline, []renewalSample{v}); status != StatusBlocked {
			t.Fatal(kind, status)
		}
	}
}
