package agent

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"runtime"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

// proofInput is the RFC 8785 enrollment document without the proof member (§6.1).
func proofInput(r api.EnrollRequest) ([]byte, error) {
	return canonjson.Marshal(struct {
		RequestID     uuid.UUID        `json:"enrollment_request_id"`
		IncarnationID uuid.UUID        `json:"node_incarnation_id"`
		CSR           string           `json:"csr_pem"`
		Platform      api.NodePlatform `json:"platform"`
		Release       string           `json:"agent_release"`
	}{r.EnrollmentRequestId, r.NodeIncarnationId, r.CsrPem, r.Platform, r.AgentRelease})
}

func signProof(key *ecdsa.PrivateKey, req api.EnrollRequest) (string, error) {
	raw, err := proofInput(req)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	sig, err := ecdsa.SignASN1(rand.Reader, key, sum[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func newKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

func marshalKey(key *ecdsa.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil
}

func parseKey(pemBytes []byte) (*ecdsa.PrivateKey, error) {
	block, rest := pem.Decode(pemBytes)
	if block == nil || len(rest) > 0 {
		return nil, errors.New("agent: identity key")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("agent: identity key")
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok || key.Curve != elliptic.P256() {
		return nil, errors.New("agent: identity key")
	}
	return key, nil
}

// newCSR builds an empty-subject P-256 CSR. The CA chooses the identity (§16.2).
func newCSR(key *ecdsa.PrivateKey) (string, error) {
	tmpl := &x509.CertificateRequest{SignatureAlgorithm: x509.ECDSAWithSHA256}
	der, err := x509.CreateCertificateRequest(rand.Reader, tmpl, key)
	if err != nil {
		return "", err
	}
	csr, err := x509.ParseCertificateRequest(der)
	if err != nil {
		return "", err
	}
	if csr.PublicKey.(*ecdsa.PublicKey).Curve != elliptic.P256() || !bytes.Equal(csr.RawSubject, []byte{0x30, 0}) || len(csr.Extensions) != 0 {
		return "", errors.New("agent: csr subject is not empty")
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der})), nil
}

func publicSHA256(key *ecdsa.PrivateKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(der)
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(sum)*2)
	for i, c := range sum {
		out[i*2] = hexdigits[c>>4]
		out[i*2+1] = hexdigits[c&0x0f]
	}
	return string(out), nil
}

func localPlatform() (api.NodePlatform, error) {
	var arch api.Architecture
	switch runtime.GOARCH {
	case "amd64":
		arch = api.ArchitectureAmd64
	case "arm64":
		arch = api.ArchitectureArm64
	default:
		return api.NodePlatform{}, errors.New("agent: unsupported architecture")
	}
	return api.NodePlatform{Arch: arch, Os: api.NodePlatformOsUbuntu, OsVersion: "24.04"}, nil
}
