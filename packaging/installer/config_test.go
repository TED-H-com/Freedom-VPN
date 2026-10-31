package installer

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/google/uuid"
)

// TEST ONLY CA generated per fixture, never committed key material.
func testServerCA(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestUPD02_ReportingInputsFailBeforeInstall(t *testing.T) {
	dir := t.TempDir()
	fx := mustRelease(t, dir, uuid.NewString(), "1")
	req := mustReq(t, dir, fx)
	for _, raw := range []string{"", "http://control.example.invalid", "https://user:password@control.example.invalid", "https://control.example.invalid/api/v1", "https://control.example.invalid?", "https://control.example.invalid#", "https://control.example.invalid:0", "https://control.example.invalid:65536", "https://control.example.invalid:", "https://127.0.0.1", "https://localhost", "https://[::1]", "https://169.254.169.254", "https://metadata.google.internal", "https://0.0.0.0"} {
		t.Run(raw, func(t *testing.T) {
			env := baseEnv(t, t.TempDir())
			bad := req
			bad.ControlURL = raw
			if _, err := Install(context.Background(), env, bad); err == nil || AsError(err).Exit != ExitUsage {
				t.Fatalf("accepted invalid control: %v", err)
			}
			if joinedCalls(env) != "" {
				t.Fatal("invalid input reached host commands")
			}
		})
	}
	for _, raw := range []string{"https://control.example.invalid/", "https://192.0.2.1:443", "https://100.64.0.1:8443", "https://[2001:db8::1]"} {
		valid := req
		valid.ControlURL = raw
		if err := validateInstall(&valid); err != nil {
			t.Fatal(err)
		}
	}
	valid := req
	valid.ControlURL = "https://127.0.0.1:8443"
	valid.AllowLoopback = true
	if err := validateInstall(&valid); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{"", "not a PEM certificate"} {
		if err := os.WriteFile(req.ServerCA, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		env := baseEnv(t, t.TempDir())
		if _, err := Install(context.Background(), env, req); err == nil || AsError(err).Exit != ExitUsage {
			t.Fatalf("bad CA: %v", err)
		}
		if joinedCalls(env) != "" {
			t.Fatal("bad CA reached host commands")
		}
	}
}

func TestUPD02_ReportingRepairPreservesIdentityAndOtherConfig(t *testing.T) {
	dir := t.TempDir()
	env := baseEnv(t, dir)
	fx := mustRelease(t, dir, uuid.NewString(), "1")
	req := mustReq(t, dir, fx)
	if _, err := Install(context.Background(), env, req); err != nil {
		t.Fatal(err)
	}
	identity := hashFile(t, env.Layout.nodeFile())
	raw, err := os.ReadFile(env.Layout.updaterConfig())
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	fields["result_url"] = ""
	fields["mtls_ca"] = env.Layout.IdentityDir + "/ca.pem"
	delete(fields, "allowed_artifact_origins")
	fields["meter_checkpoint_wait_ms"] = 1234
	raw, err = marshalIndent(fields)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env.Layout.updaterConfig(), raw, 0644); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		res, err := Install(context.Background(), env, req)
		if err != nil || res.Status != statusAlreadyCurrent {
			t.Fatalf("repair %+v %v", res, err)
		}
		cfg, err := updater.LoadConfig(env.Layout.updaterConfig())
		if err != nil {
			t.Fatal(err)
		}
		if cfg.ResultURL != req.ControlURL+"/api/v1/agent/update-results" || cfg.MTLSCA != env.Layout.serverCA() || cfg.AllowLoopback || len(cfg.AllowedArtifactOrigins) != 1 || cfg.AllowedArtifactOrigins[0] != req.ControlURL || cfg.MeterCheckpointWaitMS != 1234 {
			t.Fatalf("reporting repair: %+v", cfg)
		}
		raw, err = os.ReadFile(env.Layout.updaterConfig())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), `"boot_id"`) || identity != hashFile(t, env.Layout.nodeFile()) {
			t.Fatal("repair changed identity or pinned the boot")
		}
	}
}
