package updater

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUPD05_InstallerReportingConfigShape(t *testing.T) {
	root := t.TempDir()
	layout := testLayout(root)
	for _, dir := range []string{filepath.Dir(layout.Trust), layout.Identity} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeTrust(t, layout.Trust)
	desiredTrust := filepath.Join(filepath.Dir(layout.Trust), "desired-trust.json")
	writeTrust(t, desiredTrust)

	// TEST ONLY identity, generated for this run and trusted by this local
	// server. Enrollment/central authorization is measured by updates 11.
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "TEST ONLY node"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, clientTemplate, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	clientCert, err := x509.ParseCertificate(clientDER)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPath, keyPath := filepath.Join(layout.Identity, "cert.pem"), filepath.Join(layout.Identity, "key.pem")
	for path, block := range map[string]*pem.Block{
		certPath: {Type: "CERTIFICATE", Bytes: clientDER},
		keyPath:  {Type: "PRIVATE KEY", Bytes: keyDER},
	} {
		if err := os.WriteFile(path, pem.EncodeToMemory(block), 0600); err != nil {
			t.Fatal(err)
		}
	}
	resultID := uuid.NewString()
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/v1/agent/update-results" || req.Header.Get("Idempotency-Key") != resultID {
			t.Error("installer-shaped config did not reach the result route with its receipt identity")
		}
		if req.TLS == nil || len(req.TLS.VerifiedChains) == 0 || !bytes.Equal(req.TLS.PeerCertificates[0].Raw, clientDER) {
			t.Error("configured client certificate was not authenticated")
		}
		rw.WriteHeader(http.StatusAccepted)
		_, _ = rw.Write([]byte(`{"accepted":true,"accepted_as_history":false}`))
	}))
	clientRoots := x509.NewCertPool()
	clientRoots.AddCert(clientCert)
	srv.TLS = &tls.Config{MinVersion: tls.VersionTLS12, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: clientRoots}
	srv.StartTLS()
	defer srv.Close()
	serverRoots := x509.NewCertPool()
	serverRoots.AddCert(srv.Certificate())
	caPath := filepath.Join(filepath.Dir(layout.Trust), "ca.pem")
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw}), 0600); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, origin string
		loopback     bool
	}{
		{"control_cgnat", "https://100.64.0.10:9443", false}, // Policy only; never dial this address.
		{"test_only_loopback", srv.URL, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Literal JSON keys mirror installer.renderUpdater, including null
			// revoked_release_ids and the absent boot_id fallback. Do not marshal
			// Config here: a changed consumer tag must break this contract test.
			// Fixtures are constructed locally; no living AT evidence is read.
			fields := map[string]any{
				"layout": map[string]any{
					"releases": layout.Releases, "current": layout.Current, "identity": layout.Identity,
					"state": layout.State, "journal": layout.Journal, "lock": layout.Lock,
					"staging": layout.Staging, "trust": layout.Trust,
				},
				"node_id": uuid.NewString(), "incarnation_id": uuid.NewString(), "revoked": false,
				"platform": map[string]any{
					"os_id": "ubuntu", "os_version": "24.04", "architecture": "amd64", "updater_version": "1.0.0",
					"control_api_major": 1, "policy_schema": 1, "usage_schema": 1, "state_schema": 1,
				},
				"release_trust": layout.Trust, "desired_trust": desiredTrust,
				"desired_state": filepath.Join(layout.State, "desired-state.json"), "revoked_release_ids": nil,
				"meter_checkpoint": filepath.Join(layout.State, "meter.checkpoint"),
				"stop_command":     []string{"systemctl", "stop", "fvpn-agent.service", "fvpn-watchdog.service"},
				"start_command":    []string{"/bin/sh", "-c", "exec systemctl start fvpn-watchdog.service fvpn-agent.service", "fvpn-updater-start"},
				"result_url":       tc.origin + "/api/v1/agent/update-results",
				"mtls_cert":        certPath, "mtls_key": keyPath, "mtls_ca": caPath,
				"allow_loopback": tc.loopback, "allowed_artifact_origins": []string{tc.origin},
			}
			raw, err := json.Marshal(fields)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "updater.json")
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err := LoadConfig(path)
			if err != nil {
				t.Fatal(err)
			}
			eng, err := cfg.Engine()
			if err != nil {
				t.Fatal(err)
			}
			defer eng.Close()
			sink, ok := eng.Sender.(HTTPSink)
			if !ok || sink.URL != fields["result_url"] || sink.Client == nil || sink.Client != eng.HTTP {
				t.Fatal("installer-provisioned node can reach result_url_missing or has lost its reporting transport")
			}
			if sink.AllowLoopback != tc.loopback || !reflect.DeepEqual(sink.AllowedOrigins, []string{tc.origin}) {
				t.Fatal("rendered URL policy was not wired to HTTPSink")
			}
			policy := urlPolicy{AllowLoopback: sink.AllowLoopback, AllowedOrigins: sink.AllowedOrigins}
			if err := checkURLPolicy(sink.URL, policy); err != nil {
				t.Fatal("control origin rejected", err)
			}
			transport, ok := sink.Client.Transport.(*http.Transport)
			if !ok || transport.TLSClientConfig == nil {
				t.Fatal("TLS transport missing")
			}
			tlsConfig := transport.TLSClientConfig
			if tlsConfig.InsecureSkipVerify || tlsConfig.RootCAs == nil || !tlsConfig.RootCAs.Equal(serverRoots) || tlsConfig.RootCAs.Equal(clientRoots) {
				t.Fatal("result endpoint must use enrollment server trust, distinct from the client identity")
			}
			if len(tlsConfig.Certificates) != 1 || !bytes.Equal(tlsConfig.Certificates[0].Certificate[0], clientDER) {
				t.Fatal("enrolled client identity missing from transport")
			}
			if tc.loopback {
				ack, conflict, err := sink.Submit(t.Context(), []byte(`{"result_id":"`+resultID+`"}`))
				if err != nil || conflict || len(ack.Raw) == 0 {
					t.Fatal("installer-shaped reporting failed authenticated local delivery", err)
				}
				cfg.AllowLoopback = false
			} else {
				cfg.AllowedArtifactOrigins = nil
			}
			// An origin entry cannot grant loopback; a CGNAT control needs its
			// exact origin exception. Both invalid variants fail construction.
			if invalid, err := cfg.Engine(); err == nil {
				_ = invalid.Close()
				t.Fatal("missing URL-policy permission was accepted")
			}
		})
	}
}
