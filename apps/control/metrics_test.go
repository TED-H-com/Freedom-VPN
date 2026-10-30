package control

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"log/slog"
	"math/big"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

type boundListener struct{ name, address string }
type listenerLog struct {
	slog.Handler
	bound chan boundListener
}

func (h listenerLog) Handle(ctx context.Context, record slog.Record) error {
	if record.Message == "control listener started" {
		var listener boundListener
		record.Attrs(func(a slog.Attr) bool {
			switch a.Key {
			case "listener":
				listener.name = a.Value.String()
			case "address":
				listener.address = a.Value.String()
			}
			return true
		})
		h.bound <- listener
	}
	return h.Handler.Handle(ctx, record)
}

func metricsConfig(t *testing.T) (config.Config, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	c := config.Defaults()
	c.PublicAddress, c.AgentAddress, c.MetricsAddress = "127.0.0.1:0", "127.0.0.1:0", "127.0.0.1:0"
	c.ShutdownSeconds = 2
	dir := t.TempDir()
	c.TLSCertificateFile, c.TLSKeyFile = filepath.Join(dir, "server.crt"), filepath.Join(dir, "server.key")
	c.AgentCAFile = c.TLSCertificateFile
	cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(c.TLSCertificateFile, cert, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.TLSKeyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}), 0600); err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(cert) {
		t.Fatal("invalid test certificate")
	}
	return c, roots
}

func TestAPI01_ControlMetricsListenerLifecycle(t *testing.T) {
	c, roots := metricsConfig(t)
	registry := prometheus.NewRegistry()
	logs := listenerLog{Handler: slog.NewTextHandler(io.Discard, nil), bound: make(chan boundListener, 3)}
	app, err := Servers(t.Context(), Dependencies{Config: c, Pool: &pgxpool.Pool{}, Browser: &auth.Browser{}, Metrics: registry, Logger: slog.New(logs)}, Domain{Name: "TEST-metrics", Build: func(_ context.Context, d Dependencies) (Built, error) {
		if d.Metrics != registry {
			t.Fatal("domain received a different registry")
		}
		gauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: "fvpn_test_domain_value", Help: "TEST ONLY domain collector."})
		gauge.Set(7)
		return Built{}, d.Metrics.Register(gauge)
	}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	var runErr error
	go func() { runErr = Run(ctx, app); close(done) }()
	stop := func() {
		cancel()
		select {
		case <-done:
			if runErr != nil {
				t.Error(runErr)
			}
		case <-time.After(5 * time.Second):
			t.Error("control listeners did not stop")
		}
	}
	t.Cleanup(stop)
	addresses := map[string]string{}
	for len(addresses) < 3 {
		select {
		case bound := <-logs.bound:
			addresses[bound.name] = bound.address
		case <-done:
			t.Fatal("application stopped before binding listeners", runErr)
		case <-time.After(5 * time.Second):
			t.Fatal("missing bound listener log")
		}
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: time.Second}
	response, err := client.Get("https://" + addresses["public"] + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatal("public listener unavailable", response.Status)
	}
	response, err = client.Get("http://" + addresses["metrics"] + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if response.StatusCode != http.StatusOK || err != nil || contentType != "text/plain" {
		t.Fatal("invalid metrics response", response.Status, response.Header, err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	for _, sample := range []string{"fvpn_test_domain_value 7", `fvpn_control_http_requests_total{route="GET /healthz",status="200"} 1`} {
		if !strings.Contains(string(body), sample) {
			t.Fatalf("missing metric %q in %s", sample, body)
		}
	}
	stop()
	for name, address := range addresses {
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if err == nil {
			_ = conn.Close()
			t.Errorf("%s listener still accepts connections", name)
		}
	}
}

func TestSEC02_ControlMetricsLoopbackAndBindFailure(t *testing.T) {
	c, _ := metricsConfig(t)
	for _, address := range []string{"0.0.0.0:9091", "[::]:9091", "192.0.2.1:9091", ":9091", "invalid"} {
		c.MetricsAddress = address
		if _, err := Servers(t.Context(), Dependencies{Config: c, Pool: &pgxpool.Pool{}, Browser: &auth.Browser{}}, []Domain{}...); err == nil || !strings.Contains(err.Error(), "metrics_address") {
			t.Fatalf("unsafe metrics address %q: %v", address, err)
		}
	}
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	c.MetricsAddress = busy.Addr().String()
	var started atomic.Bool
	app, err := Servers(t.Context(), Dependencies{Config: c, Pool: &pgxpool.Pool{}, Browser: &auth.Browser{}}, Domain{Name: "TEST-runner", Build: func(context.Context, Dependencies) (Built, error) {
		return Built{Runners: []func(context.Context) error{func(context.Context) error { started.Store(true); return nil }}}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(t.Context(), app); err == nil || started.Load() {
		t.Fatal("metrics bind failure must prevent runner startup", err, started.Load())
	}
}
