//go:build e2e

package members

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type stackConfig struct {
	Prefix    string         `json:"prefix"`
	PortBase  int            `json:"port_base"`
	HostPorts map[string]int `json:"host_ports"`
	LabPrefix string         `json:"lab_prefix"`
}
type browser struct {
	client       *http.Client
	csrf, member string
}
type response struct {
	status  int
	headers http.Header
	body    []byte
	code    string
}
type step struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Command  string `json:"command"`
	At       string `json:"at"`
}
type buildInfo struct {
	Source    string                       `json:"source_commit"`
	Dirty     bool                         `json:"source_dirty"`
	Images    map[string]string            `json:"image_digests"`
	Binaries  map[string]string            `json:"binary_sha256"`
	BinaryVCS map[string]map[string]string `json:"binary_vcs"`
}
type record struct {
	e2e.Record
	Build            buildInfo         `json:"build"`
	PolicyRevision   string            `json:"policy_revision"`
	PolicyReferences map[string]string `json:"active_policy_revisions"`
	NodeIncarnation  string            `json:"node_incarnation"`
	Steps            []step            `json:"steps"`
	Blockers         []string          `json:"blockers"`
	Operator         string            `json:"operator_status,omitempty"`
}
type live struct {
	t                                        *testing.T
	ctx                                      context.Context
	root, state                              string
	stack                                    stackConfig
	transport                                *http.Transport
	pool                                     *pgxpool.Pool
	admin                                    *browser
	a, b, c                                  *browser
	build                                    buildInfo
	node, inc, plan, planRev, groupA, groupB uuid.UUID
	run                                      string
	machinePeer                              string
	dnsRuleRev                               uuid.UUID
}

