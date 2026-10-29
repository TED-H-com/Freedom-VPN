//go:build e2e

package updates

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
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// lab is the host side of one running dev stack. tests/e2e keeps its
// helpers unexported, so AT23–AT27 use this client against the same
// stack.json contract (§19.1).
type lab struct {
	root     string
	state    string
	prefix   string
	labIP    string
	originIP string
	node     string
	ports    map[string]int
	public   string
	agent    string
	idp      string
	releases string
	tr       *http.Transport
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
			short = "e2e"
		}
		state = filepath.Join(root, ".agent", "e2e-state-"+short)
	}
	var stack struct {
		Prefix    string         `json:"prefix"`
		LabPrefix string         `json:"lab_prefix"`
		OriginIP  string         `json:"origin_ip"`
		HostPorts map[string]int `json:"host_ports"`
	}
	raw, err := os.ReadFile(filepath.Join(state, "stack.json"))
	if err != nil {
		return nil, err
	}
	if json.Unmarshal(raw, &stack) != nil || stack.Prefix == "" || stack.HostPorts["control"] == 0 {
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
		"releases.example.invalid:443": fmt.Sprintf("127.0.0.1:%d", stack.HostPorts["origin"]),
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
	return &lab{
		root: root, state: state, prefix: stack.Prefix, labIP: stack.LabPrefix, originIP: stack.OriginIP,
		node: stack.Prefix + "-node", ports: stack.HostPorts,
		public: "https://portal.example.invalid:8443", agent: "https://agent.example.invalid:9443",
		idp: "https://idp.example.invalid", releases: "https://releases.example.invalid", tr: tr,
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
	http *http.Client
	csrf string
	role string
}

type reply struct {
	Status int
	Header http.Header
	Body   []byte
	Code   string
}

func (l *lab) login(ctx context.Context, subject, role string) (*session, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	s := &session{http: &http.Client{Transport: l.tr, Jar: jar, Timeout: 30 * time.Second}}
	assume, err := l.secret("assume.secret")
	if err != nil {
		return nil, err
	}
	body, _ := json.Marshal(map[string]string{"subject": subject, "role": role, "name": role})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.idp+"/e2e/assume", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-E2E-Secret", assume)
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("assume http %d", resp.StatusCode)
	}
	resp, err = s.http.Get(l.public + "/auth/login")
	if err != nil {
		return nil, err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	resp, err = s.http.Get(l.public + "/auth/session")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("session http %d", resp.StatusCode)
	}
	var view struct {
		Authenticated bool   `json:"authenticated"`
		Role          string `json:"role"`
	}
	if json.Unmarshal(raw, &view) != nil || !view.Authenticated || view.Role != role {
		return nil, fmt.Errorf("session role %q", view.Role)
	}
	s.csrf = resp.Header.Get("X-CSRF-Token")
	s.role = role
	if s.csrf == "" {
		return nil, errors.New("missing csrf")
	}
	return s, nil
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

func (l *lab) exec(timeout time.Duration, container string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", append([]string{"exec", container}, args...)...)
	out, err := cmd.CombinedOutput()
	if len(out) > 1<<20 {
		out = out[:1<<20]
	}
	return out, err
}

func (l *lab) putObject(ctx context.Context, name string, body []byte) error {
	secret, err := l.secret("origin.secret")
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, l.releases+"/e2e/object?name="+name, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("X-E2E-Secret", secret)
	client := &http.Client{Transport: l.tr, Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("origin http %d", resp.StatusCode)
	}
	return nil
}

func (l *lab) getObject(ctx context.Context, name string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.releases+"/"+name, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Transport: l.tr, Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("origin get http %d", resp.StatusCode)
	}
	return raw, nil
}

func waitUntil(ctx context.Context, every time.Duration, ok func() bool) bool {
	if ok() {
		return true
	}
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-tick.C:
			if ok() {
				return true
			}
		}
	}
}

func (l *lab) fleet(ctx context.Context, admin *session) (api.FleetReleaseState, error) {
	list, err := admin.call(ctx, http.MethodGet, l.public+"/api/v1/admin/releases", nil, nil)
	if err != nil {
		return api.FleetReleaseState{}, err
	}
	if list.Status != http.StatusOK {
		return api.FleetReleaseState{}, fmt.Errorf("releases http %d %s", list.Status, list.Code)
	}
	var page api.ReleaseList
	if json.Unmarshal(list.Body, &page) != nil {
		return api.FleetReleaseState{}, errors.New("releases body")
	}
	return page.Fleet, nil
}

