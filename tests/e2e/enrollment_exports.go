//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// EnrolledIdentityInput names private files IN the selected node container.
// RequestFile contains the exact signed api.EnrollRequest, including the CSR.
// The caller supplies/generates the P256 PKCS8 key. No key or token is returned.
type EnrolledIdentityInput struct {
	KeyFile, RequestFile, TokenFile string
	DiscardResponse                 bool
	Join                            bool
}
type EnrolledIdentityResult struct {
	EnrollHTTP        int       `json:"enroll_http"`
	BootstrapHTTP     int       `json:"bootstrap_http"`
	ErrorCode         string    `json:"error_code,omitempty"`
	RequestSHA256     string    `json:"request_sha256"`
	ResponseDiscarded bool      `json:"response_discarded"`
	Joined            bool      `json:"joined"`
	NodeID            uuid.UUID `json:"node_id"`
	IncarnationID     uuid.UUID `json:"incarnation_id"`
}

func (s *Stack) nodeSlot(slot string) (string, error) {
	if slot != "node" && slot != "node2" && slot != "node-clone" {
		return "", fmt.Errorf("unknown node slot")
	}
	return s.lab.prefix + "-" + slot, nil
}
func (s *Stack) nodeHelper(ctx context.Context, slot, command string, input, out any) error {
	container, err := s.nodeSlot(slot)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", container, "fvpn-e2e-nodeconfig", command)
	cmd.Stdin = bytes.NewReader(raw)
	raw, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("private node helper %s failed", command)
	}
	if out == nil {
		return nil
	}
	if err = json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("private node helper result invalid")
	}
	return nil
}

// GenerateEnrollmentIdentity is optional convenience for a harness. It creates
// a fresh key, CSR and signed immutable request inside the waiting node slot.
// Callers may instead stage their own files under the same private directory.
func (s *Stack) GenerateEnrollmentIdentity(ctx context.Context, slot, name string) (EnrolledIdentityInput, error) {
	if !nativeName.MatchString(name) {
		return EnrolledIdentityInput{}, fmt.Errorf("invalid identity name")
	}
	container, err := s.nodeSlot(slot)
	if err != nil {
		return EnrolledIdentityInput{}, err
	}
	wait, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if !waitUntil(wait, time.Second, func() bool {
		return s.lab.stage(container, "/var/lib/vpn-node/e2e/status.json") == "waiting_token"
	}) {
		return EnrolledIdentityInput{}, fmt.Errorf("harness enrollment requires a waiting slot")
	}
	dir := "/var/lib/vpn-node/e2e/harness/" + name
	if _, err = s.lab.docker(container, "mkdir", "-p", "/var/lib/vpn-node/e2e/harness"); err != nil {
		return EnrolledIdentityInput{}, err
	}
	err = s.nodeHelper(ctx, slot, "enrollment-generate", map[string]string{"directory": dir}, nil)
	return EnrolledIdentityInput{KeyFile: dir + "/key.pem", RequestFile: dir + "/request.json", TokenFile: dir + "/token", Join: true}, err
}

// StageEnrollmentToken transmits an API-issued secret through stdin to mode0600
// storage. Never include token in arguments, diagnostics, records or callbacks.
func (s *Stack) StageEnrollmentToken(ctx context.Context, slot string, input EnrolledIdentityInput, token string) error {
	container, err := s.nodeSlot(slot)
	if err != nil {
		return err
	}
	if !validHarnessPath(input.TokenFile) || strings.TrimSpace(token) == "" {
		return fmt.Errorf("invalid enrollment token input")
	}
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", container, "sh", "-c", `umask 077; cat > "$1"`, "sh", input.TokenFile)
	cmd.Stdin = strings.NewReader(token)
	if cmd.Run() != nil {
		return fmt.Errorf("private token staging failed")
	}
	return nil
}
func validHarnessPath(p string) bool {
	return strings.HasPrefix(p, "/var/lib/vpn-node/e2e/harness/") && filepath.Clean(p) == p && len(p) < 256
}

// JoinEnrolledIdentity submits the caller's exact persisted bytes, optionally
// loses the response before certificate persistence, or performs normal one-time
// mTLS bootstrap plus a real NetBird join. It never writes agent/database rows.
// Use a fresh waiting slot; this protocol fixture does not start the agent.
func (s *Stack) JoinEnrolledIdentity(ctx context.Context, slot string, input EnrolledIdentityInput) (EnrolledIdentityResult, error) {
	var out EnrolledIdentityResult
	if !validHarnessPath(input.KeyFile) || !validHarnessPath(input.RequestFile) || !validHarnessPath(input.TokenFile) {
		return out, fmt.Errorf("private container paths required")
	}
	err := s.nodeHelper(ctx, slot, "enrollment-join", input, &out)
	return out, err
}

type EnrollmentPeers struct {
	EnrollmentID         uuid.UUID `json:"enrollment_id"`
	OperationID          uuid.UUID `json:"operation_id"`
	NodeID               uuid.UUID `json:"node_id"`
	IncarnationID        uuid.UUID `json:"incarnation_id"`
	EnrollmentState      string    `json:"enrollment_state"`
	SetupKeys            int       `json:"setup_keys"`
	SetupKeyUses         int       `json:"setup_key_uses"`
	CorrelationGroups    int       `json:"correlation_groups"`
	Peers                int       `json:"peers"`
	ProjectedPeerPresent bool      `json:"projected_peer_present"`
	Complete             bool      `json:"complete"`
}

