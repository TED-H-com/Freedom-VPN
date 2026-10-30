package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestEnrollmentProofMatchesProductVerifier(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{SignatureAlgorithm: x509.ECDSAWithSHA256}, key)
	if err != nil {
		t.Fatal(err)
	}
	req := api.EnrollRequest{
		EnrollmentRequestId: uuid.New(), NodeIncarnationId: uuid.New(),
		CsrPem:       string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der})),
		Platform:     api.NodePlatform{Arch: api.ArchitectureAmd64, Os: api.NodePlatformOsUbuntu, OsVersion: "24.04"},
		AgentRelease: "1.2.3",
	}
	req.Proof, err = enrollmentProof(req, key)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = enrollment.VerifyProof(req); err != nil {
		t.Fatal(err)
	}
}
