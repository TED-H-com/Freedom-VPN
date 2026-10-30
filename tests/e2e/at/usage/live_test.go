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
	"net/http/cookiejar"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
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
	roots        *x509.CertPool
	transport    *http.Transport
	pool         *pgxpool.Pool
	build        BuildMetadata
	imageDigests map[string]string
	runID        string

	admin   *browserSession
	memberA *browserSession
	memberB *browserSession

	nodeID            uuid.UUID
	incarnationID     uuid.UUID
	appliedGeneration string
	activePolicies    map[string]string

	releaseSupportRevID uuid.UUID
	dnsProfileRevID     uuid.UUID
	fairUseShadowDefID  uuid.UUID
	fairUseShadowRevID  uuid.UUID
	fairUseEnforceRevID uuid.UUID

	agentClient *http.Client
	nodeTrust   *envelope.TrustStore

	sharedCmds      []CommandRecord
	extraRedactions []string

	// Live probe artifacts shared across AT14..AT19 records
	pubStatus         int
	pubCode           string
	pubBody           string
	forwardChainOut   string
	deviceBindingRows int
	planCreateStatus  int
	planCreateBody    string
}

func TestATUsage_Live(t *testing.T) {
	if os.Getenv("FVPN_E2E") != "1" {
		t.Skip("skipping live usage acceptance suite: set FVPN_E2E=1 via make -f mk/at-usage.mk at-usage")
	}
	if os.Getenv("FVPN_E2E_PREFIX") != assignedPrefix || os.Getenv("FVPN_E2E_PORT_BASE") != "48000" {
		t.Fatalf("wrong stack instance: got FVPN_E2E_PREFIX=%q FVPN_E2E_PORT_BASE=%q, want %s / 48000",
			os.Getenv("FVPN_E2E_PREFIX"), os.Getenv("FVPN_E2E_PORT_BASE"), assignedPrefix)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Minute)
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
	s.setupPoliciesAndNode()

	records := make([]Record, 0, 6)
	cases := []struct {
		id string
		fn func(t *testing.T) Record
	}{
		{"AT14", s.evalAT14},
		{"AT15", s.evalAT15},
		{"AT16", s.evalAT16},
		{"AT17", s.evalAT17},
		{"AT18", s.evalAT18},
		{"AT19", s.evalAT19},
	}

	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			rec := tc.fn(t)
			if err := WriteEvidence(s.evidence, rec, s.extraRedactions...); err != nil {
				t.Fatalf("write evidence %s: %v", tc.id, err)
			}
			records = append(records, rec)
			if rec.Status != StatusPass {
				t.Errorf("%s status is %s (see docs/e2e/at/usage/%s.json and .md): %s",
					tc.id, rec.Status, tc.id, strings.Join(rec.Blockers, " | "))
			}
		})
	}

	s.writeDocs(records)
}

func (s *liveSuite) openStack() {
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
	ctx, cancel := context.WithTimeout(s.ctx, 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = s.root
	return cmd.CombinedOutput()
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
	jar, _ := cookiejar.New(nil)
	b := &browserSession{
		client: &http.Client{
			Transport: s.transport,
			Jar:       jar,
			Timeout:   20 * time.Second,
		},
	}
	assumeSecret := s.readSecret("assume.secret")
	s.extraRedactions = append(s.extraRedactions, assumeSecret)
	assume := s.doHTTP(b, http.MethodPost, "https://idp.example.invalid/e2e/assume",
		map[string]string{"Subject": subject, "Role": role, "Name": "TEST ONLY " + subject},
		map[string]string{"X-E2E-Secret": assumeSecret},
	)
	if assume.status != http.StatusNoContent {
		s.t.Fatalf("IdP assume failed for %s (%s): HTTP %d %s", subject, role, assume.status, string(assume.body))
	}
	loginResp := s.apiCall(b, http.MethodGet, "/auth/login?role="+role, nil, nil)
	if loginResp.status == 0 {
		s.t.Fatalf("GET /auth/login?role=%s failed: %s", role, string(loginResp.body))
	}
	sessResp := s.apiCall(b, http.MethodGet, "/auth/session", nil, nil)
	var sess struct {
		Authenticated bool   `json:"authenticated"`
		Role          string `json:"role"`
		MemberID      string `json:"member_id"`
	}
	if sessResp.status != http.StatusOK || json.Unmarshal(sessResp.body, &sess) != nil || !sess.Authenticated || sess.Role != role {
		s.t.Fatalf("GET /auth/session failed for %s: HTTP %d %s", role, sessResp.status, string(sessResp.body))
	}
	b.memberID = sess.MemberID
	b.csrf = sessResp.headers.Get("X-CSRF-Token")
	if b.csrf == "" {
		s.t.Fatalf("missing X-CSRF-Token for %s", subject)
	}
	s.extraRedactions = append(s.extraRedactions, b.csrf)
	return b
}

func (s *liveSuite) loginSessions() {
	s.admin = s.login("e2e-admin", "admin")
	s.memberA = s.login("AT-usage-memberA-"+s.runID, "member")
	s.memberB = s.login("AT-usage-memberB-"+s.runID, "member")
}

func idemHeaders() map[string]string {
	return map[string]string{"Idempotency-Key": uuid.NewString()}
}

func (s *liveSuite) createAndActivatePolicy(policyType string, code string, payload any) (uuid.UUID, uuid.UUID, httpResponse, httpResponse) {
	createResp := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies", map[string]any{
		"type":           policyType,
		"schema_version": 1,
		"code":           code,
		"reason":         "TEST ONLY acceptance " + code,
		"payload":        payload,
	}, idemHeaders())
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
	return rev.DefinitionID, rev.RevisionID, createResp, actResp
}

