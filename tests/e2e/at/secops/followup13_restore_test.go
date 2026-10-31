//go:build e2e && integration

package secops

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

// The retained agent is stopped before arming. This capture belongs to the
// clean protocol restore, after the separately recorded fail-safe negative.
func (p *liveProbe) armRestoreOffer13(stack *labstack.Stack) {
	// The negative restore has closed its disposable central DB. Read only the
	// retained node here; the clean central connection does not exist yet.
	hb := p.heartbeatView()
	raw, readErr := p.docker("node", "cat", "/var/lib/vpn-node/state/desired-state.json")
	var envelope struct {
		Signed struct {
			Generation string `json:"config_generation"`
		} `json:"signed"`
	}
	if readErr != nil || json.Unmarshal(raw, &envelope) != nil || envelope.Signed.Generation == "" || hb["generation"] != envelope.Signed.Generation {
		p.t.Fatal("retained local accepted/applied generations unavailable or unequal")
	}
	retained, err := json.Marshal(map[string]any{"node_accepted_desired": envelope.Signed.Generation, "node_applied": hb["generation"], "node_heartbeat": hb, "captured_at": time.Now().UTC()})
	if err != nil {
		p.t.Fatal("retained generation evidence", err)
	}
	id, err := stack.ArmFirstDesiredOffer(p.t.Context(), "node")
	if err != nil {
		p.t.Fatal("arm first desired offer", err)
	}
	p.metadata["first_desired_offer"] = map[string]any{
		"capture_id": id, "armed_at": time.Now().UTC(), "retained_before_restore": json.RawMessage(retained),
		"arm_count": 1, "agent_stopped_before_arm": true,
		"scope": "first actual agent response after clean restore and release; direct-control HTTP probes are separate",
	}
}

func (p *liveProbe) captureRestoreOffer13(stack *labstack.Stack, node uuid.UUID, since time.Time) {
	row := p.metadata["first_desired_offer"].(map[string]any)
	var retained struct {
		Accepted string `json:"node_accepted_desired"`
		Applied  string `json:"node_applied"`
	}
	if json.Unmarshal(row["retained_before_restore"].(json.RawMessage), &retained) != nil || retained.Accepted == "" || retained.Accepted != retained.Applied {
		p.t.Fatal("retained accepted/applied generations do not agree before restore")
	}
	// Wait for the real retained-generation observation, not compiler advancement:
	// compilation happens on GET, which is still held by the shared proxy.
	ordered := false
	for until := time.Now().Add(45 * time.Second); time.Now().Before(until); {
		var raw []byte
		err := p.db.QueryRow(p.t.Context(), `SELECT jsonb_build_object('report_id',o.report_id,'received_at',o.received_at,'applied_generation',o.applied_generation::text,'central_applied',n.applied_config_generation::text) FROM node_observations o JOIN nodes n ON n.id=o.node_id WHERE o.node_id=$1 AND o.received_at >= $2 AND o.applied_generation::text=$3 AND n.applied_config_generation=o.applied_generation ORDER BY o.received_at LIMIT 1`, node, since, retained.Accepted).Scan(&raw)
		if err == nil {
			row["retained_generation_observation"] = json.RawMessage(raw)
			ordered = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	row["retained_generation_ordering_proved"] = ordered
	if !ordered {
		p.t.Fatal("no fresh retained-generation observation while first-offer hold active")
	}
	row["release_started_at"] = time.Now().UTC()
	if err := stack.ReleaseDesiredOffer(p.t.Context(), "node"); err != nil {
		p.t.Fatal("release desired offer", err)
	}
	row["released_at"] = time.Now().UTC()
	var got labstack.FirstOfferEvidence
	for until := time.Now().Add(45 * time.Second); time.Now().Before(until); {
		next, err := stack.FirstDesiredOffer(p.t.Context(), "node")
		if err == nil && next.CaptureID == row["capture_id"] {
			got = next
			row["capture"] = got
			if got.First != nil {
				first, e1 := strconv.ParseUint(got.First.Generation, 10, 64)
				prior, e2 := strconv.ParseUint(retained.Accepted, 10, 64)
				row["first_generation_above_retained"] = e1 == nil && e2 == nil && first > prior
				// Preserve bad first delivery immediately. Never re-arm or substitute a
				// later generation, successful probe or compiler row for this evidence.
				if got.First.HTTPStatus != 200 || !got.First.WriteComplete || e1 != nil || first <= prior {
					break
				}
				if got.Local != nil && got.ACK != nil {
					var receipt []byte
					err = p.db.QueryRow(p.t.Context(), `SELECT jsonb_build_object('node_id',node_id,'report_id',report_id,'received_at',received_at,'applied_generation',applied_generation::text) FROM node_observations WHERE node_id=$1 AND report_id=$2 AND applied_generation::text=$3`, node, got.ACK.ReportID, got.First.Generation).Scan(&receipt)
					if err == nil {
						row["central_receipt"] = json.RawMessage(receipt)
						break
					}
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	row["capture_finished_at"] = time.Now().UTC()
	p.t.Logf("SECOPS-21: first delivery capture=%s retained=%s first=%+v local=%+v ack=%+v", got.CaptureID, retained.Accepted, got.First, got.Local, got.ACK)
}
