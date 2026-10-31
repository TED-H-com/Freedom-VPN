//go:build e2e

package usage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	assignedPrefix   = "w4-usage"
	assignedPortBase = 48000
)

type stackConfig struct {
	Prefix    string         `json:"prefix"`
	PortBase  int            `json:"port_base"`
	HostPorts map[string]int `json:"host_ports"`
	LabPrefix string         `json:"lab_prefix"`
}

type browserSession struct {
	client   *http.Client
	csrf     string
	memberID string
	subject  string
}

type httpResponse struct {
	status  int
	headers http.Header
	body    []byte
	code    string
}

type liveSuite struct {
	t            *testing.T
	ctx          context.Context
	root         string
	stateDir     string
	evidence     string
	stack        stackConfig
	hook         *e2e.Stack
	baseline     e2e.ServingBaseline
	roots        *x509.CertPool
	transport    *http.Transport
	pool         *pgxpool.Pool
	build        BuildMetadata
	imageDigests map[string]string
	runID        string

	admin   *browserSession
	memberA *browserSession
	memberB *browserSession

	groupID            uuid.UUID
	nodeID             uuid.UUID
	incarnationID      uuid.UUID
	exitID             uuid.UUID
	exitNetID          string
	node2ID            uuid.UUID
	incarnation2ID     uuid.UUID
	exit2ID            uuid.UUID
	exit2NetID         string
	otherNodeID        uuid.UUID
	otherIncarnationID uuid.UUID

	deviceA           uuid.UUID
	deviceB           uuid.UUID
	appliedGeneration string
	activePolicies    map[string]string

	releaseSupportRevID uuid.UUID
	dnsProfileRevID     uuid.UUID
	fairUseShadowDefID  uuid.UUID
	fairUseShadowRevID  uuid.UUID
	fairUseEnforceRevID uuid.UUID
	fairUseOverrideRev  uuid.UUID
	planDefID           uuid.UUID
	planRevID           uuid.UUID

	agentClient *http.Client
	node2Client *http.Client
	nodeTrust   *envelope.TrustStore

	sharedCmds               []CommandRecord
	extraRedactions          []string
	bootstrapTransfer        transferResult
	d067BringUp              d067CycleMeasurement
	d067Restart              d067CycleMeasurement
	fupActivationTransition  fupTransitionMeasurement
	fupUsageIngestTransition fupTransitionMeasurement
	incarnationRetention     incarnationRetentionMeasurement
	firstServing             map[string]any
}

func TestATUsage_Live(t *testing.T) {
	if os.Getenv("FVPN_E2E") != "1" {
		t.Skip("skipping live usage acceptance suite: set FVPN_E2E=1 via make -f mk/at-usage.mk at-usage")
	}
	if os.Getenv("FVPN_E2E_PREFIX") != assignedPrefix || os.Getenv("FVPN_E2E_PORT_BASE") != "48000" {
		t.Fatalf("wrong stack instance: got FVPN_E2E_PREFIX=%q FVPN_E2E_PORT_BASE=%q, want %s / 48000",
			os.Getenv("FVPN_E2E_PREFIX"), os.Getenv("FVPN_E2E_PORT_BASE"), assignedPrefix)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 28*time.Minute)
	defer cancel()

	root := repoRoot(t)
	s := &liveSuite{
		t:              t,
		ctx:            ctx,
		root:           root,
		stateDir:       filepath.Join(root, ".agent/e2e-state-"+assignedPrefix),
		evidence:       filepath.Join(root, "docs/e2e/at/usage"),
		runID:          uuid.NewString()[:8],
		imageDigests:   make(map[string]string),
		activePolicies: make(map[string]string),
	}

	s.openStack()
	s.loginSessions()
	s.bootstrapCountedServingFlow()
	s.buildAndInstallUsageAdmin()

	// Execute live evaluations in order AT16 -> AT14 -> AT18 -> AT19 -> AT17 -> AT15
	// so fvpn-agent on node exclusively owns the usage sequence stream through D-067
	// Cycle 1 (bring_up), AT16 sampler restart, D-067 Cycle 2 (agent_restart), and
	// watchdog withdrawal before AT14/18/19/17/15 drive mTLS usage and observations.
	evalMap := make(map[string]Record, 6)
	t.Log("evaluating AT16")
	evalMap["AT16"] = s.evalAT16(t)
	if err := WriteEvidence(s.evidence, evalMap["AT16"], s.extraRedactions...); err != nil {
		t.Fatal(err)
	}
	t.Log("evaluating AT14")
	evalMap["AT14"] = s.evalAT14(t)
	if err := WriteEvidence(s.evidence, evalMap["AT14"], s.extraRedactions...); err != nil {
		t.Fatal(err)
	}
	t.Log("evaluating AT18")
	evalMap["AT18"] = s.evalAT18(t)
	if err := WriteEvidence(s.evidence, evalMap["AT18"], s.extraRedactions...); err != nil {
		t.Fatal(err)
	}
	t.Log("evaluating AT19")
	evalMap["AT19"] = s.evalAT19(t)
	if err := WriteEvidence(s.evidence, evalMap["AT19"], s.extraRedactions...); err != nil {
		t.Fatal(err)
	}
	t.Log("evaluating AT17")
	evalMap["AT17"] = s.evalAT17(t)
	if err := WriteEvidence(s.evidence, evalMap["AT17"], s.extraRedactions...); err != nil {
		t.Fatal(err)
	}
	t.Log("evaluating AT15")
	evalMap["AT15"] = s.evalAT15(t)
	if err := WriteEvidence(s.evidence, evalMap["AT15"], s.extraRedactions...); err != nil {
		t.Fatal(err)
	}

	orderedIDs := []string{"AT14", "AT15", "AT16", "AT17", "AT18", "AT19"}
	records := make([]Record, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		rec := evalMap[id]
		if err := WriteEvidence(s.evidence, rec, s.extraRedactions...); err != nil {
			t.Fatalf("write evidence %s: %v", id, err)
		}
		records = append(records, rec)
	}
	s.writeDocs(records)

	for _, id := range orderedIDs {
		rec := evalMap[id]
		t.Run(id, func(t *testing.T) {
			if rec.Status != StatusPass {
				t.Errorf("%s status is %s (see docs/e2e/at/usage/%s.json and .md): %s",
					id, rec.Status, id, strings.Join(rec.Blockers, " | "))
			}
		})
	}
}