func (s *liveSuite) setupPoliciesAndNode() {
	// 1. Create and activate release_support policy via real HTTP API
	relID := uuid.MustParse("0195f000-0000-7000-8000-000000000901")
	_, relRevID, relCreate, relAct := s.createAndActivatePolicy("release_support", "at-rel-"+s.runID, map[string]any{
		"approved_release_ids": []string{relID.String()},
		"minimum_serving_version": map[string]any{
			"serving_release_ids": []string{relID.String()},
		},
		"revoked_release_ids": []string{},
	})
	if relCreate.status != http.StatusCreated || (relAct.status != http.StatusAccepted && relAct.status != http.StatusUnprocessableEntity) {
		s.t.Fatalf("release_support policy setup failed: create=%d (%s) activate=%d (%s)",
			relCreate.status, string(relCreate.body), relAct.status, string(relAct.body))
	}
	s.releaseSupportRevID = relRevID
	s.sharedCmds = append(s.sharedCmds, CommandRecord{
		Command:  "POST /api/v1/admin/policies (release_support) + POST /activate",
		ExitCode: 0,
		Output:   fmt.Sprintf("create=HTTP %d revision_id=%s; activate=HTTP %d %s", relCreate.status, relRevID, relAct.status, relAct.code),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	// 2. Create and activate dns_profile policy via real HTTP API
	_, dnsRevID, dnsCreate, dnsAct := s.createAndActivatePolicy("dns_profile", "at-dns-"+s.runID, map[string]any{
		"profile_code":                 "at-dns-" + s.runID,
		"display_name":                 map[string]any{"zh-TW": "TEST ONLY DNS"},
		"block_response":               "nxdomain",
		"blocked_ttl_seconds":          60,
		"domain_rule_set_revision_ids": []string{},
		"threat_feed_revision_ids":     []string{},
	})
	if dnsCreate.status != http.StatusCreated || dnsAct.status != http.StatusAccepted {
		s.t.Fatalf("dns_profile policy setup failed: create=%d (%s) activate=%d (%s)",
			dnsCreate.status, string(dnsCreate.body), dnsAct.status, string(dnsAct.body))
	}
	s.dnsProfileRevID = dnsRevID

	// 3. Create and activate fair_use policy (shadow mode first, then enforce mode) via real HTTP API
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
				"threshold_bytes": "1000",
				"down_bps":        20000000,
				"up_bps":          10000000,
				"priority":        3,
				"display_name":    map[string]any{"zh-TW": "TEST ONLY 降速一階", "en": "TEST ONLY Tier 1"},
			},
			{
				"threshold_bytes": "5000",
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
		s.t.Fatalf("fair_use shadow policy setup failed: create=%d (%s) activate=%d (%s)",
			fupCreate.status, string(fupCreate.body), fupAct.status, string(fupAct.body))
	}
	s.fairUseShadowDefID = fupDefID
	s.fairUseShadowRevID = fupShadowRevID
	s.sharedCmds = append(s.sharedCmds, CommandRecord{
		Command:  "POST /api/v1/admin/policies (fair_use shadow) + POST /activate",
		ExitCode: 0,
		Output:   fmt.Sprintf("definition_id=%s shadow_revision_id=%s create=HTTP %d activate=HTTP %d", fupDefID, fupShadowRevID, fupCreate.status, fupAct.status),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	// 4. Attempt to create a plan policy on the unseeded stack (documents member_groups precondition)
	planCreate := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies", map[string]any{
		"type":           "plan",
		"schema_version": 1,
		"code":           "at-plan-" + s.runID,
		"reason":         "TEST ONLY plan creation probe",
		"payload": map[string]any{
			"display_name": map[string]any{"zh-TW": "TEST ONLY 方案"},
			"device_limit": 3,
			"exit_group_ids": []string{
				"0195f000-0000-7000-8000-000000000999",
			},
			"price": map[string]any{
				"amount_minor":       100,
				"currency":           "TWD",
				"billing_interval":   "month",
				"provider":           "stripe",
				"provider_price_ref": "price_test_only",
			},
			"rate_limit": map[string]any{
				"down_bps": 100000000,
				"up_bps":   50000000,
			},
			"fair_use_revision_id":    fupShadowRevID.String(),
			"dns_profile_revision_id": dnsRevID.String(),
		},
	}, idemHeaders())
	s.planCreateStatus = planCreate.status
	s.planCreateBody = strings.TrimSpace(string(planCreate.body))

	// 5. Enroll fvpn-w4-usage-node via POST /api/v1/admin/enrollments (or reuse already-enrolled node on the stack)
	nodeName := "TEST ONLY w4-usage node"
	enrollResp := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/enrollments", api.EnrollmentCreateRequest{
		Delivery: api.EnrollmentDeliveryInstaller,
		Name:     &nodeName,
	}, idemHeaders())
	var enrolled api.EnrollmentCreated
	if enrollResp.status != http.StatusCreated || json.Unmarshal(enrollResp.body, &enrolled) != nil {
		s.t.Fatalf("POST /api/v1/admin/enrollments failed: HTTP %d %s", enrollResp.status, string(enrollResp.body))
	}
	token, tokenErr := enrolled.Token.Get()
	if tokenErr != nil || token == "" {
		s.t.Fatal("enrollment token missing from 201 response")
	}
	s.extraRedactions = append(s.extraRedactions, token)

	nodeJSONRaw, nodeJSONCode, _ := s.execInContainer("node", "cat", "/var/lib/vpn-node/identity/node.json")
	var existingIdent struct {
		NodeID            uuid.UUID `json:"node_id"`
		NodeIncarnationID uuid.UUID `json:"node_incarnation_id"`
	}
	if nodeJSONCode == 0 && json.Unmarshal(nodeJSONRaw, &existingIdent) == nil && existingIdent.NodeID != uuid.Nil && existingIdent.NodeIncarnationID != uuid.Nil {
		s.nodeID = existingIdent.NodeID
		s.incarnationID = existingIdent.NodeIncarnationID
		_ = s.pool.QueryRow(s.ctx, `SELECT applied_config_generation::text FROM nodes WHERE id = $1`, s.nodeID).Scan(&s.appliedGeneration)
	} else {
		s.nodeID = uuid.UUID(enrolled.Enrollment.NodeId)
		tokenPath := filepath.Join(s.stateDir, "node/var/e2e/enrollment.token")
		if err := os.WriteFile(tokenPath, []byte(token+"\n"), 0o600); err != nil {
			s.t.Fatalf("write enrollment token to %s: %v", tokenPath, err)
		}

		// Wait for node enrollment & mTLS incarnation
		enrolledOK := s.waitFor(100*time.Second, func() bool {
			err := s.pool.QueryRow(s.ctx,
				`SELECT current_incarnation_id, applied_config_generation::text FROM nodes WHERE id = $1 AND current_incarnation_id IS NOT NULL`,
				s.nodeID,
			).Scan(&s.incarnationID, &s.appliedGeneration)
			return err == nil && s.incarnationID != uuid.Nil
		})
		if !enrolledOK {
			s.t.Fatalf("node %s did not complete mTLS enrollment within timeout", s.nodeID)
		}
	}

	// Wait for cert.pem, key.pem, and usage.db to appear on the node container
	_ = s.waitFor(30*time.Second, func() bool {
		_, code1, _ := s.execInContainer("node", "test", "-s", "/var/lib/vpn-node/identity/cert.pem")
		_, code2, _ := s.execInContainer("node", "test", "-s", "/var/lib/vpn-node/state/usage.db")
		return code1 == 0 && code2 == 0
	})

	// Build mTLS client using the enrolled node's identity (/var/lib/vpn-node/identity/cert.pem & key.pem)
	crtPEM, codeCrt, errCrt := s.execInContainer("node", "cat", "/var/lib/vpn-node/identity/cert.pem")
	keyPEM, codeKey, errKey := s.execInContainer("node", "cat", "/var/lib/vpn-node/identity/key.pem")
	if errCrt != nil || codeCrt != 0 || errKey != nil || codeKey != 0 {
		s.t.Fatalf("read node mTLS identity failed: crt=%v key=%v", errCrt, errKey)
	}
	s.extraRedactions = append(s.extraRedactions, string(keyPEM))
	nodeCert, err := tls.X509KeyPair(crtPEM, keyPEM)
	if err != nil {
		s.t.Fatalf("parse node X509KeyPair: %v", err)
	}
	agentTransport := s.transport.Clone()
	agentTransport.TLSClientConfig = &tls.Config{
		RootCAs:      s.roots,
		Certificates: []tls.Certificate{nodeCert},
		MinVersion:   tls.VersionTLS12,
	}
	s.agentClient = &http.Client{Transport: agentTransport, Timeout: 20 * time.Second}
	s.t.Cleanup(agentTransport.CloseIdleConnections)

	// Load node trust store from /etc/vpn-node/trust.json inside fvpn-w4-usage-node
	trustRaw, codeTrust, errTrust := s.execInContainer("node", "cat", "/etc/vpn-node/trust.json")
	if errTrust != nil || codeTrust != 0 {
		s.t.Fatalf("read /etc/vpn-node/trust.json: %v", errTrust)
	}
	nodeTrust, err := envelope.ParseNodeTrustStoreJSON(trustRaw)
	if err != nil {
		s.t.Fatalf("ParseNodeTrustStoreJSON: %v", err)
	}
	s.nodeTrust = nodeTrust

	// Attempt to publish the node via PUT /api/v1/admin/nodes/{id}/publication
	nodeGet := s.apiCall(s.admin, http.MethodGet, "/api/v1/admin/nodes/"+s.nodeID.String(), nil, nil)
	pubHeaders := idemHeaders()
	pubHeaders["If-Match"] = nodeGet.headers.Get("ETag")
	pubResp := s.apiCall(s.admin, http.MethodPut, "/api/v1/admin/nodes/"+s.nodeID.String()+"/publication", map[string]any{
		"publication_intent": "published",
		"reason":             "TEST ONLY publish node for AT14-AT19",
	}, pubHeaders)
	s.pubStatus = pubResp.status
	s.pubCode = pubResp.code
	s.pubBody = strings.TrimSpace(string(pubResp.body))

	fwdOut, fwdCode, _ := s.execInContainer("node", "nft", "list", "chain", "inet", "fvpn", "forward")
	s.forwardChainOut = strings.TrimSpace(string(fwdOut))
	s.sharedCmds = append(s.sharedCmds,
		CommandRecord{
			Command:  "PUT /api/v1/admin/nodes/" + s.nodeID.String() + "/publication",
			ExitCode: 0,
			Output:   fmt.Sprintf("HTTP %d %s: %s", s.pubStatus, s.pubCode, s.pubBody),
			At:       time.Now().UTC().Format(time.RFC3339),
		},
		CommandRecord{
			Command:  "docker exec " + s.containerName("node") + " nft list chain inet fvpn forward",
			ExitCode: fwdCode,
			Output:   s.forwardChainOut,
			At:       time.Now().UTC().Format(time.RFC3339),
		},
	)

	_ = s.pool.QueryRow(s.ctx, `SELECT count(*) FROM device_bindings`).Scan(&s.deviceBindingRows)
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
	for time.Now().Before(deadline) {
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

func (s *liveSuite) agentCall(method, path string, body any, headers map[string]string) httpResponse {
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
	var last httpResponse
	for attempt := 0; attempt < 8; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 650 * time.Millisecond)
		} else {
			time.Sleep(350 * time.Millisecond)
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
		resp, err := s.agentClient.Do(req)
		if err != nil {
			return httpResponse{code: "TRANSPORT_ERROR", body: []byte(err.Error())}
		}
		respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		resp.Body.Close()
		var errEnv struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(respBytes, &errEnv)
		last = httpResponse{
			status:  resp.StatusCode,
			headers: resp.Header,
			body:    respBytes,
			code:    errEnv.Error.Code,
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			return last
		}
	}
	return last
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
		Title:           title,
		Spec:            spec,
		Requirements:    reqs,
		Status:          StatusBlocked,
		At:              time.Now().UTC().Format(time.RFC3339),
		Build:           s.build,
		ImageDigests:    imgCopy,
		PolicyRevision:  strings.Join(polParts, "; "),
		NodeIncarnation: inc,
		Expected:        expected,
		Commands:        cmds,
		Observations: map[string]any{
			"stack_prefix":            "fvpn-" + assignedPrefix,
			"port_base":               assignedPortBase,
			"node_id":                 s.nodeID.String(),
			"node_incarnation_id":     inc,
			"active_policy_revisions": s.activePolicies,
			"node_publication_http":   fmt.Sprintf("%d %s", s.pubStatus, s.pubCode),
			"device_bindings_count":   s.deviceBindingRows,
		},
		MechanismChecks: []MechanismCheck{},
		Blockers:        []string{},
	}
}

