//go:build e2e && integration

package secops

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	e2e "github.com/TED-H-com/Freedom-VPN/tests/e2e"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/idplogin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const livePrefix = "fvpn-w4-secops-"
const livePublic = "https://portal.example.invalid:8443"
const liveAgent = "https://agent.example.invalid:9443"

type liveSession struct {
	client       *http.Client
	csrf, member string
	cred         idplogin.Cred
}
type liveReply struct {
	status int
	code   string
	body   []byte
	header http.Header
}
type liveProbe struct {
	t                     *testing.T
	root, state           string
	tr                    *http.Transport
	db                    *pgx.Conn
	observations          []map[string]any
	metadata              map[string]any
	secrets               [][]byte
	groupID, planRevision uuid.UUID
	bindings              map[uuid.UUID]api.DeviceBinding
	securityComplete      bool
}

// This probe consumes the existing dev-stack state and documented IdP/enrollment
// hooks. It never prepares a second implementation of the lab (§19.1).
func TestAT28_AT31_LiveStack(t *testing.T) {
	if os.Getenv("FVPN_SECOPS_LIVE") != "1" {
		t.Skip("requires assigned running w4-secops dev stack; supporting tests are separate")
	}
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal("repository root")
	}
	p := &liveProbe{t: t, root: root, state: filepath.Join(root, ".agent/e2e-state-w4-secops"), metadata: map[string]any{"started_at": time.Now().UTC(), "hardware": "shared development host; not candidate hardware"}}
	p.open()
	defer p.db.Close(context.Background())
	defer p.tr.CloseIdleConnections()
	defer p.finish()
	admin := p.login("e2e-admin", "admin")
	members := make([]*liveSession, 20)
	for i := range members {
		members[i] = p.login(fmt.Sprintf("secops-member-%02d", i), "member")
		time.Sleep(2 * time.Second)
	}
	p.metadata["members_authenticated"] = len(members)
	p.securityComplete = true
	p.roles(admin, members[0])
	p.planInputs(admin)
	p.grantMembers(admin, members)
	p.releaseSupport(admin)
	nodeA := p.enrollAgent(admin)
	nodeB, nodeClient := p.enrollHost(admin)
	p.identities(admin, nodeClient, nodeB, nodeA)
	devices := p.admitMember(members)
	p.ownership(admin, members, devices)
	p.serving(admin, nodeA)
	p.concurrentLogins(members)
	if p.metadata["counted_flow"] == true {
		p.load(members)
	} else {
		p.metadata["capacity_subset"] = map[string]any{"status": "blocked", "reason": "Follow-up 2: no capacity measurement without a counted serving flow; prior API diagnostics remain historical"}
	}
	if os.Getenv("FVPN_SECOPS_FOLLOWUP3") != "1" {
		p.ledger(admin, nodeClient, nodeB, members, devices)
	} else {
		p.metadata["physical_restore"] = "Historical Follow-up 2 record preserved; command and migration compatibility reviewed separately. No new restore or replay measurement."
	}
	if len(devices[0]) > 0 {
		p.memberOperation(members)
	}
	p.revoke(admin, nodeClient, nodeB)
	if os.Getenv("FVPN_SECOPS_LOGIN_BURST") == "1" {
		p.loginThresholds(members)
	}
}