func (s *liveSuite) openStack() {
	hook, err := e2e.OpenStack()
	if err != nil {
		s.t.Fatalf("e2e.OpenStack: %v", err)
	}
	s.hook = hook
	s.t.Cleanup(hook.Close)

	raw, err := os.ReadFile(filepath.Join(s.stateDir, "stack.json"))
	if err != nil {
		s.t.Fatalf("read stack.json: %v", err)
	}
	if err := json.Unmarshal(raw, &s.stack); err != nil {
		s.t.Fatalf("unmarshal stack.json: %v", err)
	}
	if s.stack.Prefix != "fvpn-"+assignedPrefix || s.stack.PortBase != assignedPortBase {
		s.t.Fatalf("unexpected stack config: prefix=%q port_base=%d", s.stack.Prefix, s.stack.PortBase)
	}

	caPEM, err := os.ReadFile(filepath.Join(s.stateDir, "ca.pem"))
	if err != nil {
		s.t.Fatalf("read ca.pem: %v", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(caPEM) {
		s.t.Fatal("failed to parse lab CA PEM")
	}
	s.roots = roots

	ports := map[string]int{
		"portal.example.invalid:8443":  48443,
		"agent.example.invalid:9443":   49443,
		"idp.example.invalid:443":      48444,
		"releases.example.invalid:443": 48445,
		"api.stripe.com:443":           48446,
		"netbird.example.invalid:443":  48480,
	}
	s.transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:    roots,
			MinVersion: tls.VersionTLS12,
		},
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			port, ok := ports[addr]
			if !ok {
				return nil, fmt.Errorf("destination %q outside assigned %s lab", addr, assignedPrefix)
			}
			return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, fmt.Sprintf("127.0.0.1:%d", port))
		},
	}
	s.t.Cleanup(s.transport.CloseIdleConnections)

	pgPass := s.readSecret("postgres.password")
	s.extraRedactions = append(s.extraRedactions, pgPass)
	dbURL := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword("fvpn", pgPass),
		Host:     "127.0.0.1:53432",
		Path:     "fvpn",
		RawQuery: "sslmode=disable",
	}
	cfg, err := pgxpool.ParseConfig(dbURL.String())
	if err != nil {
		s.t.Fatalf("parse lab DB config: %v", err)
	}
	cfg.MaxConns = 4
	pool, err := pgxpool.NewWithConfig(s.ctx, cfg)
	if err != nil || pool.Ping(s.ctx) != nil {
		s.t.Fatalf("connect lab DB: %v", err)
	}
	s.pool = pool
	s.t.Cleanup(pool.Close)

	s.collectBuildMetadata()
}

func (s *liveSuite) collectBuildMetadata() {
	s.build = BuildMetadata{
		GoVersion: runtime.Version(),
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
		Binaries:  make(map[string]string),
	}
	out, err := s.runHostCmd("git", "rev-parse", "HEAD")
	if err != nil {
		s.t.Fatalf("git rev-parse HEAD: %v", err)
	}
	s.build.GitCommit = strings.TrimSpace(string(out))
	statusOut, _ := s.runHostCmd("git", "status", "--porcelain", "--", "tests/e2e/at/usage", "mk/at-usage.mk", ".github/workflows/at-usage.yml")
	s.build.GitDirty = len(bytes.TrimSpace(statusOut)) > 0

	services := []string{"control", "worker", "node", "member", "postgres", "idp", "netbird", "gateway", "payment", "origin", "adguard", "coredns"}
	for _, svc := range services {
		cname := s.containerName(svc)
		runOut, _ := s.runHostCmd("docker", "inspect", "--format", "{{.State.Running}}", cname)
		if strings.TrimSpace(string(runOut)) != "true" {
			_, _ = s.runHostCmd("docker", "start", cname)
		}
		out, err := s.runHostCmd("docker", "inspect", "--format", "{{.Image}}", cname)
		if err != nil {
			s.t.Fatalf("docker inspect %s: %v", cname, err)
		}
		s.imageDigests[svc] = strings.TrimSpace(string(out))
	}

	binaries := []string{"fvpn-control", "fvpn-worker", "fvpn-agent", "fvpn-updater"}
	for _, bin := range binaries {
		binPath := filepath.Join(s.root, ".agent/e2e-build", bin)
		raw, err := os.ReadFile(binPath)
		if err != nil {
			s.t.Fatalf("read binary %s: %v", binPath, err)
		}
		sum := sha256.Sum256(raw)
		s.build.Binaries[bin] = hex.EncodeToString(sum[:])
	}
}

func (s *liveSuite) containerName(service string) string {
	return "fvpn-" + assignedPrefix + "-" + service
}

func (s *liveSuite) readSecret(name string) string {
	raw, err := os.ReadFile(filepath.Join(s.stateDir, "secrets", name))
	if err != nil {
		s.t.Fatalf("read lab secret %s: %v", name, err)
	}
	return strings.TrimSpace(string(raw))
}

func (s *liveSuite) runHostCmd(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = s.root
	return cmd.CombinedOutput()
}

func (s *liveSuite) cleanupDocker(args ...string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(s.ctx), 40*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = s.root
	_ = cmd.Run()
}

func (s *liveSuite) execInContainer(service string, args ...string) ([]byte, int, error) {
	fullArgs := append([]string{"exec", s.containerName(service)}, args...)
	out, err := s.runHostCmd("docker", fullArgs...)
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return out, exitCode, err
}