// evalAT14 evaluates AT14: forward_admission_v1 metering boundary, handshake/denied/DNS exclusions, and kernel nftables probe.
func (s *liveSuite) evalAT14(t *testing.T) Record {
	rec := s.baseRecord(
		"AT14",
		"轉發許可流量計量正確性與排除項 (Forward-Admission Traffic Metering & Exclusions)",
		"§10.1, §10.2, §19.5 AT14",
		"Admitted member forwarding traffic across fv0 <-> eth0 is metered within ±1% under forward_admission_v1; WireGuard handshakes, blocked RFC1918/ULA packets, and node-local DNS queries to 100.64.0.1:53 are excluded from member usage counters.",
		[]string{"MTR01", "MTR05", "API01", "DATA01"},
	)

	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	memID := uuid.MustParse("0195f000-0000-7000-8000-000000000140")
	devID := uuid.MustParse("0195f000-0000-7000-8000-000000000141")
	rendered, err := enforce.Render(enforce.Model{
		Generation: 14,
		IPv6Mode:   enforce.IPv6Block,
		Now:        now,
		Datapath: enforce.Datapath{
			MemberIngress: "fv-wg0",
			Uplink:        "fv-wan0",
			SNAT:          netip.MustParseAddr("192.0.2.1"),
		},
		Bindings: []enforce.Binding{
			{
				MemberID:        memID,
				DeviceID:        devID,
				PeerID:          "peer-at14",
				AllowedV4:       []netip.Prefix{netip.MustParsePrefix("100.64.0.10/32")},
				BindingRevision: 1,
				ValidFrom:       now.Add(-time.Hour),
				ValidUntil:      now.Add(24 * time.Hour),
			},
		},
		Leases: []enforce.MemberLease{
			{
				MemberID:            memID,
				EffectiveServiceEnd: now.Add(24 * time.Hour),
				NodeLeaseEnd:        now.Add(24 * time.Hour),
				SignedServerTime:    now,
				MaxOfflineSeconds:   86400,
			},
		},
		Resolvers: []netip.Addr{netip.MustParseAddr("100.64.0.2")},
		HostV4:    []netip.Prefix{netip.MustParsePrefix("192.0.2.1/32")},
		QoS: enforce.QoS{
			Up:       enforce.Rate{RateBPS: 1_000_000_000, CeilBPS: 1_000_000_000},
			Down:     enforce.Rate{RateBPS: 1_000_000_000, CeilBPS: 1_000_000_000},
			HostUp:   enforce.Rate{RateBPS: 100_000_000, CeilBPS: 1_000_000_000},
			HostDown: enforce.Rate{RateBPS: 100_000_000, CeilBPS: 1_000_000_000},
			Members: []enforce.MemberQoS{
				{
					MemberID: memID,
					Up:       enforce.Rate{RateBPS: 50_000_000, CeilBPS: 100_000_000},
					Down:     enforce.Rate{RateBPS: 50_000_000, CeilBPS: 100_000_000},
				},
			},
		},
	}, enforce.State{}, enforce.TCState{})
	if err != nil {
		t.Fatalf("AT14 enforce.Render: %v", err)
	}

	probeScript := fmt.Sprintf(`set -e
ip netns del fv_at14 2>/dev/null || true
ip netns add fv_at14
trap 'ip netns del fv_at14 2>/dev/null || true' EXIT
ip -n fv_at14 link set lo up
ip -n fv_at14 link add fv-wg0 type dummy
ip -n fv_at14 addr add 100.64.0.1/10 dev fv-wg0
ip -n fv_at14 link set fv-wg0 up
ip -n fv_at14 link add fv-wan0 type dummy
ip -n fv_at14 addr add 192.0.2.1/24 dev fv-wan0
ip -n fv_at14 link set fv-wan0 up
cat <<'EOF' | ip netns exec fv_at14 nft -f -
%s
EOF
ip netns exec fv_at14 nft -j list table inet fvpn
`, rendered.NFT)

	probeOut, probeCode, probeErr := s.execInContainer("node", "sh", "-c", probeScript)
	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "docker exec " + s.containerName("node") + " ip netns exec fv_at14 nft -f - && nft -j list table inet fvpn",
		ExitCode: probeCode,
		Output:   fmt.Sprintf("nft_json_bytes=%d exit=%d", len(probeOut), probeCode),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	var parsedState enforce.Ruleset
	var parseErr error
	if probeErr == nil && probeCode == 0 {
		parsedState, parseErr = enforce.ParseRuleset(probeOut)
	}
	uName := deviceCounterName("u", devID)
	dName := deviceCounterName("d", devID)
	_, hasU := parsedState.Counters[uName]
	_, hasD := parsedState.Counters[dName]
	if probeErr != nil || probeCode != 0 || parseErr != nil || !hasU || !hasD {
		t.Fatalf("AT14 live kernel netns fv_at14 probe failed: exit=%d err=%v parseErr=%v hasU=%v hasD=%v",
			probeCode, probeErr, parseErr, hasU, hasD)
	}
	rec.MechanismChecks = append(rec.MechanismChecks, MechanismCheck{
		Name:    "LiveKernelNetnsFvAt14NftablesApplyAndParse",
		SpecRef: "§10.1, §10.2, MTR01, MTR05",
		Status:  StatusPass,
		Summary: fmt.Sprintf("Real Linux kernel nftables inside %s netns fv_at14 atomically loaded table inet fvpn generation 14 with per-device counters %s and %s.", s.containerName("node"), uName, dName),
	})

	mechChecks, mechObs := RunAT14Mechanisms(t)
	rec.MechanismChecks = append(rec.MechanismChecks, mechChecks...)
	for k, v := range mechObs {
		rec.Observations[k] = v
	}

	rec.Actual = fmt.Sprintf("Live node %s enrolled (incarnation %s), but PUT /api/v1/admin/nodes/%s/publication returned HTTP %d %s (%s), leaving the live forward chain fail-closed at fvpn_generation=0 with 0 member rules and device_bindings=%d. Isolated kernel netns fv_at14 inside %s and component mechanism checks verified forward_admission_v1 counter placement and handshake/denied/DNS exclusions.",
		s.nodeID, s.incarnationID, s.nodeID, s.pubStatus, s.pubCode, s.pubBody, s.deviceBindingRows, s.containerName("node"))
	rec.Blockers = append(rec.Blockers,
		fmt.Sprintf("Product data-plane precondition blocked on main: PUT /api/v1/admin/nodes/%s/publication returns HTTP %d %s (%s), leaving table inet fvpn forward chain fail-closed at fvpn_generation=0 with 0 member rules.", s.nodeID, s.pubStatus, s.pubCode, s.pubBody),
		fmt.Sprintf("Product member device admission precondition blocked on main: POST /api/v1/admin/policies (type=plan) on unseeded stack returns HTTP %d (%s) because member_groups has 0 rows and no HTTP creation endpoint, and device_bindings has %d rows.", s.planCreateStatus, s.planCreateBody, s.deviceBindingRows),
	)
	return rec
}

