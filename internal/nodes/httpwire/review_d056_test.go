//go:build integration

package httpwire_test

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func TestReviewD056_OwningNodeReceivesPublishedDocument(t *testing.T) {
	f, enrolled, cert, _, _ := publicationFixture(t)
	doc, _ := publishResolverProfile(t, f, nil, 60)
	stateW := f.call("GET", "/api/v1/agent/desired-state", nil, cert, nil)
	if stateW.Code != http.StatusOK {
		t.Fatalf("desired state %d %s", stateW.Code, stateW.Body.String())
	}
	state := decode[api.DesiredStateEnvelope](t, stateW, http.StatusOK)
	ref, refErr := state.Signed.Dns.ResolverDocument.Get()
	path := "/api/v1/agent/resolver-documents/" + doc.Reference.DocumentId.String()
	got := f.call("GET", path, nil, cert, nil)
	_, otherEnrolled, otherCert := f.enroll(f.create(nil))
	other := f.call("GET", path, nil, otherCert, nil)
	if refErr != nil || ref != doc.Reference || got.Code != http.StatusOK || !bytes.Equal(got.Body.Bytes(), doc.Canonical) || other.Code != http.StatusNotFound {
		// Regression proof: the production compiler must issue the published
		// reference and authorize only its owning node to fetch canonical bytes.
		t.Fatalf("published resolver document was not delivered to the owning node: node=%s otherNode=%s refErr=%v ref=%+v want=%+v ownerStatus=%d ownerCache=%q ownerType=%q ownerBody=%s otherStatus=%d otherBody=%s desiredHasRef=%t",
			enrolled.NodeId, otherEnrolled.NodeId, refErr, ref, doc.Reference, got.Code, got.Header().Get("Cache-Control"), got.Header().Get("Content-Type"), got.Body.String(), other.Code, other.Body.String(), strings.Contains(stateW.Body.String(), doc.Reference.DocumentId.String()))
	}
	if got.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("content type %q", got.Header().Get("Content-Type"))
	}
	if cache := got.Header().Get("Cache-Control"); cache != "private, max-age=0" && cache != "no-store" {
		t.Fatalf("cache %q", cache)
	}
}

func TestReviewD056_DocumentFetchAuthAndIdenticalNotFound(t *testing.T) {
	f, enrolled, cert, _, _ := publicationFixture(t)
	doc, _ := publishResolverProfile(t, f, nil, 60)
	unknown := "/api/v1/agent/resolver-documents/" + uuid.NewString()
	published := "/api/v1/agent/resolver-documents/" + doc.Reference.DocumentId.String()
	missingDoc := decode[api.ErrorEnvelope](t, f.call("GET", unknown, nil, cert, nil), http.StatusNotFound)
	unissued := decode[api.ErrorEnvelope](t, f.call("GET", published, nil, cert, nil), http.StatusNotFound)
	if missingDoc.Error.Code != api.ErrorCodeNOTFOUND || unissued.Error.Code != missingDoc.Error.Code || unissued.Error.Message != missingDoc.Error.Message {
		t.Fatalf("not-found oracle: %+v vs %+v", missingDoc.Error, unissued.Error)
	}
	if f.call("GET", unknown, nil, cert, nil).Header().Get("Cache-Control") != "no-store" {
		t.Fatal("document error was cacheable")
	}
	// dns_resolver_documents rejects DELETE (immutable_row). A row that is gone
	// after authorization is ResolverDocumentForNode's ErrNoRows branch
	// (resolver_documents.go:110-112), which returns the same NotFound sentinel.
	if _, err := f.r.Pool.Exec(t.Context(), `DELETE FROM dns_resolver_documents WHERE id=$1`, doc.Reference.DocumentId); err == nil || !strings.Contains(err.Error(), "immutable record") {
		t.Fatalf("resolver document row was deletable: %v", err)
	}
	kept := decode[api.ErrorEnvelope](t, f.call("GET", published, nil, cert, nil), http.StatusNotFound)
	if kept.Error.Code != missingDoc.Error.Code || kept.Error.Message != missingDoc.Error.Message {
		t.Fatalf("published id disclosed after refused delete: %+v", kept.Error)
	}
	detail := f.call("GET", "/api/v1/admin/nodes/"+enrolled.NodeId.String(), nil, nil, nil)
	decode[api.NodeDetail](t, detail, http.StatusOK)
	rev := f.call("POST", "/api/v1/admin/nodes/"+enrolled.NodeId.String()+"/revoke", api.ReasonRequest{Reason: "TEST ONLY review revoke"}, nil, map[string]string{"If-Match": detail.Header().Get("ETag"), "Idempotency-Key": uuid.NewString()})
	if rev.Code != http.StatusAccepted {
		t.Fatalf("revoke %d %s", rev.Code, rev.Body.String())
	}
	revokedDoc := decode[api.ErrorEnvelope](t, f.call("GET", published, nil, cert, nil), http.StatusForbidden)
	revokedState := decode[api.ErrorEnvelope](t, f.call("GET", "/api/v1/agent/desired-state", nil, cert, nil), http.StatusForbidden)
	if revokedDoc.Error.Code != api.ErrorCodeIDENTITYREVOKED || revokedDoc.Error.Code != revokedState.Error.Code || revokedDoc.Error.Message != revokedState.Error.Message {
		t.Fatalf("revoked document auth differs: doc=%+v state=%+v", revokedDoc.Error, revokedState.Error)
	}
	// node_certificates also rejects DELETE. An unknown certificate is one this
	// CA did not issue. Both agent routes must reject it the same way.
	_, foreignCA := pkitest.Authority(t)
	_, csr := pkitest.CSR(t)
	issued, err := foreignCA.Issue(t.Context(), pki.NodeIdentity{NodeID: uuid.New(), IncarnationID: uuid.New()}, csr)
	if err != nil {
		t.Fatal(err)
	}
	unknownCert := decode[api.ErrorEnvelope](t, f.call("GET", published, nil, issued.Certificate, nil), http.StatusUnauthorized)
	unknownState := decode[api.ErrorEnvelope](t, f.call("GET", "/api/v1/agent/desired-state", nil, issued.Certificate, nil), http.StatusUnauthorized)
	if unknownCert.Error.Code != api.ErrorCodeUNAUTHENTICATED || unknownCert.Error.Code != unknownState.Error.Code || unknownCert.Error.Message != unknownState.Error.Message {
		t.Fatalf("unknown certificate auth differs: doc=%+v state=%+v", unknownCert.Error, unknownState.Error)
	}
}

