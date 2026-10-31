package pki_test

import (
	"context"
	"crypto/x509"
	"net/url"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/google/uuid"
)

// TestReviewR15_IssueSetsTheNodeURNAndRejectsACSRURI is the enrollment CSR
// boundary on the issuing path. CA.Issue writes urn:fvpn:node:<node id> from
// the node id. A client-chosen URI SAN is rejected by ParseCSR before a
// certificate is minted, so a CSR cannot choose the ingress budget key.
func TestReviewR15_IssueSetsTheNodeURNAndRejectsACSRURI(t *testing.T) {
	_, ca := pkitest.Authority(t)
	k, csr := pkitest.CSR(t)
	id := pki.NodeIdentity{NodeID: uuid.New(), IncarnationID: uuid.New()}
	issued, err := ca.Issue(context.Background(), id, csr)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ca.Verify(issued.Certificate)
	if err != nil || got != id {
		t.Fatalf("identity=%v error=%v", got, err)
	}
	var uris []string
	for _, u := range issued.Certificate.URIs {
		uris = append(uris, u.String())
	}
	wantNode := "urn:fvpn:node:" + id.NodeID.String()
	wantInc := "urn:fvpn:incarnation:" + id.IncarnationID.String()
	if len(uris) != 2 || uris[0] != wantNode || uris[1] != wantInc {
		t.Fatalf("issued URIs %v, want [%s %s]", uris, wantNode, wantInc)
	}
	attacker := "urn:fvpn:node:" + uuid.NewString()
	bad := pkitest.CSRWith(t, k, &x509.CertificateRequest{URIs: []*url.URL{{Scheme: "urn", Opaque: attacker[4:]}}})
	if _, err := ca.Issue(context.Background(), id, bad); err == nil {
		t.Fatal("CA.Issue minted a leaf from a CSR that carried a client-chosen URI SAN")
	}
}