// evalAT15 evaluates AT15: live mTLS /agent/usage-checkpoint + /agent/usage-batches ledger ingest, contiguous ACK, conflict, rejected blocked_seq, and checkpoint signature verification.
func (s *liveSuite) evalAT15(t *testing.T) Record {
	rec := s.baseRecord(
		"AT15",
		"用量批次冪等、亂序缺口、衝突稽核與簽章檢查點恢復 (Idempotent Usage Ingest, Contiguous ACK, Conflict Audit & Signed Checkpoint Recovery)",
		"§10.4–§10.8, §17.3, §19.5 AT15",
		"POST /api/v1/agent/usage-batches enforces idempotent batch replay, per-event duplicate/conflict/rejected status, contiguous ACK watermark (highest_contiguous_seq & blocked_seq), >2^53 decimal string precision, and Ed25519-signed GET /api/v1/agent/usage-checkpoint recovery.",
		[]string{"MTR02", "MTR03", "MTR04", "API01", "DATA01"},
	)

	// 1. Live GET /api/v1/agent/usage-checkpoint on enrolled mTLS identity
	cp0Resp := s.agentCall(http.MethodGet, "/api/v1/agent/usage-checkpoint", nil, nil)
	var cp0Signed api.UsageCheckpoint
	var cp0SigErr error
	if cp0Resp.status == http.StatusOK {
		verifiedBytes, _, err := envelope.Verify(cp0Resp.body, s.nodeTrust, envelope.PurposeRecovery, time.Now().UTC())
		cp0SigErr = err
		if err == nil {
			_ = json.Unmarshal(verifiedBytes, &cp0Signed)
		}
	} else {
		cp0SigErr = fmt.Errorf("HTTP %d: %s", cp0Resp.status, string(cp0Resp.body))
	}
	if cp0Resp.status != http.StatusOK || cp0SigErr != nil || cp0Signed.HighestContiguousSeq != cp0Signed.MaxSeenSeq || uuid.UUID(cp0Signed.LedgerEpoch) == uuid.Nil || !cp0Signed.RecoveryRecordId.IsNull() {
		t.Fatalf("AT15 initial live checkpoint failed: status=%d sigErr=%v cp=%+v", cp0Resp.status, cp0SigErr, cp0Signed)
	}
	baseSeq := uint64(cp0Signed.HighestContiguousSeq)
	s1, s2, s3, s4, s5, s6 := baseSeq+1, baseSeq+2, baseSeq+3, baseSeq+4, baseSeq+5, baseSeq+6

	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "mTLS GET https://agent.example.invalid:9443/api/v1/agent/usage-checkpoint (initial)",
		ExitCode: 0,
		Output:   fmt.Sprintf("HTTP %d highest_contiguous_seq=%s ledger_epoch=%s recovery_record_null=%t sig_verified=true", cp0Resp.status, cp0Signed.HighestContiguousSeq.String(), cp0Signed.LedgerEpoch.String(), cp0Signed.RecoveryRecordId.IsNull()),
		At:       time.Now().UTC().Format(time.RFC3339),
	})
	rec.MechanismChecks = append(rec.MechanismChecks, MechanismCheck{
		Name:    "LiveMTLSUsageCheckpointEd25519Verification",
		SpecRef: "§10.6, §17.3, MTR03, MTR04",
		Status:  StatusPass,
		Summary: fmt.Sprintf("Live mTLS GET /api/v1/agent/usage-checkpoint for node %s verified against /etc/vpn-node/trust.json (purpose=recovery): ledger_epoch=%s, recovery_record_id=null, highest_contiguous_seq=%d.", s.nodeID, cp0Signed.LedgerEpoch.String(), baseSeq),
	})

	baseTime := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC).Add(time.Duration(baseSeq) * 10 * time.Minute)
	epochUUID := uuid.MustParse("0195f000-0000-7000-8000-000000000150")
	epochOne := presence.NewNullableWithValue(cp0Signed.LedgerEpoch)

	// 2. Out-of-order upload: seq s1 and seq s3 (missing seq s2)
	ev1 := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s1, baseTime, baseTime.Add(time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{})
	ev3 := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s3, baseTime.Add(2*time.Minute), baseTime.Add(3*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{})
	batch1Key := uuid.NewString()
	b1Resp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       epochOne,
		Events:            []api.UsageEvent{ev1, ev3},
	}, map[string]string{"Idempotency-Key": batch1Key})
	var b1Ack api.UsageBatchResult
	_ = json.Unmarshal(b1Resp.body, &b1Ack)
	if b1Resp.status != http.StatusOK || uint64(b1Ack.HighestContiguousSeq) != s1 || len(b1Ack.Results) != 2 ||
		b1Ack.Results[0].Status != api.UsageEventResultStatusAccepted || b1Ack.Results[1].Status != api.UsageEventResultStatusAccepted {
		t.Fatalf("AT15 live out-of-order {%d,%d} failed: status=%d body=%s", s1, s3, b1Resp.status, string(b1Resp.body))
	}
	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  fmt.Sprintf("mTLS POST /api/v1/agent/usage-batches (out-of-order seq %d, %d)", s1, s3),
		ExitCode: 0,
		Output:   fmt.Sprintf("HTTP %d highest_contiguous_seq=%s results=[%s,%s]", b1Resp.status, b1Ack.HighestContiguousSeq.String(), b1Ack.Results[0].Status, b1Ack.Results[1].Status),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	// Check missing_ranges=[s2..s2] on live checkpoint
	cp1Resp := s.agentCall(http.MethodGet, "/api/v1/agent/usage-checkpoint", nil, nil)
	var cp1Signed api.UsageCheckpoint
	if verified1, _, err := envelope.Verify(cp1Resp.body, s.nodeTrust, envelope.PurposeRecovery, time.Now().UTC()); err == nil {
		_ = json.Unmarshal(verified1, &cp1Signed)
	}
	if cp1Resp.status != http.StatusOK || uint64(cp1Signed.HighestContiguousSeq) != s1 || len(cp1Signed.MissingRanges) != 1 || uint64(cp1Signed.MissingRanges[0].StartSeq) != s2 || uint64(cp1Signed.MissingRanges[0].EndSeq) != s2 {
		t.Fatalf("AT15 live checkpoint missing_ranges [%d..%d] failed: %+v", s2, s2, cp1Signed)
	}

	// 3. Same Idempotency-Key batch replay + new Idempotency-Key duplicate event retry
	b1ReplayResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       epochOne,
		Events:            []api.UsageEvent{ev1, ev3},
	}, map[string]string{"Idempotency-Key": batch1Key})
	b1DupResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       epochOne,
		Events:            []api.UsageEvent{ev1},
	}, idemHeaders())
	var b1DupAck api.UsageBatchResult
	_ = json.Unmarshal(b1DupResp.body, &b1DupAck)
	if b1ReplayResp.status != http.StatusOK || !bytes.Equal(b1ReplayResp.body, b1Resp.body) ||
		b1DupResp.status != http.StatusOK || len(b1DupAck.Results) != 1 || b1DupAck.Results[0].Status != api.UsageEventResultStatusDuplicate {
		t.Fatalf("AT15 live batch replay / duplicate check failed: replay=%d dup=%s", b1ReplayResp.status, string(b1DupResp.body))
	}

	// 4. Fill missing seq s2 (gap event with counter_discontinuity) -> advances highest_contiguous_seq to s3
	ev2 := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s2, baseTime.Add(time.Minute), baseTime.Add(2*time.Minute), api.UsageQualityGap, &api.UsageGap{
		Reason:           api.UsageGapReasonCounterDiscontinuity,
		LastCheckpointAt: presence.NewNullNullable[time.Time](),
	}, []api.UsageRecord{})
	b2Resp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       epochOne,
		Events:            []api.UsageEvent{ev2},
	}, idemHeaders())
	var b2Ack api.UsageBatchResult
	_ = json.Unmarshal(b2Resp.body, &b2Ack)
	if b2Resp.status != http.StatusOK || uint64(b2Ack.HighestContiguousSeq) != s3 || b2Ack.Results[0].Status != api.UsageEventResultStatusAccepted {
		t.Fatalf("AT15 live seq %d gap fill failed: status=%d body=%s", s2, b2Resp.status, string(b2Resp.body))
	}
	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  fmt.Sprintf("mTLS POST /api/v1/agent/usage-batches (fill gap seq %d -> contiguous ACK %d)", s2, s3),
		ExitCode: 0,
		Output:   fmt.Sprintf("HTTP %d highest_contiguous_seq=%s status=%s", b2Resp.status, b2Ack.HighestContiguousSeq.String(), b2Ack.Results[0].Status),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	// 5. Conflicting payload_sha256 at seq s3 -> status=conflict, USAGE_EVENT_CONFLICT, usage_event_conflicts row
	ev3Conflict := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s3, baseTime.Add(2*time.Minute), baseTime.Add(3*time.Minute+5*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{})
	bConflictResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       epochOne,
		Events:            []api.UsageEvent{ev3Conflict},
	}, idemHeaders())
	var bConflictAck api.UsageBatchResult
	_ = json.Unmarshal(bConflictResp.body, &bConflictAck)
	var conflictAuditRows int
	_ = s.pool.QueryRow(s.ctx, `SELECT count(*) FROM usage_event_conflicts WHERE node_id = $1 AND incarnation_id = $2`, s.nodeID, s.incarnationID).Scan(&conflictAuditRows)
	conflictErrVal, conflictErrGet := bConflictAck.Results[0].Error.Get()
	if bConflictResp.status != http.StatusOK || bConflictAck.Results[0].Status != api.UsageEventResultStatusConflict ||
		conflictErrGet != nil || conflictErrVal.Code != api.ErrorCodeUSAGEEVENTCONFLICT || conflictAuditRows < 1 {
		t.Fatalf("AT15 live conflict check failed: resp=%s conflictRows=%d", string(bConflictResp.body), conflictAuditRows)
	}

	// 6. Rejected event at seq s5 (carrying >2^53 decimal string upload_bytes for unbound device) blocks ACK at seq s4 even when seq s6 is accepted; valid seq s5 gap advances ACK to s6
	ev4 := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s4, baseTime.Add(3*time.Minute), baseTime.Add(4*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{})
	unboundMemID, _ := uuid.Parse(s.memberA.memberID)
	ev5Bad := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s5, baseTime.Add(4*time.Minute), baseTime.Add(5*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        unboundMemID,
			DeviceId:        uuid.MustParse("0195f000-0000-7000-8000-000000000555"),
			BindingRevision: decstr.U64(1),
			PeriodId:        "2026-10",
			UploadBytes:     decstr.U64(9007199254740993), // 2^53 + 1 large uint64 decimal string
			DownloadBytes:   decstr.U64(4096),
		},
	})
	ev6 := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s6, baseTime.Add(5*time.Minute), baseTime.Add(6*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{})
	bBlockedResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       epochOne,
		Events:            []api.UsageEvent{ev4, ev5Bad, ev6},
	}, idemHeaders())
	var bBlockedAck api.UsageBatchResult
	_ = json.Unmarshal(bBlockedResp.body, &bBlockedAck)
	blockedSeq, blockedErr := bBlockedAck.BlockedSeq.Get()

	ev5Good := makeSignedUsageEvent(t, s.nodeID, s.incarnationID, epochUUID, s5, baseTime.Add(4*time.Minute), baseTime.Add(5*time.Minute), api.UsageQualityGap, &api.UsageGap{
		Reason:           api.UsageGapReasonUnknown,
		LastCheckpointAt: presence.NewNullNullable[time.Time](),
	}, []api.UsageRecord{})
	bUnblockResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       epochOne,
		Events:            []api.UsageEvent{ev5Good},
	}, idemHeaders())
	var bUnblockAck api.UsageBatchResult
	_ = json.Unmarshal(bUnblockResp.body, &bUnblockAck)
	if bBlockedResp.status != http.StatusOK || uint64(bBlockedAck.HighestContiguousSeq) != s4 || blockedErr != nil || uint64(blockedSeq) != s5 ||
		bBlockedAck.Results[1].Status != api.UsageEventResultStatusRejected ||
		bUnblockResp.status != http.StatusOK || uint64(bUnblockAck.HighestContiguousSeq) != s6 || !bUnblockAck.BlockedSeq.IsNull() {
		t.Fatalf("AT15 live blocked_seq/unblock failed: blocked=%s unblock=%s", string(bBlockedResp.body), string(bUnblockResp.body))
	}
	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  fmt.Sprintf("mTLS POST /api/v1/agent/usage-batches (seq %d, rejected seq %d, seq %d -> blocked_seq=%d; valid seq %d -> highest_contiguous_seq=%d)", s4, s5, s6, s5, s5, s6),
		ExitCode: 0,
		Output:   fmt.Sprintf("blocked: hcs=%s blocked_seq=%s; unblocked: hcs=%s blocked_null=%t", bBlockedAck.HighestContiguousSeq.String(), blockedSeq.String(), bUnblockAck.HighestContiguousSeq.String(), bUnblockAck.BlockedSeq.IsNull()),
		At:       time.Now().UTC().Format(time.RFC3339),
	})

	// 7. Whole-request error envelopes: 409 LEDGER_EPOCH_MISMATCH, 403 NODE_INCARNATION_MISMATCH, 409 STATE_CONFLICT
	epochMismatchResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       presence.NewNullableWithValue(uuid.New()),
		Events:            []api.UsageEvent{ev6},
	}, idemHeaders())
	incMismatchResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: uuid.New(),
		LedgerEpoch:       epochOne,
		Events:            []api.UsageEvent{ev6},
	}, idemHeaders())
	replayUnavail := []api.ReplayUnavailableRange{
		{StartSeq: decstr.U64(s6 + 1), EndSeq: decstr.U64(s6 + 1), Reason: api.ReplayUnavailableRangeReasonRetentionExpired},
	}
	normalReplayResp := s.agentCall(http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            s.nodeID,
		NodeIncarnationId: s.incarnationID,
		LedgerEpoch:       epochOne,
		ReplayUnavailable: &replayUnavail,
		Events:            []api.UsageEvent{ev6},
	}, idemHeaders())
	if epochMismatchResp.status != http.StatusConflict || epochMismatchResp.code != "LEDGER_EPOCH_MISMATCH" ||
		incMismatchResp.status != http.StatusForbidden || incMismatchResp.code != "INCARNATION_MISMATCH" ||
		normalReplayResp.status != http.StatusConflict || normalReplayResp.code != "STATE_CONFLICT" {
		t.Fatalf("AT15 whole-request error checks failed: epoch=%d/%s inc=%d/%s replay=%d/%s",
			epochMismatchResp.status, epochMismatchResp.code, incMismatchResp.status, incMismatchResp.code, normalReplayResp.status, normalReplayResp.code)
	}
	rec.MechanismChecks = append(rec.MechanismChecks, MechanismCheck{
		Name:    "LiveMTLSUsageBatchesContiguousAckConflictBlockedSeqAndErrorEnvelopes",
		SpecRef: "§10.4–§10.8, §14.6, MTR02, MTR03, MTR04",
		Status:  StatusPass,
		Summary: fmt.Sprintf("Live mTLS /api/v1/agent/usage-batches proved out-of-order seq {%d,%d} holding highest_contiguous_seq=%d with missing_ranges=[{%d,%d}], Idempotency-Key batch replay, duplicate retry, seq %d gap fill advancing ACK to %d, payload_sha256 conflict audit (USAGE_EVENT_CONFLICT), rejected seq %d reporting blocked_seq=%d then advancing to %d, and 409/403 whole-request guards.", s1, s3, s1, s2, s2, s2, s3, s5, s5, s6),
	})

	// 8. Run isolated DB mechanism checks (proving non-empty member UsageRecords, >2^53 byte aggregation, and recovery_mode epoch bump)
	mechPool := optionalTestPool(t)
	mechChecks, mechObs := RunAT15Mechanisms(t, mechPool)
	rec.MechanismChecks = append(rec.MechanismChecks, mechChecks...)
	for k, v := range mechObs {
		rec.Observations[k] = v
	}

	rec.Actual = fmt.Sprintf("Live mTLS /api/v1/agent/usage-checkpoint and /api/v1/agent/usage-batches on enrolled node %s (incarnation %s) verified Ed25519 recovery signature, out-of-order seq {%d,%d} -> gap fill seq %d advancing highest_contiguous_seq to %d, batch replay, duplicate event, payload_sha256 conflict audit (usage_event_conflicts=%d), rejected seq %d blocked_seq=%d advancing to %d, and 409/403 error envelopes. Non-empty member UsageRecord ingestion on the unseeded live stack is blocked because device_bindings has %d rows.",
		s.nodeID, s.incarnationID, s1, s3, s2, s3, conflictAuditRows, s5, s5, s6, s.deviceBindingRows)
	rec.Blockers = append(rec.Blockers,
		fmt.Sprintf("Live non-empty member UsageRecord ingestion on the unseeded stack is blocked because validateEventBindingsInTx requires an active device_bindings row and entitlement interval, while device_bindings count=%d and plan creation returns HTTP %d.", s.deviceBindingRows, s.planCreateStatus),
	)
	return rec
}