// EnrollmentPeerInventory reads complete NetBird collections, the enrollment
// API and node detail. Correlation uses the product's deterministic enrollment
// key/group names and projected peer, never hostname/IP guessing. Complete is
// false until the one-off key usage and every correlated peer are read back.
// It does NOT claim complete incarnation history (ENR-03's separate API gap).
func (s *Stack) EnrollmentPeerInventory(ctx context.Context, admin BrowserSession, id uuid.UUID) (EnrollmentPeers, error) {
	out := EnrollmentPeers{EnrollmentID: id}
	resp, err := s.Call(ctx, admin, "GET", "/api/v1/admin/enrollments/"+id.String(), nil, nil)
	var en api.Enrollment
	if err != nil || resp.Status != 200 || json.Unmarshal(resp.Body, &en) != nil {
		return out, fmt.Errorf("enrollment inventory HTTP %d", resp.Status)
	}
	out.NodeID = en.NodeId
	out.IncarnationID, _ = en.NodeIncarnationId.Get()
	out.EnrollmentState = string(en.State)
	resp, err = s.Call(ctx, admin, "GET", "/api/v1/admin/audit-events?resource_type=enrollment&resource_id="+id.String()+"&limit=100", nil, nil)
	var audits api.AuditEventList
	if err != nil || resp.Status != 200 || json.Unmarshal(resp.Body, &audits) != nil {
		return out, fmt.Errorf("enrollment audit HTTP %d", resp.Status)
	}
	for _, ev := range audits.Items {
		if ev.Action != "enrollment.consumed" {
			continue
		}
		op, opErr := ev.OperationId.Get()
		if opErr == nil && uuid.UUID(op) != uuid.Nil {
			out.OperationID = uuid.UUID(op)
		}
	}
	if out.OperationID == uuid.Nil {
		return out, fmt.Errorf("enrollment operation unavailable")
	}
	resp, err = s.Call(ctx, admin, "GET", "/api/v1/admin/nodes/"+en.NodeId.String(), nil, nil)
	var detail api.NodeDetail
	if err != nil || resp.Status != 200 || json.Unmarshal(resp.Body, &detail) != nil {
		return out, fmt.Errorf("node inventory HTTP %d", resp.Status)
	}
	peer, _ := detail.NetbirdPeerId.Get()
	f := &probeFixture{lab: s.lab}
	var keys []struct {
		Name string `json:"name"`
		Used int    `json:"used_times"`
	}
	if err = f.netbirdAPI(ctx, "GET", "/setup-keys", nil, &keys); err != nil {
		return out, err
	}
	// CreateNodeSetupKey names the key and correlation group from the
	// provisioning operation id, not the enrollment id.
	sum := sha256.Sum256([]byte(out.OperationID.String()))
	suffix := hex.EncodeToString(sum[:12])
	for _, k := range keys {
		if k.Name == "fvpn-enroll-"+suffix {
			out.SetupKeys++
			out.SetupKeyUses += k.Used
		}
	}
	var groups []struct {
		Name  string    `json:"name"`
		Peers nbPeerIDs `json:"peers"`
	}
	if err = f.netbirdAPI(ctx, "GET", "/groups", nil, &groups); err != nil {
		return out, err
	}
	ids := map[string]bool{}
	if peer != "" {
		ids[peer] = true
	}
	for _, g := range groups {
		if g.Name == "fvpn-enroll-peer-"+suffix {
			out.CorrelationGroups++
			for _, id := range g.Peers {
				ids[id] = true
			}
		}
	}
	var peers []struct {
		ID string `json:"id"`
	}
	if err = f.netbirdAPI(ctx, "GET", "/peers", nil, &peers); err != nil {
		return out, err
	}
	for _, p := range peers {
		if ids[p.ID] {
			out.Peers++
		}
		if p.ID == peer && peer != "" {
			out.ProjectedPeerPresent = true
		}
	}
	// Peer discovery removes the empty correlation group after it binds the
	// peer. A remaining group must not name any other peer.
	out.Complete = out.SetupKeys == 1 && out.SetupKeyUses == 1 && out.Peers == 1 && len(ids) == 1 && out.ProjectedPeerPresent && out.CorrelationGroups <= 1
	return out, nil
}

// nbPeerIDs accepts NetBird group peers as strings or {"id": "..."} objects.
type nbPeerIDs []string

func (p *nbPeerIDs) UnmarshalJSON(raw []byte) error {
	*p = nil
	if string(raw) == "null" {
		return nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return err
	}
	for _, item := range items {
		var id string
		if json.Unmarshal(item, &id) == nil {
			if id == "" {
				return fmt.Errorf("group peer id unavailable")
			}
			*p = append(*p, id)
			continue
		}
		var obj struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(item, &obj) != nil || obj.ID == "" {
			return fmt.Errorf("group peer id unavailable")
		}
		*p = append(*p, obj.ID)
	}
	return nil
}
