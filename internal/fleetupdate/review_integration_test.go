//go:build integration

package fleetupdate

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// §13.8 / AT23, AT26: receipt time cannot order applied release generations.
func TestAT23_ResultAfterRoutineHeartbeat(t *testing.T) {
	for _, observed := range []decstr.U64{0, 1, 2} {
		t.Run(observed.String(), func(t *testing.T) {
			l := newLab(t)
			i := l.node("amd64", "published", "registered", true)
			l.batch(l.new)
			w := l.intent(i)
			if w.Intent.Generation != 1 {
				t.Fatal(w.Intent.Generation)
			}
			current := l.old
			if observed >= w.Intent.Generation {
				current = l.new
			}
			if err := l.s.Observe(l.ctx, i, Observation{Sequence: 2, AppliedUpdateGeneration: observed, CurrentReleaseID: &current, LastGoodReleaseID: &current, ObservedAt: time.Now().UTC()}); err != nil {
				t.Fatal(err)
			}
			var receipt time.Time
			if err := l.s.pool.QueryRow(l.ctx, `SELECT received_at FROM fleet_node_state WHERE node_id=$1`, i.NodeID).Scan(&receipt); err != nil || !receipt.After(w.Intent.IssuedAt) {
				t.Fatal(receipt, err)
			}
			ack := l.report(i, w, "succeeded")
			if ack.AppliedToCurrentObservation != (observed < 1) || ack.ObservedUpdateGeneration != max(observed, 1) {
				t.Fatal(ack)
			}
			view, err := l.s.NodeVersionView(l.ctx, i.NodeID)
			if err != nil || view.CurrentReleaseID == nil || *view.CurrentReleaseID != l.new || view.LastGoodReleaseID == nil || *view.LastGoodReleaseID != l.new {
				t.Fatal(view, err)
			}
		})
	}
}

func TestAT24_IntentRefreshAndTargetNames(t *testing.T) {
	l := newLab(t)
	i := l.node("amd64", "published", "registered", true)
	b := l.batch(l.new)
	first := l.intent(i)
	second := l.intent(i)
	var issued, valid time.Time
	if err := l.s.pool.QueryRow(l.ctx, `SELECT issued_at,authorization_valid_until FROM update_jobs WHERE id=$1`, first.Intent.JobId).Scan(&issued, &valid); err != nil {
		t.Fatal(err)
	}
	if !first.Intent.IssuedAt.Equal(issued) || !second.Intent.IssuedAt.Equal(issued) || !second.Intent.AuthorizationValidUntil.Equal(valid) {
		t.Fatal(first.Intent, second.Intent, issued, valid)
	}
	got := l.get(b.BatchId)
	if name, err := got.Targets[0].NodeName.Get(); err != nil || name != "TEST ONLY node" {
		t.Fatal(name, err)
	}
	// An enrolled record can legitimately have no exit/display name yet.
	if _, err := l.s.pool.Exec(l.ctx, `INSERT INTO nodes(lifecycle_state) VALUES('enrolling')`); err != nil {
		t.Fatal(err)
	}
	next := l.batch(l.new)
	named, unnamed := 0, 0
	for _, target := range l.get(next.BatchId).Targets {
		if target.NodeName.IsNull() {
			unnamed++
		} else {
			named++
		}
	}
	if named != 1 || unnamed != 1 {
		t.Fatal(named, unnamed)
	}
}