func (p *liveProbe) open() {
	revision, e := exec.Command("git", "rev-parse", "HEAD").Output()
	if e != nil {
		p.t.Fatal("source revision")
	}
	p.metadata["source_revision"] = strings.TrimSpace(string(revision))
	sourceFiles, er := filepath.Glob(filepath.Join(p.root, "tests/e2e/at/secops/*.go"))
	if er != nil {
		p.t.Fatal("probe source inventory")
	}
	hash := sha256.New()
	for _, name := range sourceFiles {
		raw, e := os.ReadFile(name)
		if e != nil {
			p.t.Fatal("probe source digest")
		}
		_, _ = hash.Write(raw)
	}
	p.metadata["probe_source_sha256"] = fmt.Sprintf("%x", hash.Sum(nil))
	raw, err := os.ReadFile(filepath.Join(p.state, "stack.json"))
	if err != nil {
		p.t.Fatal("stack state absent")
	}
	var stack struct {
		Prefix string         `json:"prefix"`
		Ports  map[string]int `json:"host_ports"`
	}
	if json.Unmarshal(raw, &stack) != nil || stack.Prefix != "fvpn-w4-secops" || stack.Ports["control"] != 30443 || stack.Ports["postgres"] != 35432 {
		p.t.Fatal("refusing unassigned stack")
	}
	ca, err := os.ReadFile(filepath.Join(p.state, "ca.pem"))
	if err != nil {
		p.t.Fatal("lab TLS CA absent")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca) {
		p.t.Fatal("lab TLS CA invalid")
	}
	dial := map[string]int{"portal.example.invalid:8443": 30443, "agent.example.invalid:9443": 31443, "idp.example.invalid:443": 30444, "netbird.example.invalid:443": stack.Ports["gateway"], "releases.example.invalid:443": stack.Ports["origin"], "api.stripe.com:443": stack.Ports["payment"]}
	p.tr = &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots}, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		port, ok := dial[address]
		if !ok {
			return nil, fmt.Errorf("destination outside controlled lab")
		}
		return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, fmt.Sprintf("127.0.0.1:%d", port))
	}}
	cfg, err := pgx.ParseConfig("host=127.0.0.1 port=35432 dbname=fvpn user=fvpn sslmode=disable")
	if err != nil {
		p.t.Fatal("database configuration")
	}
	cfg.Password = p.secret("postgres.password")
	p.db, err = pgx.ConnectConfig(p.t.Context(), cfg)
	if err != nil {
		p.t.Fatal("own stack database connection failed")
	}
	entries, _ := os.ReadDir(filepath.Join(p.state, "secrets"))
	for _, e := range entries {
		if !e.IsDir() {
			b, er := os.ReadFile(filepath.Join(p.state, "secrets", e.Name()))
			if er == nil && len(bytes.TrimSpace(b)) >= 16 {
				p.secrets = append(p.secrets, bytes.TrimSpace(b))
			}
		}
	}
}
func (p *liveProbe) secret(name string) string {
	b, e := os.ReadFile(filepath.Join(p.state, "secrets", name))
	if e != nil {
		p.t.Fatal("missing lab secret")
	}
	return strings.TrimSpace(string(b))
}
func (p *liveProbe) client() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Transport: p.tr, Jar: jar, Timeout: 15 * time.Second}
}
func (p *liveProbe) call(s *liveSession, method, path string, body any, h map[string]string) liveReply {
	var b []byte
	var err error
	if raw, ok := body.([]byte); ok {
		b = raw
	} else if body != nil {
		b, err = json.Marshal(body)
		if err != nil {
			p.t.Fatal("probe request marshal")
		}
	}
	if !strings.HasPrefix(path, "https://") {
		path = livePublic + path
	}
	req, err := http.NewRequestWithContext(p.t.Context(), method, path, bytes.NewReader(b))
	if err != nil {
		p.t.Fatal("probe request")
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method != "GET" {
		req.Header.Set("Origin", livePublic)
		req.Header.Set("X-CSRF-Token", s.csrf)
		req.Header.Set("Idempotency-Key", uuid.NewString())
	}
	for k, v := range h {
		if v == "" {
			req.Header.Del(k)
		} else {
			req.Header.Set(k, v)
		}
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return liveReply{status: 0, code: transportClass(err)}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return liveReply{status: 0, code: "READ_FAILED"}
	}
	var e struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &e)
	return liveReply{status: resp.StatusCode, code: e.Error.Code, body: raw, header: resp.Header}
}
func (p *liveProbe) check(name string, r liveReply, status int, code string) {
	ok := r.status == status && (code == "" || r.code == code)
	p.observations = append(p.observations, map[string]any{"name": name, "expected_status": status, "actual_status": r.status, "expected_code": code, "actual_code": r.code, "passed": ok, "utc": time.Now().UTC()})
	if !ok {
		p.t.Errorf("%s: status=%d code=%s; want %d %s", name, r.status, r.code, status, code)
	}
}
func (p *liveProbe) login(subject, role string) *liveSession {
	s := &liveSession{client: p.client()}
	s.client.Timeout = 30 * time.Second
	s.client.CheckRedirect = idplogin.RedirectLimit(20)
	cred, err := idplogin.CredFor(p.t.Context(), s.client, "https://netbird.example.invalid/api", p.secret("netbird.token"), p.state, subject)
	if err != nil {
		p.t.Fatal("lab user unavailable")
	}
	if err := idplogin.PortalLogin(p.t.Context(), s.client, livePublic, cred.Email, cred.Password); err != nil {
		p.metadata["login_failure"] = map[string]any{"role": role, "error_class": transportClass(err), "utc": time.Now().UTC()}
		p.t.Fatal("IdP login failed: ", transportClass(err))
	}
	r := p.call(s, "GET", "/auth/session?role="+role, nil, nil)
	var v struct {
		Authenticated bool   `json:"authenticated"`
		Role          string `json:"role"`
		Member        string `json:"member_id"`
	}
	if json.Unmarshal(r.body, &v) != nil || !v.Authenticated || v.Role != role {
		p.t.Fatalf("live OIDC session failed: role=%s status=%d code=%s authenticated=%t returned_role=%s", role, r.status, r.code, v.Authenticated, v.Role)
	}
	s.cred = cred
	p.secrets = append(p.secrets, []byte(cred.Password))
	s.csrf = r.header.Get("X-CSRF-Token")
	s.member = v.Member
	p.secrets = append(p.secrets, []byte(s.csrf))
	req, _ := http.NewRequest("GET", livePublic, nil)
	for _, c := range s.client.Jar.Cookies(req.URL) {
		p.secrets = append(p.secrets, []byte(c.Value))
	}
	return s
}
func (p *liveProbe) roles(admin, member *liveSession) {
	spec, err := gen.GetSpec()
	if err != nil {
		p.t.Fatal(err)
	}
	anonymous := &liveSession{client: p.client()}
	paths := make([]string, 0, len(spec.Paths.Map()))
	for path := range spec.Paths.Map() {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	parameter := regexp.MustCompile(`\{[^}]+\}`)
	for _, path := range paths {
		item := spec.Paths.Value(path)
		for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
			op := item.GetOperation(method)
			if op == nil || strings.EqualFold(op.OperationID, "EnrollNode") || strings.Contains(path, "webhook") {
				continue
			}
			target := "/api/v1" + parameter.ReplaceAllString(path, "00000000-0000-0000-0000-000000000001")
			r := p.call(anonymous, method, target, json.RawMessage(`{}`), nil)
			p.check("anonymous "+method+" "+path, r, 401, "")
			if strings.HasPrefix(path, "/admin/") {
				r = p.call(member, method, target, json.RawMessage(`{}`), nil)
				p.check("member to admin "+method+" "+path, r, 401, "UNAUTHENTICATED")
			}
			if strings.HasPrefix(path, "/member/") {
				r = p.call(admin, method, target, json.RawMessage(`{}`), nil)
				p.check("admin to member "+method+" "+path, r, 401, "UNAUTHENTICATED")
			}
			time.Sleep(550 * time.Millisecond)
		}
	}
	p.check("CSRF missing", p.call(admin, "POST", "/api/v1/admin/enrollments", map[string]string{"delivery": "installer"}, map[string]string{"X-CSRF-Token": ""}), 403, "")
	p.check("CSRF invalid", p.call(admin, "POST", "/api/v1/admin/enrollments", map[string]string{"delivery": "installer"}, map[string]string{"X-CSRF-Token": uuid.NewString()}), 403, "")
	p.check("CSRF cross-origin", p.call(admin, "POST", "/api/v1/admin/enrollments", map[string]string{"delivery": "installer"}, map[string]string{"Origin": "https://example.invalid"}), 403, "")
	p.check("invalid bearer", p.call(anonymous, "GET", "/api/v1/admin/nodes", nil, map[string]string{"Authorization": "Bearer " + uuid.NewString()}), 401, "")
	validEnrollment, _ := pkitest.Request(p.t)
	p.check("invalid enrollment credential", p.call(anonymous, "POST", "/api/v1/agent/enroll", validEnrollment, map[string]string{"Authorization": "Enrollment " + uuid.NewString()}), 401, "")
	p.check("oversize body", p.call(admin, "POST", "/api/v1/admin/enrollments", []byte(`{"delivery":"installer","name":"`+strings.Repeat("x", (1<<20)+1)+`"}`), nil), 413, "")
	p.check("schema invalid", p.call(admin, "POST", "/api/v1/admin/enrollments", json.RawMessage(`{"delivery":"invalid"}`), nil), 422, "")
}
func (p *liveProbe) enrollment(admin *liveSession, name string) (api.EnrollmentCreated, string) {
	key := uuid.NewString()
	body := map[string]any{"delivery": "installer", "name": name, "region": "TEST ONLY lab", "member_group_ids": []uuid.UUID{p.groupID}}
	r := p.call(admin, "POST", "/api/v1/admin/enrollments", body, map[string]string{"Idempotency-Key": key})
	p.check("create enrollment "+name, r, 201, "")
	var e api.EnrollmentCreated
	if json.Unmarshal(r.body, &e) != nil {
		p.t.Fatal("enrollment response")
	}
	token, err := e.Token.Get()
	if err != nil || token == "" {
		p.t.Fatal("enrollment token absent")
	}
	p.secrets = append(p.secrets, []byte(token))
	p.check("enrollment secret no-store", liveReply{status: r.status, code: r.header.Get("Cache-Control")}, 201, "no-store")
	replay := p.call(admin, "POST", "/api/v1/admin/enrollments", body, map[string]string{"Idempotency-Key": key})
	p.check("enrollment duplicate", replay, 201, "")
	var repeat api.EnrollmentCreated
	if json.Unmarshal(replay.body, &repeat) != nil || repeat.Enrollment.EnrollmentId != e.Enrollment.EnrollmentId || repeat.SecretAvailable || bytes.Contains(replay.body, []byte(token)) {
		p.t.Error("enrollment replay leaked secret or duplicated resource")
	}
	body["name"] = name + " changed"
	p.check("enrollment idempotency conflict", p.call(admin, "POST", "/api/v1/admin/enrollments", body, map[string]string{"Idempotency-Key": key}), 409, "IDEMPOTENCY_CONFLICT")
	return e, token
}
func (p *liveProbe) docker(service string, args ...string) ([]byte, error) {
	return exec.CommandContext(p.t.Context(), "docker", append([]string{"exec", livePrefix + service}, args...)...).CombinedOutput()
}
func (p *liveProbe) enrollAgent(admin *liveSession) uuid.UUID {
	e, token := p.enrollment(admin, "secops-agent")
	path := filepath.Join(p.state, "node/var/e2e/enrollment.token")
	if err := os.WriteFile(path, []byte(token+"\n"), 0600); err != nil {
		p.t.Fatal("node enrollment hook")
	}
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		b, err := p.docker("node", "cat", "/var/lib/vpn-node/e2e/status.json")
		if err == nil && bytes.Contains(b, []byte("enrolled")) {
			return e.Enrollment.NodeId
		}
		time.Sleep(time.Second)
	}
	p.t.Fatal("node enrollment stage timeout")
	return uuid.Nil
}
func (p *liveProbe) enrollHost(admin *liveSession) (api.EnrollResponse, *liveSession) {
	_, token := p.enrollment(admin, "secops-node-b")
	req, key := pkitest.Request(p.t)
	client := &liveSession{client: p.client()}
	r := p.call(client, "POST", "/api/v1/agent/enroll", req, map[string]string{"Authorization": "Enrollment " + token})
	p.check("node B certificate enrollment", r, 201, "")
	var e api.EnrollResponse
	if json.Unmarshal(r.body, &e) != nil || e.CertificatePem == "" {
		p.t.Fatal("node B certificate absent")
	}
	r2 := p.call(client, "POST", "/api/v1/agent/enroll", req, map[string]string{"Authorization": "Enrollment " + token})
	p.check("node B CSR replay", r2, 200, "")
	var replay api.EnrollResponse
	_ = json.Unmarshal(r2.body, &replay)
	if replay.CertificateSerial != e.CertificateSerial {
		p.t.Error("CSR replay changed serial")
	}
	leaf := pkitest.Certificate(p.t, e.CertificatePem)
	tr := p.tr.Clone()
	tr.TLSClientConfig.Certificates = []tls.Certificate{{Certificate: [][]byte{leaf.Raw}, PrivateKey: key}}
	p.t.Cleanup(tr.CloseIdleConnections)
	client = &liveSession{client: &http.Client{Transport: tr, Timeout: 15 * time.Second}}
	// Sign an expired leaf using only this disposable stack's test CA. Never set a clock.
	configRaw, err := os.ReadFile(filepath.Join(p.state, "config/nodes.json"))
	if err != nil {
		p.t.Fatal("node test settings")
	}
	var cfg struct {
		Key string `json:"ca_key_id"`
	}
	_ = json.Unmarshal(configRaw, &cfg)
	provider, err := signing.OpenFileProvider(filepath.Join(p.state, "vault"))
	if err != nil {
		p.t.Fatal("test CA provider")
	}
	signer, err := provider.CertificateSigner(p.t.Context(), cfg.Key)
	if err != nil {
		p.t.Fatal("test CA signer")
	}
	caRaw, err := os.ReadFile(filepath.Join(p.state, "secrets/agent-ca.pem"))
	if err != nil {
		p.t.Fatal("test CA certificate")
	}
	caBlock, _ := pem.Decode(caRaw)
	ca, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		p.t.Fatal("test CA parse")
	}
	template := *leaf
	template.NotBefore = time.Now().Add(-time.Hour)
	template.NotAfter = time.Now().Add(-time.Minute)
	expired, err := x509.CreateCertificate(rand.Reader, &template, ca, key.Public(), signer)
	if err != nil {
		p.t.Fatal("expired certificate fixture")
	}
	expiredTR := p.tr.Clone()
	expiredTR.TLSClientConfig.Certificates = []tls.Certificate{{Certificate: [][]byte{expired}, PrivateKey: key}}
	defer expiredTR.CloseIdleConnections()
	p.check("expired certificate actual TLS", p.call(&liveSession{client: &http.Client{Transport: expiredTR, Timeout: 5 * time.Second}}, "GET", liveAgent+"/api/v1/agent/desired-state", nil, nil), 0, "TRANSPORT_REFUSED")
	keyDER, er := x509.MarshalECPrivateKey(key)
	if er != nil {
		p.t.Fatal("key scan canary")
	}
	p.secrets = append(p.secrets, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))
	p.metadata["node_incarnation"] = e.NodeIncarnationId
	return e, client
}
func (p *liveProbe) identities(admin, node *liveSession, b api.EnrollResponse, a uuid.UUID) {
	certPEM, er := p.docker("node", "cat", "/var/lib/vpn-node/identity/cert.pem")
	if er != nil {
		p.t.Fatal("node A certificate")
	}
	keyPEM, er := p.docker("node", "cat", "/var/lib/vpn-node/identity/key.pem")
	if er != nil {
		p.t.Fatal("node A key fixture")
	}
	p.secrets = append(p.secrets, keyPEM)
	pair, er := tls.X509KeyPair(certPEM, keyPEM)
	if er != nil {
		p.t.Fatal("node A key pair")
	}
	trA := p.tr.Clone()
	trA.TLSClientConfig.Certificates = []tls.Certificate{pair}
	defer trA.CloseIdleConnections()
	rA := p.call(&liveSession{client: &http.Client{Transport: trA, Timeout: 15 * time.Second}}, "GET", liveAgent+"/api/v1/agent/desired-state", nil, map[string]string{"X-Node-ID": b.NodeId.String()})
	p.check("node A certificate keeps own configuration", rA, 200, "")
	var aDoc struct {
		Signed struct {
			NodeID uuid.UUID `json:"node_id"`
		} `json:"signed"`
	}
	_ = json.Unmarshal(rA.body, &aDoc)
	if aDoc.Signed.NodeID != a {
		p.t.Error("node A cross-node identity")
	}
	p.check("certificate required", p.call(&liveSession{client: p.client()}, "GET", liveAgent+"/api/v1/agent/desired-state", nil, nil), 0, "TRANSPORT_REFUSED")
	for _, id := range []uuid.UUID{a, b.NodeId} {
		r := p.call(node, "GET", liveAgent+"/api/v1/agent/desired-state", nil, map[string]string{"X-Node-ID": id.String()})
		p.check("desired state forged header "+id.String(), r, 200, "")
		var doc struct {
			Signed struct {
				NodeID     uuid.UUID `json:"node_id"`
				Inc        uuid.UUID `json:"node_incarnation_id"`
				Generation string    `json:"config_generation"`
			} `json:"signed"`
		}
		_ = json.Unmarshal(r.body, &doc)
		if doc.Signed.NodeID != b.NodeId || doc.Signed.Inc != b.NodeIncarnationId {
			p.t.Error("cross-node desired state")
		}
		p.metadata["policy_revision"] = doc.Signed.Generation
		trustRaw, er := os.ReadFile(filepath.Join(p.state, "node/etc/trust.json"))
		if er != nil {
			p.t.Fatal("node trust fixture")
		}
		trust, er := envelope.ParseNodeTrustStoreJSON(trustRaw)
		if er != nil {
			p.t.Fatal("node trust parse")
		}
		if _, _, er = envelope.Verify(r.body, trust, envelope.PurposeDesiredState, time.Now()); er != nil {
			p.t.Error("desired state signature verification")
		}
	}
	forgedEvent, err := usage.WithComputedHash(api.UsageEvent{SchemaVersion: 1, NodeId: a, NodeIncarnationId: b.NodeIncarnationId, MeterEpochId: uuid.New(), EventSeq: 1, IntervalStart: time.Now().UTC().Add(-time.Minute), IntervalEnd: time.Now().UTC(), MeteringVersion: api.MeteringVersionForwardAdmissionV1, Quality: api.UsageQualityComplete, Records: []api.UsageRecord{}})
	if err != nil {
		p.t.Fatal("forged event fixture")
	}
	r := p.call(node, "POST", liveAgent+"/api/v1/agent/usage-batches", api.UsageBatchRequest{NodeId: a, NodeIncarnationId: b.NodeIncarnationId, LedgerEpoch: presence.NewNullNullable[uuid.UUID](), Events: []api.UsageEvent{forgedEvent}}, nil)
	p.check("forged usage node_id", r, 403, "")
	bootstrap := p.call(node, "POST", liveAgent+"/api/v1/agent/bootstrap", nil, nil)
	for attempts := 0; bootstrap.status == 409 && attempts < 10; attempts++ {
		time.Sleep(time.Second)
		bootstrap = p.call(node, "POST", liveAgent+"/api/v1/agent/bootstrap", nil, nil)
	}
	p.check("node B bootstrap once", bootstrap, 200, "")
	var secret struct {
		SetupKey string `json:"setup_key"`
	}
	_ = json.Unmarshal(bootstrap.body, &secret)
	if secret.SetupKey != "" {
		p.secrets = append(p.secrets, []byte(secret.SetupKey))
	}
	p.check("bootstrap secret no-store", liveReply{status: bootstrap.status, code: bootstrap.header.Get("Cache-Control")}, 200, "no-store")
	p.check("node B bootstrap consumed", p.call(node, "POST", liveAgent+"/api/v1/agent/bootstrap", nil, nil), 410, "BOOTSTRAP_CONSUMED")
	_ = admin
}

