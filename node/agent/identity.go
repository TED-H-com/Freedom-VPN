package agent

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

func (c Config) keyPath() string    { return filepath.Join(c.IdentityDir, "key.pem") }
func (c Config) csrPath() string    { return filepath.Join(c.IdentityDir, "csr.pem") }
func (c Config) certPath() string   { return filepath.Join(c.IdentityDir, "cert.pem") }
func (c Config) caPath() string     { return filepath.Join(c.IdentityDir, "ca.pem") }
func (c Config) nodePath() string   { return filepath.Join(c.IdentityDir, "node.json") }
func (c Config) enrollPath() string { return filepath.Join(c.IdentityDir, "enrollment.json") }
func (c Config) setupKeyPath() string {
	return filepath.Join(c.IdentityDir, "setup.key")
}
func (c Config) setupMetaPath() string {
	return filepath.Join(c.IdentityDir, "setup.meta")
}
func (c Config) renewCSRPath() string {
	return filepath.Join(c.IdentityDir, "renew.csr")
}

type enrollmentFile struct {
	EnrollmentRequestID string `json:"enrollment_request_id"`
	NodeIncarnationID   string `json:"node_incarnation_id"`
}

type nodeFile struct {
	NodeID            string    `json:"node_id"`
	NodeIncarnationID string    `json:"node_incarnation_id"`
	Serial            string    `json:"certificate_serial"`
	NotAfter          time.Time `json:"certificate_not_after"`
}

func loadOrCreateKey(path string) (*ecdsa.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err == nil {
		return parseKey(b)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	key, err := newKey()
	if err != nil {
		return nil, err
	}
	raw, err := marshalKey(key)
	if err != nil {
		return nil, err
	}
	if err := writeAtomic(path, raw, 0o600); err != nil {
		return nil, err
	}
	return key, nil
}

func writeEnrollmentFile(path string, request, inc uuid.UUID) error {
	raw, err := json.Marshal(enrollmentFile{EnrollmentRequestID: request.String(), NodeIncarnationID: inc.String()})
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return writeAtomic(path, raw, 0o600)
}

func writeNodeFile(path string, n nodeFile) error {
	raw, err := json.Marshal(n)
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return writeAtomic(path, raw, 0o600)
}

func loadNodeFile(path string) (nodeFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nodeFile{}, err
	}
	var n nodeFile
	if err := json.Unmarshal(b, &n); err != nil {
		return nodeFile{}, err
	}
	return n, nil
}

func certFromPEM(raw []byte) (*x509.Certificate, error) {
	block, rest := pem.Decode(raw)
	if block == nil || block.Type != "CERTIFICATE" || len(bytes.TrimSpace(rest)) > 0 {
		return nil, errors.New("agent: certificate")
	}
	return x509.ParseCertificate(block.Bytes)
}

func keyMatches(cert *x509.Certificate, key *ecdsa.PrivateKey) bool {
	pub, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok || key == nil {
		return false
	}
	return pub.Equal(&key.PublicKey)
}

func tlsPair(certPEM, keyPEM []byte) (tls.Certificate, error) {
	return tls.X509KeyPair(certPEM, keyPEM)
}