func (s *liveSuite) doHTTP(b *browserSession, method, fullURL string, body any, headers map[string]string) httpResponse {
	select {
	case <-s.ctx.Done():
		return httpResponse{code: "CANCELLED"}
	case <-time.After(550 * time.Millisecond):
	}
	var rawBody []byte
	if body != nil {
		switch v := body.(type) {
		case []byte:
			rawBody = v
		default:
			var err error
			rawBody, err = json.Marshal(body)
			if err != nil {
				s.t.Fatalf("marshal request body: %v", err)
			}
		}
	}
	req, err := http.NewRequestWithContext(s.ctx, method, fullURL, bytes.NewReader(rawBody))
	if err != nil {
		s.t.Fatalf("new request %s %s: %v", method, fullURL, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Origin", "https://portal.example.invalid:8443")
	client := &http.Client{Transport: s.transport, Timeout: 20 * time.Second}
	if b != nil {
		client = b.client
		if b.csrf != "" {
			req.Header.Set("X-CSRF-Token", b.csrf)
		}
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return httpResponse{code: "TRANSPORT_ERROR", body: []byte(err.Error())}
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return httpResponse{status: resp.StatusCode, code: "READ_ERROR"}
	}
	var errEnv struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(respBytes, &errEnv)
	return httpResponse{
		status:  resp.StatusCode,
		headers: resp.Header,
		body:    respBytes,
		code:    errEnv.Error.Code,
	}
}

func (s *liveSuite) apiCall(b *browserSession, method, path string, body any, headers map[string]string) httpResponse {
	return s.doHTTP(b, method, "https://portal.example.invalid:8443"+path, body, headers)
}

func (s *liveSuite) login(subject, role string) *browserSession {
	bs, err := s.hook.Login(s.ctx, subject, role)
	if err != nil {
		s.t.Fatalf("s.hook.Login(%s, %s) failed: %v", subject, role, err)
	}
	if bs.CSRF == "" {
		s.t.Fatalf("missing X-CSRF-Token for %s", subject)
	}
	s.extraRedactions = append(s.extraRedactions, bs.CSRF)
	return &browserSession{
		client:   bs.Client,
		csrf:     bs.CSRF,
		memberID: bs.MemberID,
		subject:  subject,
	}
}

func (s *liveSuite) loginSessions() {
	s.admin = s.login("e2e-admin", "admin")
	memberSuffix := s.runID
	if matches, _ := filepath.Glob(filepath.Join(s.stateDir, "netbird", "users", "at-usage-membera-*.json")); len(matches) > 0 {
		base := filepath.Base(matches[0])
		base = strings.TrimPrefix(base, "at-usage-membera-")
		base = strings.TrimSuffix(base, ".json")
		if base != "" {
			memberSuffix = base
		}
	}
	s.memberA = s.login("AT-usage-memberA-"+memberSuffix, "member")
	s.memberB = s.login("AT-usage-memberB-"+memberSuffix, "member")
}

func idemHeaders() map[string]string {
	return map[string]string{"Idempotency-Key": uuid.NewString()}
}

func (s *liveSuite) createAndActivatePolicy(policyType string, code string, payload any) (uuid.UUID, uuid.UUID, httpResponse, httpResponse) {
	var existingDefID uuid.UUID
	if policyType == "dns_profile" || policyType == "fair_use" || policyType == "plan" {
		_ = s.pool.QueryRow(s.ctx, `SELECT id, code FROM policy_definitions WHERE type = $1 ORDER BY id LIMIT 1`, policyType).Scan(&existingDefID, &code)
	}
	reqBody := map[string]any{
		"type":           policyType,
		"schema_version": 1,
		"reason":         "TEST ONLY acceptance " + code,
		"payload":        payload,
	}
	if existingDefID != uuid.Nil {
		reqBody["definition_id"] = existingDefID.String()
	} else {
		reqBody["code"] = code
	}
	createResp := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies", reqBody, idemHeaders())
	var rev struct {
		RevisionID   uuid.UUID `json:"revision_id"`
		DefinitionID uuid.UUID `json:"definition_id"`
	}
	_ = json.Unmarshal(createResp.body, &rev)
	if createResp.status != http.StatusCreated {
		return rev.DefinitionID, rev.RevisionID, createResp, httpResponse{}
	}

	var version string
	if err := s.pool.QueryRow(s.ctx, `SELECT resource_version::text FROM policy_definitions WHERE id = $1`, rev.DefinitionID).Scan(&version); err != nil {
		s.t.Fatalf("query policy_definitions version for %s: %v", rev.DefinitionID, err)
	}
	actHeaders := idemHeaders()
	actHeaders["If-Match"] = `"` + version + `"`
	actResp := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies/"+rev.DefinitionID.String()+"/activate", map[string]any{
		"revision_id": rev.RevisionID,
		"reason":      "TEST ONLY activate " + code,
	}, actHeaders)
	if actResp.status == http.StatusAccepted {
		_ = s.waitFor(45*time.Second, func() bool {
			var activeID uuid.UUID
			err := s.pool.QueryRow(s.ctx, `SELECT active_revision_id FROM policy_definitions WHERE id = $1 AND active_revision_id IS NOT NULL`, rev.DefinitionID).Scan(&activeID)
			return err == nil && activeID == rev.RevisionID
		})
	}
	return rev.DefinitionID, rev.RevisionID, createResp, actResp
}

func (s *liveSuite) activateFairUseShadowRevision(reason string) uuid.UUID {
	effAt := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	shadowPayload := map[string]any{
		"mode":             "shadow",
		"effective_at":     effAt,
		"effective_period": "immediate",
		"accounting_basis": map[string]any{
			"period_type":      "utc_calendar_month",
			"metering_version": "forward_admission_v1",
			"measure":          "total_bytes",
		},
		"minimum_rate": map[string]any{
			"down_bps": 1000000,
			"up_bps":   1000000,
		},
		"tiers": []map[string]any{
			{
				"threshold_bytes": "0",
				"down_bps":        100000000,
				"up_bps":          50000000,
				"priority":        1,
				"display_name":    map[string]any{"zh-TW": "TEST ONLY 標準速率", "en": "TEST ONLY Standard"},
			},
			{
				"threshold_bytes": "2000",
				"down_bps":        20000000,
				"up_bps":          10000000,
				"priority":        3,
				"display_name":    map[string]any{"zh-TW": "TEST ONLY 降速一階", "en": "TEST ONLY Tier 1"},
			},
			{
				"threshold_bytes": "6000",
				"down_bps":        5000000,
				"up_bps":          2000000,
				"priority":        5,
				"display_name":    map[string]any{"zh-TW": "TEST ONLY 降速二階", "en": "TEST ONLY Tier 2"},
			},
		},
		"overrides": []map[string]any{},
	}
	createResp := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies", map[string]any{
		"type":           "fair_use",
		"schema_version": 1,
		"definition_id":  s.fairUseShadowDefID.String(),
		"reason":         reason,
		"payload":        shadowPayload,
	}, idemHeaders())
	var rev struct {
		RevisionID uuid.UUID `json:"revision_id"`
	}
	if createResp.status != http.StatusCreated || json.Unmarshal(createResp.body, &rev) != nil || rev.RevisionID == uuid.Nil {
		s.t.Fatalf("activateFairUseShadowRevision create failed: HTTP %d %s", createResp.status, string(createResp.body))
	}
	var version string
	if err := s.pool.QueryRow(s.ctx, `SELECT resource_version::text FROM policy_definitions WHERE id = $1`, s.fairUseShadowDefID).Scan(&version); err != nil {
		s.t.Fatalf("activateFairUseShadowRevision query version: %v", err)
	}
	actHeaders := idemHeaders()
	actHeaders["If-Match"] = `"` + version + `"`
	actResp := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies/"+s.fairUseShadowDefID.String()+"/activate", map[string]any{
		"revision_id": rev.RevisionID,
		"reason":      reason,
	}, actHeaders)
	if actResp.status != http.StatusAccepted {
		s.t.Fatalf("activateFairUseShadowRevision activate failed: HTTP %d %s", actResp.status, string(actResp.body))
	}
	s.fairUseShadowRevID = rev.RevisionID
	return rev.RevisionID
}

