// Package pkitest creates ephemeral keys; no test private key is committed.
package pkitest

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

func Authority(t testing.TB) (*signing.FileProvider, *pki.CA) {
	t.Helper()
	ctx := context.Background()
	p, err := signing.InitializeFileProvider(filepath.Join(t.TempDir(), "vault"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := p.Rotate(ctx, signing.NodeCA, 30*24*time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.Rotate(ctx, signing.BootstrapSecret, 30*24*time.Hour, time.Hour); err != nil {
		t.Fatal(err)
	}
	der, err := pki.CreateDevCA(ctx, p, id, 30*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	ca, err := pki.NewCA(ctx, p, id, der, pki.TestCertificateValidity)
	if err != nil {
		t.Fatal(err)
	}
	return p, ca
}
func CSR(t testing.TB) (*ecdsa.PrivateKey, string) {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return k, CSRWith(t, k, &x509.CertificateRequest{})
}
func CSRWith(t testing.TB, k *ecdsa.PrivateKey, template *x509.CertificateRequest) string {
	t.Helper()
	der, err := x509.CreateCertificateRequest(rand.Reader, template, k)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der}))
}
func Request(t testing.TB) (api.EnrollRequest, *ecdsa.PrivateKey) {
	t.Helper()
	k, csr := CSR(t)
	r := api.EnrollRequest{EnrollmentRequestId: uuid.New(), NodeIncarnationId: uuid.New(), CsrPem: csr, Platform: api.NodePlatform{Os: "ubuntu", OsVersion: "24.04", Arch: "amd64"}, AgentRelease: "1.0.0"}
	SignRequest(t, &r, k)
	return r, k
}

// SignRequest uses an independent map representation to exercise the public
// proof contract rather than the server's ProofInput implementation.
func SignRequest(t testing.TB, r *api.EnrollRequest, k *ecdsa.PrivateKey) {
	t.Helper()
	raw, err := canonjson.Marshal(map[string]any{"enrollment_request_id": r.EnrollmentRequestId, "node_incarnation_id": r.NodeIncarnationId, "csr_pem": r.CsrPem, "platform": r.Platform, "agent_release": r.AgentRelease})
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	sig, err := ecdsa.SignASN1(rand.Reader, k, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	r.Proof = base64.StdEncoding.EncodeToString(sig)
}
func Certificate(t testing.TB, p string) *x509.Certificate {
	t.Helper()
	b, _ := pem.Decode([]byte(p))
	if b == nil {
		t.Fatal("invalid certificate PEM")
	}
	c, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
