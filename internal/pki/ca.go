// Package pki issues and authenticates node-only ECDSA P-256 identities.
package pki

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"math/big"
	"net/url"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

// TEST ONLY defaults. Production certificate validity and overlap are 待決策.
const TestCertificateValidity = 24 * time.Hour
const TestRenewalOverlap = 5 * time.Minute

type NodeIdentity struct {
	NodeID        uuid.UUID
	IncarnationID uuid.UUID
}
type AgentAuthenticator interface {
	Authenticate(context.Context, *x509.Certificate) (NodeIdentity, error)
}
type Revocations interface {
	IdentityDenied(context.Context, string) (bool, error)
	DenyIdentity(context.Context, string) error
}

func invalidCSR() error { return apierr.New(apierr.ValidationFailed, "Invalid node CSR or proof.") }

// ParseCSR permits an empty subject and no attributes/extensions. The CA alone
// chooses identity SANs; a node never supplies names or personal data (§16.2).
func ParseCSR(input string) (*x509.CertificateRequest, error) {
	if len(input) > 8192 || !bytes.HasPrefix([]byte(input), []byte("-----BEGIN CERTIFICATE REQUEST-----")) {
		return nil, invalidCSR()
	}
	b, rest := pem.Decode([]byte(input))
	if b == nil || b.Type != "CERTIFICATE REQUEST" || len(b.Headers) > 0 || len(bytes.TrimSpace(rest)) > 0 {
		return nil, invalidCSR()
	}
	csr, err := x509.ParseCertificateRequest(b.Bytes)
	if err != nil {
		return nil, invalidCSR()
	}
	key, ok := csr.PublicKey.(*ecdsa.PublicKey)
	if !ok || key.Curve != elliptic.P256() || csr.CheckSignature() != nil || csr.SignatureAlgorithm != x509.ECDSAWithSHA256 || !bytes.Equal(csr.RawSubject, []byte{0x30, 0}) || len(csr.Extensions) > 0 || len(csr.Attributes) > 0 {
		return nil, invalidCSR()
	}
	// x509 ignores unrecognized PKCS#10 attributes; inspect the raw ASN.1 too.
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
		return nil, invalidCSR()
	}
	return csr, nil
}
func Fingerprint(csr *x509.CertificateRequest) []byte {
	h := sha256.Sum256(csr.RawSubjectPublicKeyInfo)
	return h[:]
}

type CA struct {
	provider signing.Provider
	keyID    string
	cert     *x509.Certificate
	roots    *x509.CertPool
	validity time.Duration
	now      func() time.Time
}
type Issued struct {
	Certificate *x509.Certificate
	PEM         string
	CAChainPEM  string
	Issuer      string
}

// NewCA takes a public CA certificate and an opaque provider key reference.
func NewCA(ctx context.Context, provider signing.Provider, keyID string, der []byte, validity time.Duration) (*CA, error) {
	if provider == nil || validity <= 0 {
		return nil, errors.New("pki: explicit certificate validity required")
	}
	c, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, errors.New("pki: invalid CA certificate")
	}
	pub, ok := c.PublicKey.(*ecdsa.PublicKey)
	if !ok || pub.Curve != elliptic.P256() || !c.IsCA || !c.BasicConstraintsValid || c.KeyUsage&x509.KeyUsageCertSign == 0 || c.CheckSignatureFrom(c) != nil {
		return nil, errors.New("pki: invalid P-256 root CA")
	}
	signer, err := provider.CertificateSigner(ctx, keyID)
	if err != nil {
		return nil, err
	}
	got, err := x509.MarshalPKIXPublicKey(signer.Public())
	if err != nil || !bytes.Equal(got, c.RawSubjectPublicKeyInfo) {
		return nil, errors.New("pki: CA key mismatch")
	}
	roots := x509.NewCertPool()
	roots.AddCert(c)
	return &CA{provider: provider, keyID: keyID, cert: c, roots: roots, validity: validity, now: time.Now}, nil
}

// CreateDevCA generates a public, self-signed development CA certificate.
// The private key has already been generated in the provider and never leaves it.
func CreateDevCA(ctx context.Context, p signing.Provider, keyID string, validity time.Duration) ([]byte, error) {
	if validity <= 0 {
		return nil, errors.New("pki: explicit CA validity required")
	}
	k, err := p.CertificateSigner(ctx, keyID)
	if err != nil {
		return nil, err
	}
	serial, err := randomSerial()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	t := &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: "Freedom VPN development CA (TEST ONLY)"}, NotBefore: now, NotAfter: now.Add(validity), IsCA: true, BasicConstraintsValid: true, MaxPathLenZero: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign, SignatureAlgorithm: x509.ECDSAWithSHA256}
	return x509.CreateCertificate(rand.Reader, t, t, k.Public(), k)
}
func randomSerial() (*big.Int, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(b)
	if n.Sign() == 0 {
		return randomSerial()
	}
	return n, nil
}
func (c *CA) ChainPEM() string {
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c.cert.Raw}))
}
func (c *CA) Issuer() string { h := sha256.Sum256(c.cert.Raw); return hex.EncodeToString(h[:]) }