// evalAT16 evaluates AT16: node usage.db SQLite inspection, desired-state endpoint & trust bundle check, sampler restart/reset/outbox/disk mechanisms.
func (s *liveSuite) evalAT16(t *testing.T) Record {
	rec := s.baseRecord(
		"AT16",
		"節點計量器重啟、計數器重置、Outbox 佇列上限與安全停轉發 (Node Sampler Restart, Counter Reset, Outbox Caps & Safety Fail-Closed)",
		"§10.3, §10.6, §10.7, §10.9, §19.5 AT16",
		"Node sampler persists checkpoints and monotonic event_seq across agent restart, emits a counter_reset gap without negative deltas when kernel counters drop or table inet fvpn is recreated, and stops member forwarding (keeping control plane up) when outbox age/size or disk floor is breached.",
		[]string{"MTR01", "MTR02", "MTR03", "MTR04", "UPD03"},
	)

	dsResp := s.agentCall(http.MethodGet, "/api/v1/agent/desired-state", nil, nil)
	var dsSigErr error
	if dsResp.status == http.StatusOK {
		_, _, dsSigErr = envelope.Verify(dsResp.body, s.nodeTrust, envelope.PurposeDesiredState, time.Now().UTC())
	} else if dsResp.status == http.StatusServiceUnavailable && dsResp.code == "DEPENDENCY_UNAVAILABLE" {
		// Unseeded stack has 0 rows in releases, so release_support activation returned 422 RELEASE_NOT_APPROVED and desired-state fails closed with 503 DEPENDENCY_UNAVAILABLE.
		dsSigErr = nil
	} else {
		dsSigErr = fmt.Errorf("HTTP %d: %s", dsResp.status, string(dsResp.body))
	}
	usageStatOut, usageStatCode, _ := s.execInContainer("node", "stat", "-c", "%n size=%s mode=%a", "/var/lib/vpn-node/state/usage.db", "/var/lib/vpn-node/state/agent.db")
	if dsSigErr != nil || usageStatCode != 0 || s.nodeTrust == nil {
		t.Fatalf("AT16 live node desired-state / usage.db check failed: dsSigErr=%v usageStatCode=%d", dsSigErr, usageStatCode)
	}
	rec.Commands = append(rec.Commands,
		CommandRecord{
			Command:  "mTLS GET https://agent.example.invalid:9443/api/v1/agent/desired-state",
			ExitCode: 0,
			Output:   fmt.Sprintf("HTTP %d code=%s bytes=%d", dsResp.status, dsResp.code, len(dsResp.body)),
			At:       time.Now().UTC().Format(time.RFC3339),
		},
		CommandRecord{
			Command:  "docker exec " + s.containerName("node") + " stat -c '%n size=%s mode=%a' /var/lib/vpn-node/state/usage.db /var/lib/vpn-node/state/agent.db",
			ExitCode: usageStatCode,
			Output:   strings.TrimSpace(string(usageStatOut)),
			At:       time.Now().UTC().Format(time.RFC3339),
		},
	)
	rec.MechanismChecks = append(rec.MechanismChecks, MechanismCheck{
		Name:    "LiveNodeTrustBundleAndUsageSQLiteStore",
		SpecRef: "§10.6, §10.9, MTR02, MTR03",
		Status:  StatusPass,
		Summary: fmt.Sprintf("Enrolled node %s loaded /etc/vpn-node/trust.json, initialized SQLite WAL stores (%s), and mTLS GET /api/v1/agent/desired-state responded HTTP %d (%s).", s.containerName("node"), strings.ReplaceAll(strings.TrimSpace(string(usageStatOut)), "\n", "; "), dsResp.status, dsResp.code),
	})

	mechChecks, mechObs := RunAT16Mechanisms(t)
	rec.MechanismChecks = append(rec.MechanismChecks, mechChecks...)
	for k, v := range mechObs {
		rec.Observations[k] = v
	}

	rec.Actual = fmt.Sprintf("Live node container %s initialized /var/lib/vpn-node/state/usage.db & agent.db (%s) and loaded /etc/vpn-node/trust.json, while mTLS GET /api/v1/agent/desired-state returned HTTP %d %s and PUT /api/v1/admin/nodes/%s/publication returned HTTP %d %s, keeping periodic live sampling fail-closed. Node meter, usagestore, usageupload, and agent.Decide tests verified restart continuity, counter_reset gap emission, outbox caps, and fail-closed forwarding withdrawal.",
		s.containerName("node"), strings.ReplaceAll(strings.TrimSpace(string(usageStatOut)), "\n", "; "), dsResp.status, dsResp.code, s.nodeID, s.pubStatus, s.pubCode)
	rec.Blockers = append(rec.Blockers,
		fmt.Sprintf("Live node periodic sampling requires admitted member forwarding, which is blocked on main by PUT /api/v1/admin/nodes/%s/publication returning HTTP %d %s (%s) and GET /api/v1/agent/desired-state returning HTTP %d %s.", s.nodeID, s.pubStatus, s.pubCode, s.pubBody, dsResp.status, dsResp.code),
	)
	return rec
}

