package fakecontrol

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"regexp"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

// proofInput is the RFC 8785 enrollment document without proof (§6.1).
// It matches the central enrollment proof input field-for-field. This package
// does not import the central enrollment service.
func proofInput(r api.EnrollRequest) ([]byte, error) {
	return canonjson.Marshal(struct {
		RequestID     uuid.UUID        `json:"enrollment_request_id"`
		IncarnationID uuid.UUID        `json:"node_incarnation_id"`
		CSR           string           `json:"csr_pem"`
		Platform      api.NodePlatform `json:"platform"`
		Release       string           `json:"agent_release"`
	}{r.EnrollmentRequestId, r.NodeIncarnationId, r.CsrPem, r.Platform, r.AgentRelease})
}

func verifyEnroll(r api.EnrollRequest) (*x509.CertificateRequest, error) {
	if r.EnrollmentRequestId == uuid.Nil || r.NodeIncarnationId == uuid.Nil || !r.Platform.Os.Valid() || !r.Platform.Arch.Valid() || r.Platform.OsVersion != "24.04" || r.AgentRelease == "latest" || !regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z.+_-]{0,63}$`).MatchString(r.AgentRelease) || len(r.Proof) > 256 {
		return nil, errInvalid
	}
	csr, err := parseCSR(r.CsrPem)
	if err != nil {
		return nil, errInvalid
	}
	input, err := proofInput(r)
	if err != nil {
		return nil, errInvalid
	}
	sum := sha256.Sum256(input)
	proof, err := base64.StdEncoding.Strict().DecodeString(r.Proof)
	if err != nil || base64.StdEncoding.EncodeToString(proof) != r.Proof {
		return nil, errInvalid
	}
	pub, ok := csr.PublicKey.(*ecdsa.PublicKey)
	if !ok || !ecdsa.VerifyASN1(pub, sum[:], proof) {
		return nil, errInvalid
	}
	return csr, nil
}

func parseCSR(input string) (*x509.CertificateRequest, error) {
	if len(input) > 8192 || !bytes.HasPrefix([]byte(input), []byte("-----BEGIN CERTIFICATE REQUEST-----")) {
		return nil, errInvalid
	}
	b, rest := pem.Decode([]byte(input))
	if b == nil || b.Type != "CERTIFICATE REQUEST" || len(b.Headers) > 0 || len(bytes.TrimSpace(rest)) > 0 {
		return nil, errInvalid
	}
	csr, err := x509.ParseCertificateRequest(b.Bytes)
	if err != nil {
		return nil, errInvalid
	}
	key, ok := csr.PublicKey.(*ecdsa.PublicKey)
	if !ok || key.Curve != elliptic.P256() || csr.CheckSignature() != nil || csr.SignatureAlgorithm != x509.ECDSAWithSHA256 || !bytes.Equal(csr.RawSubject, []byte{0x30, 0}) || len(csr.Extensions) > 0 {
		return nil, errInvalid
	}
	var raw struct {
		Info struct {
			Version    int
			Subject    asn1.RawValue
			Public     asn1.RawValue
			Attributes []asn1.RawValue `asn1:"tag:0"`
		}
		Algorithm asn1.RawValue
		Signature asn1.BitString
	}
	if rest, err := asn1.Unmarshal(b.Bytes, &raw); err != nil || len(rest) > 0 || raw.Info.Version != 0 || len(raw.Info.Attributes) > 0 {
		return nil, errInvalid
	}
	return csr, nil
}

func fingerprint(csr *x509.CertificateRequest) string {
	sum := sha256.Sum256(csr.RawSubjectPublicKeyInfo)
	return base64.StdEncoding.EncodeToString(sum[:])
}
