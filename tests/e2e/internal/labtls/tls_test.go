package labtls

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"testing"
	"time"
)

func TestServerCertVerifiesAgainstAuthority(t *testing.T) {
	now := time.Now().UTC()
	ca, err := NewAuthority("E2E Lab CA", now.Add(-time.Minute), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	certPEM, keyPEM, err := ServerCert(ca, "portal.example.invalid", []string{"portal.example.invalid", "agent.example.invalid"}, []net.IP{net.ParseIP("192.0.3.10")}, now.Add(-time.Minute), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca.CertPEM) {
		t.Fatal("ca pem")
	}
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: pool, DNSName: "portal.example.invalid", CurrentTime: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: pool, DNSName: "agent.example.invalid", CurrentTime: now}); err != nil {
		t.Fatal(err)
	}
}
