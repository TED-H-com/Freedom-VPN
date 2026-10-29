package pki_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"net/url"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/google/uuid"
)

func TestAT28_CSRBoundary(t *testing.T) {
	k, good := pkitest.CSR(t)
	if _, err := pki.ParseCSR(good); err != nil {
		t.Fatal(err)
	}
	wrong, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := pem.Decode([]byte(good))
	b.Bytes[len(b.Bytes)-1] ^= 1
	cases := map[string]string{
		"wrong_curve":       pkitest.CSRWith(t, wrong, &x509.CertificateRequest{}),
		"bad_signature":     string(pem.EncodeToMemory(b)),
		"extra_SAN":         pkitest.CSRWith(t, k, &x509.CertificateRequest{URIs: []*url.URL{{Scheme: "urn", Opaque: "fvpn:node:" + uuid.NewString()}}}),
		"personal_subject":  pkitest.CSRWith(t, k, &x509.CertificateRequest{Subject: pkix.Name{CommonName: "example.invalid"}}),
		"unknown_extension": pkitest.CSRWith(t, k, &x509.CertificateRequest{ExtraExtensions: []pkix.Extension{{Id: asn1.ObjectIdentifier{1, 2, 3, 4}, Value: []byte{5, 0}}}}),
		"trailing_PEM":      good + good,
		"oversize":          good + string(make([]byte, 8192)),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := pki.ParseCSR(input); err == nil {
				t.Fatal("unsafe CSR accepted")
			}
		})
	}
}
func TestAT28_CertificateSANRoundTrip(t *testing.T) {
	_, ca := pkitest.Authority(t)
	_, csr := pkitest.CSR(t)
	id := pki.NodeIdentity{NodeID: uuid.New(), IncarnationID: uuid.New()}
	issued, err := ca.Issue(context.Background(), id, csr)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ca.Verify(issued.Certificate)
	if err != nil || got != id {
		t.Fatalf("identity=%v error=%v", got, err)
	}
	c := issued.Certificate
	if len(c.ExtKeyUsage) != 1 || c.ExtKeyUsage[0] != x509.ExtKeyUsageClientAuth || c.SerialNumber.BitLen() > 128 || c.SerialNumber.Sign() <= 0 {
		t.Fatal("certificate scope or serial incorrect")
	}
	// Mutating parsed fields cannot override the signed DER identity.
	c.URIs[0] = &url.URL{Scheme: "urn", Opaque: "fvpn:node:" + uuid.NewString()}
	got, err = ca.Verify(c)
	if err != nil || got != id {
		t.Fatal("mutable parsed fields trusted")
	}
	_, wrongCA := pkitest.Authority(t)
	if _, err = wrongCA.Verify(c); err == nil {
		t.Fatal("wrong CA trusted")
	}
}
