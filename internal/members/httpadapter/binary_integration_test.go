//go:build integration

package httpadapter_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/billing"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
)

type synchronizedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *synchronizedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}
func (b *synchronizedBuffer) String() string { b.mu.Lock(); defer b.mu.Unlock(); return b.b.String() }

func TestAPI01_BinariesActivateConfiguredMembersThroughComposition(t *testing.T) {
	pool, dsn := dbtest.NewUnmigratedWithDSN(t)
	if err := db.Migrate(t.Context(), pool); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	write := func(name string, b []byte) string {
		path := filepath.Join(dir, name)
		if e := os.WriteFile(path, b, 0600); e != nil {
			t.Fatal(e)
		}
		return path
	}
	random := func() []byte {
		b := make([]byte, 32)
		if _, e := rand.Read(b); e != nil {
			t.Fatal(e)
		}
		return b
	}
	webhookSecret := hex.EncodeToString(random())
	zero, live := 0, false
	// TEST ONLY explicit settings, never constructor defaults. No external API
	// traffic: this test exercises ingress and a harmless subscription-free sweep.
	members := config.Members{Provider: "stripe", APIKeyFile: write("payment-key", []byte(hex.EncodeToString(random()))), WebhookSecretFiles: []string{write("webhook-secret", []byte(webhookSecret))}, VaultActiveKey: "TEST-ONLY", VaultKeyFiles: map[string]string{"TEST-ONLY": write("vault-key", random())}, LiveMode: &live, PriceIDs: []string{"TEST-ONLY-price"}, ReturnURLs: []string{testOrigin + "/billing/return"}, FullRefundPolicy: "revoke_original_period", GraceSeconds: &zero, TrialMaxSeconds: &zero, GrantMaxSeconds: 86400, DeviceLimit: 3, PortalTTLSeconds: 60, BillingLeaseSeconds: 60, MaxWebhookBytes: 1 << 20, ReconcileSeconds: 3600, DeviceSyncSeconds: 3600, NetBirdDeployment: "TEST-ONLY", NetBirdTokenFile: write("netbird-key", []byte("TEST ONLY synthetic token")), OverlayScope: "TEST-ONLY"}
	nb := &netbirdFake{objects: map[string]map[string]map[string]any{"peers": {}, "users": {}, "groups": {}, "routes": {}, "policies": {}, "dns/nameservers": {}}}
	nbServer := httptest.NewTLSServer(nb)
	t.Cleanup(nbServer.Close)
	members.NetBirdURL = nbServer.URL + "/api"
	raw, e := json.Marshal(members)
	if e != nil {
		t.Fatal(e)
	}
	c := testTLSConfig(t, config.Defaults())
	c.DatabaseURLFile = write("database-url", []byte(dsn))
	c.MembersConfigFile = write("members.json", raw)
	nodes := nodestest.New(t, nodestest.NetBird{URL: members.NetBirdURL, Deployment: members.NetBirdDeployment, TokenFile: members.NetBirdTokenFile})
	c.NodesConfigFile, c.AgentCAFile = nodes.Path, nodes.Nodes.CACertificateFile
	c.PublicAddress = availableAddress(t)
	c.MetricsAddress = availableAddress(t)
	c.ShutdownSeconds = 3
	c.WorkerConcurrency = 1
	c.OIDCClientID = "TEST-ONLY-portal"
	c.PublicOrigin = testOrigin
	idp := newIDP(t, true)
	c.OIDCIssuer = idp.server.URL
	raw, e = json.Marshal(c)
	if e != nil {
		t.Fatal(e)
	}
	configFile := write("central.json", raw)
	trusted := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: idp.server.Certificate().Raw})
	trusted = append(trusted, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: nbServer.Certificate().Raw})...)
	trustFile := write("test-roots.pem", trusted)
	roots := x509.NewCertPool()
	cert, e := os.ReadFile(c.TLSCertificateFile)
	if e != nil {
		t.Fatal(e)
	}
	if !roots.AppendCertsFromPEM(cert) {
		t.Fatal("test CA")
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
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
	start := func(name string) (*exec.Cmd, *synchronizedBuffer, chan error) {
		t.Helper()
		binary := filepath.Join(dir, name)
		build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "./apps/"+strings.TrimPrefix(name, "fvpn-")+"/cmd/"+name)
		build.Dir = root
		if out, e := build.CombinedOutput(); e != nil {
			t.Fatalf("build %s: %v\n%s", name, e, out)
		}
		cmd := exec.Command(binary, "-config", configFile)
		cmd.Env = env
		logs := &synchronizedBuffer{}
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
					t.Errorf("binary %s exited: %v", name, e)
				}
			case <-time.After(10 * time.Second):
				_ = cmd.Process.Kill()
				<-done
				t.Error("binary shutdown timeout")
			}
		})
		return cmd, logs, done
	}
	_, controlLogs, _ := start("fvpn-control")
	base := "https://" + c.PublicAddress
	waitReady(t, client, base+"/readyz")
	f := &fixture{t: t, pool: pool, idp: idp}
	// The fixture's signed OIDC login uses the actual TLS listener and binary.
	f.auth = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, e := http.NewRequestWithContext(r.Context(), r.Method, base+r.URL.RequestURI(), r.Body)
		if e != nil {
			t.Error(e)
			return
		}
		req.Header = r.Header.Clone()
		res, e := client.Do(req)
		if e != nil {
			t.Error(e)
			return
		}
		defer res.Body.Close()
		for k, values := range res.Header {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(res.StatusCode)
		_, _ = io.Copy(w, res.Body)
	})
	session := f.login("TEST-binary-member", "member")
	for _, path := range []string{"/api/v1/member/me", "/api/v1/member/devices"} {
		req, _ := http.NewRequestWithContext(t.Context(), "GET", base+path, nil)
		req.AddCookie(session.cookie)
		res, e := client.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != 200 {
			t.Fatalf("binary member route HTTP %d: %s", res.StatusCode, body)
		}
	}
	// Significant whitespace proves the provider sees the original body and the
	// exact signature that entered Servers' registered middleware.
	body := "{ \n \"id\":\"TEST-binary-webhook\",\"type\":\"customer.subscription.updated\",\"api_version\":\"" + billing.StripeAPIVersion + "\",\"livemode\":false,\"data\":{\"object\":{\"id\":\"TEST-sub\",\"customer\":\"TEST-customer\"}} }"
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write([]byte(timestamp + "." + body))
	signature := "t=" + timestamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))
	for i, header := range []string{signature, signature, "invalid"} {
		req, _ := http.NewRequestWithContext(t.Context(), "POST", base+"/api/v1/webhooks/stripe", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Stripe-Signature", header)
		res, e := client.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		got, _ := io.ReadAll(res.Body)
		res.Body.Close()
		want := 202
		if i == 2 {
			want = 401
		}
		if res.StatusCode != want {
			t.Fatalf("webhook HTTP %d: %s", res.StatusCode, got)
		}
	}
	if f.count(`SELECT count(*) FROM inbox_events`) != 1 {
		t.Fatal("binary webhook replay duplicated")
	}
	// Do not dispatch the synthetic invoice to Stripe. Defer it beyond this run;
	// the worker's own startup reconciliation remains real, durable outbox work.
	f.exec(`UPDATE outbox_events SET available_at=clock_timestamp()+interval '1 day' WHERE event_type='billing.inbox_received'`)
	_, workerLogs, _ := start("fvpn-worker")
	waitReady(t, &http.Client{Timeout: time.Second}, "http://"+c.MetricsAddress+"/readyz")
	deadline := time.Now().Add(10 * time.Second)
	for f.count(`SELECT count(*) FROM outbox_events WHERE event_type IN ('billing.reconcile','members.reconcile','devices.discover') AND delivered_at IS NOT NULL`) < 3 {
		if time.Now().After(deadline) {
			var summary string
			_ = pool.QueryRow(t.Context(), `SELECT coalesce(string_agg(event_type || ':attempt=' || attempt::text || ':delivered=' || (delivered_at IS NOT NULL)::text || ':error=' || coalesce(last_error_code,''), ', '),'') FROM outbox_events`).Scan(&summary)
			t.Fatal("binary scheduler/context did not deliver maintenance work", summary, workerLogs.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
	for _, check := range []struct {
		logs   *synchronizedBuffer
		values []string
	}{
		{controlLogs, []string{"control-core", "members", "GetMemberSelf", "ReceiveProviderWebhook"}},
		{workerLogs, []string{"members", "billing.inbox_received", "devices.discover"}},
	} {
		for _, value := range check.values {
			if !strings.Contains(check.logs.String(), value) {
				t.Errorf("startup registration log missing %s", value)
			}
		}
	}
	// The binaries must reject absent or invalid settings before listening, and
	// identify only the setting name in logs (never its path or wrapped error).
	for _, setting := range []string{"", write("invalid-members.json", []byte(`{}`))} {
		bad := c
		bad.MembersConfigFile = setting
		raw, e := json.Marshal(bad)
		if e != nil {
			t.Fatal(e)
		}
		badPath := write("invalid-central.json", raw)
		for _, name := range []string{"fvpn-control", "fvpn-worker"} {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			cmd := exec.CommandContext(ctx, filepath.Join(dir, name), "-config", badPath)
			cmd.Env = env
			out, e := cmd.CombinedOutput()
			cancel()
			if e == nil || !bytes.Contains(out, []byte(`"setting":"members_config_file"`)) {
				t.Fatalf("%s did not reject/name members_config_file: %v", name, e)
			}
			if bytes.Contains(out, []byte(webhookSecret)) || bytes.Contains(out, []byte(dsn)) {
				t.Fatal("failed startup leaked secret")
			}
		}
	}
	for _, secret := range []string{webhookSecret, signature, dsn} {
		if strings.Contains(controlLogs.String()+workerLogs.String(), secret) {
			t.Fatal("startup/request log leaked secret")
		}
	}
}

func availableAddress(t *testing.T) string {
	t.Helper()
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	address := listener.Addr().String()
	if e = listener.Close(); e != nil {
		t.Fatal(e)
	}
	return address
}
func waitReady(t *testing.T, client *http.Client, url string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	for {
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		res, e := client.Do(req)
		if e == nil {
			res.Body.Close()
			if res.StatusCode == 200 {
				return
			}
		}
		select {
		case <-ctx.Done():
			t.Fatal("binary listener not ready", ctx.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}
}
