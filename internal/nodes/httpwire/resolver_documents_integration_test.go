//go:build integration

package httpwire_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

type unexpectedDocumentSource struct{ t *testing.T }

func (s unexpectedDocumentSource) LatestPublished(context.Context, uuid.UUID) (nodes.ResolverDocument, error) {
	s.t.Fatal("unauthorized request consulted publisher")
	return nodes.ResolverDocument{}, nil
}
func (s unexpectedDocumentSource) DocumentByID(context.Context, uuid.UUID) (nodes.ResolverDocument, error) {
	s.t.Fatal("unauthorized request consulted publisher")
	return nodes.ResolverDocument{}, nil
}

func TestD056_HTTPResolverDocumentUnauthorizedDoesNotConsultPublisher(t *testing.T) {
	r, enrolled, _ := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
	r.ResolverDocuments = unexpectedDocumentSource{t}
	f := setupRuntime(t, r)
	cert := pkitest.Certificate(t, enrolled.Response.CertificatePem)
	path := "/api/v1/agent/resolver-documents/" + uuid.NewString()
	for _, headers := range []map[string]string{nil, {"X-Node-ID": uuid.NewString()}} {
		failure := decode[api.ErrorEnvelope](t, f.call("GET", path, nil, cert, headers), http.StatusNotFound)
		if failure.Error.Code != api.ErrorCodeNOTFOUND || failure.Error.Message != "Resolver document not found." {
			t.Fatalf("document existence disclosed: %+v", failure)
		}
	}
	if w := f.call("GET", path, nil, nil, nil); w.Code == http.StatusOK || w.Code == http.StatusNotImplemented {
		t.Fatalf("missing mTLS identity: %d", w.Code)
	}
}
