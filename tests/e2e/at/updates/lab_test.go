//go:build e2e

package updates_test

// This adapter contains transport, enrollment and read-only inspection only.
// Release construction, signing, registration and support-policy activation are
// owned by tests/e2e.PrepareServingBaseline; keep that pipeline shared.
import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// lab is the host side of one running dev stack.
type lab struct {
	root      string
	state     string
	prefix    string
	node      string
	member    string
	control   string
	ports     map[string]int
	public    string
	agent     string
	idp       string
	netbird   string
	releases  string
	transport *http.Transport
}

func openLab() (*lab, error) {
	root, err := repoRoot()
	if err != nil {
		return nil, err
	}
	state := os.Getenv("FVPN_E2E_STATE")
	if state == "" {
		short := os.Getenv("FVPN_E2E_PREFIX")
		if short == "" {
			short = "w4-updates"
		}
		state = filepath.Join(root, ".agent", "e2e-state-"+short)
	}
	relative, err := filepath.Rel(root, state)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, errors.New("state outside worktree")
	}
	var stack struct {
		Prefix    string         `json:"prefix"`
		HostPorts map[string]int `json:"host_ports"`
	}
	rawStack, err := os.ReadFile(filepath.Join(state, "stack.json"))
	if err != nil {
		return nil, err
	}
	if json.Unmarshal(rawStack, &stack) != nil || stack.Prefix != "fvpn-w4-updates" || stack.HostPorts["control"] == 0 {
		return nil, errors.New("stack identity")
	}
	pemBytes, err := os.ReadFile(filepath.Join(state, "ca.pem"))
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemBytes) {
		return nil, errors.New("lab ca")
	}
	dial := map[string]string{
		"portal.example.invalid:8443":  fmt.Sprintf("127.0.0.1:%d", stack.HostPorts["control"]),
		"agent.example.invalid:9443":   fmt.Sprintf("127.0.0.1:%d", stack.HostPorts["agent"]),
		"idp.example.invalid:443":      fmt.Sprintf("127.0.0.1:%d", stack.HostPorts["idp"]),
		"netbird.example.invalid:443":  fmt.Sprintf("127.0.0.1:%d", stack.HostPorts["gateway"]),
		"api.stripe.com:443":           fmt.Sprintf("127.0.0.1:%d", stack.HostPorts["payment"]),
		"releases.example.invalid:443": fmt.Sprintf("127.0.0.1:%d", stack.HostPorts["origin"]),
	}
	tr := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if mapped, ok := dial[addr]; ok {
				addr = mapped
			} else {
				return nil, errors.New("non-lab address refused")
			}
			return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, addr)
		},
		TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
	}
	return &lab{
		root: root, state: state, prefix: stack.Prefix, ports: stack.HostPorts,
		node: stack.Prefix + "-node", member: stack.Prefix + "-member", control: stack.Prefix + "-control",
		public: "https://portal.example.invalid:8443", agent: "https://agent.example.invalid:9443",
		idp: "https://idp.example.invalid", netbird: "https://netbird.example.invalid",
		releases: "https://releases.example.invalid", transport: tr,
	}, nil
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found")
		}
		dir = parent
	}
}

func (l *lab) secret(name string) (string, error) {
	b, err := os.ReadFile(filepath.Join(l.state, "secrets", name))
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "", errors.New("empty secret")
	}
	return s, nil
}

type session struct {
	http     *http.Client
	csrf     string
	role     string
	memberID string
}

func (l *lab) newSession() (*session, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &session{http: &http.Client{Transport: l.transport, Jar: jar, Timeout: 30 * time.Second}}, nil
}

func (l *lab) login(ctx context.Context, subject, role string) (*session, error) {
	s, err := l.newSession()
	if err != nil {
		return nil, err
	}
	s.http.CheckRedirect = idplogin.RedirectLimit(20)
	token, err := l.secret("netbird.token")
	if err != nil {
		return nil, err
	}
	cred, err := idplogin.CredFor(ctx, s.http, l.netbird+"/api", token, l.state, subject)
	if err != nil {
		return nil, err
	}
	if err := idplogin.PortalLogin(ctx, s.http, l.public, cred.Email, cred.Password); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.public+"/auth/session", nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("session HTTP %d", resp.StatusCode)
	}
	var view struct {
		Authenticated bool   `json:"authenticated"`
		Role          string `json:"role"`
		MemberID      string `json:"member_id"`
	}
	if json.Unmarshal(raw, &view) != nil || !view.Authenticated || view.Role != role {
		return nil, fmt.Errorf("session role %q", view.Role)
	}
	s.csrf = resp.Header.Get("X-CSRF-Token")
	s.role = role
	s.memberID = view.MemberID
	if s.csrf == "" {
		return nil, errors.New("missing csrf")
	}
	return s, nil
}

type reply struct {
	Status int
	Header http.Header
	Body   []byte
	Code   string
}

func (s *session) call(ctx context.Context, method, url string, body any, headers map[string]string) (reply, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return reply{}, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return reply{}, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Origin", "https://portal.example.invalid:8443")
	req.Header.Set("X-CSRF-Token", s.csrf)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return reply{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return reply{}, err
	}
	return reply{Status: resp.StatusCode, Header: resp.Header.Clone(), Body: raw, Code: errorCode(raw)}, nil
}

func errorCode(body []byte) string {
	if i := bytes.IndexByte(body, '{'); i > 0 {
		body = body[i:]
	}
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &env) != nil {
		return ""
	}
	return env.Error.Code
}

