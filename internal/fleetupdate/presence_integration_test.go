//go:build integration

package fleetupdate

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestUPD04_TargetAttentionAndConfirmation(t *testing.T) {
	l := newLab(t)
	i := l.node("amd64", "published", "registered", true)
	if err := l.s.Observe(l.ctx, i, Observation{Sequence: 1, CurrentReleaseID: &l.old, LastGoodReleaseID: &l.old, ObservedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	batch := l.batch(l.new)
	work := l.intent(i)
	assertFlags := func(attention, unconfirmed bool) {
		t.Helper()
		got := l.get(batch.BatchId)
		raw, err := json.Marshal(got)
		if err != nil {
			t.Fatal("required response member missing:", err)
		}
		if err = l.f.Validator.ValidateBody("UpdateBatch", raw); err != nil {
			t.Fatal(err)
		}
		target := got.Targets[0]
		if target.NeedsAttention != attention || target.Unconfirmed != unconfirmed {
			t.Fatalf("attention=%t unconfirmed=%t; want %t %t", target.NeedsAttention, target.Unconfirmed, attention, unconfirmed)
		}
	}
	assertFlags(false, true) // An older heartbeat cannot confirm a newly issued job.
	l.report(i, work, "failed")
	assertFlags(true, true)
	if err := l.s.Observe(l.ctx, i, Observation{Sequence: 2, AppliedUpdateGeneration: work.Intent.Generation, CurrentReleaseID: &l.new, LastGoodReleaseID: &l.new, ObservedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	assertFlags(true, false)
}

// Each mapping uses its own unfinished job; completed evidence remains immutable.
func TestUPD04_TargetAttentionMappings(t *testing.T) {
	for _, tc := range []struct {
		state                         string
		mismatch, watchdog, attention bool
	}{
		{"rolled_back", false, false, true},
		{"already_current", true, false, true},
		{"already_current", false, false, false},
		{"checking", false, true, true},
	} {
		t.Run(tc.state+fmt.Sprint(tc.mismatch), func(t *testing.T) {
			l := newLab(t)
			i := l.node("amd64", "published", "registered", true)
			batch := l.batch(l.new)
			work := l.intent(i)
			if tc.state == "rolled_back" {
				if _, err := l.s.pool.Exec(l.ctx, `UPDATE update_jobs SET state='checking' WHERE id=$1`, work.Intent.JobId); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := l.s.pool.Exec(l.ctx, `UPDATE update_jobs SET state=$2,needs_attention=$3 WHERE id=$1`, work.Intent.JobId, tc.state, tc.watchdog); err != nil {
				t.Fatal(err)
			}
			release := l.new
			if tc.mismatch {
				release = l.old
			}
			if _, err := l.s.pool.Exec(l.ctx, `UPDATE nodes SET current_release_id=$2,resource_version=resource_version+1 WHERE id=$1`, i.NodeID, release); err != nil {
				t.Fatal(err)
			}
			target := l.get(batch.BatchId).Targets[0]
			if target.NeedsAttention != tc.attention {
				t.Fatalf("attention=%t want %t", target.NeedsAttention, tc.attention)
			}
		})
	}
}