func (s *liveSuite) nodeMutation(nodeID uuid.UUID, method, suffix string, body any) httpResponse {
	basePath := "/api/v1/admin/nodes/" + nodeID.String()
	var got httpResponse
	for range 4 {
		old := s.apiCall(s.admin, http.MethodGet, basePath, nil, nil)
		h := idemHeaders()
		h["If-Match"] = old.headers.Get("ETag")
		got = s.apiCall(s.admin, method, basePath+suffix, body, h)
		if got.status != http.StatusPreconditionFailed {
			return got
		}
	}
	return got
}

func (s *liveSuite) retireStaleNodesExcept(keepIDs ...uuid.UUID) {
	rows, err := s.pool.Query(s.ctx, `
		SELECT id
		FROM nodes
		WHERE id <> ALL($1::uuid[])
		  AND (lifecycle_state = 'registered' OR current_incarnation_id IS NOT NULL)
	`, keepIDs)
	if err != nil {
		return
	}
	var staleIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if rows.Scan(&id) == nil {
			staleIDs = append(staleIDs, id)
		}
	}
	rows.Close()
	if len(staleIDs) == 0 {
		return
	}
	for _, id := range staleIDs {
		var wasPublished bool
		_ = s.pool.QueryRow(s.ctx, `SELECT COALESCE(publication_intent, '') = 'published' FROM exits WHERE node_id = $1`, id).Scan(&wasPublished)
		if wasPublished {
			_ = s.nodeMutation(id, http.MethodPut, "/publication", map[string]any{
				"publication_intent": "disabled",
				"group_ids":          []string{},
				"reason":             "TEST ONLY retire stale node publication",
			})
		}
		_ = s.nodeMutation(id, http.MethodPost, "/revoke", map[string]any{
			"reason": "TEST ONLY retire stale node incarnation",
		})
		_ = s.nodeMutation(id, http.MethodDelete, "", map[string]any{
			"reason": "TEST ONLY retire stale node",
		})
	}
	_ = s.waitFor(15*time.Second, func() bool {
		var count int
		_ = s.pool.QueryRow(s.ctx, `
			SELECT count(*)
			FROM nodes n
			LEFT JOIN exits e ON e.node_id = n.id
			WHERE n.id <> ALL($1::uuid[])
			  AND n.lifecycle_state = 'registered'
			  AND n.current_incarnation_id IS NOT NULL
			  AND COALESCE(e.publication_intent, '') = 'published'
		`, keepIDs).Scan(&count)
		return count == 0
	})
}

func (s *liveSuite) ensureUnpublishedReadyBaseline(nodeService string, nodeID uuid.UUID, perMemDownBPS, perMemUpBPS int64) {
	// Stop fvpn-agent first so any in-memory e.applyError from a previous serving attempt is cleared on restart.
	s.stopNodeAgent(nodeService)

	var currentIntent string
	_ = s.pool.QueryRow(s.ctx, `SELECT COALESCE(publication_intent, '') FROM exits WHERE node_id = $1`, nodeID).Scan(&currentIntent)
	if currentIntent == "published" {
		disResp := s.nodeMutation(nodeID, http.MethodPut, "/publication", map[string]any{
			"publication_intent": "disabled",
			"group_ids":          []string{s.groupID.String()},
			"reason":             "TEST ONLY reset publication before publish " + nodeService,
		})
		if disResp.status != http.StatusAccepted {
			s.t.Fatalf("PUT /api/v1/admin/nodes/%s/publication (disabled) failed: HTTP %d %s", nodeID, disResp.status, string(disResp.body))
		}
	}

	patchResp := s.nodeMutation(nodeID, http.MethodPatch, "", map[string]any{
		"capacity": map[string]any{
			"up_bps":              1000000000,
			"down_bps":            1000000000,
			"member_up_bps_cap":   perMemUpBPS,
			"member_down_bps_cap": perMemDownBPS,
		},
	})
	if patchResp.status != http.StatusOK && patchResp.status != http.StatusAccepted {
		s.t.Fatalf("PATCH /api/v1/admin/nodes/%s failed: HTTP %d %s", nodeID, patchResp.status, string(patchResp.body))
	}

	// Wait for outbox events (node.publication_changed / node.updated) to compile the non-serving desired state
	_ = s.waitFor(20*time.Second, func() bool {
		var pending int
		var intent string
		_ = s.pool.QueryRow(s.ctx, `SELECT count(*) FROM outbox_events WHERE delivered_at IS NULL AND dead_lettered_at IS NULL AND aggregate_id = $1 AND event_type IN ('node.publication_changed', 'node.updated')`, nodeID).Scan(&pending)
		_ = s.pool.QueryRow(s.ctx, `SELECT COALESCE(envelope->'signed'->>'publication_intent', '') FROM desired_state_envelopes WHERE node_id = $1 ORDER BY generation DESC LIMIT 1`, nodeID).Scan(&intent)
		return pending == 0 && (intent == "unpublished" || intent == "disabled")
	})

	s.startNodeAgent(nodeService)

	ready := s.waitFor(90*time.Second, func() bool {
		var ok bool
		err := s.pool.QueryRow(s.ctx, `
			SELECT n.provisioning_state = 'ready'
			   AND n.safety_status = 'passed'
			   AND n.desired_config_generation > 0
			   AND n.applied_config_generation >= n.desired_config_generation
			   AND COALESCE((
				SELECT applied_generation >= n.desired_config_generation AND safety_status = 'passed' AND report_payload->>'provisioning_state' = 'ready'
				FROM node_observations
				WHERE node_id = n.id AND incarnation_id = n.current_incarnation_id
				ORDER BY received_at DESC
				LIMIT 1
			), false)
			FROM nodes n
			WHERE n.id = $1
		`, nodeID).Scan(&ok)
		return err == nil && ok
	})
	if !ready {
		s.t.Fatalf("node %s (%s) did not reach unpublished baseline observation safety_status=passed & provisioning_state=ready", nodeID, nodeService)
	}
}

