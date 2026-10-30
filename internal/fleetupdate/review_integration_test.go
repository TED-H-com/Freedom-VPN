//go:build integration

package fleetupdate

import (
	"errors"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
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
			// r11: a heartbeat can seed current, but only a result can advance
			// last-good. Even a same-generation success must write that evidence;
			// a result below the restore high-water still cannot move either.
			if ack.AppliedToCurrentObservation != (observed <= 1) || ack.ObservedUpdateGeneration != max(observed, 1) {
				t.Fatal(ack)
			}
			lastGood := l.new
			if observed > w.Intent.Generation {
				lastGood = l.old
			}
			view, err := l.s.NodeVersionView(l.ctx, i.NodeID)
			if err != nil || view.CurrentReleaseID == nil || *view.CurrentReleaseID != l.new || view.LastGoodReleaseID == nil || *view.LastGoodReleaseID != lastGood {
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

func TestAT26_AppliedPolicyEvidenceIsImmutable(t *testing.T) {
	l := newLab(t)
	i := l.node("amd64", "published", "registered", true)
	l.batch(l.new)
	w := l.intent(i)
	generation := decstr.MustParse("9007199254740993")
	oldPolicy, newPolicy := uuid.New(), uuid.New()
	for offset, ref := range []uuid.UUID{oldPolicy, newPolicy} {
		if _, err := l.s.pool.Exec(l.ctx, `INSERT INTO desired_states(node_id,incarnation_id,generation,policy_refs,binding_revision,release_id,config_hash,schema_version,signed_at) VALUES($1,$2,$3::numeric,$4,0,$5,$6,1,clock_timestamp())`, i.NodeID, i.IncarnationID, (generation + decstr.U64(offset)).String(), []uuid.UUID{ref}, l.new, make([]byte, 32)); err != nil {
			t.Fatal(err)
		}
	}
	r := result(w, "succeeded")
	r.PolicyGeneration = generation
	if _, err := l.s.SubmitResult(l.ctx, i, payload(t, r)); err != nil {
		t.Fatal(err)
	}
	var stored string
	var refs []uuid.UUID
	if err := l.s.pool.QueryRow(l.ctx, `SELECT policy_generation::text,policy_refs FROM update_results WHERE result_id=$1`, r.ResultId).Scan(&stored, &refs); err != nil || stored != generation.String() || len(refs) != 1 || refs[0] != oldPolicy {
		t.Fatal(stored, refs, err)
	}
	r.PolicyGeneration++
	if _, err := l.s.SubmitResult(l.ctx, i, payload(t, r)); !errors.Is(err, apierr.New(apierr.AckConflict, "")) {
		t.Fatal(err)
	}
	r.ResultId = uuid.New()
	if ack, err := l.s.SubmitResult(l.ctx, i, payload(t, r)); err != nil || !ack.AcceptedAsHistory {
		t.Fatal(ack, err)
	}
	target := nullableValue(w.Intent.TargetId)
	evidence, err := l.s.Evidence(l.ctx, target)
	if err != nil || evidence.PolicyGeneration == nil || *evidence.PolicyGeneration != generation || len(evidence.PolicyRefs) != 1 || evidence.PolicyRefs[0] != oldPolicy {
		t.Fatal(evidence, err)
	}
	// Late history keeps its own applied config without rewriting terminal evidence.
	if err := l.s.pool.QueryRow(l.ctx, `SELECT policy_generation::text,policy_refs FROM update_results WHERE result_id=$1`, r.ResultId).Scan(&stored, &refs); err != nil || stored != r.PolicyGeneration.String() || len(refs) != 1 || refs[0] != newPolicy {
		t.Fatal(stored, refs, err)
	}
}
