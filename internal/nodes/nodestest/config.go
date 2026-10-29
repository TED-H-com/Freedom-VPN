// Package nodestest provisions TEST ONLY node-domain dependencies for composition
// and binary tests. It has no dependency on either application's composition root.
package nodestest

import (
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
)

// NetBird allows a binary fixture to reuse its local TLS fake, deployment and
// token file. The caller supplies TLS trust for any actual adapter requests.
// Leave URL empty to start a local TLS fake that rejects unexpected requests.
type NetBird struct {
	URL, Deployment, TokenFile string
}

type Fixture struct {
	Nodes config.Nodes
	Path  string
}

// New writes a private configuration, ephemeral authority and synthetic token
// under t.TempDir. All lifetimes and resolver addresses below are TEST ONLY.
// It never changes process environment or supplies a production default.
func New(t testing.TB, nb NetBird) Fixture {
	t.Helper()
	dir := t.TempDir()
	write := func(name string, data []byte) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	if nb.URL == "" {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			t.Error("unexpected NetBird request: supply a behavior-specific local fake")
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		t.Cleanup(server.Close)
		nb.URL = server.URL + "/api"
	}
	if nb.Deployment == "" {
		nb.Deployment = "TEST ONLY"
	}
	if nb.TokenFile == "" {
		nb.TokenFile = write("netbird-token", []byte("TEST ONLY synthetic token"))
	}
	vaultPath := filepath.Join(dir, "vault")
	vault, err := signing.InitializeFileProvider(vaultPath)
	if err != nil {
		t.Fatal(err)
	}
	key, err := vault.Rotate(t.Context(), signing.NodeCA, 48*time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = vault.Rotate(t.Context(), signing.BootstrapSecret, 48*time.Hour, time.Hour); err != nil {
		t.Fatal(err)
	}
	der, err := pki.CreateDevCA(t.Context(), vault, key, 48*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	n := config.Nodes{
		ProviderDirectory: vaultPath,
		CACertificateFile: write("node-ca.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		CAKeyID:           key, CertificateSeconds: 86400, RenewalOverlapSeconds: 300,
		BootstrapSeconds: 60, InstallerSeconds: 900, PreinstalledSeconds: 604800,
		AuthorizationSeconds: 300, OnlineSeconds: 180, StuckSeconds: 600,
		ReconcileSeconds: 1, FetchSeconds: 10, DesiredStateSeconds: 600,
		ReleaseOrigins:  []string{"https://releases.example.invalid"},
		DNSResolverIPv4: []string{"100.64.0.53"}, DNSResolverIPv6: []string{"2001:db8::53"},
		NetBirdURL: nb.URL, NetBirdDeployment: nb.Deployment, NetBirdTokenFile: nb.TokenFile,
	}
	if err := n.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	return Fixture{Nodes: n, Path: write("nodes.json", raw)}
}