func (p *liveProbe) ownership(admin *liveSession, members []*liveSession, devices [][]uuid.UUID) {
	for i, m := range members {
		r := p.call(m, "GET", "/api/v1/member/devices", nil, nil)
		p.check(fmt.Sprintf("member %02d own devices", i), r, 200, "")
		for j, ds := range devices {
			for _, id := range ds {
				if bytes.Contains(r.body, []byte(id.String())) != (i == j) {
					p.t.Error("device list ownership")
				}
			}
		}
		time.Sleep(550 * time.Millisecond)
	}
	if len(devices[0]) > 0 {
		p.check("foreign device revoke", p.call(members[1], "DELETE", "/api/v1/member/devices/"+devices[0][0].String(), nil, nil), 404, "")
	} else {
		p.securityComplete = false
		p.metadata["device_idor_blocker"] = "No product-admitted member A device; no invented foreign UUID positive control"
	}
	r := p.call(members[0], "GET", "/api/v1/member/usage?member_id="+members[1].member, nil, nil)
	p.check("usage remains caller scoped with foreign query", r, 200, "")
	if bytes.Contains(r.body, []byte(members[1].member)) {
		p.t.Error("foreign usage identity exposed")
	}
	p.check("member PATCH needs If-Match", p.call(admin, "PATCH", "/api/v1/admin/members/"+members[0].member, map[string]any{"admin_suspended": true}, nil), 428, "")
	p.check("member PATCH stale ETag", p.call(admin, "PATCH", "/api/v1/admin/members/"+members[0].member, map[string]any{"admin_suspended": true}, map[string]string{"If-Match": "\"0\""}), 412, "")
}
func (p *liveProbe) load(members []*liveSession) {
	// Fixed diagnostic subset, not the unexecuted 300-second mixed/relay profile.
	var mu sync.Mutex
	latencies := []float64{}
	codes := map[int]int{}
	wrong := 0
	start := time.Now()
	var wg sync.WaitGroup
	for _, m := range members {
		wg.Go(func() {
			for range 10 {
				before := time.Now()
				r := p.call(m, "GET", "/api/v1/member/me", nil, nil)
				elapsed := float64(time.Since(before).Microseconds()) / 1000
				mu.Lock()
				latencies = append(latencies, elapsed)
				codes[r.status]++
				if r.status == 200 && !bytes.Contains(r.body, []byte(m.member)) {
					wrong++
				}
				mu.Unlock()
				time.Sleep(time.Second)
			}
		})
	}
	wg.Wait()
	sort.Float64s(latencies)
	stats, err := exec.CommandContext(p.t.Context(), "docker", "stats", "--no-stream", "--format", "{{json .}}", livePrefix+"control", livePrefix+"worker", livePrefix+"postgres", livePrefix+"node").Output()
	var samples []map[string]any
	if err == nil {
		for _, line := range bytes.Split(bytes.TrimSpace(stats), []byte("\n")) {
			var s map[string]any
			if json.Unmarshal(line, &s) == nil {
				delete(s, "ID")
				delete(s, "Container")
				samples = append(samples, s)
			}
		}
	}
	p.metadata["capacity_subset"] = map[string]any{"profile": "20 already authenticated members; 10 concurrent rounds, one read/member/second; product-admitted device count recorded separately", "duration_seconds": time.Since(start).Seconds(), "requests": len(latencies), "http_status_counts": codes, "read_p50_ms": latencies[len(latencies)/2], "read_p95_ms": latencies[(len(latencies)*95+99)/100-1], "read_p99_ms": latencies[(len(latencies)*99+99)/100-1], "wrong_member_responses": wrong, "resource_snapshot_after_load": samples, "simultaneous_oidc": false, "separate_concurrent_oidc_probe": true, "capacity_claim": false, "relay_load": false, "policy_latency_seconds": nil, "usage_latency_seconds": nil}
	if wrong != 0 {
		p.t.Error("load ownership violation")
	}
	// Saturate this principal deliberately; bounded flood only to loopback.
	limited := 0
	retry := false
	for range 80 {
		r := p.call(members[0], "GET", "/api/v1/member/me", nil, nil)
		if r.status == 429 {
			limited++
			retry = retry || r.header.Get("Retry-After") != ""
		}
	}
	p.metadata["rate_limit"] = map[string]any{"requests": 80, "limited": limited, "retry_after_present": retry}
	if limited == 0 || !retry {
		p.t.Error("rate limit not effective")
	}
	time.Sleep(3 * time.Second)
}
func (p *liveProbe) ledger(admin, node *liveSession, b api.EnrollResponse, members []*liveSession, devices [][]uuid.UUID) {
	if len(p.bindings) == 0 {
		p.metadata["ledger_blocker"] = "No product-projected binding; no usage batch sent"
		return
	}
	p.physicalSource(admin, node, b, members[0], devices[0][0])
	raw, err := os.ReadFile(filepath.Join(p.root, ".agent/secops/retained.json"))
	var batch api.UsageBatchRequest
	if err != nil || json.Unmarshal(raw, &batch) != nil {
		p.t.Fatal("retained fixture read")
	}
	r := p.call(node, "POST", liveAgent+"/api/v1/agent/usage-batches", batch, nil)
	p.check("usage duplicate window", r, 200, "")
	var ack api.UsageBatchResult
	if json.Unmarshal(r.body, &ack) != nil {
		p.t.Fatal("duplicate result")
	}
	duplicate := 0
	for _, result := range ack.Results {
		if result.Status == api.UsageEventResultStatusDuplicate {
			duplicate++
		}
	}
	if ack.HighestContiguousSeq != 10 || duplicate != 10 {
		p.t.Fatal("ledger duplicate replay")
	}
	p.metadata["usage_final_ack"] = ack.HighestContiguousSeq
	p.metadata["usage_duplicate_events"] = duplicate
	p.metadata["recovery"] = p.metadata["physical_source"]
	var ledgerRows int
	if p.db.QueryRow(p.t.Context(), `SELECT count(*) FROM usage_events WHERE node_id=$1 AND incarnation_id=$2`, b.NodeId, b.NodeIncarnationId).Scan(&ledgerRows) != nil || ledgerRows != 10 {
		p.t.Error("ledger receipt count")
	}
	p.metadata["synthetic_ledger_rows"] = ledgerRows
	visibleStart := time.Now()
	var total decstr.U64
	for time.Since(visibleStart) < 60*time.Second {
		v := p.call(members[0], "GET", "/api/v1/member/usage", nil, nil)
		var page struct {
			Totals struct {
				Total decstr.U64 `json:"total_bytes"`
			} `json:"totals"`
		}
		if v.status == 200 && json.Unmarshal(v.body, &page) == nil {
			total = page.Totals.Total
		}
		if total == 3000 {
			break
		}
		time.Sleep(2 * time.Second)
	}
	p.metadata["synthetic_usage_visibility"] = map[string]any{"total_bytes": total, "expected_bytes": decstr.U64(3000), "poll_seconds": time.Since(visibleStart).Seconds(), "exact": total == 3000, "kernel_sampling": false}
	if total != 3000 {
		p.t.Error("synthetic usage projection did not converge")
	}

}
func (p *liveProbe) revoke(admin, node *liveSession, b api.EnrollResponse) {
	// Revoke only B after retaining its batch; isolated restore predates revocation.
	r := p.call(admin, "GET", "/api/v1/admin/nodes/"+b.NodeId.String(), nil, nil)
	revoked := p.call(admin, "POST", "/api/v1/admin/nodes/"+b.NodeId.String()+"/revoke", map[string]string{"reason": "TEST ONLY secops revoked certificate"}, map[string]string{"If-Match": r.header.Get("ETag")})
	p.check("node revoke accepted", revoked, 202, "")
	p.check("revoked certificate refuses desired state", p.call(node, "GET", liveAgent+"/api/v1/agent/desired-state", nil, nil), 403, "IDENTITY_REVOKED")
}
func (p *liveProbe) finish() {
	for key := range p.metadata {
		if strings.HasPrefix(key, "private_") {
			delete(p.metadata, key)
		}
	}
	p.metadata["finished_at"] = time.Now().UTC()
	p.metadata["checks"] = p.observations
	p.metadata["security_matrix_complete"] = p.securityComplete
	p.metadata["projector_owned_rows_written"] = []string{}
	p.metadata["go_test_failed"] = p.t.Failed()
	for _, nodeSvc := range []string{"node", "node2"} {
		for _, file := range []string{"key.pem", "setup.key"} {
			raw, er := p.docker(nodeSvc, "cat", "/var/lib/vpn-node/identity/"+file)
			if er == nil {
				p.secrets = append(p.secrets, bytes.TrimSpace(raw))
			}
		}
	}
	services := []string{"postgres", "idp", "payment", "origin", "netbird", "gateway", "coredns", "adguard", "control", "worker", "node-netns", "node", "member"}
	if os.Getenv("FVPN_SECOPS_FOLLOWUP7") == "1" {
		for _, service := range []string{"node2-netns", "node2-adguard", "node2"} {
			if exec.CommandContext(p.t.Context(), "docker", "inspect", livePrefix+service).Run() == nil {
				services = append(services, service)
			}
		}
	}
	extra, inventoryErr := exec.CommandContext(p.t.Context(), "docker", "ps", "-a", "--filter", "name=^fvpn-w4-secops-member-", "--format", "{{.Names}}").Output()
	if inventoryErr != nil {
		p.t.Error("native peer log inventory failed")
	}
	for _, name := range strings.Fields(string(extra)) {
		if strings.HasPrefix(name, livePrefix+"member-") {
			services = append(services, strings.TrimPrefix(name, livePrefix))
		}
	}
	streams := []map[string]any{}
	images := []map[string]string{}
	logDir := filepath.Join(p.root, ".agent/secops/captured-logs")
	if os.MkdirAll(logDir, 0700) != nil {
		p.t.Error("private diagnostic directory")
	}
	scan := func(name string, raw []byte, err error) {
		file := strings.NewReplacer("/", "_", " ", "_").Replace(name) + ".log"
		if os.WriteFile(filepath.Join(logDir, file), raw, 0600) != nil {
			p.t.Error("private diagnostic capture")
		}
		findings := 0
		for _, secret := range p.secrets {
			if len(secret) >= 16 && bytes.Contains(raw, secret) {
				findings++
			}
		}
		if bytes.Contains(raw, []byte("PRIVATE KEY-----")) {
			findings++
		}
		deviceFlowLines := 0
		for _, line := range bytes.Split(raw, []byte("\n")) {
			if bytes.Contains(line, []byte("user_code=")) || bytes.Contains(line, []byte("verification_uri")) || bytes.Contains(line, []byte("device_code=")) {
				deviceFlowLines++
			}
		}
		streams = append(streams, map[string]any{"stream": name, "bytes": len(raw), "read_ok": err == nil, "known_secret_matches": findings, "device_flow_key_lines": deviceFlowLines})
		if findings > 0 {
			p.t.Error("secret in stream " + name)
		}
	}
	for _, service := range services {
		raw, err := exec.CommandContext(p.t.Context(), "docker", "logs", livePrefix+service).CombinedOutput()
		scan(service+" stdout/stderr", raw, err)
		id, er := exec.CommandContext(p.t.Context(), "docker", "inspect", "--format", "{{.Image}}", livePrefix+service).Output()
		if er == nil {
			images = append(images, map[string]string{"service": service, "image_id": strings.TrimSpace(string(id))})
		}
	}
	for _, service := range services {
		if service != "node" && service != "node2" && service != "member" && !strings.HasPrefix(service, "member-") {
			continue
		}
		listing, err := p.docker(service, "find", "/var/lib/vpn-node/e2e", "/var/log/netbird", "-type", "f", "-name", "*.log")
		if err != nil {
			p.t.Error("log stream inventory failed")
		}
		for _, name := range strings.Fields(string(listing)) {
			raw, er := p.docker(service, "cat", name)
			scan(service+" file "+name, raw, er)
		}
	}
	rows, err := p.db.Query(p.t.Context(), "SELECT row_to_json(a)::text FROM audit_events a")
	var audit bytes.Buffer
	count := 0
	if err == nil {
		for rows.Next() {
			var s string
			if rows.Scan(&s) == nil {
				audit.WriteString(s)
				count++
			}
		}
		err = rows.Err()
		rows.Close()
	}
	scan("audit_events", audit.Bytes(), err)
	p.metadata["audit_rows_scanned"] = count
	p.metadata["secret_scan"] = streams
	p.metadata["known_secret_canaries"] = len(p.secrets)
	p.metadata["image_digests"] = images
	p.metadata["go_test_failed"] = p.t.Failed()
	out, err := json.MarshalIndent(p.metadata, "", "  ")
	if err != nil {
		p.t.Error("evidence marshal")
		return
	}
	if p.metadata["physical_source"] != nil {
		if os.WriteFile(filepath.Join(p.root, "docs/e2e/at/secops/pitr-source.json"), append(out, '\n'), 0644) != nil {
			p.t.Error("PITR evidence write")
		}
	}
	name := "live.json"
	if os.Getenv("FVPN_SECOPS_FOLLOWUP7") == "1" {
		name = "followup7-live.json"
	}
	if os.Getenv("FVPN_SECOPS_PITR") == "1" {
		name = "pitr-source.json"
	}
	if os.Getenv("FVPN_SECOPS_SCALE_ONLY") == "1" {
		name = "native-scale-live.json"
	}
	if phase := os.Getenv("FVPN_SECOPS_FOLLOWUP6"); phase == "profile" || phase == "helper" {
		name = "followup6-" + phase + "-live.json"
	}
	if os.WriteFile(filepath.Join(p.root, "docs/e2e/at/secops", name), append(out, '\n'), 0644) != nil {
		p.t.Error("evidence write")
	}
}

