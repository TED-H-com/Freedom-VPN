//go:build integration

package fleetupdate

import (
	"encoding/json"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func TestUPD05_TargetPolicyEvidenceUnknownZeroEmptyAndHistory(t *testing.T) {
	for _, tc := range []struct {
		name       string
		generation decstr.U64
		history    bool
	}{
		{"zero with unknown history", 0, false},
		{"reported with unknown history", 9, false},
		{"known empty references", 12, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := newLab(t)
			i := l.node("amd64", "published", "registered", true)
			unknown := l.node("arm64", "unpublished", "enrolling", false)
			batch := l.batch(l.new)
			work := l.intent(i)
			if tc.history {
				if _, err := l.s.pool.Exec(l.ctx, `INSERT INTO desired_states(node_id,incarnation_id,generation,policy_refs,binding_revision,release_id,config_hash,schema_version,signed_at) VALUES($1,$2,$3::numeric,'{}',0,$4,$5,1,clock_timestamp())`, i.NodeID, i.IncarnationID, tc.generation.String(), l.new, make([]byte, 32)); err != nil {
					t.Fatal(err)
				}
			}
			r := result(work, "succeeded")
			r.PolicyGeneration = tc.generation
			if _, err := l.s.SubmitResult(l.ctx, i, payload(t, r)); err != nil {
				t.Fatal(err)
			}
			// Late history and a newer policy must never overwrite accepted evidence.
			r.ResultId = uuid.New()
			r.PolicyGeneration++
			if ack, err := l.s.SubmitResult(l.ctx, i, payload(t, r)); err != nil || !ack.AcceptedAsHistory {
				t.Fatal(ack, err)
			}
			got := l.get(batch.BatchId)
			raw, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			if err = l.f.Validator.ValidateBody("UpdateBatch", raw); err != nil {
				t.Fatal(err)
			}
			for _, target := range got.Targets {
				if target.NodeId == unknown.NodeID {
					if !target.PolicyGeneration.IsSpecified() || !target.PolicyGeneration.IsNull() || !target.PolicyRefs.IsSpecified() || !target.PolicyRefs.IsNull() {
						t.Fatal("unknown evidence not explicit null", target)
					}
					continue
				}
				generation, err := target.PolicyGeneration.Get()
				if err != nil || generation != tc.generation {
					t.Fatal("generation changed or unknown inferred", target, err)
				}
				if tc.history {
					refs, err := target.PolicyRefs.Get()
					if err != nil || refs == nil || len(refs) != 0 {
						t.Fatal("known empty history lost", target, err)
					}
				} else if !target.PolicyRefs.IsNull() {
					t.Fatal("borrowed policy refs", target)
				}
				// Single-target Evidence and batched page share result-id selection.
				evidence, err := l.s.Evidence(l.ctx, target.TargetId)
				if err != nil || evidence.PolicyGeneration == nil || *evidence.PolicyGeneration != generation {
					t.Fatal("evidence mapping drift", evidence, err)
				}
			}
		})
	}
}
