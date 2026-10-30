//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

// Stack exposes the same setup path used by acceptance. API and Docker effects
// are limited to the prepared FVPN_E2E_STATE prefix. Close does not tear it down.
type Stack struct{ lab *lab }
type BrowserSession struct {
	Client         *http.Client
	CSRF, MemberID string
}
type Response = reply
type BuiltRelease struct {
	ID             uuid.UUID
	Directory      string
	Manifest       []byte
	ManifestSHA256 string
}
type EnrolledNode struct{ NodeID, IncarnationID uuid.UUID }

func OpenStack() (*Stack, error) {
	l, err := openLab()
	if err != nil {
		return nil, err
	}
	return &Stack{lab: l}, nil
}
func (s *Stack) Close()            { s.lab.transport.CloseIdleConnections() }
func (s *Stack) StateDir() string  { return s.lab.state }
func (s *Stack) PublicURL() string { return s.lab.public }
func (s *Stack) Login(ctx context.Context, subject, role string) (BrowserSession, error) {
	b, err := s.lab.login(ctx, subject, role)
	if err != nil {
		return BrowserSession{}, err
	}
	return BrowserSession{b.http, b.csrf, b.memberID}, nil
}
func (s *Stack) Call(ctx context.Context, b BrowserSession, method, path string, body any, headers map[string]string) (Response, error) {
	if !strings.HasPrefix(path, "/api/v1/") {
		return Response{}, fmt.Errorf("API path required")
	}
	return (&session{http: b.Client, csrf: b.CSRF}).call(ctx, method, s.lab.public+path, body, headers)
}

// BuildRelease builds both architectures and signs the real bundle. The caller
// owns dir and removes it after RegisterRelease completes.
func (s *Stack) BuildRelease(ctx context.Context, dir, version, sequence, build string, options ReleaseOptions) (BuiltRelease, error) {
	commit, err := exec.CommandContext(ctx, "git", "-C", s.lab.root, "rev-parse", "HEAD").Output()
	if err != nil {
		return BuiltRelease{}, err
	}
	id, raw, err := s.lab.buildRelease(ctx, dir, version, sequence, strings.TrimSpace(string(commit)), build, options)
	if err != nil {
		return BuiltRelease{}, err
	}
	hash, err := canonjson.HashHex(raw)
	return BuiltRelease{id, dir, raw, hash}, err
}
func (s *Stack) RegisterRelease(ctx context.Context, admin BrowserSession, b BuiltRelease) error {
	if err := s.lab.uploadRelease(ctx, b.ID, b.Directory, b.Manifest); err != nil {
		return err
	}
	resp, err := s.Call(ctx, admin, "POST", "/api/v1/admin/releases", map[string]any{"manifest": json.RawMessage(b.Manifest), "manifest_url": s.lab.releases + "/r/" + b.ID.String() + "/manifest.json"}, map[string]string{"Idempotency-Key": uuid.NewString()})
	if err != nil || (resp.Status != 201 && resp.Status != 202) {
		return fmt.Errorf("register: %s", httpDetail(resp, err))
	}
	wait, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	if !waitUntil(wait, time.Second, func() bool {
		resp, err := s.Call(wait, admin, "GET", "/api/v1/admin/releases", nil, nil)
		var page api.ReleaseList
		if err != nil || resp.Status != 200 || json.Unmarshal(resp.Body, &page) != nil {
			return false
		}
		for _, v := range page.Items {
			if v.ReleaseId == b.ID && v.VerificationState == "verified" {
				return true
			}
		}
		return false
	}) {
		return fmt.Errorf("release verification timed out")
	}
	return nil
}
func (s *Stack) CandidateConfig(id uuid.UUID, version string) error {
	return s.lab.candidateConfig(id, version)
}
func (s *Stack) ActivatePolicy(ctx context.Context, admin BrowserSession, kind, code string, payload any) (uuid.UUID, uuid.UUID, error) {
	return s.lab.createPolicy(ctx, &session{http: admin.Client, csrf: admin.CSRF}, kind, code, payload)
}

// Enroll issues a real installer token, triggers the standard entrypoint, and
// waits for the real bootstrap/NetBird join. It never exports the identity key.
func (s *Stack) Enroll(ctx context.Context, admin BrowserSession, name string) (EnrolledNode, error) {
	resp, err := s.Call(ctx, admin, "POST", "/api/v1/admin/enrollments", api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, Name: &name}, map[string]string{"Idempotency-Key": uuid.NewString()})
	var out api.EnrollmentCreated
	if err != nil || resp.Status != 201 || json.Unmarshal(resp.Body, &out) != nil {
		return EnrolledNode{}, fmt.Errorf("enroll: %s", httpDetail(resp, err))
	}
	token, err := out.Token.Get()
	if err != nil || !out.SecretAvailable {
		return EnrolledNode{}, fmt.Errorf("enrollment secret absent")
	}
	dir := filepath.Join(s.lab.state, "node", "var", "e2e")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return EnrolledNode{}, err
	}
	if err = atomicHook(filepath.Join(dir, "enrollment.token"), []byte(token+"\n")); err != nil {
		return EnrolledNode{}, err
	}
	wait, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if !waitUntil(wait, time.Second, func() bool { return s.lab.stage(s.lab.node, "/var/lib/vpn-node/e2e/netbird.json") == "netbird_up" }) {
		return EnrolledNode{}, fmt.Errorf("enrollment/join timed out")
	}
	raw, err := s.lab.docker(s.lab.node, "cat", "/var/lib/vpn-node/identity/node.json")
	var identity struct {
		Node        uuid.UUID `json:"node_id"`
		Incarnation uuid.UUID `json:"node_incarnation_id"`
	}
	if err != nil || json.Unmarshal(raw, &identity) != nil || identity.Node != out.Enrollment.NodeId || identity.Incarnation == uuid.Nil {
		return EnrolledNode{}, fmt.Errorf("enrolled identity mismatch")
	}
	return EnrolledNode{identity.Node, identity.Incarnation}, nil
}
func atomicHook(path string, raw []byte) error {
	if err := os.WriteFile(path+".tmp", raw, 0600); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}

// Service invokes the lab process supervisor. PauseAgent is required to keep a
// deliberate stop in force; the entrypoint otherwise restarts the agent.
func (s *Stack) Service(ctx context.Context, action, release string) error {
	if action != "start" && action != "stop" {
		return fmt.Errorf("unknown service action")
	}
	args := []string{"exec", s.lab.node, "agent-service.sh", action}
	if action == "start" {
		if release != "/usr/local" && !strings.HasPrefix(release, "/opt/vpn-node/releases/") {
			return fmt.Errorf("invalid release path")
		}
		args = append(args, release)
	}
	return exec.CommandContext(ctx, "docker", args...).Run()
}
func (s *Stack) PauseAgent(ctx context.Context, pause bool) error {
	action := "touch"
	if !pause {
		action = "rm"
	}
	return exec.CommandContext(ctx, "docker", "exec", s.lab.node, action, "/var/lib/vpn-node/e2e/pause-agent").Run()
}
