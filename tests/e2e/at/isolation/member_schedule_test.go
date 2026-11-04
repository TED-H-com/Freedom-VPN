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
		if status != StatusPass || p.Deadline.Format(time.RFC3339) != "2026-11-04T03:08:48Z" || evidence != "2026-11-04T03:08:36.030226Z" {
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
		for name, offsets := range map[string][]time.Duration{
			"scheduled":       {0, 30 * time.Second, 60 * time.Second},
			"earliest jitter": {0, 27 * time.Second, 54 * time.Second},
			"latest jitter":   {0, 33 * time.Second, 66 * time.Second},
			"late step":       {0, 34 * time.Second, 68 * time.Second},
			"early nominal":   {0, 10500 * time.Millisecond, 40500 * time.Millisecond},
			"early min":       {0, 10500 * time.Millisecond, 37500 * time.Millisecond},
			"early max":       {0, 10500 * time.Millisecond, 43500 * time.Millisecond},
			"early late step": {0, 10500 * time.Millisecond, 44500 * time.Millisecond},
		} {
			t.Run(name+"/"+fraction.String(), func(t *testing.T) {
				var samples []renewalSample
				for _, offset := range offsets {
					s := renewalSample{At: anchor.Add(offset + 2*time.Second)}
					s.Lease.Issued, s.Lease.Expires, s.Lease.Accepted = anchor.Add(offset), anchor.Add(offset+time.Minute), true
					samples = append(samples, s)
				}
				if status, detail := replayMemberWindow(plan.Deadline, samples); status != StatusPass {
					t.Fatal(status, detail)
				}
			})
		}
	}
}

func TestAT13_MemberScheduleMissingWindowStillBlocks(t *testing.T) {
	a := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	p, _ := PlanMemberExpiry(a, a.Add(time.Minute), a, true)
	var samples []renewalSample
	for _, offset := range []time.Duration{0, 20 * time.Second, 51 * time.Second} {
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