func repository(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "../../../.."))
	if _, err = os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatal("repository root missing")
	}
	return root
}
func TestATMembers_Live(t *testing.T) {
	if os.Getenv("FVPN_E2E") != "1" {
		t.Skip("set FVPN_E2E=1 through at-members")
	}
	if os.Getenv("FVPN_E2E_PREFIX") != "w4-members" || os.Getenv("FVPN_E2E_PORT_BASE") != "24000" {
		t.Fatal("wrong assigned instance")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Minute)
	defer cancel()
	l := &live{t: t, ctx: ctx, root: repository(t), run: uuid.NewString()[:8]}
	l.state = filepath.Join(l.root, ".agent/e2e-state-w4-members")
	l.open()
	l.admin = l.login("e2e-admin", "admin")
	l.a = l.login("AT-member-A-"+l.run, "member")
	l.b = l.login("AT-member-B-"+l.run, "member")
	l.c = l.login("AT-member-C-"+l.run, "member")
	l.seedPlan()
	// Cases share a single fresh stack. Each gathers its own evidence and leaves
	// faults restored before the next; no failure suppresses independent cases.
	for _, task := range []struct {
		id  string
		run func(*record)
	}{
		{"AT05", l.enrollment}, {"AT01", l.identity}, {"AT02", l.devices}, {"AT03", l.payment}, {"AT04", l.entitlements}, {"AT08", l.publication}, {"AT09", l.exits}, {"AT20", l.dnsRules}, {"AT21", l.dnsFailure}, {"AT22", l.privacy},
	} {
		t.Run(task.id, func(t *testing.T) {
			r := l.newRecord(task.id)
			task.run(&r)
			l.write(&r)
			if r.Status != e2e.StatusPass {
				t.Errorf("%s: %s; see docs/e2e/at/members/%s.json", task.id, r.Status, task.id)
			}
		})
	}
}
func (l *live) open() {
	raw, err := os.ReadFile(filepath.Join(l.state, "stack.json"))
	if err != nil || json.Unmarshal(raw, &l.stack) != nil || l.stack.Prefix != "fvpn-w4-members" || l.stack.PortBase != 24000 {
		l.t.Fatal("assigned stack missing")
	}
	pem, err := os.ReadFile(filepath.Join(l.state, "ca.pem"))
	if err != nil {
		l.t.Fatal("lab CA missing")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pem) {
		l.t.Fatal("lab CA invalid")
	}
	ports := map[string]int{"portal.example.invalid:8443": 24443, "agent.example.invalid:9443": 25443, "idp.example.invalid:443": 24444, "netbird.example.invalid:443": 24480, "api.stripe.com:443": 24446, "releases.example.invalid:443": 24445}
	// Exact lab-only destinations: redirects and provider calls cannot escape to
	// a real Internet endpoint, including the lab's Stripe hostname alias.
	l.transport = &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}, DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		p, ok := ports[addr]
		if !ok {
			return nil, errors.New("destination outside assigned lab")
		}
		return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, fmt.Sprintf("127.0.0.1:%d", p))
	}}
	l.t.Cleanup(l.transport.CloseIdleConnections)
	dbURL := url.URL{Scheme: "postgres", User: url.UserPassword("fvpn", l.secret("postgres.password")), Host: "127.0.0.1:29432", Path: "fvpn", RawQuery: "sslmode=disable"}
	cfg, err := pgxpool.ParseConfig(dbURL.String())
	if err != nil {
		l.t.Fatal("lab DB config")
	}
	cfg.MaxConns = 4
	l.pool, err = pgxpool.NewWithConfig(l.ctx, cfg)
	if err != nil || l.pool.Ping(l.ctx) != nil {
		l.t.Fatal("lab DB unavailable")
	}
	l.t.Cleanup(l.pool.Close)
	l.build = buildInfo{Images: map[string]string{}, Binaries: map[string]string{}, BinaryVCS: map[string]map[string]string{}}
	out, err := l.command("git", "rev-parse", "HEAD")
	if err != nil {
		l.t.Fatal("git commit missing")
	}
	l.build.Source = strings.TrimSpace(string(out))
	out, _ = l.command("git", "status", "--porcelain")
	l.build.Dirty = len(out) > 0
	for _, s := range []string{"control", "worker", "node", "member", "postgres", "idp", "netbird", "gateway", "payment", "origin", "adguard", "coredns"} {
		out, err = l.docker("inspect", "--format", "{{.Image}}", l.name(s))
		if err != nil {
			l.t.Fatalf("assigned container %s unavailable", s)
		}
		l.build.Images[s] = strings.TrimSpace(string(out))
	}
	for _, name := range []string{"fvpn-control", "fvpn-worker", "fvpn-agent", "fvpn-updater"} {
		raw, err := os.ReadFile(filepath.Join(l.root, ".agent/e2e-build", name))
		if err == nil {
			sum := sha256.Sum256(raw)
			l.build.Binaries[name] = hex.EncodeToString(sum[:])
			info, err := buildinfo.ReadFile(filepath.Join(l.root, ".agent/e2e-build", name))
			if err == nil {
				settings := map[string]string{"go_version": info.GoVersion}
				for _, setting := range info.Settings {
					if strings.HasPrefix(setting.Key, "vcs.") {
						settings[setting.Key] = setting.Value
					}
				}
				l.build.BinaryVCS[name] = settings
			}
		}
	}
}
func (l *live) secret(name string) string {
	raw, err := os.ReadFile(filepath.Join(l.state, "secrets", name))
	if err != nil {
		l.t.Fatalf("required lab secret %s unavailable", name)
	}
	return strings.TrimSpace(string(raw))
}
func (l *live) command(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(l.ctx, 40*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = l.root
	return cmd.CombinedOutput()
}
func (l *live) name(service string) string            { return "fvpn-w4-members-" + service }
func (l *live) docker(args ...string) ([]byte, error) { return l.command("docker", args...) }
func (l *live) inside(service string, args ...string) ([]byte, error) {
	return l.docker(append([]string{"exec", l.name(service)}, args...)...)
}
func (l *live) request(b *browser, method, endpoint string, body any, headers map[string]string) response {
	// Respect the stock anonymous 2/s bucket, including browser redirects.
	select {
	case <-l.ctx.Done():
		return response{code: "CANCELLED"}
	case <-time.After(600 * time.Millisecond):
	}
	var raw []byte
	var err error
	if body != nil {
		if bts, ok := body.([]byte); ok {
			raw = bts
		} else {
			raw, err = json.Marshal(body)
			if err != nil {
				l.t.Fatal("request encode")
			}
		}
	}
	req, err := http.NewRequestWithContext(l.ctx, method, endpoint, bytes.NewReader(raw))
	if err != nil {
		l.t.Fatal("request shape")
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Origin", "https://portal.example.invalid:8443")
	client := &http.Client{Transport: l.transport, Timeout: 20 * time.Second}
	if b != nil {
		client = b.client
		req.Header.Set("X-CSRF-Token", b.csrf)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return response{code: "TRANSPORT_UNAVAILABLE"}
	}
	defer resp.Body.Close()
	raw, err = io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return response{code: "RESPONSE_READ_FAILED"}
	}
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &env)
	return response{resp.StatusCode, resp.Header, raw, env.Error.Code}
}
func (l *live) api(b *browser, method, path string, body any, headers map[string]string) response {
	return l.request(b, method, "https://portal.example.invalid:8443"+path, body, headers)
}
func commandHeaders() map[string]string {
	return map[string]string{"Idempotency-Key": uuid.NewString()}
}
func (l *live) login(subject, role string) *browser {
	jar, _ := cookiejar.New(nil)
	b := &browser{client: &http.Client{Transport: l.transport, Jar: jar, Timeout: 30 * time.Second, CheckRedirect: idplogin.RedirectLimit(20)}}
	cred, err := idplogin.CredFor(l.ctx, b.client, "https://netbird.example.invalid/api", l.secret("netbird.token"), l.state, subject)
	if err != nil {
		l.t.Fatal("lab user unavailable")
	}
	if idplogin.PortalLogin(l.ctx, b.client, "https://portal.example.invalid:8443", cred.Email, cred.Password) != nil {
		l.t.Fatal("browser redirect transport")
	}
	r := l.api(b, "GET", "/auth/session", nil, nil)
	var s struct {
		Authenticated bool   `json:"authenticated"`
		Role          string `json:"role"`
		MemberID      string `json:"member_id"`
	}
	if r.status != 200 || json.Unmarshal(r.body, &s) != nil || !s.Authenticated || s.Role != role {
		l.t.Fatalf("real browser login unavailable: status=%d role=%s", r.status, role)
	}
	b.member = s.MemberID
	b.csrf = r.headers.Get("X-CSRF-Token")
	if b.csrf == "" {
		l.t.Fatal("session CSRF missing")
	}
	return b
}
func (l *live) count(query string, args ...any) int {
	var n int
	if err := l.pool.QueryRow(l.ctx, query, args...).Scan(&n); err != nil {
		l.t.Fatal("lab count query failed")
	}
	return n
}
func (l *live) wait(timeout time.Duration, fn func() bool) bool {
	ctx, cancel := context.WithTimeout(l.ctx, timeout)
	defer cancel()
	for {
		if fn() {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(time.Second):
		}
	}
}
func (l *live) newRecord(id string) record {
	for _, c := range catalogue() {
		if c.id == id {
			r := record{Record: e2e.Record{ID: id, Title: c.title, Spec: []string{"§19.1", "§19 " + id, c.requirements}, Status: e2e.StatusPass, At: time.Now().UTC().Format(time.RFC3339)}, Build: l.build, Steps: []step{}, Blockers: []string{}}
			if c.operator {
				r.Operator = "needs operator run"
			}
			return r
		}
	}
	panic("unknown AT")
}
func (r *record) check(name, expected, actual, command string, ok bool) {
	status := "pass"
	if !ok {
		status = "fail"
		r.Status = e2e.StatusFail
	}
	r.Steps = append(r.Steps, step{name, status, expected, e2e.Redact(actual), command, time.Now().UTC().Format(time.RFC3339)})
}
func (r *record) block(reason string) {
	r.Blockers = append(r.Blockers, reason)
	if r.Status != e2e.StatusFail {
		r.Status = e2e.StatusBlocked
	}
}
func (l *live) write(r *record) {
	r.PolicyReferences = map[string]string{}
	rows, err := l.pool.Query(l.ctx, `SELECT type,active_revision_id::text FROM policy_definitions WHERE active_revision_id IS NOT NULL ORDER BY type`)
	if err == nil {
		for rows.Next() {
			var kind, id string
			if rows.Scan(&kind, &id) == nil {
				r.PolicyReferences[kind] = id
			}
		}
		rows.Close()
	}
	r.NodeIncarnation = "unavailable"
	r.PolicyRevision = "unavailable"
	if l.node != uuid.Nil {
		var inc *string
		var generation string
		err := l.pool.QueryRow(l.ctx, `SELECT current_incarnation_id::text,applied_config_generation::text FROM nodes WHERE id=$1`, l.node).Scan(&inc, &generation)
		if err == nil {
			if inc != nil {
				r.NodeIncarnation = *inc
			}
			r.PolicyRevision = generation
		}
	}
	for _, s := range r.Steps {
		r.Observations = append(r.Observations, e2e.Observation{Name: s.Name, Detail: fmt.Sprintf("%s; expected: %s; actual: %s; command: %s", s.Status, s.Expected, s.Actual, s.Command)})
	}
	for _, b := range r.Blockers {
		r.Observations = append(r.Observations, e2e.Observation{Name: "blocked", Detail: b})
	}
	r.Observations = append(r.Observations, e2e.Observation{Name: "provenance", Detail: fmt.Sprintf("source=%s; applied generation=%s; incarnation=%s; image/binary digests in JSON", r.Build.Source, r.PolicyRevision, r.NodeIncarnation)})
	if r.Operator != "" {
		r.Observations = append(r.Observations, e2e.Observation{Name: "operator", Detail: r.Operator + "; see OPERATORS.md"})
	}
	dir := filepath.Join(l.root, "docs/e2e/at/members")
	if err := e2e.Write(dir, r.Record); err != nil {
		l.t.Fatal("evidence writer")
	}
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		l.t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, r.ID+".json"), append(raw, '\n'), 0644); err != nil {
		l.t.Fatal(err)
	}
	page := filepath.Join(dir, r.ID+".md")
	raw, err = os.ReadFile(page)
	if err != nil {
		l.t.Fatal(err)
	}
	if err = os.WriteFile(page, []byte(strings.TrimRight(string(raw), "\n")+"\n"), 0644); err != nil {
		l.t.Fatal(err)
	}
}
func detail(r response) string { return fmt.Sprintf("HTTP %d %s", r.status, r.code) }