func (s *liveSuite) configureAndPublishNode(nodeService string, nodeID, incID uuid.UUID, nodeClient *http.Client, perMemDownBPS, perMemUpBPS int64) (uuid.UUID, string) {
	s.ensureUnpublishedReadyBaseline(nodeService, nodeID, perMemDownBPS, perMemUpBPS)

	var oldDesired uint64
	_ = s.pool.QueryRow(s.ctx, `SELECT desired_config_generation::bigint FROM nodes WHERE id = $1`, nodeID).Scan(&oldDesired)

	pubResp := s.nodeMutation(nodeID, http.MethodPut, "/publication", map[string]any{
		"publication_intent": "published",
		"group_ids":          []string{s.groupID.String()},
		"reason":             "TEST ONLY publish " + nodeService + " for AT14-AT19",
	})
	if pubResp.status != http.StatusAccepted {
		s.t.Fatalf("PUT /api/v1/admin/nodes/%s/publication failed: HTTP %d %s", nodeID, pubResp.status, string(pubResp.body))
	}

	var exitID uuid.UUID
	var targetGen uint64
	if !s.waitFor(45*time.Second, func() bool {
		err1 := s.pool.QueryRow(s.ctx, `SELECT id FROM exits WHERE node_id = $1 AND publication_intent = 'published'`, nodeID).Scan(&exitID)
		err2 := s.pool.QueryRow(s.ctx, `SELECT generation::bigint FROM desired_state_envelopes WHERE node_id = $1 AND incarnation_id = $2 AND envelope->'signed'->>'publication_intent' = 'published' ORDER BY generation DESC LIMIT 1`, nodeID, incID).Scan(&targetGen)
		return err1 == nil && err2 == nil && exitID != uuid.Nil && targetGen > oldDesired
	}) {
		s.t.Fatalf("node %s (%s) published desired_state_envelope did not compile beyond %d", nodeID, nodeService, oldDesired)
	}

	_, _, _, _ = s.ackNodeGeneration(nodeID, incID, nodeClient, targetGen)

	return exitID, exitNetworkID(exitID)
}

