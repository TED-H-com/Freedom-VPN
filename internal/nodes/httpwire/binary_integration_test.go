//go:build integration

package httpwire_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/internal/policies/policiestest"
	"github.com/TED-H-com/Freedom-VPN/internal/usage/usagetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

type processLog struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *processLog) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}
func (b *processLog) String() string { b.mu.Lock(); defer b.mu.Unlock(); return b.b.String() }
func loopbackAddress(t *testing.T) string {
	t.Helper()
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	a := l.Addr().String()
	if e = l.Close(); e != nil {
		t.Fatal(e)
	}
	return a
}
func TestAPI01_NodeBinariesUseCompositionAndCleanShutdown(t *testing.T) {
	pool, dsn := dbtest.NewUnmigratedWithDSN(t)
	if e := db.Migrate(t.Context(), pool); e != nil {
		t.Fatal(e)
	}
	cfg, n := domainConfig(t)
	dir := t.TempDir()
	// The only remote APIs reachable by these processes are loopback test servers.
	var mu sync.Mutex
	var key map[string]any
	groups := map[string]map[string]any{}
	creates := 0
	const setupSecret = "TEST ONLY binary setup secret"
	nb := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /api/peers", "GET /api/users", "GET /api/routes", "GET /api/policies", "GET /api/dns/nameservers":
			io.WriteString(w, "[]")
		case "GET /api/groups":
			list := []any{}
			for _, group := range groups {
				list = append(list, group)
			}
			json.NewEncoder(w).Encode(list)
		case "POST /api/groups":
			var group map[string]any
			json.NewDecoder(r.Body).Decode(&group)
			id := fmt.Sprintf("test-group-%d", len(groups)+1)
			group["id"] = id
			groups[id] = group
			json.NewEncoder(w).Encode(group)
		case "GET /api/groups/test-group-1", "GET /api/groups/test-group-2":
			json.NewEncoder(w).Encode(groups[strings.TrimPrefix(r.URL.Path, "/api/groups/")])
		case "GET /api/setup-keys":
			if key == nil {
				io.WriteString(w, "[]")
			} else {
				json.NewEncoder(w).Encode([]any{key})
			}
		case "POST /api/setup-keys":
			creates++
			json.NewDecoder(r.Body).Decode(&key)
			key["used_times"], key["revoked"], key["state"] = 0, false, "valid"
			key["id"] = "test-key"
			key["key"] = setupSecret
			key["expires"] = time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
			json.NewEncoder(w).Encode(key)
			delete(key, "key")
		case "GET /api/setup-keys/test-key":
			json.NewEncoder(w).Encode(key)
		default:
			t.Errorf("unexpected external call: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(500)
		}
	}))
	t.Cleanup(nb.Close)
	var issuer string
	idp := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/.well-known/openid-configuration" {
			t.Errorf("unexpected OIDC call %s", r.URL.Path)
			w.WriteHeader(500)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token", "jwks_uri": issuer + "/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
	}))
	issuer = idp.URL
	t.Cleanup(idp.Close)
	n.NetBirdURL = nb.URL + "/api"
	cfg.NodesConfigFile = jsonFixture(t, dir, "nodes.json", n)
	random := make([]byte, 32)
	if _, e := rand.Read(random); e != nil {
		t.Fatal(e)
	}
	zero, live := 0, false
	// Explicit TEST ONLY member configuration keeps both production domains active.
	members := config.Members{Provider: "stripe", APIKeyFile: writeFixture(t, dir, "payment-key", []byte(hex.EncodeToString(random))), WebhookSecretFiles: []string{writeFixture(t, dir, "webhook-secret", []byte(hex.EncodeToString(random)))}, VaultActiveKey: "TEST ONLY", VaultKeyFiles: map[string]string{"TEST ONLY": writeFixture(t, dir, "billing-key", random)}, LiveMode: &live, PriceIDs: []string{"TEST ONLY price"}, ReturnURLs: []string{cfg.PublicOrigin + "/billing/return"}, FullRefundPolicy: "revoke_original_period", GraceSeconds: &zero, TrialMaxSeconds: &zero, GrantMaxSeconds: 86400, DeviceLimit: 3, PortalTTLSeconds: 60, BillingLeaseSeconds: 60, MaxWebhookBytes: 1 << 20, ReconcileSeconds: 3600, DeviceSyncSeconds: 3600, NetBirdURL: n.NetBirdURL, NetBirdDeployment: n.NetBirdDeployment, NetBirdTokenFile: n.NetBirdTokenFile, OverlayScope: "TEST ONLY"}
	cfg.MembersConfigFile = jsonFixture(t, dir, "members.json", members)
	// TEST ONLY upstream: the idle DNS worker does not dial it.
	cfg.DNSConfigFile = writeFixture(t, dir, "dns.json", []byte(`{"upstreams":["192.0.2.53:53"]}`))
	cfg.PoliciesConfigFile = policiestest.New(t).Path
	cfg.UsageConfigFile = usagetest.New(t).Path
	cfg.DatabaseURLFile = writeFixture(t, dir, "database-url", []byte(dsn))
	cfg.OIDCIssuer = issuer
	cfg.OIDCClientID = "TEST ONLY portal"
	cfg.PublicAddress = loopbackAddress(t)
	cfg.AgentAddress = loopbackAddress(t)
	cfg.MetricsAddress = loopbackAddress(t)
	cfg.AgentRate = config.Bucket{PerSecond: 100, Burst: 100}
	cfg.IngressRate = cfg.AgentRate
	cfg.WorkerConcurrency = 1
	path := jsonFixture(t, dir, "central.json", cfg)
	trusted := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: idp.Certificate().Raw})
	trusted = append(trusted, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: nb.Certificate().Raw})...)
	trustFile := writeFixture(t, dir, "roots.pem", trusted)
	env := []string{}
	for _, s := range os.Environ() {
		if !strings.HasPrefix(s, "FVPN_") && !strings.HasPrefix(s, "SSL_CERT_FILE=") && !strings.HasPrefix(s, "SSL_CERT_DIR=") {
			env = append(env, s)
		}
	}
	env = append(env, "SSL_CERT_FILE="+trustFile, "SSL_CERT_DIR="+dir)
	root, e := filepath.Abs("../../..")
	if e != nil {
		t.Fatal(e)
	}
	start := func(name, metricsAddress string) *processLog {
		t.Helper()
		binary := filepath.Join(dir, name)
		build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "./apps/"+strings.TrimPrefix(name, "fvpn-")+"/cmd/"+name)
		build.Dir = root
		if output, e := build.CombinedOutput(); e != nil {
			t.Fatalf("build: %v\n%s", e, output)
		}
		cmd := exec.Command(binary, "-config", path)
		cmd.Env = append([]string{"FVPN_METRICS_ADDRESS=" + metricsAddress}, env...)
		logs := &processLog{}
		cmd.Stdout = logs
		cmd.Stderr = logs
		if e := cmd.Start(); e != nil {
			t.Fatal(e)
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		t.Cleanup(func() {
			_ = cmd.Process.Signal(syscall.SIGTERM)
			select {
			case e := <-done:
				if e != nil {
					t.Errorf("%s exit: %v\n%s", name, e, logs.String())
				}
			case <-time.After(5 * time.Second):
				_ = cmd.Process.Kill()
				<-done
				t.Errorf("%s shutdown deadline exceeded", name)
			}
		})
		return logs
	}
	serverPEM, e := os.ReadFile(cfg.TLSCertificateFile)
	if e != nil {
		t.Fatal(e)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(serverPEM) {
		t.Fatal("server roots")
	}
	clientFor := func(cert *api.EnrollResponse, private *ecdsa.PrivateKey) *http.Client {
		transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots}}
		if cert != nil {
			leaf := pkitest.Certificate(t, cert.CertificatePem)
			transport.TLSClientConfig.Certificates = []tls.Certificate{{Certificate: [][]byte{leaf.Raw}, PrivateKey: private}}
		}
		t.Cleanup(transport.CloseIdleConnections)
		return &http.Client{Transport: transport, Timeout: time.Second}
	}
	publicClient := clientFor(nil, nil)
	controlLogs := start("fvpn-control", cfg.MetricsAddress)
	waitUntil(t, func() bool {
		res, e := publicClient.Get("https://" + cfg.PublicAddress + "/readyz")
		if e != nil {
			return false
		}
		res.Body.Close()
		return res.StatusCode == 200
	})
	runtime, e := httpwire.Open(t.Context(), pool, n, nil)
	if e != nil {
		t.Fatal(e)
	}
	token, e := runtime.Enrollment.CreateEnrollment(t.Context(), enrollment.Command{Principal: "admin:" + uuid.NewString(), Key: uuid.New()}, api.EnrollmentCreateRequest{Delivery: "installer"})
	if e != nil {
		t.Fatal(e)
	}
	request, private := pkitest.Request(t)
	raw, e := json.Marshal(request)
	if e != nil {
		t.Fatal(e)
	}
	req, e := http.NewRequestWithContext(t.Context(), "POST", "https://"+cfg.PublicAddress+"/api/v1/agent/enroll", bytes.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	req.Header.Set("Authorization", "Enrollment "+mustValue(t, token.Token))
	req.Header.Set("Content-Type", "application/json")
	res, e := publicClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	var enrolled api.EnrollResponse
	e = json.NewDecoder(res.Body).Decode(&enrolled)
	res.Body.Close()
	if e != nil || res.StatusCode != 201 {
		t.Fatal("binary enrollment", res.StatusCode, e)
	}
	agentClient := clientFor(&enrolled, private)
	// Choose the worker's metrics port after control has bound its own listener.
	workerLogs := start("fvpn-worker", loopbackAddress(t))
	waitUntil(t, func() bool {
		req, e := http.NewRequestWithContext(t.Context(), "POST", "https://"+cfg.AgentAddress+"/api/v1/agent/bootstrap", nil)
		if e != nil {
			t.Fatal(e)
		}
		req.Header.Set("X-Node-ID", uuid.NewString())
		res, e := agentClient.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		if res.StatusCode == 409 {
			return false
		}
		if res.StatusCode != 200 {
			t.Fatalf("binary bootstrap %d", res.StatusCode)
		}
		var key api.AgentBootstrap
		if e = json.NewDecoder(res.Body).Decode(&key); e != nil {
			t.Fatal(e)
		}
		if key.SetupKey != setupSecret || res.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("one-time delivery differs")
		}
		return true
	})
	req, e = http.NewRequestWithContext(t.Context(), "POST", "https://"+cfg.AgentAddress+"/api/v1/agent/bootstrap", nil)
	if e != nil {
		t.Fatal(e)
	}
	res, e = agentClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	res.Body.Close()
	if res.StatusCode != 410 {
		t.Fatal("bootstrap replay", res.StatusCode)
	}
	waitUntil(t, func() bool {
		var n int
		if e := pool.QueryRow(t.Context(), `SELECT count(*) FROM outbox_events WHERE event_type='node.reconcile' AND delivered_at IS NOT NULL AND delivery_receipt IS NOT NULL`).Scan(&n); e != nil {
			t.Fatal(e)
		}
		return n > 0
	})
	mu.Lock()
	count := creates
	mu.Unlock()
	if count != 1 {
		t.Fatal("duplicate remote key", count)
	}
	for _, logs := range []*processLog{controlLogs, workerLogs} {
		text := logs.String()
		if !strings.Contains(text, `"domain":"nodes"`) || !strings.Contains(text, `"domain":"members"`) {
			t.Fatal("production domains missing from process logs")
		}
		for _, secret := range []string{setupSecret, mustValue(t, token.Token), dsn, hex.EncodeToString(random)} {
			if strings.Contains(text, secret) {
				t.Fatal("process logged secret")
			}
		}
	}
}
