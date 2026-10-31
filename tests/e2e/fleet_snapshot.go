//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

type EnrollingFleetSnapshot struct {
	PendingRequestID             uuid.UUID `json:"pending_request_id"`
	KeySHA256                    string    `json:"key_sha256"`
	OfflineAttemptFailed         bool      `json:"offline_attempt_failed"`
	IncarnationUnboundAtSnapshot bool      `json:"incarnation_unbound_at_snapshot"`
	LocalStage                   string    `json:"local_stage"`
	BeforeLifecycle              string    `json:"before_lifecycle"`
	NodeID                       uuid.UUID `json:"node_id"`
	IncarnationID                uuid.UUID `json:"incarnation_id"`
	RetainedAt                   time.Time `json:"retained_at"`
	BootedAt                     time.Time `json:"booted_at"`
	EligibleAt                   time.Time `json:"eligible_at"`
	BeforeState                  string    `json:"before_state"`
	AfterState                   string    `json:"after_state"`
	SameIdentity                 bool      `json:"same_identity"`
	Eligibility                  string    `json:"eligibility"`
}

// RetainEnrollingThirdNode retains a real agent-generated pending identity while
// the API-created node remains centrally enrolling/pending. A closed loopback
// transport makes the production enroll CLI persist key/CSR/request/incarnation
// without consuming the token or receiving a certificate. Boot resumes those
// exact durable files through the normal entrypoint. No domain row is seeded.
func (s *Stack) RetainEnrollingThirdNode(ctx context.Context, admin BrowserSession) (out EnrollingFleetSnapshot, err error) {
	if err = s.startExitSlot(ctx, "node3", ".24", false, true); err != nil {
		return out, err
	}
	name := "TEST ONLY retained enrolling fleet target"
	resp, err := s.Call(ctx, admin, "POST", "/api/v1/admin/enrollments", api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryPreinstalled, Name: &name}, map[string]string{"Idempotency-Key": uuid.NewString()})
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
	// The regular entrypoint is held. Only this production CLI invocation owns
	// the pending SQLite enrollment, using a transport which cannot reach API.
	configPath := filepath.Join(s.lab.state, "node3/etc/agent-enroll.json")
	raw, err := os.ReadFile(configPath)
	var config map[string]any
	if err != nil || json.Unmarshal(raw, &config) != nil {
		return out, fmt.Errorf("fleet snapshot: prepared config")
	}
	config["control_url"] = "https://127.0.0.1:1"
	raw, err = json.Marshal(config)
	if err != nil {
		return out, err
	}
	if err = os.WriteFile(filepath.Join(s.lab.state, "node3/etc/agent-retain.json"), raw, 0600); err != nil {
		return out, err
	}
	attempt, attemptCancel := context.WithTimeout(ctx, 20*time.Second)
	failure := exec.CommandContext(attempt, "docker", "exec", container, "fvpn-agent", "enroll", "-config", "/etc/vpn-node/agent-retain.json", "-token-file", "/var/lib/vpn-node/e2e/enrollment.token").Run()
	attemptCancel()
	var exit *exec.ExitError
	out.OfflineAttemptFailed = errors.As(failure, &exit) && exit.ExitCode() == 1
	if !out.OfflineAttemptFailed {
		return out, fmt.Errorf("fleet snapshot: expected offline enrollment exit 1")
	}
	raw, err = s.lab.docker(container, "cat", "/var/lib/vpn-node/identity/enrollment.json")
	var pending struct {
		Request     uuid.UUID `json:"enrollment_request_id"`
		Incarnation uuid.UUID `json:"node_incarnation_id"`
	}
	if err != nil || json.Unmarshal(raw, &pending) != nil || pending.Request == uuid.Nil || pending.Incarnation == uuid.Nil {
		return out, fmt.Errorf("fleet snapshot: pending identity unavailable")
	}
	out.NodeID, out.IncarnationID, out.PendingRequestID = created.Enrollment.NodeId, pending.Incarnation, pending.Request
	out.KeySHA256, err = s.pendingIdentityHash(ctx, container)
	if err != nil {
		return out, err
	}
	resp, err = s.Call(ctx, admin, "GET", "/api/v1/admin/nodes/"+out.NodeID.String(), nil, nil)
	var detail api.NodeDetail
	if err != nil || resp.Status != 200 || json.Unmarshal(resp.Body, &detail) != nil {
		return out, fmt.Errorf("fleet snapshot: node readback")
	}
	out.BeforeState = string(detail.ProvisioningState)
	out.LocalStage = "enrolling_snapshot"
	out.BeforeLifecycle = string(detail.LifecycleState)
	out.IncarnationUnboundAtSnapshot = detail.CurrentIncarnationId.IsSpecified() && detail.CurrentIncarnationId.IsNull()
	if out.BeforeState != "pending" || out.BeforeLifecycle != "enrolling" || !out.IncarnationUnboundAtSnapshot {
		return out, fmt.Errorf("fleet snapshot: unexpected central enrollment state")
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
		hash, hashErr := s.pendingIdentityHash(wait, s.lab.prefix+"-node3")
		centralInc, centralErr := detail.CurrentIncarnationId.Get()
		out.SameIdentity = e == nil && json.Unmarshal(raw, &identity) == nil && identity.Node == out.NodeID && identity.Incarnation == out.IncarnationID && hashErr == nil && hash == out.KeySHA256 && centralErr == nil && centralInc == out.IncarnationID
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

func (s *Stack) pendingIdentityHash(ctx context.Context, container string) (string, error) {
	raw, err := exec.CommandContext(ctx, "docker", "exec", container, "sha256sum", "/var/lib/vpn-node/identity/key.pem").Output()
	parts := strings.Fields(string(raw))
	if err != nil || len(parts) != 2 || len(parts[0]) != 64 {
		return "", fmt.Errorf("fleet snapshot: key digest unavailable")
	}
	return parts[0], nil
}