// Issue is deliberately outside database transactions so a later remote
// provider never blocks a request transaction on a KMS call (§14.5).
func (c *CA) Issue(ctx context.Context, id NodeIdentity, csrPEM string) (Issued, error) {
	if id.NodeID == uuid.Nil || id.IncarnationID == uuid.Nil {
		return Issued{}, invalidCSR()
	}
	csr, err := ParseCSR(csrPEM)
	if err != nil {
		return Issued{}, err
	}
	now := c.now().UTC().Truncate(time.Second)
	until := now.Add(c.validity)
	if now.Before(c.cert.NotBefore) || !now.Before(c.cert.NotAfter) {
		return Issued{}, errors.New("pki: CA outside validity window")
	}
	if until.After(c.cert.NotAfter) {
		until = c.cert.NotAfter
	}
	serial, err := randomSerial()
	if err != nil {
		return Issued{}, err
	}
	t := &x509.Certificate{SerialNumber: serial, NotBefore: now, NotAfter: until, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, BasicConstraintsValid: true, SignatureAlgorithm: x509.ECDSAWithSHA256, URIs: []*url.URL{{Scheme: "urn", Opaque: "fvpn:node:" + id.NodeID.String()}, {Scheme: "urn", Opaque: "fvpn:incarnation:" + id.IncarnationID.String()}}}
	k, err := c.provider.CertificateSigner(ctx, c.keyID)
	if err != nil {
		return Issued{}, err
	}
	der, err := x509.CreateCertificate(rand.Reader, t, c.cert, csr.PublicKey, k)
	if err != nil {
		return Issued{}, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return Issued{}, err
	}
	return Issued{cert, string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), c.ChainPEM(), c.Issuer()}, nil
}
func Identity(cert *x509.Certificate) (NodeIdentity, error) {
	var id NodeIdentity
	if cert == nil || len(cert.URIs) != 2 || len(cert.DNSNames) > 0 || len(cert.IPAddresses) > 0 || len(cert.EmailAddresses) > 0 || !bytes.Equal(cert.RawSubject, []byte{0x30, 0}) {
		return id, apierr.New(apierr.Unauthenticated, "Invalid node certificate.")
	}
	for _, u := range cert.URIs {
		if u == nil {
			return NodeIdentity{}, apierr.New(apierr.Unauthenticated, "Invalid node certificate.")
		}
		for _, entry := range []struct {
			prefix string
			dst    *uuid.UUID
		}{{"urn:fvpn:node:", &id.NodeID}, {"urn:fvpn:incarnation:", &id.IncarnationID}} {
			raw := u.String()
			if len(raw) > len(entry.prefix) && raw[:len(entry.prefix)] == entry.prefix {
				v, err := uuid.Parse(raw[len(entry.prefix):])
				if err != nil || raw != entry.prefix+v.String() || *entry.dst != uuid.Nil {
					return NodeIdentity{}, apierr.New(apierr.Unauthenticated, "Invalid node certificate.")
				}
				*entry.dst = v
			}
		}
	}
	if id.NodeID == uuid.Nil || id.IncarnationID == uuid.Nil {
		return NodeIdentity{}, apierr.New(apierr.Unauthenticated, "Invalid node certificate.")
	}
	return id, nil
}

func (c *CA) Verify(cert *x509.Certificate) (NodeIdentity, error) {
	if cert == nil {
		return NodeIdentity{}, apierr.New(apierr.Unauthenticated, "Client certificate required.")
	}
	// Reparse DER: mutable fields on a caller's x509.Certificate cannot override
	// the identities and serial actually signed by the CA.
	cert, err := x509.ParseCertificate(cert.Raw)
	if err != nil {
		return NodeIdentity{}, apierr.New(apierr.Unauthenticated, "Invalid node certificate.")
	}
	pub, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok || pub.Curve != elliptic.P256() || cert.IsCA || len(cert.ExtKeyUsage) != 1 || cert.ExtKeyUsage[0] != x509.ExtKeyUsageClientAuth || len(cert.UnknownExtKeyUsage) > 0 || cert.KeyUsage != x509.KeyUsageDigitalSignature {
		return NodeIdentity{}, apierr.New(apierr.Unauthenticated, "Invalid node certificate.")
	}
	if _, err = cert.Verify(x509.VerifyOptions{Roots: c.roots, CurrentTime: c.now(), KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); err != nil {
		return NodeIdentity{}, apierr.New(apierr.Unauthenticated, "Invalid node certificate.")
	}
	return Identity(cert)
}