// Shared exported lab hook: real signed binaries, release verification and
// non-empty release-support activation through APIs. Never seed policy rows.
func (p *liveProbe) releaseSupport(admin *liveSession) {
	baseline, err := e2e.PrepareServingBaseline(p.t.Context(), admin.client, admin.csrf, false)
	if err != nil {
		p.t.Fatal("shared serving baseline failed: ", err)
	}
	p.metadata["serving_baseline"] = baseline
}

func (p *liveProbe) memberOperation(members []*liveSession) {
	r := p.call(members[0], "GET", "/api/v1/member/devices", nil, nil)
	var page struct {
		Items []struct {
			DeviceID uuid.UUID `json:"device_id"`
		} `json:"items"`
	}
	if json.Unmarshal(r.body, &page) != nil || len(page.Items) == 0 {
		p.t.Fatal("own device fixture missing")
	}
	r = p.call(members[0], "DELETE", "/api/v1/member/devices/"+page.Items[0].DeviceID.String(), nil, nil)
	p.check("own device revoke accepted", r, 202, "")
	var op api.Operation
	_ = json.Unmarshal(r.body, &op)
	if op.OperationId == uuid.Nil {
		p.t.Fatal("own operation absent")
	}
	p.check("own operation visible", p.call(members[0], "GET", "/api/v1/member/operations/"+op.OperationId.String(), nil, nil), 200, "")
	p.check("foreign operation hidden", p.call(members[1], "GET", "/api/v1/member/operations/"+op.OperationId.String(), nil, nil), 404, "")
}

// Never publish a url.Error string: callback URLs can contain OAuth state.
func transportClass(err error) string {
	var netErr net.Error
	var verify *tls.CertificateVerificationError
	switch {
	case errors.As(err, &netErr) && netErr.Timeout():
		return "TRANSPORT_TIMEOUT"
	case errors.As(err, &verify):
		return "TLS_VERIFY_FAILED"
	case errors.Is(err, syscall.ECONNREFUSED):
		return "CONNECTION_REFUSED"
	default:
		return "TRANSPORT_REFUSED"
	}
}
