package control

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"io/fs"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/policies/policiestest"
)

func TestAT28_AgentTLSRequiresValidCertificate(t *testing.T) {
	key, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	now := time.Now()
	ca := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "TEST ONLY CA"}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	der, e := x509.CreateCertificate(rand.Reader, ca, ca, &key.PublicKey, key)
	if e != nil {
		t.Fatal(e)
	}
	root, e := x509.ParseCertificate(der)
	if e != nil {
		t.Fatal(e)
	}
	roots := x509.NewCertPool()
	roots.AddCert(root)
	issue := func(client, expired bool) tls.Certificate {
		k, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if e != nil {
			t.Fatal(e)
		}
		end := now.Add(time.Hour)
		if expired {
			end = now.Add(-time.Minute)
		}
		template := &x509.Certificate{SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "TEST ONLY identity"}, NotBefore: now.Add(-time.Hour), NotAfter: end, IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)}, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
		if client {
			template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		}
		der, e := x509.CreateCertificate(rand.Reader, template, root, &k.PublicKey, key)
		if e != nil {
			t.Fatal(e)
		}
		return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: k}
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.TLS.VerifiedChains) == 0 {
			t.Error("unverified connection reached handler")
		}
		w.WriteHeader(200)
	}))
	server.TLS = AgentTLSConfig(issue(false, false), roots)
	server.Config.ErrorLog = nil
	server.StartTLS()
	defer server.Close()
	for _, tc := range []struct {
		name  string
		certs []tls.Certificate
		valid bool
	}{{"no cert", nil, false}, {"expired", []tls.Certificate{issue(true, true)}, false}, {"valid", []tls.Certificate{issue(true, false)}, true}} {
		t.Run(tc.name, func(t *testing.T) {
			transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots, Certificates: tc.certs}}
			defer transport.CloseIdleConnections()
			client := &http.Client{Transport: transport, Timeout: time.Second}
			response, e := client.Get(server.URL)
			if e == nil {
				defer response.Body.Close()
			}
			if (e == nil) != tc.valid {
				t.Fatal("TLS acceptance differs", e)
			}
		})
	}
}

func TestD045_PoliciesAndUsageDomainsRequireConfigAndSigningFiles(t *testing.T) {
	ctx := context.Background()
	c := config.Defaults()

	// 1. Missing policies_config_file fails with SettingError naming policies_config_file.
	_, err := Servers(ctx, Dependencies{Config: c}, policies(), usage())
	var setting *config.SettingError
	if !errors.As(err, &setting) || setting.Setting() != "policies_config_file" {
		t.Fatalf("missing policies_config_file: got %v", err)
	}

	// 2. Missing usage_config_file fails with SettingError naming usage_config_file.
	c.PoliciesConfigFile = policiestest.New(t).Path
	_, err = Servers(ctx, Dependencies{Config: c}, policies(), usage())
	setting = nil
	if !errors.As(err, &setting) || setting.Setting() != "usage_config_file" {
		t.Fatalf("missing usage_config_file: got %v", err)
	}

	// 3. Missing signing_config_file inside usage_config_file fails with SettingError naming signing_config_file.
	dir := t.TempDir()
	usageNoSigning := filepath.Join(dir, "usage-no-signing.json")
	if err := os.WriteFile(usageNoSigning, []byte(`{
		"raw_usage_retention_days": 90,
		"rollup_retention_months": 24,
		"audit_retention_days": 365,
		"projection_batch_size": 256
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c.UsageConfigFile = usageNoSigning
	_, err = Servers(ctx, Dependencies{Config: c}, policies(), usage())
	setting = nil
	if !errors.As(err, &setting) || setting.Setting() != "signing_config_file" {
		t.Fatalf("missing signing_config_file in usage file: got %v", err)
	}

	// 4. Unreadable signing_config_file inside usage_config_file fails with SettingError naming signing_config_file.
	usageBadVault := filepath.Join(dir, "usage-bad-vault.json")
	if err := os.WriteFile(usageBadVault, []byte(`{
		"signing_config_file": "`+filepath.Join(dir, "nonexistent-vault")+`",
		"raw_usage_retention_days": 90,
		"rollup_retention_months": 24,
		"audit_retention_days": 365,
		"projection_batch_size": 256
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c.UsageConfigFile = usageBadVault
	_, err = Servers(ctx, Dependencies{Config: c}, usage())
	setting = nil
	if !errors.As(err, &setting) || setting.Setting() != "signing_config_file" {
		t.Fatalf("unreadable signing_config_file: got %v", err)
	}
}

func TestD045_NoProductionSigningSeedsOrTestOnlyKeys(t *testing.T) {
	for _, root := range []string{"../../apps", "../../internal"} {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				name := d.Name()
				if strings.HasSuffix(name, "test") || name == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			src := string(raw)
			if strings.Contains(src, "ed25519.NewKeyFromSeed") {
				t.Errorf("production file %s contains ed25519.NewKeyFromSeed", path)
			}
			for _, line := range strings.Split(src, "\n") {
				lower := strings.ToLower(line)
				if strings.Contains(line, "TEST ONLY") && (strings.Contains(lower, "seed") || strings.Contains(lower, "ed25519") || strings.Contains(lower, "signing")) {
					t.Errorf("production file %s contains TEST ONLY signing seed: %s", path, strings.TrimSpace(line))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
}