func TestReviewD056_EvidenceIsReportedWithoutIssuanceCheck(t *testing.T) {
	runtime, enrolled, report := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
	f := setupRuntime(t, runtime)
	cert := pkitest.Certificate(t, enrolled.Response.CertificatePem)
	path := "/api/v1/admin/nodes/" + report.NodeId.String()
	type detail struct {
		api.NodeDetail
		Evidence presence.Nullable[api.DnsPolicyEvidence] `json:"dns_policy_evidence"`
	}
	neverIssued := uuid.New()
	ev := api.DnsPolicyEvidence{
		AppliedAt:     presence.NewNullableWithValue(time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)),
		ContentSha256: strings.Repeat("ab", 32),
		DocumentId:    neverIssued,
		ErrorCode:     presence.NewNullNullable[api.ErrorCode](),
		Feeds:         []api.DnsFeedEvidence{},
		Generation:    decstr.U64(9),
		Outcome:       api.DnsPolicyEvidenceOutcomeApplied,
	}
	report.DnsPolicyEvidence.Set(ev)
	check := api.NodeCheck{Name: api.NodeCheckNameDnsPolicy, Status: api.HealthCheckStatusPassed, ObservedAt: report.ObservedAt}
	check.ErrorCode.SetNull()
	report.Checks = append(report.Checks, check)
	decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, map[string]string{"Idempotency-Key": uuid.NewString()}), http.StatusAccepted)
	got := decode[detail](t, f.call("GET", path, nil, nil, nil), http.StatusOK)
	projected, err := got.Evidence.Get()
	if err != nil || projected.DocumentId != neverIssued || projected.Outcome != api.DnsPolicyEvidenceOutcomeApplied || projected.Generation != ev.Generation {
		t.Fatalf("unissued evidence was not projected: %+v %v", got.Evidence, err)
	}
	// A later report with a lower generation replaces the projection. Nothing requires monotonicity.
	report.ReportId = uuid.New()
	report.ObservedAt = report.ObservedAt.Add(time.Second)
	ev.Generation = decstr.U64(2)
	report.DnsPolicyEvidence.Set(ev)
	decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, map[string]string{"Idempotency-Key": uuid.NewString()}), http.StatusAccepted)
	got = decode[detail](t, f.call("GET", path, nil, nil, nil), http.StatusOK)
	projected, err = got.Evidence.Get()
	if err != nil || projected.Generation != decstr.U64(2) {
		t.Fatalf("lower generation rejected or not projected: %+v %v", projected, err)
	}
	report.ReportId = uuid.New()
	report.ObservedAt = report.ObservedAt.Add(time.Second)
	ev.Feeds = []api.DnsFeedEvidence{{FeedCode: "feed", ContentSha256: strings.Repeat("cd", 32), RuleCount: 1}, {FeedCode: "feed", ContentSha256: strings.Repeat("cd", 32), RuleCount: 1}}
	report.DnsPolicyEvidence.Set(ev)
	dup := f.call("POST", "/api/v1/agent/observations", report, cert, map[string]string{"Idempotency-Key": uuid.NewString()})
	if dup.Code != http.StatusUnprocessableEntity {
		t.Fatalf("duplicate feed accepted: %d %s", dup.Code, dup.Body.String())
	}
	ev.Outcome = api.DnsPolicyEvidenceOutcomeFailed
	report.ReportId = uuid.New()
	report.DnsPolicyEvidence.Set(ev)
	feeds := make([]api.DnsFeedEvidence, 33)
	for i := range feeds {
		feeds[i] = api.DnsFeedEvidence{FeedCode: "f" + decstr.U64(uint64(i)).String(), ContentSha256: strings.Repeat("ab", 32), RuleCount: 1}
	}
	ev.Feeds = feeds
	ev.Outcome = api.DnsPolicyEvidenceOutcomeApplied
	report.DnsPolicyEvidence.Set(ev)
	// The passing check would also be inconsistent with a non-applied outcome; this body is applied with 33 feeds.
	tooMany := f.call("POST", "/api/v1/agent/observations", report, cert, map[string]string{"Idempotency-Key": uuid.NewString()})
	if tooMany.Code != http.StatusUnprocessableEntity {
		t.Fatalf("TODO(review): 33 feed evidence was stored: %d %s", tooMany.Code, tooMany.Body.String())
	}
}