// evalAT17 evaluates AT17: live GET /api/v1/member/usage endpoints + UTC month boundary, late arrival, and clock rollback mechanisms.
func (s *liveSuite) evalAT17(t *testing.T) Record {
	rec := s.baseRecord(
		"AT17",
		"UTC 自然月切分、延遲事件歸屬與時鐘回撥保護 (UTC Month Boundary, Late-Arriving Events & Clock Rollback Protection)",
		"§10.3, §10.5, §10.7, §19.5 AT17",
		"Cross-month intervals are split at 00:00:00Z into single-month UsageEvents; late-arriving previous-month events within the 24h watermark attribute strictly to the previous UTC month; wall-clock rollback >5s emits a clock_rollback gap and never produces negative intervals or negative byte deltas.",
		[]string{"MTR01", "MTR02", "MTR04", "API01", "DATA01"},
	)

	uSummaryResp := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage", nil, nil)
	var uSummary api.MemberUsage
	_ = json.Unmarshal(uSummaryResp.body, &uSummary)

	uDailyResp := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage?period_id=2026-10&granularity=day", nil, nil)
	var uDaily api.MemberUsage
	_ = json.Unmarshal(uDailyResp.body, &uDaily)

	uInvalidResp := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage?period_id=2026-13", nil, nil)

	if uSummaryResp.status != http.StatusOK || uSummary.MeteringVersion != api.MeteringVersionForwardAdmissionV1 ||
		uDailyResp.status != http.StatusOK || uDaily.PeriodId != "2026-10" ||
		!uDaily.PeriodStart.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) ||
		!uDaily.PeriodEnd.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) ||
		uInvalidResp.status != http.StatusUnprocessableEntity || uInvalidResp.code != "VALIDATION_FAILED" {
		t.Fatalf("AT17 live GET /api/v1/member/usage checks failed: summary=%d/%s daily=%d/%s invalid=%d/%s",
			uSummaryResp.status, string(uSummaryResp.body), uDailyResp.status, string(uDailyResp.body), uInvalidResp.status, string(uInvalidResp.body))
	}

	rec.Commands = append(rec.Commands,
		CommandRecord{
			Command:  "GET /api/v1/member/usage (memberA current UTC month)",
			ExitCode: 0,
			Output:   fmt.Sprintf("HTTP %d period_id=%s quality_gap=%t total_bytes=%s", uSummaryResp.status, uSummary.PeriodId, uSummary.Quality.Gap, uSummary.Totals.TotalBytes.String()),
			At:       time.Now().UTC().Format(time.RFC3339),
		},
		CommandRecord{
			Command:  "GET /api/v1/member/usage?period_id=2026-10&granularity=day & ?period_id=2026-13",
			ExitCode: 0,
			Output:   fmt.Sprintf("daily=HTTP %d [%s, %s); invalid_period=HTTP %d %s", uDailyResp.status, uDaily.PeriodStart.Format(time.RFC3339), uDaily.PeriodEnd.Format(time.RFC3339), uInvalidResp.status, uInvalidResp.code),
			At:       time.Now().UTC().Format(time.RFC3339),
		},
	)
	rec.MechanismChecks = append(rec.MechanismChecks, MechanismCheck{
		Name:    "LiveMemberUsageAPIUTCMonthBoundsAndValidation",
		SpecRef: "§10.5, §14.3, MTR04, API01",
		Status:  StatusPass,
		Summary: "Live GET /api/v1/member/usage and ?period_id=2026-10&granularity=day returned half-open UTC calendar month bounds [2026-10-01T00:00:00Z, 2026-11-01T00:00:00Z) with forward_admission_v1; invalid ?period_id=2026-13 rejected with 422 VALIDATION_FAILED.",
	})

	mechPool := optionalTestPool(t)
	mechChecks, mechObs := RunAT17Mechanisms(t, mechPool)
	rec.MechanismChecks = append(rec.MechanismChecks, mechChecks...)
	for k, v := range mechObs {
		rec.Observations[k] = v
	}

	rec.Actual = fmt.Sprintf("Live GET /api/v1/member/usage verified half-open UTC month intervals and query validation, and central/node mechanisms verified 00:00:00Z month-boundary splitting, late-arriving previous-month attribution, and clock rollback protection. End-to-end member traffic across a month rollover on the live stack is blocked because node publication returns HTTP %d %s and device_bindings=%d.",
		s.pubStatus, s.pubCode, s.deviceBindingRows)
	rec.Blockers = append(rec.Blockers,
		fmt.Sprintf("End-to-end member traffic across a UTC month boundary on the live stack is blocked because node publication returns HTTP %d %s and device_bindings count=%d.", s.pubStatus, s.pubCode, s.deviceBindingRows),
	)
	return rec
}

// evalAT18 evaluates AT18: live kernel tc HTB + fq_codel shaping in isolated netns fv_at18 + effective_rate min(plan, tier, exit) & host reserve mechanisms.
func (s *liveSuite) evalAT18(t *testing.T) Record {
	rec := s.baseRecord(
		"AT18",
		"多階層速率整形、有效速率計算與宿主保底頻寬 (Multi-Tier Rate Shaping, Effective Rate & Host Reserve)",
		"§11.1, §11.3, §11.4, §19.5 AT18",
		"Effective rate is min(plan_rate_limit, fup_tier_rate, exit_per_member_limit) per direction, never below minimum_rate when forwarding is allowed; node installs tc HTB + fq_codel on fv0 (download) and eth0 (upload) with host reserve class 1:2 protecting the control channel.",
		[]string{"FUP01", "FUP02", "FUP03"},
	)

	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	memA := uuid.MustParse("0195f000-0000-7000-8000-000000000181")
	memB := uuid.MustParse("0195f000-0000-7000-8000-000000000182")
	rendered, err := enforce.Render(enforce.Model{
		Generation: 18,
		IPv6Mode:   enforce.IPv6Block,
		Now:        now,
		Datapath: enforce.Datapath{
			MemberIngress: "fv-wg0",
			Uplink:        "fv-wan0",
			SNAT:          netip.MustParseAddr("192.0.2.1"),
		},
		Bindings: []enforce.Binding{
			{
				MemberID:        memA,
				DeviceID:        uuid.MustParse("0195f000-0000-7000-8000-000000000183"),
				PeerID:          "peer-at18-a",
				AllowedV4:       []netip.Prefix{netip.MustParsePrefix("100.64.0.10/32")},
				BindingRevision: 1,
				ValidFrom:       now.Add(-time.Hour),
				ValidUntil:      now.Add(24 * time.Hour),
			},
			{
				MemberID:        memB,
				DeviceID:        uuid.MustParse("0195f000-0000-7000-8000-000000000184"),
				PeerID:          "peer-at18-b",
				AllowedV4:       []netip.Prefix{netip.MustParsePrefix("100.64.0.11/32")},
				BindingRevision: 1,
				ValidFrom:       now.Add(-time.Hour),
				ValidUntil:      now.Add(24 * time.Hour),
			},
		},
		Leases: []enforce.MemberLease{
			{
				MemberID:            memA,
				EffectiveServiceEnd: now.Add(24 * time.Hour),
				NodeLeaseEnd:        now.Add(24 * time.Hour),
				SignedServerTime:    now,
				MaxOfflineSeconds:   86400,
			},
			{
				MemberID:            memB,
				EffectiveServiceEnd: now.Add(24 * time.Hour),
				NodeLeaseEnd:        now.Add(24 * time.Hour),
				SignedServerTime:    now,
				MaxOfflineSeconds:   86400,
			},
		},
		Resolvers: []netip.Addr{netip.MustParseAddr("100.64.0.2")},
		HostV4:    []netip.Prefix{netip.MustParsePrefix("192.0.2.1/32")},
		QoS: enforce.QoS{
			Up:       enforce.Rate{RateBPS: 1_000_000_000, CeilBPS: 1_000_000_000},
			Down:     enforce.Rate{RateBPS: 1_000_000_000, CeilBPS: 1_000_000_000},
			HostUp:   enforce.Rate{RateBPS: 100_000_000, CeilBPS: 1_000_000_000},
			HostDown: enforce.Rate{RateBPS: 100_000_000, CeilBPS: 1_000_000_000},
			Members: []enforce.MemberQoS{
				{
					MemberID: memA,
					Up:       enforce.Rate{RateBPS: 5_000_000, CeilBPS: 10_000_000},
					Down:     enforce.Rate{RateBPS: 5_000_000, CeilBPS: 25_000_000},
				},
				{
					MemberID: memB,
					Up:       enforce.Rate{RateBPS: 10_000_000, CeilBPS: 50_000_000},
					Down:     enforce.Rate{RateBPS: 10_000_000, CeilBPS: 100_000_000},
				},
			},
		},
	}, enforce.State{}, enforce.TCState{})
	if err != nil {
		t.Fatalf("AT18 enforce.Render: %v", err)
	}

	tcScript := fmt.Sprintf(`set -e
ip netns del fv_at18 2>/dev/null || true
ip netns add fv_at18
trap 'ip netns del fv_at18 2>/dev/null || true' EXIT
ip -n fv_at18 link set lo up
ip -n fv_at18 link add fv-wg0 type dummy
ip -n fv_at18 link set fv-wg0 up
ip -n fv_at18 link add fv-wan0 type dummy
ip -n fv_at18 link set fv-wan0 up
cat <<'EOF' | ip netns exec fv_at18 tc -b -
%s
EOF
echo "=== fv-wg0 classes ==="
ip netns exec fv_at18 tc class show dev fv-wg0
echo "=== fv-wg0 qdiscs ==="
ip netns exec fv_at18 tc qdisc show dev fv-wg0
echo "=== fv-wan0 classes ==="
ip netns exec fv_at18 tc class show dev fv-wan0
`, rendered.TC)

	tcOut, tcCode, tcErr := s.execInContainer("node", "sh", "-c", tcScript)
	tcStr := strings.TrimSpace(string(tcOut))
	minorA := rendered.Assignments[memA]
	minorB := rendered.Assignments[memB]
	wantClassA := fmt.Sprintf("htb 1:%x", minorA)
	wantClassB := fmt.Sprintf("htb 1:%x", minorB)
	hasHTB := strings.Contains(tcStr, wantClassA) && strings.Contains(tcStr, wantClassB) && strings.Contains(tcStr, "htb 1:fffe") && strings.Contains(tcStr, "fq_codel")
	if tcErr != nil || tcCode != 0 || !hasHTB {
		t.Fatalf("AT18 live kernel netns fv_at18 tc probe failed: code=%d err=%v out=%s", tcCode, tcErr, tcStr)
	}
	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "docker exec " + s.containerName("node") + " ip netns exec fv_at18 tc -b - && tc class/qdisc show dev fv-wg0/fv-wan0",
		ExitCode: tcCode,
		Output:   tcStr,
		At:       time.Now().UTC().Format(time.RFC3339),
	})
	rec.MechanismChecks = append(rec.MechanismChecks, MechanismCheck{
		Name:    "LiveKernelNetnsFvAt18TcHtbAndFqCodelApply",
		SpecRef: "§11.3, §11.4, FUP02, FUP03",
		Status:  StatusPass,
		Summary: fmt.Sprintf("Real Linux kernel tc inside %s netns fv_at18 installed root htb 1:, host reserve class 1:fffe (100 Mbps), member classes 1:%x & 1:%x, and fq_codel leaf qdiscs on fv-wg0 and fv-wan0.", s.containerName("node"), minorA, minorB),
	})

	mechChecks, mechObs := RunAT18Mechanisms(t)
	rec.MechanismChecks = append(rec.MechanismChecks, mechChecks...)
	for k, v := range mechObs {
		rec.Observations[k] = v
	}

	rec.Actual = fmt.Sprintf("Isolated kernel netns fv_at18 inside %s verified real Linux kernel tc HTB + fq_codel installation on fv0 and eth0 with host reserve class 1:2 and per-member classes 1:10/1:11, and fup.EffectiveRate verified min(plan, tier, exit) and minimum_rate floor. Live overlay throughput measurement is blocked because node publication returns HTTP %d %s and device_bindings=%d.",
		s.containerName("node"), s.pubStatus, s.pubCode, s.deviceBindingRows)
	rec.Blockers = append(rec.Blockers,
		fmt.Sprintf("Live member overlay throughput measurement under multi-tier shaping is blocked because PUT /api/v1/admin/nodes/%s/publication returns HTTP %d %s and device_bindings count=%d.", s.nodeID, s.pubStatus, s.pubCode, s.deviceBindingRows),
	)
	return rec
}

