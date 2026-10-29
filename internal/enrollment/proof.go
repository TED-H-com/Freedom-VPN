package enrollment

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"regexp"

	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

// ProofInput implements the literal EnrollRequest contract: RFC8785 of the
// entire request without proof (including the exact PEM string). UUIDs in this
// typed service API use their canonical lowercase representation.
func ProofInput(r api.EnrollRequest) ([]byte, error) {
	return canonjson.Marshal(struct {
		RequestID     uuid.UUID        `json:"enrollment_request_id"`
		IncarnationID uuid.UUID        `json:"node_incarnation_id"`
		CSR           string           `json:"csr_pem"`
		Platform      api.NodePlatform `json:"platform"`
		Release       string           `json:"agent_release"`
	}{r.EnrollmentRequestId, r.NodeIncarnationId, r.CsrPem, r.Platform, r.AgentRelease})
}
func VerifyProof(r api.EnrollRequest) (*x509.CertificateRequest, []byte, error) {
	if r.EnrollmentRequestId == uuid.Nil || r.NodeIncarnationId == uuid.Nil || !r.Platform.Os.Valid() || !r.Platform.Arch.Valid() || r.Platform.OsVersion != "24.04" || r.AgentRelease == "latest" || !regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z.+_-]{0,63}$`).MatchString(r.AgentRelease) || len(r.Proof) > 256 {
		return nil, nil, apierr.New(apierr.ValidationFailed, "Invalid enrollment request.")
	}
	csr, err := pki.ParseCSR(r.CsrPem)
	if err != nil {
		return nil, nil, err
	}
	input, err := ProofInput(r)
	if err != nil {
		return nil, nil, apierr.New(apierr.ValidationFailed, "Invalid enrollment request.")
	}
	hash := sha256.Sum256(input)
	proof, err := base64.StdEncoding.Strict().DecodeString(r.Proof)
	if err != nil || base64.StdEncoding.EncodeToString(proof) != r.Proof || !ecdsa.VerifyASN1(csr.PublicKey.(*ecdsa.PublicKey), hash[:], proof) {
		return nil, nil, apierr.New(apierr.ValidationFailed, "Invalid node CSR or proof.")
	}
	return csr, hash[:], nil
}
