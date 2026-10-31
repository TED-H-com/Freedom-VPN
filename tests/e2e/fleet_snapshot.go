//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

type EnrollingFleetSnapshot struct {
	NodeID        uuid.UUID `json:"node_id"`
	IncarnationID uuid.UUID `json:"incarnation_id"`
	RetainedAt    time.Time `json:"retained_at"`
	BootedAt      time.Time `json:"booted_at"`
	EligibleAt    time.Time `json:"eligible_at"`
	BeforeState   string    `json:"before_state"`
	AfterState    string    `json:"after_state"`
	SameIdentity  bool      `json:"same_identity"`
	Eligibility   string    `json:"eligibility"`
}

// RetainEnrollingThirdNode enrolls through the real entrypoint and holds before
// bootstrap. It stops node3 with its identity/certificate/key in private durable
// bind mounts. This retained directory snapshot is the only identity used on
// BootEnrollingThirdNode; there is no fabricated node or incarnation row.
func (s *Stack) RetainEnrollingThirdNode(ctx context.Context, admin BrowserSession) (out EnrollingFleetSnapshot, err error) {
	if err = s.startExitSlot(ctx, "node3", ".24", false, true); err != nil {
		return out, err
	}
	name := "TEST ONLY retained enrolling fleet target"
	resp, err := s.Call(ctx, admin, "POST", "/api/v1/admin/enrollments", api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, Name: &name}, map[string]string{"Idempotency-Key": uuid.NewString()})
	var created api.EnrollmentCreated
	if err != nil || resp.Status != 201 || json.Unmarshal(resp.Body, &created) != nil {
		return out, fmt.Errorf("fleet snapshot: enrollment HTTP %d", resp.Status)
	}
	token, err := created.Token.Get()
	if err != nil {
		return out, fmt.Errorf("fleet snapshot: enrollment secret unavailable")
	}
	if err = atomicHook(filepath.Join(s.lab.state, "node3/var/e2e/enrollment.token"), []byte(token+"\n")); err != nil {
		return out, err
	}
	token = ""
	wait, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	container := s.lab.prefix + "-node3"
	if !waitUntil(wait, time.Second, func() bool {
		return s.lab.stage(container, "/var/lib/vpn-node/e2e/status.json") == "enrolling_snapshot"
	}) {
		return out, fmt.Errorf("fleet snapshot: pre-bootstrap hold deadline")
	}
	raw, err := s.lab.docker(container, "cat", "/var/lib/vpn-node/identity/node.json")
	var identity struct {
		Node        uuid.UUID `json:"node_id"`
		Incarnation uuid.UUID `json:"node_incarnation_id"`
	}
	if err != nil || json.Unmarshal(raw, &identity) != nil || identity.Node != created.Enrollment.NodeId || identity.Incarnation == uuid.Nil {
		return out, fmt.Errorf("fleet snapshot: identity mismatch")
	}
	out.NodeID, out.IncarnationID = identity.Node, identity.Incarnation
	resp, err = s.Call(ctx, admin, "GET", "/api/v1/admin/nodes/"+out.NodeID.String(), nil, nil)
	var detail api.NodeDetail
	if err != nil || resp.Status != 200 || json.Unmarshal(resp.Body, &detail) != nil {
		return out, fmt.Errorf("fleet snapshot: node readback")
	}
	out.BeforeState = string(detail.ProvisioningState)
	if out.BeforeState != "enrolling" {
		return out, fmt.Errorf("fleet snapshot: expected enrolling, observed %s", out.BeforeState)
	}
	if err = exec.CommandContext(ctx, "docker", "stop", "-t", "10", container).Run(); err != nil {
		return out, fmt.Errorf("fleet snapshot: stop failed")
	}
	out.RetainedAt = time.Now().UTC()
	out.Eligibility = "first admin API observation of provisioning_state=ready with the retained incarnation; publication/update assignment is separate"
	// Metadata is private too; the export is returned explicitly to the caller.
	raw, err = json.Marshal(out)
	if err != nil {
		return out, err
	}
	err = atomicHook(filepath.Join(s.lab.state, "node3/snapshot.json"), raw)
	return out, err
}

// BootEnrollingThirdNode resumes the retained snapshot mid-run. EligibleAt is
// a measured read-back timestamp, never the instant docker start returned.
func (s *Stack) BootEnrollingThirdNode(ctx context.Context, admin BrowserSession) (out EnrollingFleetSnapshot, err error) {
	raw, err := os.ReadFile(filepath.Join(s.lab.state, "node3/snapshot.json"))
	if err != nil || json.Unmarshal(raw, &out) != nil || out.NodeID == uuid.Nil || out.IncarnationID == uuid.Nil {
		return out, fmt.Errorf("fleet snapshot: retained metadata required")
	}
	if err = os.Remove(filepath.Join(s.lab.state, "node3/var/e2e/hold-enrolling")); err != nil {
		return out, err
	}
	if err = exec.CommandContext(ctx, "docker", "start", s.lab.prefix+"-node3").Run(); err != nil {
		return out, err
	}
	out.BootedAt = time.Now().UTC()
	wait, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	ready := waitUntil(wait, time.Second, func() bool {
		resp, e := s.Call(wait, admin, "GET", "/api/v1/admin/nodes/"+out.NodeID.String(), nil, nil)
		var detail api.NodeDetail
		if e != nil || resp.Status != 200 || json.Unmarshal(resp.Body, &detail) != nil {
			return false
		}
		out.AfterState = string(detail.ProvisioningState)
		var identity struct {
			Node        uuid.UUID `json:"node_id"`
			Incarnation uuid.UUID `json:"node_incarnation_id"`
		}
		raw, e := s.lab.docker(s.lab.prefix+"-node3", "cat", "/var/lib/vpn-node/identity/node.json")
		out.SameIdentity = e == nil && json.Unmarshal(raw, &identity) == nil && identity.Node == out.NodeID && identity.Incarnation == out.IncarnationID
		if out.SameIdentity && out.AfterState == "ready" {
			out.EligibleAt = time.Now().UTC()
			return true
		}
		return false
	})
	if !ready {
		return out, fmt.Errorf("fleet snapshot: eligibility deadline (state=%s)", out.AfterState)
	}
	return out, nil
}