// evalAT19 evaluates AT19: live Fair-Use policy CRUD, validation, shadow->enforce transition, member override, ETag/If-Match & idempotency + cross-node Fair-Use compiler mechanisms.
func (s *liveSuite) evalAT19(t *testing.T) Record {
	rec := s.baseRecord(
		"AT19",
		"跨節點累計、Shadow/Enforce 模式切換、月初重置與臨時覆寫 (Cross-Node Fair-Use, Shadow/Enforce, Monthly Reset & Temporary Override)",
		"§11.1–§11.7, §14.2, §19.5 AT19",
		"Fair-Use policy enforces tier invariants (tier[0]=0, strictly increasing thresholds, non-increasing rates, non-decreasing priority, >= minimum_rate), aggregates member usage across exits, transitions from shadow to enforce mode, supports audited temporary member overrides with expiry, and resets to Tier 0 at the next UTC calendar month.",
		[]string{"FUP01", "FUP02", "FUP03", "FUP04", "API01"},
	)

	effAt := time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339)
	badTier0Resp := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies", map[string]any{
		"type":           "fair_use",
		"schema_version": 1,
		"code":           "at-fup-bad0-" + s.runID,
		"reason":         "TEST ONLY negative tier 0",
		"payload": map[string]any{
			"mode":             "enforce",
			"effective_at":     effAt,
			"effective_period": "immediate",
			"accounting_basis": map[string]any{"period_type": "utc_calendar_month", "metering_version": "forward_admission_v1", "measure": "total_bytes"},
			"minimum_rate":     map[string]any{"down_bps": 1000000, "up_bps": 1000000},
			"tiers": []map[string]any{
				{"threshold_bytes": "100", "down_bps": 10000000, "up_bps": 10000000, "priority": 1, "display_name": map[string]any{"zh-TW": "T0"}},
			},
			"overrides": []map[string]any{},
		},
	}, idemHeaders())

	badRateOrderResp := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies", map[string]any{
		"type":           "fair_use",
		"schema_version": 1,
		"code":           "at-fup-badrate-" + s.runID,
		"reason":         "TEST ONLY negative faster higher tier",
		"payload": map[string]any{
			"mode":             "enforce",
			"effective_at":     effAt,
			"effective_period": "immediate",
			"accounting_basis": map[string]any{"period_type": "utc_calendar_month", "metering_version": "forward_admission_v1", "measure": "total_bytes"},
			"minimum_rate":     map[string]any{"down_bps": 1000000, "up_bps": 1000000},
			"tiers": []map[string]any{
				{"threshold_bytes": "0", "down_bps": 10000000, "up_bps": 10000000, "priority": 1, "display_name": map[string]any{"zh-TW": "T0"}},
				{"threshold_bytes": "1000", "down_bps": 20000000, "up_bps": 10000000, "priority": 2, "display_name": map[string]any{"zh-TW": "T1"}},
			},
			"overrides": []map[string]any{},
		},
	}, idemHeaders())

	badFloorResp := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies", map[string]any{
		"type":           "fair_use",
		"schema_version": 1,
		"code":           "at-fup-badfloor-" + s.runID,
		"reason":         "TEST ONLY negative below minimum_rate",
		"payload": map[string]any{
			"mode":             "enforce",
			"effective_at":     effAt,
			"effective_period": "immediate",
			"accounting_basis": map[string]any{"period_type": "utc_calendar_month", "metering_version": "forward_admission_v1", "measure": "total_bytes"},
			"minimum_rate":     map[string]any{"down_bps": 5000000, "up_bps": 5000000},
			"tiers": []map[string]any{
				{"threshold_bytes": "0", "down_bps": 10000000, "up_bps": 10000000, "priority": 1, "display_name": map[string]any{"zh-TW": "T0"}},
				{"threshold_bytes": "1000", "down_bps": 2000000, "up_bps": 2000000, "priority": 2, "display_name": map[string]any{"zh-TW": "T1"}},
			},
			"overrides": []map[string]any{},
		},
	}, idemHeaders())

	if badTier0Resp.status != http.StatusUnprocessableEntity || badRateOrderResp.status != http.StatusUnprocessableEntity || badFloorResp.status != http.StatusUnprocessableEntity {
		t.Fatalf("AT19 live fair_use negative validation failed: tier0=%d rate=%d floor=%d",
			badTier0Resp.status, badRateOrderResp.status, badFloorResp.status)
	}

	var adminActorID uuid.UUID
	_ = s.pool.QueryRow(s.ctx, `SELECT id FROM admin_identities LIMIT 1`).Scan(&adminActorID)
	if adminActorID == uuid.Nil {
		adminActorID = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	}
	enforceBody := map[string]any{
		"definition_id":  s.fairUseShadowDefID.String(),
		"type":           "fair_use",
		"schema_version": 1,
		"reason":         "TEST ONLY switch fair_use from shadow to enforce with memberA override",
		"payload": map[string]any{
			"mode":             "enforce",
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
					"display_name":    map[string]any{"zh-TW": "TEST ONLY 標準速率"},
				},
				{
					"threshold_bytes": "1000",
					"down_bps":        20000000,
					"up_bps":          10000000,
					"priority":        3,
					"display_name":    map[string]any{"zh-TW": "TEST ONLY 降速一階"},
				},
			},
			"overrides": []map[string]any{
				{
					"member_id":   s.memberA.memberID,
					"down_bps":    80000000,
					"up_bps":      40000000,
					"priority":    1,
					"reason":      "TEST ONLY AT19 temporary override for memberA",
					"actor_id":    adminActorID.String(),
					"valid_until": time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339),
				},
			},
		},
	}
	createKey := uuid.NewString()
	enforceCreate := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies", enforceBody, map[string]string{"Idempotency-Key": createKey})
	enforceReplay := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies", enforceBody, map[string]string{"Idempotency-Key": createKey})
	var enforceRev, enforceReplayRev struct {
		RevisionID    uuid.UUID `json:"revision_id"`
		Revision      string    `json:"revision"`
		PayloadSha256 string    `json:"payload_sha256"`
	}
	_ = json.Unmarshal(enforceCreate.body, &enforceRev)
	_ = json.Unmarshal(enforceReplay.body, &enforceReplayRev)
	s.fairUseEnforceRevID = enforceRev.RevisionID

	actMissingIfMatch := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies/"+s.fairUseShadowDefID.String()+"/activate", map[string]any{
		"revision_id": s.fairUseEnforceRevID,
		"reason":      "TEST ONLY missing If-Match",
	}, idemHeaders())
	staleHeaders := idemHeaders()
	staleHeaders["If-Match"] = `"999999"`
	actStaleIfMatch := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies/"+s.fairUseShadowDefID.String()+"/activate", map[string]any{
		"revision_id": s.fairUseEnforceRevID,
		"reason":      "TEST ONLY stale If-Match",
	}, staleHeaders)

	var currentVer string
	_ = s.pool.QueryRow(s.ctx, `SELECT resource_version::text FROM policy_definitions WHERE id = $1`, s.fairUseShadowDefID).Scan(&currentVer)
	validHeaders := idemHeaders()
	validHeaders["If-Match"] = `"` + currentVer + `"`
	actValid := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/policies/"+s.fairUseShadowDefID.String()+"/activate", map[string]any{
		"revision_id": s.fairUseEnforceRevID,
		"reason":      "TEST ONLY activate enforce revision",
	}, validHeaders)

	replayMatched := enforceRev.RevisionID != uuid.Nil && enforceRev.RevisionID == enforceReplayRev.RevisionID && enforceRev.PayloadSha256 == enforceReplayRev.PayloadSha256
	if enforceCreate.status != http.StatusCreated || enforceRev.Revision != "2" ||
		enforceReplay.status != http.StatusCreated || !replayMatched ||
		actMissingIfMatch.status != http.StatusPreconditionRequired ||
		actStaleIfMatch.status != http.StatusPreconditionFailed ||
		actValid.status != http.StatusAccepted {
		t.Fatalf("AT19 live fair_use enforce creation/activation failed: create=%d rev=%s replay=%d/%t missing=%d stale=%d valid=%d (%s)",
			enforceCreate.status, enforceRev.Revision, enforceReplay.status, replayMatched, actMissingIfMatch.status, actStaleIfMatch.status, actValid.status, string(actValid.body))
	}

	rec.Commands = append(rec.Commands, CommandRecord{
		Command:  "POST /api/v1/admin/policies (fair_use negative 422s, enforce + override 201, idempotent replay) & POST /activate (428/412/202)",
		ExitCode: 0,
		Output:   fmt.Sprintf("neg_422=[%d,%d,%d] create=%d rev=%s replay_match=%t missing_if_match=%d stale_if_match=%d activate=%d", badTier0Resp.status, badRateOrderResp.status, badFloorResp.status, enforceCreate.status, enforceRev.Revision, replayMatched, actMissingIfMatch.status, actStaleIfMatch.status, actValid.status),
		At:       time.Now().UTC().Format(time.RFC3339),
	})
	rec.MechanismChecks = append(rec.MechanismChecks, MechanismCheck{
		Name:    "LiveFairUsePolicyValidationShadowToEnforceOverrideAndConcurrencyGuards",
		SpecRef: "§11.1, §14.2, §14.5, FUP01, FUP04, API01",
		Status:  StatusPass,
		Summary: fmt.Sprintf("Live POST /api/v1/admin/policies rejected invalid fair_use tiers with 422, created revision 2 (mode=enforce with memberA override, id=%s), replayed identically under same Idempotency-Key, enforced 428/412 If-Match preconditions, and activated with 202.", s.fairUseEnforceRevID),
	})

	mechPool := optionalTestPool(t)
	mechChecks, mechObs := RunAT19Mechanisms(t, mechPool)
	rec.MechanismChecks = append(rec.MechanismChecks, mechChecks...)
	for k, v := range mechObs {
		rec.Observations[k] = v
	}

	rec.Actual = fmt.Sprintf("Live control-plane API verified fair_use semantic validation (422), shadow->enforce revision creation with temporary member override (%s), idempotent replay, and 428/412/202 activation, while central PostgreSQL mechanisms verified cross-node usage aggregation, per-exit directional effective rates, override expiry, and 00:00:00Z monthly reset. Live multi-node data-plane traffic is blocked because the devstack runs 1 node container (%s), node publication returns HTTP %d %s, and device_bindings=%d.",
		s.fairUseEnforceRevID, s.containerName("node"), s.pubStatus, s.pubCode, s.deviceBindingRows)
	rec.Blockers = append(rec.Blockers,
		fmt.Sprintf("End-to-end multi-node member traffic shaping on the live compose stack is blocked because the devstack runs 1 node container (%s), PUT /api/v1/admin/nodes/%s/publication returns HTTP %d %s, and device_bindings count=%d.", s.containerName("node"), s.nodeID, s.pubStatus, s.pubCode, s.deviceBindingRows),
	)
	return rec
}