func (s *liveSuite) bootstrapCountedServingFlow() {
	// 1. Real-release serving baseline (D-050)
	var existingRelID, existingSupRevID uuid.UUID
	_ = s.pool.QueryRow(s.ctx, `SELECT id FROM releases ORDER BY created_at DESC LIMIT 1`).Scan(&existingRelID)
	_ = s.pool.QueryRow(s.ctx, `SELECT active_revision_id FROM policy_definitions WHERE type = 'release_support' AND active_revision_id IS NOT NULL LIMIT 1`).Scan(&existingSupRevID)
	if existingRelID != uuid.Nil && existingSupRevID != uuid.Nil {
		s.baseline = e2e.ServingBaseline{
			Boot:            existingRelID,
			SupportRevision: existingSupRevID,
			Digests:         map[string]string{},
		}
		s.releaseSupportRevID = existingSupRevID
	} else {
		baseline, err := e2e.PrepareServingBaseline(s.ctx, s.admin.client, s.admin.csrf, false)
		if err != nil {
			s.t.Fatalf("e2e.PrepareServingBaseline: %v", err)
		}
		s.baseline = baseline
		s.releaseSupportRevID = baseline.SupportRevision
	}
	s.sharedCmds = append(s.sharedCmds, CommandRecord{
		Command:  "e2e.PrepareServingBaseline(ctx, admin.Client, admin.CSRF, false)",
		ExitCode: 0,
		Output:   fmt.Sprintf("boot_release_id=%s support_revision_id=%s digests=%d", s.baseline.Boot, s.baseline.SupportRevision, len(s.baseline.Digests)),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	// 2. D-057 member group via POST /api/v1/admin/groups
	var existingGroupID uuid.UUID
	_ = s.pool.QueryRow(s.ctx, `SELECT id FROM member_groups ORDER BY id ASC LIMIT 1`).Scan(&existingGroupID)
	if existingGroupID != uuid.Nil {
		s.groupID = existingGroupID
		s.sharedCmds = append(s.sharedCmds, CommandRecord{
			Command:  "POST /api/v1/admin/groups",
			ExitCode: 0,
			Output:   fmt.Sprintf("HTTP 201 group_id=%s", s.groupID),
			At:       time.Now().UTC().Format(time.RFC3339),
		})
	} else {
		grpResp := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/groups", api.MemberGroupCreate{
			Code: "at-usage-grp-" + s.runID,
			Name: "TEST ONLY Usage Exit Group " + s.runID,
		}, idemHeaders())
		var grp api.MemberGroup
		if grpResp.status != http.StatusCreated || json.Unmarshal(grpResp.body, &grp) != nil || grp.GroupId == uuid.Nil {
			s.t.Fatalf("POST /api/v1/admin/groups failed: HTTP %d %s", grpResp.status, string(grpResp.body))
		}
		s.groupID = grp.GroupId
		s.sharedCmds = append(s.sharedCmds, CommandRecord{
			Command:  "POST /api/v1/admin/groups",
			ExitCode: 0,
			Output:   fmt.Sprintf("HTTP %d group_id=%s code=%s", grpResp.status, s.groupID, grp.Code),
			At:       time.Now().UTC().Format(time.RFC3339),
		})
	}

	// 3. Activate domain_rule_set, dns_profile, fair_use (shadow mode), & plan (with exit_group_ids)
	_, ruleRevID, ruleCreate, ruleAct := s.createAndActivatePolicy("domain_rule_set", "at-rules-"+s.runID, map[string]any{
		"rules": []map[string]any{
			{
				"rule_id":            uuid.NewString(),
				"profile_id":         nil,
				"domain":             "feed-01.example.invalid",
				"include_subdomains": false,
				"action":             "allow",
				"reason":             "false_positive",
				"revision":           "1",
			},
		},
	})
	if ruleCreate.status != http.StatusCreated || ruleAct.status != http.StatusAccepted {
		s.t.Fatalf("domain_rule_set setup failed: create=%d (%s) activate=%d (%s)", ruleCreate.status, string(ruleCreate.body), ruleAct.status, string(ruleAct.body))
	}

	_, dnsRevID, dnsCreate, dnsAct := s.createAndActivatePolicy("dns_profile", "at-dns-"+s.runID, map[string]any{
		"profile_code":                 "at-dns-" + s.runID,
		"display_name":                 map[string]any{"zh-TW": "TEST ONLY DNS"},
		"block_response":               "nxdomain",
		"blocked_ttl_seconds":          60,
		"domain_rule_set_revision_ids": []string{ruleRevID.String()},
		"threat_feed_revision_ids":     []string{},
	})
	if dnsCreate.status != http.StatusCreated || dnsAct.status != http.StatusAccepted {
		s.t.Fatalf("dns_profile setup failed: create=%d activate=%d", dnsCreate.status, dnsAct.status)
	}
	s.dnsProfileRevID = dnsRevID

	effAt := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	shadowPayload := map[string]any{
		"mode":             "shadow",
		"effective_at":     effAt,
		"effective_period": "immediate",
		"accounting_basis": map[string]any{
			"period_type":      "utc_calendar_month",
			"metering_version": "forward_admission_v1",
			"measure":          "total_bytes",
		},
		"minimum_rate": map[string]any{
			"down_bps": 1000000,
			"up_bps":   1000000,
		},
		"tiers": []map[string]any{
			{
				"threshold_bytes": "0",
				"down_bps":        100000000,
				"up_bps":          50000000,
				"priority":        1,
				"display_name":    map[string]any{"zh-TW": "TEST ONLY 標準速率", "en": "TEST ONLY Standard"},
			},
			{
				"threshold_bytes": "2000",
				"down_bps":        20000000,
				"up_bps":          10000000,
				"priority":        3,
				"display_name":    map[string]any{"zh-TW": "TEST ONLY 降速一階", "en": "TEST ONLY Tier 1"},
			},
			{
				"threshold_bytes": "6000",
				"down_bps":        5000000,
				"up_bps":          2000000,
				"priority":        5,
				"display_name":    map[string]any{"zh-TW": "TEST ONLY 降速二階", "en": "TEST ONLY Tier 2"},
			},
		},
		"overrides": []map[string]any{},
	}
	fupDefID, fupShadowRevID, fupCreate, fupAct := s.createAndActivatePolicy("fair_use", "at-fup-"+s.runID, shadowPayload)
	if fupCreate.status != http.StatusCreated || fupAct.status != http.StatusAccepted {
		s.t.Fatalf("fair_use shadow setup failed: create=%d activate=%d", fupCreate.status, fupAct.status)
	}
	s.fairUseShadowDefID = fupDefID
	s.fairUseShadowRevID = fupShadowRevID
	_ = s.waitFor(25*time.Second, func() bool {
		var pending int
		var mode string
		err1 := s.pool.QueryRow(s.ctx, `SELECT count(*) FROM outbox_events WHERE event_type IN ('policy.activated', 'fup.evaluate') AND delivered_at IS NULL AND dead_lettered_at IS NULL`).Scan(&pending)
		err2 := s.pool.QueryRow(s.ctx, `SELECT COALESCE((SELECT mode FROM fup_member_states WHERE member_id = $1 ORDER BY period_start DESC LIMIT 1), 'shadow')`, s.memberA.memberID).Scan(&mode)
		return err1 == nil && err2 == nil && pending == 0 && mode == "shadow"
	})

	planDefID, planRevID, planCreate, planAct := s.createAndActivatePolicy("plan", "at-plan-"+s.runID, map[string]any{
		"display_name":   map[string]any{"zh-TW": "TEST ONLY 計量驗收方案"},
		"device_limit":   3,
		"exit_group_ids": []string{s.groupID.String()},
		"price": map[string]any{
			"amount_minor":       100,
			"currency":           "TWD",
			"billing_interval":   "month",
			"provider":           "stripe",
			"provider_price_ref": "price_test_only",
		},
		"rate_limit": map[string]any{
			"down_bps": 80000000,
			"up_bps":   40000000,
		},
		"fair_use_revision_id":    fupShadowRevID.String(),
		"dns_profile_revision_id": dnsRevID.String(),
	})
	if planCreate.status != http.StatusCreated || planAct.status != http.StatusAccepted {
		s.t.Fatalf("plan policy setup failed: create=%d (%s) activate=%d (%s)",
			planCreate.status, string(planCreate.body), planAct.status, string(planAct.body))
	}
	s.planDefID = planDefID
	s.planRevID = planRevID
	s.sharedCmds = append(s.sharedCmds, CommandRecord{
		Command:  "POST /api/v1/admin/policies (dns_profile, fair_use shadow, plan with exit_group_ids) + POST /activate",
		ExitCode: 0,
		Output:   fmt.Sprintf("dns_rev=%s fair_use_shadow_rev=%s plan_rev=%s", dnsRevID, fupShadowRevID, planRevID),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	// 4. Enroll primary node (or reuse if already enrolled on this stack) and build mTLS client
	nodeJSONRaw, nodeJSONCode, _ := s.execInContainer("node", "cat", "/var/lib/vpn-node/identity/node.json")
	var existingIdent struct {
		NodeID            uuid.UUID `json:"node_id"`
		NodeIncarnationID uuid.UUID `json:"node_incarnation_id"`
	}
	if nodeJSONCode == 0 && json.Unmarshal(nodeJSONRaw, &existingIdent) == nil && existingIdent.NodeID != uuid.Nil && existingIdent.NodeIncarnationID != uuid.Nil {
		s.nodeID = existingIdent.NodeID
		s.incarnationID = existingIdent.NodeIncarnationID
	} else {
		enrolled, err := s.hook.Enroll(s.ctx, s.adminHook(), "TEST ONLY w4-usage node")
		if err != nil {
			s.t.Fatalf("s.hook.Enroll: %v", err)
		}
		s.nodeID = enrolled.NodeID
		s.incarnationID = enrolled.IncarnationID
	}
	s.agentClient = s.buildNodeMTLSClient("node")
	s.retireStaleNodesExcept(s.nodeID)

	trustRaw, codeTrust, errTrust := s.execInContainer("node", "cat", "/etc/vpn-node/trust.json")
	if errTrust != nil || codeTrust != 0 {
		s.t.Fatalf("read /etc/vpn-node/trust.json: %v", errTrust)
	}
	nodeTrust, err := envelope.ParseNodeTrustStoreJSON(trustRaw)
	if err != nil {
		s.t.Fatalf("ParseNodeTrustStoreJSON: %v", err)
	}
	s.nodeTrust = nodeTrust

	// 5. Grant entitlements and admit native devices for memberA and memberB BEFORE publishing node so
	// PUT /api/v1/admin/nodes/{id}/publication compiles a populated generation (len(device_bindings) == 2).
	for _, mem := range []*browserSession{s.memberA, s.memberB} {
		checkResp := s.apiCall(s.admin, http.MethodGet, "/api/v1/admin/members/"+mem.memberID, nil, nil)
		var existingMem api.MemberDetail
		if checkResp.status == http.StatusOK && json.Unmarshal(checkResp.body, &existingMem) == nil && existingMem.Entitlement.State == "active" {
			continue
		}
		grantResp := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/members/"+mem.memberID+"/grants", api.GrantCreateRequest{
			Source:         api.GrantSourceAdminGrant,
			Reason:         "TEST ONLY bootstrap grant",
			PlanRevisionId: &s.planRevID,
			ValidUntil:     time.Now().UTC().Add(24 * time.Hour),
		}, idemHeaders())
		if grantResp.status != http.StatusAccepted {
			s.t.Fatalf("grant for %s failed: HTTP %d %s", mem.memberID, grantResp.status, string(grantResp.body))
		}
		active := s.waitFor(45*time.Second, func() bool {
			r := s.apiCall(s.admin, http.MethodGet, "/api/v1/admin/members/"+mem.memberID, nil, nil)
			var m api.MemberDetail
			return r.status == http.StatusOK && json.Unmarshal(r.body, &m) == nil && m.Entitlement.State == "active"
		})
		if !active {
			s.t.Fatalf("member %s did not reach service_status=active", mem.memberID)
		}
	}

	var existingDevA uuid.UUID
	_ = s.pool.QueryRow(s.ctx, `SELECT id FROM member_devices WHERE member_id = $1 AND status = 'authorized' ORDER BY created_at DESC LIMIT 1`, s.memberA.memberID).Scan(&existingDevA)
	if existingDevA != uuid.Nil {
		s.deviceA = existingDevA
	} else {
		peerA, err := s.hook.JoinNativeMember(s.ctx, s.memberAHook(), s.memberA.subject, s.containerName("member"), "fvpn-member-e2e")
		if err != nil || !peerA.Authorized || peerA.DeviceID == uuid.Nil {
			s.t.Fatalf("JoinNativeMember(memberA) failed: err=%v peer=%+v", err, peerA)
		}
		s.deviceA = peerA.DeviceID
	}

	var existingDevB uuid.UUID
	_, inspectErrB := s.runHostCmd("docker", "inspect", s.containerName("member-b"))
	if inspectErrB == nil {
		_ = s.pool.QueryRow(s.ctx, `SELECT id FROM member_devices WHERE member_id = $1 AND status = 'authorized' ORDER BY created_at DESC LIMIT 1`, s.memberB.memberID).Scan(&existingDevB)
	}
	if existingDevB != uuid.Nil {
		s.deviceB = existingDevB
	} else {
		s.cleanupDocker("rm", "-f", s.containerName("member-b"))
		s.cleanupDocker("volume", "rm", "-f", s.containerName("member-b"))
		memberBContainer, err := s.hook.StartNativePeer(s.ctx, "b")
		if err != nil {
			s.t.Fatalf("StartNativePeer(b) failed: %v", err)
		}
		peerB, err := s.hook.JoinNativeMember(s.ctx, s.memberBHook(), s.memberB.subject, memberBContainer, memberBContainer)
		if err != nil || !peerB.Authorized || peerB.DeviceID == uuid.Nil {
			s.t.Fatalf("JoinNativeMember(memberB) failed: err=%v peer=%+v", err, peerB)
		}
		s.deviceB = peerB.DeviceID
	}

	// 6. Ensure primary node is at unpublished ready baseline (safety_status=passed, provisioning_state=ready) with finite capacity (60M down / 30M up)
	s.ensureUnpublishedReadyBaseline("node", s.nodeID, 60000000, 30000000)

	// 7. Publish node into s.groupID and measure D-067 Cycle 1 (bring_up) provisional-lease window
	// (applying -> awaiting_member_grants -> ready -> populated lease -> VerifyServing / data-plane probe).
	s.d067BringUp = s.measureD067ProvisionalWindow("bring_up", "node", s.nodeID, s.incarnationID, s.agentClient, func() {
		pubResp := s.nodeMutation(s.nodeID, http.MethodPut, "/publication", map[string]any{
			"publication_intent": "published",
			"group_ids":          []string{s.groupID.String()},
			"reason":             "TEST ONLY publish node for AT14-AT19",
		})
		if pubResp.status != http.StatusAccepted {
			s.t.Fatalf("PUT /api/v1/admin/nodes/%s/publication failed: HTTP %d %s", s.nodeID, pubResp.status, string(pubResp.body))
		}
	}, true)

	_ = s.pool.QueryRow(s.ctx, `SELECT id FROM exits WHERE node_id = $1`, s.nodeID).Scan(&s.exitID)
	s.exitNetID = exitNetworkID(s.exitID)
	s.bootstrapTransfer = s.d067BringUp.FirstCountedTransfer
	s.captureFirstServing()
	hbOut, _, _ := s.execInContainer("node", "cat", "/run/vpn-node/heartbeat")
	s.sharedCmds = append(s.sharedCmds,
		CommandRecord{
			Command:  "PATCH /api/v1/admin/nodes/{id} + JoinNativeMember(memberA, memberB) + PUT /publication + D-067 Cycle 1 (bring_up) provisional-lease window",
			ExitCode: 0,
			Output:   fmt.Sprintf("node_id=%s exit_id=%s deviceA=%s deviceB=%s provisional_lease=%s (ttl=%ds, members=%d) ready_obs=%s (reasons=%v, checks=%v) populated_lease=%s (members=%d) ready_to_populated_ms=%d apply_blocker=%s (%s) heartbeat=%q", s.nodeID, s.exitID, s.deviceA, s.deviceB, s.d067BringUp.ProvisionalLeaseID, s.d067BringUp.ProvisionalLeaseTTLSeconds, s.d067BringUp.ProvisionalLeaseMemberCount, s.d067BringUp.ReadyObservationReportID, s.d067BringUp.ForwardingBlockReasons, s.d067BringUp.CheckErrorCodes, s.d067BringUp.PopulatedLeaseID, s.d067BringUp.PopulatedLeaseMemberCount, s.d067BringUp.ReadyToPopulatedLeaseMs, s.d067BringUp.ApplyBlockerErrorCode, s.d067BringUp.ApplyBlockerObservationReportID, strings.TrimSpace(string(hbOut))),
			At:       time.Now().UTC().Format(time.RFC3339),
		},
		CommandRecord{
			Command:  "Baseline HTTPS probe through exit node (member -> fvnb0 -> fvmb0 -> fvup0 -> origin)",
			ExitCode: 0,
			Output:   fmt.Sprintf("counted=%t generation=%s delta_up=%s delta_down=%s origin_src=%s", s.bootstrapTransfer.Counted, s.bootstrapTransfer.Generation, s.bootstrapTransfer.DeltaUp, s.bootstrapTransfer.DeltaDown, s.bootstrapTransfer.OriginSource),
			At:       time.Now().UTC().Format(time.RFC3339),
		},
	)
	s.refreshPolicyRevisions()
}

