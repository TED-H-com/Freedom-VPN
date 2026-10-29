//go:build e2e

package isolation

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// stack is the host side of one running dev stack. It uses the state
// directory and loopback ports documented in docs/e2e/README.md. It does
// not copy the scenario runner in tests/e2e.
type stack struct {
	root      string
	state     string
	prefix    string
	labPrefix string
	originIP  string
	node      string
	member    string
	ports     map[string]int
	public    string
	idp       string
	transport *http.Transport
}

func openStack() (*stack, error) {
	root, err := repoRoot()
	if err != nil {
		return nil, err
	}
	state := os.Getenv("FVPN_E2E_STATE")
	if state == "" {
		short := os.Getenv("FVPN_E2E_PREFIX")
		if short == "" {
			short = "w4-isolation"
		}
		state = filepath.Join(root, ".agent", "e2e-state-"+short)
	}
	var id struct {
		Prefix    string         `json:"prefix"`
		LabPrefix string         `json:"lab_prefix"`
		OriginIP  string         `json:"origin_ip"`
		HostPorts map[string]int `json:"host_ports"`
	}
	raw, err := os.ReadFile(filepath.Join(state, "stack.json"))
	if err != nil {
		return nil, err
	}
	if json.Unmarshal(raw, &id) != nil || id.Prefix == "" || id.HostPorts["control"] == 0 {
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
		"portal.example.invalid:8443":  fmt.Sprintf("127.0.0.1:%d", id.HostPorts["control"]),
		"agent.example.invalid:9443":   fmt.Sprintf("127.0.0.1:%d", id.HostPorts["agent"]),
		"idp.example.invalid:443":      fmt.Sprintf("127.0.0.1:%d", id.HostPorts["idp"]),
		"netbird.example.invalid:443":  fmt.Sprintf("127.0.0.1:%d", id.HostPorts["gateway"]),
		"releases.example.invalid:443": fmt.Sprintf("127.0.0.1:%d", id.HostPorts["origin"]),
	}
	tr := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if mapped, ok := dial[addr]; ok {
				addr = mapped
			}
			return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, addr)
		},
		TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
	}
	return &stack{
		root: root, state: state, prefix: id.Prefix, labPrefix: id.LabPrefix, originIP: id.OriginIP,
		ports: id.HostPorts, node: id.Prefix + "-node", member: id.Prefix + "-member",
		public: "https://portal.example.invalid:8443", idp: "https://idp.example.invalid",
		transport: tr,
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

func (s *stack) healthy(ctx context.Context) bool {
	client := &http.Client{Transport: s.transport, Timeout: 8 * time.Second}
	resp, err := client.Get(s.public + "/readyz")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (s *stack) secret(name string) (string, error) {
	b, err := os.ReadFile(filepath.Join(s.state, "secrets", name))
	if err != nil {
		return "", err
	}
	out := strings.TrimSpace(string(b))
	if out == "" {
		return "", errors.New("empty secret")
	}
	return out, nil
}

type session struct {
	http     *http.Client
	csrf     string
	role     string
	memberID string
}

type reply struct {
	Status int
	Header http.Header
	Body   []byte
	Code   string
}

func (s *stack) login(ctx context.Context, subject, role string) (*session, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	sess := &session{http: &http.Client{Transport: s.transport, Jar: jar, Timeout: 30 * time.Second}}
	assume, err := s.secret("assume.secret")
	if err != nil {
		return nil, err
	}
	body, _ := json.Marshal(map[string]string{"subject": subject, "role": role, "name": role})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.idp+"/e2e/assume", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-E2E-Secret", assume)
	resp, err := sess.http.Do(req)
	if err != nil {
		return nil, err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("assume HTTP %d", resp.StatusCode)
	}
	resp, err = sess.http.Get(s.public + "/auth/login")
	if err != nil {
		return nil, err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	resp, err = sess.http.Get(s.public + "/auth/session")
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
	sess.csrf = resp.Header.Get("X-CSRF-Token")
	sess.role = role
	sess.memberID = view.MemberID
	if sess.csrf == "" {
		return nil, errors.New("missing csrf")
	}
	return sess, nil
}

func (sess *session) call(ctx context.Context, method, url string, body any, headers map[string]string) (reply, error) {
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
	req.Header.Set("X-CSRF-Token", sess.csrf)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := sess.http.Do(req)
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

func (s *stack) docker(container string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", append([]string{"exec", container}, args...)...)
	out, err := cmd.CombinedOutput()
	if len(out) > 1<<20 {
		out = out[:1<<20]
	}
	return out, err
}

func (s *stack) stage(container, path string) string {
	out, err := s.docker(container, "cat", path)
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

func (s *stack) db(ctx context.Context) (*pgx.Conn, error) {
	password, err := s.secret("postgres.password")
	if err != nil {
		return nil, err
	}
	conn, err := pgx.Connect(ctx, fmt.Sprintf("postgres://fvpn:%s@127.0.0.1:%d/fvpn?sslmode=disable", password, s.ports["postgres"]))
	if err != nil {
		return nil, errors.New("postgres connection failed")
	}
	return conn, nil
}

// ensureReleaseSupport inserts the empty release-support row the lab agent
// needs before it will hold a setup key. The payload is TEST ONLY.
func (s *stack) ensureReleaseSupport(ctx context.Context) error {
	conn, err := s.db(ctx)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	sum, err := canonjson.Hash([]byte(`{}`))
	if err != nil {
		return err
	}
	var existing uuid.UUID
	err = conn.QueryRow(ctx, `SELECT id FROM policy_definitions WHERE type='release_support' AND code='e2e empty support'`).Scan(&existing)
	if err == nil {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	definition, revision := uuid.New(), uuid.New()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO policy_definitions(id,type,code) VALUES($1,'release_support','e2e empty support')`, definition); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO policy_revisions(id,definition_id,type,revision,schema_version,payload,payload_hash,signing_key_ref,created_by) VALUES($1,$2,'release_support',1,1,'{}',$3,'e2e key','e2e actor')`, revision, definition, sum[:]); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE policy_definitions SET active_revision_id=$2, resource_version=resource_version+1 WHERE id=$1`, definition, revision); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *stack) forwardChain() ForwardView {
	out, err := s.docker(s.node, "nft", "list", "chain", "inet", "fvpn", "forward")
	if err != nil {
		return ForwardView{Generation: -1}
	}
	return ParseForwardChain(string(out))
}

func (s *stack) desired() (raw []byte, verified bool) {
	path := filepath.Join(s.state, "node", "var", "state", "desired-state.json")
	b, err := os.ReadFile(path)
	if err != nil || len(b) == 0 {
		b, err = s.docker(s.node, "cat", "/var/lib/vpn-node/state/desired-state.json")
		if err != nil {
			return nil, false
		}
	}
	trustRaw, err := os.ReadFile(filepath.Join(s.state, "node", "etc", "trust.json"))
	if err != nil {
		return b, false
	}
	ts, err := envelope.ParseNodeTrustStoreJSON(trustRaw)
	if err != nil {
		return b, false
	}
	_, _, err = envelope.Verify(b, ts, envelope.PurposeDesiredState, time.Now())
	return b, err == nil
}

type desiredView struct {
	Generation        string
	Incarnation       string
	PublicationIntent string
	Bindings          int
	AuthorizationGen  string
	DNSGeneration     string
	Resolvers         int
}

func parseDesired(raw []byte) desiredView {
	var env struct {
		Signed struct {
			ConfigGeneration  string            `json:"config_generation"`
			NodeIncarnationID string            `json:"node_incarnation_id"`
			PublicationIntent string            `json:"publication_intent"`
			DeviceBindings    []json.RawMessage `json:"device_bindings"`
			PolicyRefs        struct {
				Authorization string `json:"authorization_generation"`
				DNS           string `json:"dns_generation"`
			} `json:"policy_refs"`
			DNS struct {
				ResolverIPv4 []string `json:"resolver_ipv4"`
			} `json:"dns"`
		} `json:"signed"`
	}
	if json.Unmarshal(raw, &env) != nil {
		return desiredView{}
	}
	return desiredView{
		Generation: env.Signed.ConfigGeneration, Incarnation: env.Signed.NodeIncarnationID,
		PublicationIntent: env.Signed.PublicationIntent, Bindings: len(env.Signed.DeviceBindings),
		AuthorizationGen: env.Signed.PolicyRefs.Authorization, DNSGeneration: env.Signed.PolicyRefs.DNS,
		Resolvers: len(env.Signed.DNS.ResolverIPv4),
	}
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
		return itoa(resp.Status) + " " + resp.Code
	}
	return itoa(resp.Status)
}

func itoa(n int) string {
	return strconvItoa(n)
}

func strconvItoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [16]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func writeAtomic(path string, b []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// publishOnce sends PUT /admin/nodes/{id}/publication (§5.2, §14.2).
func publishOnce(ctx context.Context, admin *session, public, nodeID, etag, key string) (reply, error) {
	return admin.call(ctx, http.MethodPut, public+"/api/v1/admin/nodes/"+nodeID+"/publication", api.PublicationRequest{
		PublicationIntent: api.PublicationIntentPublished,
		Reason:            "at isolation acceptance",
	}, map[string]string{"Idempotency-Key": key, "If-Match": etag})
}

func nodeUUID(id openapi_types.UUID) string {
	return uuid.UUID(id).String()
}