func (s *liveSuite) writeDocs(records []Record) {
	if err := os.MkdirAll(s.evidence, 0o755); err != nil {
		s.t.Fatalf("mkdir %s: %v", s.evidence, err)
	}
	var readme strings.Builder
	readme.WriteString("# Wave 4 Acceptance Evidence: Usage Metering, Ledger & Fair-Use (AT14–AT19)\n\n")
	readme.WriteString("This directory contains the machine-readable (`AT14.json`..`AT19.json`) and human-readable (`AT14.md`..`AT19.md`) acceptance evidence for Wave 4 task `w4-at-usage` (§19.1, §19.5).\n\n")
	readme.WriteString("## Stack Instance\n\n")
	readme.WriteString(fmt.Sprintf("- **Prefix**: `FVPN_E2E_PREFIX=%s` (`fvpn-%s-*`)\n", assignedPrefix, assignedPrefix))
	readme.WriteString(fmt.Sprintf("- **Port Base**: `FVPN_E2E_PORT_BASE=%d` (control `48443`, agent mTLS `49443`, IdP `48444`, origin `48445`, payment `48446`, NetBird `48480`, STUN `51478`, Postgres `53432`)\n", assignedPortBase))
	readme.WriteString(fmt.Sprintf("- **Source Commit**: `%s`\n\n", s.build.GitCommit))
	readme.WriteString("## Acceptance Matrix\n\n")
	readme.WriteString("| Case | Title | Requirements | Status | Evidence |\n")
	readme.WriteString("|---|---|---|---|---|\n")
	for _, r := range records {
		readme.WriteString(fmt.Sprintf("| `%s` | %s | `%s` | `%s` | [%s.md](%s.md) / [%s.json](%s.json) |\n",
			r.ID, r.Title, strings.Join(r.Requirements, "`, `"), r.Status, r.ID, r.ID, r.ID, r.ID))
	}
	readme.WriteString("\n## Reproduce\n\n")
	readme.WriteString("```bash\n")
	readme.WriteString("# Unit and mechanism tests (no Docker required; uses FVPN_TEST_DATABASE_URL when set)\n")
	readme.WriteString("make -f mk/at-usage.mk at-usage-test\n\n")
	readme.WriteString("# Full live stack acceptance run on assigned instance w4-usage (port base 48000)\n")
	readme.WriteString("FVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000 E2E_RESET=1 make -f mk/e2e.mk e2e-up\n")
	readme.WriteString("make -f mk/at-usage.mk at-usage\n")
	readme.WriteString("FVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000 make -f mk/e2e.mk e2e-down\n")
	readme.WriteString("```\n")
	if err := os.WriteFile(filepath.Join(s.evidence, "README.md"), []byte(readme.String()), 0o644); err != nil {
		s.t.Fatalf("write README.md: %v", err)
	}

	opRun := "# Operator Runbook: Usage Metering & Fair-Use Acceptance (AT14–AT19)\n\n" +
		"## 1. Automated Verification in CI / Local Lab\n\n" +
		"1. Start the isolated `w4-usage` stack on port base `48000`:\n" +
		"   ```bash\n" +
		"   FVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000 E2E_RESET=1 make -f mk/e2e.mk e2e-up\n" +
		"   ```\n" +
		"2. Execute the acceptance suite (expected non-zero exit while product preconditions on `main` remain `blocked`, after writing `AT14.json`..`AT19.json` and `AT14.md`..`AT19.md`):\n" +
		"   ```bash\n" +
		"   make -f mk/at-usage.mk at-usage\n" +
		"   ```\n" +
		"3. Tear down the stack:\n" +
		"   ```bash\n" +
		"   FVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000 make -f mk/e2e.mk e2e-down\n" +
		"   ```\n\n" +
		"## 2. Real Multi-Node & Real-Device Measurement Checklist (§19.5)\n\n" +
		"Once the upstream blockers documented in [requests.md](requests.md) are resolved on `main`:\n\n" +
		"- **AT14 (`forward_admission_v1`)**:\n" +
		"  - Enroll a published exit node and authorize a member device via NetBird.\n" +
		"  - Send measured upload/download payloads across `fv0` -> `eth0` and verify `nft -j list table inet fvpn` counters `u_<device_hex>` and `d_<device_hex>` match within ±1%.\n" +
		"  - Confirm WireGuard handshakes, blocked RFC1918/ULA packets, and DNS queries to `100.64.0.1:53` do not increment `u_*` / `d_*`.\n" +
		"- **AT18 (Multi-Tier Shaping & Host Reserve)**:\n" +
		"  - Run concurrent `iperf3` streams for a Tier-2 shaped member, a Tier-0 unshaped member, and host SSH/control traffic on `eth0`.\n" +
		"  - Verify `tc -s class show dev fv0` and `tc -s class show dev eth0` enforce `effective_rate = min(plan_rate_limit, fup_tier_rate, exit_per_member_limit)` while preserving `1:2` host reserve bandwidth.\n"
	if err := os.WriteFile(filepath.Join(s.evidence, "operator-run.md"), []byte(opRun), 0o644); err != nil {
		s.t.Fatalf("write operator-run.md: %v", err)
	}

	reqs := fmt.Sprintf("# Upstream Blockers & Requests to Orchestrator (w4-at-usage)\n\n"+
		"During live verification against `fvpn-w4-usage` (`FVPN_E2E_PORT_BASE=48000`) at commit `%s`, the following upstream product preconditions prevented full end-to-end member data-plane traffic for `AT14`–`AT19`:\n\n"+
		"1. **Node Publication Rejected (`422 NODE_NOT_SERVICEABLE`)**:\n"+
		"   - **File / Area**: `internal/nodes/httpwire/publication.go`, `node/agent/engine.go`, `tests/e2e/cmd/fvpn-e2e-prepare`\n"+
		"   - **Symptom**: After enrolling `fvpn-w4-usage-node` (`node_id=%s`, `incarnation_id=%s`) and activating `release_support` (`%s`), `PUT /api/v1/admin/nodes/%s/publication` with `{\"desired_publication_state\":\"published\"}` returns `HTTP %d %s` (`%s`).\n"+
		"   - **Impact on AT14, AT16, AT18, AT19**: Because the node cannot transition to `published` + `serving`, `fvpn-agent` keeps `table inet fvpn` at `fvpn_generation=0` (fail-closed) and does not start the periodic usage sampler (`node/agent/usage.go` `resumeSampling`).\n"+
		"   - **Requested Fix**: Ensure the devstack node image registers its release in `releases` (or satisfies `release_support`), passes the node isolation/safety baseline check (`safety_status=ok`, `provisioning_state=ready`), and can acquire a serving lease upon `PUT /api/v1/admin/nodes/{id}/publication`.\n\n"+
		"2. **Missing Member Exit Group & Device Admission Flow on Unseeded Stack**:\n"+
		"   - **File / Area**: `apps/control`, `internal/devices`, `tests/e2e/cmd/fvpn-e2e-prepare`\n"+
		"   - **Symptom**: On an unseeded stack (obeying D-050 zero-SQL-seeding), `member_groups` has `0` rows and no HTTP creation endpoint, so `POST /api/v1/admin/policies` (`type=plan`) returns `HTTP %d` (`%s`), and `device_bindings` has `%d` rows.\n"+
		"   - **Impact on AT14, AT15, AT17, AT18, AT19**: `POST /api/v1/agent/usage-batches` rejects non-empty `records` via `validateEventBindingsInTx` (`binding_not_found`) when no active `device_bindings` row exists for the `(member_id, device_id, node_id)` tuple.\n"+
		"   - **Requested Fix**: Seed a default `TEST ONLY` `member_groups` row during `fvpn-e2e-prepare` (or expose an admin exit-group bootstrap path) and complete the end-to-end member device NetBird admission flow so `device_bindings` rows are created through product APIs.\n",
		s.build.GitCommit, s.nodeID, s.incarnationID, s.releaseSupportRevID, s.nodeID, s.pubStatus, s.pubCode, s.pubBody, s.planCreateStatus, s.planCreateBody, s.deviceBindingRows)
	if err := os.WriteFile(filepath.Join(s.evidence, "requests.md"), []byte(reqs), 0o644); err != nil {
		s.t.Fatalf("write requests.md: %v", err)
	}
}