func (l *lab) db(ctx context.Context) (*pgx.Conn, error) {
	password, err := l.secret("postgres.password")
	if err != nil {
		return nil, err
	}
	conn, err := pgx.Connect(ctx, fmt.Sprintf("postgres://fvpn:%s@127.0.0.1:%d/fvpn?sslmode=disable", password, l.ports["postgres"]))
	if err != nil {
		return nil, errors.New("postgres connection failed")
	}
	return conn, nil
}

func (l *lab) docker(container string, args ...string) ([]byte, error) {
	if !strings.HasPrefix(container, "fvpn-w4-updates-") {
		return nil, errors.New("non-owned container")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", append([]string{"exec", container}, args...)...)
	out, err := cmd.CombinedOutput()
	if len(out) > 1<<20 {
		out = out[:1<<20]
	}
	return out, err
}

func (l *lab) stage(container, path string) string {
	out, err := l.docker(container, "cat", path)
	if err != nil {
		return ""
	}
	var v struct {
		Stage string `json:"stage"`
	}
	if json.Unmarshal(out, &v) != nil {
		return ""
	}
	return v.Stage
}

func waitUntil(ctx context.Context, every time.Duration, ok func() bool) bool {
	for {
		if ok() {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(every):
		}
	}
}

func httpDetail(resp reply, err error) string {
	if err != nil {
		return "transport"
	}
	if resp.Code != "" {
		return strconv.Itoa(resp.Status) + " " + resp.Code
	}
	return strconv.Itoa(resp.Status)
}

func writeAtomic(path string, b []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

type runner struct {
	t             *testing.T
	lab           *lab
	nodeID, incID uuid.UUID
}

func (r *runner) enroll(ctx context.Context, admin *session) {
	name, region := "TEST ONLY updates", "lab"
	resp, err := admin.call(ctx, "POST", r.lab.public+"/api/v1/admin/enrollments", api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, Name: &name, Region: &region}, map[string]string{"Idempotency-Key": uuid.NewString()})
	var created api.EnrollmentCreated
	if err != nil || resp.Status != 201 || json.Unmarshal(resp.Body, &created) != nil || !created.SecretAvailable {
		return
	}
	token, err := created.Token.Get()
	if err != nil || writeAtomic(filepath.Join(r.lab.state, r.lab.nodeSlot(), "var", "e2e", "enrollment.token"), []byte(token+"\n")) != nil {
		return
	}
	wait, cancel := context.WithTimeout(ctx, 7*time.Minute)
	defer cancel()
	if !waitUntil(wait, 2*time.Second, func() bool {
		return r.lab.stage(r.lab.node, "/var/lib/vpn-node/e2e/status.json") == "enrolled" && r.lab.stage(r.lab.node, "/var/lib/vpn-node/e2e/netbird.json") == "netbird_up"
	}) {
		return
	}
	raw, err := r.lab.docker(r.lab.node, "cat", "/var/lib/vpn-node/identity/node.json")
	var identity struct {
		Incarnation uuid.UUID `json:"node_incarnation_id"`
	}
	if err != nil || json.Unmarshal(raw, &identity) != nil {
		return
	}
	r.nodeID, r.incID = created.Enrollment.NodeId, identity.Incarnation
}
func (r *runner) writeUpdaterConfig() error {
	boot, err := r.lab.docker(r.lab.node, "cat", "/etc/vpn-node/boot-id")
	if err != nil {
		return err
	}
	bootID, err := uuid.Parse(strings.TrimSpace(string(boot)))
	if err != nil {
		return err
	}
	cfg := map[string]any{
		"node_id": r.nodeID.String(), "incarnation_id": r.incID.String(), "boot_id": bootID.String(),
		"revoked": false,
		"platform": map[string]any{
			"os_id": "ubuntu", "os_version": "24.04", "architecture": "amd64", "updater_version": "1.0.0",
			"control_api_major": 1, "policy_schema": 1, "usage_schema": 1, "state_schema": 1,
		},
		"release_trust": "/etc/vpn-node/trust.json", "desired_trust": "/etc/vpn-node/trust.json",
		"desired_state":    "/var/lib/vpn-node/state/desired-state.json",
		"meter_checkpoint": "/var/lib/vpn-node/e2e/meter.json",
		"stop_command":     []string{"/usr/local/bin/agent-service.sh", "stop"}, "start_command": []string{"/usr/local/bin/agent-service.sh", "start"},
		"result_url":               r.lab.agent + "/api/v1/agent/update-results",
		"allowed_artifact_origins": []string{r.lab.agent},
		"mtls_cert":                "/var/lib/vpn-node/identity/cert.pem",
		"mtls_key":                 "/var/lib/vpn-node/identity/key.pem",
		"mtls_ca":                  "/etc/vpn-node/ca.pem",
	}
	// Preserve lab-supplied probe inputs; never invent a peer or repair the
	// running agent's config path to make issuance pass.
	if raw, err := r.lab.docker(r.lab.node, "cat", "/etc/vpn-node/updater-recover.json"); err == nil {
		var inherited map[string]json.RawMessage
		if err := json.Unmarshal(raw, &inherited); err != nil {
			return err
		}
		for _, key := range []string{"probe_peer", "agent_config"} {
			if value, ok := inherited[key]; ok {
				cfg[key] = value
			}
		}
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(r.lab.state, r.lab.nodeSlot(), "var", "e2e", "updater.json"), append(raw, '\n'))
}

func (l *lab) nodeSlot() string {
	if l.node == l.prefix+"-node2" {
		return "node2"
	}
	return "node"
}