func (s *liveSuite) refreshPolicyRevisions() {
	rows, err := s.pool.Query(s.ctx, `SELECT type, active_revision_id::text FROM policy_definitions WHERE active_revision_id IS NOT NULL ORDER BY type`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t, id string
			if rows.Scan(&t, &id) == nil {
				s.activePolicies[t] = id
			}
		}
	}
	if s.nodeID != uuid.Nil {
		_ = s.pool.QueryRow(s.ctx, `SELECT applied_config_generation::text FROM nodes WHERE id = $1`, s.nodeID).Scan(&s.appliedGeneration)
	}
}

func (s *liveSuite) waitFor(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	iter := 0
	for time.Now().Before(deadline) {
		if iter%5 == 0 {
			wName := s.containerName("worker")
			runOut, _ := s.runHostCmd("docker", "inspect", "--format", "{{.State.Running}}", wName)
			if strings.TrimSpace(string(runOut)) != "true" {
				_, _ = s.runHostCmd("docker", "start", wName)
			}
		}
		iter++
		if cond() {
			return true
		}
		select {
		case <-s.ctx.Done():
			return false
		case <-time.After(1 * time.Second):
		}
	}
	return false
}

// agentCallWithClient paces mTLS requests at >=650ms to respect D-060 limiter classes
// (SubmitUsageBatch 2/s burst 10, 8 MiB body) and asserts each response once.
func (s *liveSuite) agentCallWithClient(client *http.Client, method, path string, body any, headers map[string]string) httpResponse {
	select {
	case <-s.ctx.Done():
		return httpResponse{code: "CANCELLED"}
	case <-time.After(650 * time.Millisecond):
	}
	var rawBody []byte
	if body != nil {
		switch v := body.(type) {
		case []byte:
			rawBody = v
		default:
			var err error
			rawBody, err = json.Marshal(body)
			if err != nil {
				s.t.Fatalf("marshal agent body: %v", err)
			}
		}
	}
	req, err := http.NewRequestWithContext(s.ctx, method, "https://agent.example.invalid:9443"+path, bytes.NewReader(rawBody))
	if err != nil {
		s.t.Fatalf("new agent request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return httpResponse{code: "TRANSPORT_ERROR", body: []byte(err.Error())}
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	var errEnv struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(respBytes, &errEnv)
	return httpResponse{
		status:  resp.StatusCode,
		headers: resp.Header,
		body:    respBytes,
		code:    errEnv.Error.Code,
	}
}

func (s *liveSuite) agentCall(method, path string, body any, headers map[string]string) httpResponse {
	return s.agentCallWithClient(s.agentClient, method, path, body, headers)
}

func (s *liveSuite) baseRecord(id, title, spec, expected string, reqs []string) Record {
	s.refreshPolicyRevisions()
	polParts := make([]string, 0, len(s.activePolicies)+1)
	polParts = append(polParts, fmt.Sprintf("applied_config_generation=%s", s.appliedGeneration))
	for k, v := range s.activePolicies {
		polParts = append(polParts, fmt.Sprintf("%s=%s", k, v))
	}
	inc := "unavailable"
	if s.incarnationID != uuid.Nil {
		inc = s.incarnationID.String()
	}
	cmds := make([]CommandRecord, len(s.sharedCmds))
	copy(cmds, s.sharedCmds)
	imgCopy := make(map[string]string, len(s.imageDigests))
	for k, v := range s.imageDigests {
		imgCopy[k] = v
	}
	return Record{
		ID:              id,
		RunID:           s.runID,
		Title:           title,
		Spec:            spec,
		Requirements:    reqs,
		Status:          StatusPass,
		At:              time.Now().UTC().Format(time.RFC3339),
		Build:           s.build,
		ImageDigests:    imgCopy,
		PolicyRevision:  strings.Join(polParts, "; "),
		NodeIncarnation: inc,
		Expected:        expected,
		Commands:        cmds,
		Observations: map[string]any{
			"run_id":                  s.runID,
			"stack_prefix":            "fvpn-" + assignedPrefix,
			"port_base":               assignedPortBase,
			"node_id":                 s.nodeID.String(),
			"node_incarnation_id":     inc,
			"exit_id":                 s.exitID.String(),
			"exit_network_id":         s.exitNetID,
			"group_id":                s.groupID.String(),
			"device_a_id":             s.deviceA.String(),
			"device_b_id":             s.deviceB.String(),
			"active_policy_revisions": s.activePolicies,
			"bootstrap_transfer":      s.bootstrapTransfer,
			"first_serving_apply":     s.firstServing,
		},
		MechanismChecks: []MechanismCheck{},
		Blockers:        []string{},
	}
}
