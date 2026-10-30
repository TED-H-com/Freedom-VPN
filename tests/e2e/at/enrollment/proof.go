package enrollment

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

// holder exists only in memory. No key, CSR, proof or token is written to records.
type holder struct {
	key         *ecdsa.PrivateKey
	body        []byte
	fingerprint string
	incarnation uuid.UUID
	proofHash   string
}

func newHolder() (holder, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return holder{}, err
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{}, key)
	if err != nil {
		return holder{}, err
	}
	inc := uuid.New()
	// Independent map encoder follows the public contract, not enrollment.Service.
	doc := map[string]any{
		"enrollment_request_id": uuid.NewString(), "node_incarnation_id": inc.String(),
		"csr_pem":       string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr})),
		"platform":      map[string]string{"os": "ubuntu", "os_version": "24.04", "arch": "amd64"},
		"agent_release": "0.1.0",
	}
	canonical, err := canonjson.Marshal(doc)
	if err != nil {
		return holder{}, err
	}
	digest := sha256.Sum256(canonical)
	proof, err := ecdsa.SignASN1(rand.Reader, key, digest[:])
	if err != nil {
		return holder{}, err
	}
	doc["proof"] = base64.StdEncoding.EncodeToString(proof)
	body, err := json.Marshal(doc)
	if err != nil {
		return holder{}, err
	}
	public, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return holder{}, err
	}
	return holder{key, body, hash(public), inc, hex.EncodeToString(digest[:])}, nil
}

func hash(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }

var errDropped = errors.New("TEST ONLY enrollment success response discarded")

// dropSuccess is a transport fault boundary. The caller receives no response;
// the observer retains only public identifiers/digests for later comparison.
type dropSuccess struct {
	base            http.RoundTripper
	dropped         bool
	status          int
	incarnation     uuid.UUID
	node            uuid.UUID
	certificateHash string
}

func (d *dropSuccess) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := d.base.RoundTrip(req)
	if err != nil || d.dropped || resp.StatusCode != http.StatusCreated {
		return resp, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var result api.EnrollResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, errors.New("invalid discarded response")
	}
	d.dropped, d.status = true, resp.StatusCode
	d.incarnation, d.node = result.NodeIncarnationId, result.NodeId
	d.certificateHash = hash([]byte(result.CertificatePem))
	return nil, errDropped
}