func (l *lab) createBatch(ctx context.Context, admin *session, release uuid.UUID, reason, key string) (reply, api.UpdateBatchAccepted, error) {
	fleet, err := l.fleet(ctx, admin)
	if err != nil {
		return reply{}, api.UpdateBatchAccepted{}, err
	}
	return l.postBatch(ctx, admin, api.UpdateBatchCreateRequest{
		ReleaseId:          openapi_types.UUID(release),
		Scope:              api.UpdateBatchCreateRequestScopeAllNodes,
		ExpectedGeneration: fleet.Generation,
		SetFleetDefault:    api.UpdateBatchCreateRequestSetFleetDefaultTrue,
		Reason:             reason,
	}, key)
}

// postBatch sends one exact body. Idempotency replay must reuse that body:
// createBatch reads a new fleet generation after the first accept, so a
// second createBatch is a different payload (§14.5, AT26).
func (l *lab) postBatch(ctx context.Context, admin *session, req api.UpdateBatchCreateRequest, key string) (reply, api.UpdateBatchAccepted, error) {
	resp, err := admin.call(ctx, http.MethodPost, l.public+"/api/v1/admin/update-batches", req, map[string]string{"Idempotency-Key": key})
	if err != nil {
		return resp, api.UpdateBatchAccepted{}, err
	}
	var accepted api.UpdateBatchAccepted
	_ = json.Unmarshal(resp.Body, &accepted)
	return resp, accepted, nil
}

func (l *lab) batch(ctx context.Context, admin *session, id uuid.UUID) (api.UpdateBatch, error) {
	resp, err := admin.call(ctx, http.MethodGet, l.public+"/api/v1/admin/update-batches/"+id.String(), nil, nil)
	if err != nil {
		return api.UpdateBatch{}, err
	}
	if resp.Status != http.StatusOK {
		return api.UpdateBatch{}, fmt.Errorf("batch http %d %s", resp.Status, resp.Code)
	}
	var got api.UpdateBatch
	if json.Unmarshal(resp.Body, &got) != nil {
		return api.UpdateBatch{}, errors.New("batch body")
	}
	return got, nil
}

func (l *lab) getNode(ctx context.Context, admin *session, id uuid.UUID) (reply, api.NodeDetail, error) {
	resp, err := admin.call(ctx, http.MethodGet, l.public+"/api/v1/admin/nodes/"+id.String(), nil, nil)
	if err != nil {
		return resp, api.NodeDetail{}, err
	}
	var detail api.NodeDetail
	_ = json.Unmarshal(resp.Body, &detail)
	return resp, detail, nil
}

// insertReleaseSupport seeds the empty release_support policy the lab
// documents. Eligible batches require it. This is the same statement the
// dev-stack harness uses; it does not drop a base guard.
func (l *lab) insertReleaseSupport(ctx context.Context) error {
	conn, err := l.db(ctx)
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

// allowReleases writes both allowlists without bumping fleet generation.
// There is no admin API that sets approved and supported together (§13.2).
// The dev-stack README documents this statement.
func (l *lab) allowReleases(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return errors.New("no releases")
	}
	conn, err := l.db(ctx)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, `
		UPDATE fleet_config
		SET supported_release_policy='allowlist',
		    supported_release_ids=$1::uuid[],
		    approved_release_ids=$1::uuid[],
		    resource_version=resource_version+1
		WHERE id='00000000-0000-0000-0000-000000000001'`, ids)
	return err
}

func (l *lab) query(ctx context.Context, sql string, args ...any) ([]string, error) {
	conn, err := l.db(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)
	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		parts := make([]string, len(vals))
		for i, v := range vals {
			if v == nil {
				parts[i] = ""
				continue
			}
			parts[i] = fmt.Sprint(v)
		}
		out = append(out, strings.Join(parts, " "))
	}
	return out, rows.Err()
}

func (l *lab) scalar(ctx context.Context, sql string, args ...any) (string, error) {
	conn, err := l.db(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close(ctx)
	var v any
	if err := conn.QueryRow(ctx, sql, args...).Scan(&v); err != nil {
		return "", err
	}
	if v == nil {
		return "", nil
	}
	return fmt.Sprint(v), nil
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func httpDetail(r reply, err error) string {
	if err != nil {
		return err.Error()
	}
	code := r.Code
	if code == "" {
		code = "-"
	}
	return fmt.Sprintf("http %d %s", r.Status, code)
}
