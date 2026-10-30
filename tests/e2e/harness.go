//go:build e2e

package e2e

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

	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
			short = "e2e"
		}
		state = filepath.Join(root, ".agent", "e2e-state-"+short)
	}
	var stack struct {
		Prefix    string         `json:"prefix"`
		HostPorts map[string]int `json:"host_ports"`
	}
	rawStack, err := os.ReadFile(filepath.Join(state, "stack.json"))
	if err != nil {
		return nil, err
	}
	if json.Unmarshal(rawStack, &stack) != nil || stack.Prefix == "" || stack.HostPorts["control"] == 0 {
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

func (l *lab) bindingSubject(ctx context.Context, memberID string) (issuer, subject string, err error) {
	conn, err := l.db(ctx)
	if err != nil {
		return "", "", err
	}
	defer conn.Close(ctx)
	err = conn.QueryRow(ctx, `SELECT issuer, subject FROM identity_bindings WHERE member_id = $1::uuid`, memberID).Scan(&issuer, &subject)
	return issuer, subject, err
}

func (l *lab) projectionCounts(ctx context.Context, memberID string) (users, devices, bindings int, external string, err error) {
	conn, err := l.db(ctx)
	if err != nil {
		return 0, 0, 0, "", err
	}
	defer conn.Close(ctx)
	err = conn.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM netbird_user_bindings WHERE member_id = $1::uuid),
		(SELECT count(*) FROM member_devices WHERE member_id = $1::uuid),
		(SELECT count(*) FROM device_bindings WHERE member_id = $1::uuid)`, memberID).Scan(&users, &devices, &bindings)
	if err != nil {
		return 0, 0, 0, "", err
	}
	_ = conn.QueryRow(ctx, `SELECT external_user_id FROM netbird_user_bindings WHERE member_id = $1::uuid LIMIT 1`, memberID).Scan(&external)
	return users, devices, bindings, external, nil
}

func (l *lab) docker(container string, args ...string) ([]byte, error) {
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

func (l *lab) fileNonEmpty(container, path string) bool {
	_, err := l.docker(container, "test", "-s", path)
	return err == nil
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
	client := &http.Client{Transport: l.transport, Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("origin HTTP %d", resp.StatusCode)
	}
	return nil
}

func (l *lab) signedRelease(dir, product, sequence, commit, buildID string) (uuid.UUID, []byte, error) {
	key, err := release.LoadKey(filepath.Join(l.state, "release", "key.json"))
	if err != nil {
		return uuid.Nil, nil, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, nil, err
	}
	amd := filepath.Join(dir, "amd64.tar.zst")
	arm := filepath.Join(dir, "arm64.tar.zst")
	sbom := filepath.Join(dir, "sbom.json")
	if err := os.WriteFile(sbom, []byte("{}\n"), 0o644); err != nil {
		return uuid.Nil, nil, err
	}
	base := "https://releases.example.invalid/r/" + id.String()
	signed, _, err := release.BuildManifest(release.BuildInput{
		ReleaseID: id.String(), ProductVersion: product, ReleaseSequence: sequence,
		SourceCommit: commit, BuildID: buildID,
		Components: api.ManifestComponents{NodeAgent: "1.0.0", NodeUpdater: "1.0.0", Netbird: "0.82.0", Enforcement: "1.0.0"},
		Compatibility: api.ManifestCompatibility{
			ControlApiMajor: 1, PolicySchemaVersions: []int{1}, UsageEventSchemaVersions: []int{1},
			MinimumUpdaterVersion: "1.0.0", StateSchemaFrom: []int{1}, StateSchemaTo: 1,
			RequiresServiceRestart: true, MigrationId: presence.NewNullNullable[string](),
		},
		Rollback: api.ManifestRollback{
			AllowedReleaseIds: []uuid.UUID{}, StateReadableByPrevious: true,
			RequiredFreeBytes: decstr.MustParse("4096"), HealthProfile: "vpn-node-standard-v1",
		},
		Evidence: release.EvidenceInput{
			SBOMURL: base + "/sbom.json", SBOMPath: sbom,
			NoticesURL: base + "/NOTICE.txt", SourceURL: base + "/source.txt",
		},
		Artifacts: []release.ArtifactInput{
			{OSID: "ubuntu", OSVersions: []string{"24.04"}, Architecture: "amd64", Path: amd, URL: base + "/amd64.tar.zst", InstalledSizeBytes: "1000"},
			{OSID: "ubuntu", OSVersions: []string{"24.04"}, Architecture: "arm64", Path: arm, URL: base + "/arm64.tar.zst", InstalledSizeBytes: "1000"},
		},
	}, time.Now().UTC())
	if err != nil {
		return uuid.Nil, nil, err
	}
	_, canonical, err := release.Sign(signed, key)
	if err != nil {
		return uuid.Nil, nil, err
	}
	return id, canonical, nil
}

func (l *lab) uploadRelease(ctx context.Context, id uuid.UUID, dir string, canonical []byte) error {
	names := []string{"amd64.tar.zst", "arm64.tar.zst", "sbom.json"}
	for _, name := range names {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		if err := l.putObject(ctx, "r/"+id.String()+"/"+name, b); err != nil {
			return err
		}
	}
	if err := l.putObject(ctx, "r/"+id.String()+"/NOTICE.txt", []byte("e2e\n")); err != nil {
		return err
	}
	if err := l.putObject(ctx, "r/"+id.String()+"/source.txt", []byte("e2e\n")); err != nil {
		return err
	}
	return l.putObject(ctx, "r/"+id.String()+"/manifest.json", canonical)
}

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

func (l *lab) allowReleases(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return errors.New("no releases")
	}
	conn, err := l.db(ctx)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	// Eligible requires both arrays (§13.2). Generation stays put so the
	// batch's expected_generation still matches GET /admin/releases.
	_, err = conn.Exec(ctx, `
		UPDATE fleet_config
		SET supported_release_policy='allowlist',
		    supported_release_ids=$1::uuid[],
		    approved_release_ids=$1::uuid[],
		    resource_version=resource_version+1
		WHERE id='00000000-0000-0000-0000-000000000001'`, ids)
	return err
}

func (l *lab) verifyDesired(raw []byte) error {
	trustRaw, err := os.ReadFile(filepath.Join(l.state, "node", "etc", "trust.json"))
	if err != nil {
		return err
	}
	ts, err := envelope.ParseNodeTrustStoreJSON(trustRaw)
	if err != nil {
		return err
	}
	_, _, err = envelope.Verify(raw, ts, envelope.PurposeDesiredState, time.Now())
	return err
}

// forwardView is the inet fvpn forward chain, not the input or output policy.
type forwardView struct {
	Present      bool
	Policy       string
	Baseline     bool
	Generation   int
	MemberAccept int
	Excerpt      string
}

func (l *lab) forwardChain() forwardView {
	out, err := l.docker(l.node, "nft", "list", "chain", "inet", "fvpn", "forward")
	if err != nil {
		return forwardView{}
	}
	return parseForwardChain(string(out))
}

func parseForwardChain(text string) forwardView {
	start := strings.Index(text, "chain forward")
	if start < 0 {
		return forwardView{}
	}
	body := text[start:]
	if end := strings.Index(body, "\n\t}"); end > 0 {
		body = body[:end]
	}
	view := forwardView{Present: true}
	var kept []string
	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || trim == "chain forward {" {
			continue
		}
		if strings.Contains(trim, "hook forward") {
			switch {
			case strings.Contains(trim, "policy drop"):
				view.Policy = "drop"
			case strings.Contains(trim, "policy accept"):
				view.Policy = "accept"
			}
		}
		if strings.Contains(trim, "fvpn_baseline=1") {
			view.Baseline = true
		}
		if n, ok := forwardGeneration(trim); ok && n > view.Generation {
			view.Generation = n
		}
		if memberAcceptRule(trim) {
			view.MemberAccept++
		}
		if len(kept) < 6 {
			kept = append(kept, trim)
		}
	}
	view.Excerpt = strings.Join(kept, " | ")
	if len(view.Excerpt) > 400 {
		view.Excerpt = view.Excerpt[:400]
	}
	return view
}

// memberAcceptRule counts a member-path accept on fvmb0. NetBird inserts
// fvnb0 accepts into the same chain while the baseline marker is still set,
// and the input and output chains are policy accept even when forward drops.
func forwardGeneration(line string) (int, bool) {
	const key = "fvpn_generation="
	i := strings.Index(line, key)
	if i < 0 {
		return 0, false
	}
	n := 0
	digits := false
	for _, c := range line[i+len(key):] {
		if c < '0' || c > '9' {
			break
		}
		digits = true
		n = n*10 + int(c-'0')
	}
	if !digits {
		return 0, false
	}
	return n, true
}

func memberAcceptRule(line string) bool {
	if strings.Contains(line, "policy ") || strings.Contains(line, "drop") || !strings.Contains(line, "accept") {
		return false
	}
	if !strings.Contains(line, `"fvmb0"`) {
		return false
	}
	return strings.Contains(line, "iifname") || strings.Contains(line, "oifname") || strings.Contains(line, "ip saddr") || strings.Contains(line, "ip daddr")
}

func scanWords(blob string) map[string]bool {
	words := []string{"locally_succeeded", "rolled_back", "HASH_MISMATCH", "SIGNATURE_INVALID", "ISOLATION_BASELINE_MISSING", "recovery_required", "artifact url address"}
	out := map[string]bool{}
	for _, w := range words {
		out[w] = strings.Contains(blob, w)
	}
	return out
}
