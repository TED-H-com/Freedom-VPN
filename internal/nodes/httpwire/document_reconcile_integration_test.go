//go:build integration

package httpwire_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/apps/worker"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	dnsstore "github.com/TED-H-com/Freedom-VPN/internal/dnspublish/store"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

func TestD056_WorkerDeliversNewlyEligibleResolverDocumentWithoutChurn(t *testing.T) {
	r, enrolled, report := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
	f := setupRuntime(t, r)
	cert := pkitest.Certificate(t, enrolled.Response.CertificatePem)
	if _, err := r.Nodes.SubmitObservation(t.Context(), report.NodeId, report.NodeIncarnationId, report); err != nil {
		t.Fatal(err)
	}
	// Release support, node publication, DNS revisions and document eligibility all
	// use their owning services. No desired-state or DNS projection is fabricated.
	rf := releasetest.New(t)
	r.Releases = &releases.Service{Pool: r.Pool, Validator: rf.Validator}
	if _, err := r.Releases.Register(t.Context(), releases.Command{ActorID: f.p.ID, IdempotencyKey: uuid.New()}, rf.Request(t, rf.Manifest)); err != nil {
		t.Fatal(err)
	}
	store := policies.NewStore(r.Pool, policies.WithReleaseService(r.Releases), policies.WithUpstreamDNS([]string{"192.0.2.53:53"}))
	support := api.ReleaseSupportPayload{ApprovedReleaseIds: []uuid.UUID{rf.Manifest.ReleaseId}, RevokedReleaseIds: []uuid.UUID{}}
	support.MinimumServingVersion.ServingReleaseIds = support.ApprovedReleaseIds
	create := func(typ api.PolicyType, code string, payload any) *policies.RevisionRecord {
		t.Helper()
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		rev, err := store.CreateRevision(t.Context(), policies.CreateRevisionInput{Code: &code, Type: typ, SchemaVersion: 1, Reason: "TEST ONLY", ActorID: f.p.ID, Payload: raw})
		if err != nil {
			t.Fatal(err)
		}
		return rev
	}
	activate := func(definition, revision uuid.UUID) {
		t.Helper()
		_, err := store.ActivateRevision(t.Context(), policies.ActivateRevisionInput{DefinitionID: definition, RevisionID: revision, ExpectedVersion: "1", Reason: "TEST ONLY", ActorID: f.p.ID})
		if err != nil {
			t.Fatal(err)
		}
	}
	rev := create(api.PolicyTypeReleaseSupport, "test-support", support)
	activate(rev.DefinitionID, rev.RevisionID)
	f.publicationCommand(report.NodeId, api.PublicationIntentPublished)
	initial := decode[api.DesiredStateEnvelope](t, f.call("GET", "/api/v1/agent/desired-state", nil, cert, nil), http.StatusOK)
	if !initial.Signed.Dns.ResolverDocument.IsNull() {
		t.Fatal("invented unpublished document")
	}
	_, other, otherCert := f.enroll(f.create(nil))
	decode[api.DesiredStateEnvelope](t, f.call("GET", "/api/v1/agent/desired-state", nil, otherCert, nil), http.StatusOK)
	profile := create(api.PolicyTypeDnsProfile, "test-profile", api.DnsProfilePayload{ProfileCode: "test-profile", DisplayName: api.LocalizedText{ZhTW: "TEST ONLY"}, DomainRuleSetRevisionIds: []uuid.UUID{}, ThreatFeedRevisionIds: []uuid.UUID{}, BlockResponse: api.DnsProfilePayloadBlockResponseNxdomain, BlockedTtlSeconds: 60})
	activate(profile.DefinitionID, profile.RevisionID)
	read := func(node uuid.UUID) (api.DesiredStateEnvelope, []byte) {
		t.Helper()
		var raw []byte
		if err := r.Pool.QueryRow(t.Context(), `SELECT envelope FROM desired_state_envelopes WHERE node_id=$1`, node).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		var state api.DesiredStateEnvelope
		if err := json.Unmarshal(raw, &state); err != nil {
			t.Fatal(err)
		}
		trust, err := r.Provider.TrustBundle(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err = envelope.Verify(raw, trust, envelope.PurposeDesiredState, time.Now()); err != nil {
			t.Fatal(err)
		}
		return state, raw
	}
	r.Config.ReconcileSeconds = 3600 // Document scheduling is independently capped at 60s.
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- worker.RunNodeSchedule(ctx, r) }()
	waitUntil(t, func() bool {
		var count int
		err := r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM outbox_events WHERE event_type='node.resolver_documents.reconcile' AND delivered_at IS NULL`).Scan(&count)
		return err == nil && count == 1
	})
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	dispatcher, err := worker.New(r.Pool, worker.NodeJobs(r, nil, nil, 3), worker.Config{Concurrency: 1, Attempts: 3, Lease: time.Minute, JobTimeout: 30 * time.Second, Poll: time.Millisecond, Backoff: time.Millisecond, MaxBackoff: time.Second}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	cycle := func() {
		t.Helper()
		f.sql(`UPDATE outbox_events SET available_at=clock_timestamp()+interval '1 day' WHERE event_type<>'node.resolver_documents.reconcile'`)
		f.sql(`UPDATE outbox_events SET available_at=clock_timestamp() WHERE event_type='node.resolver_documents.reconcile' AND delivered_at IS NULL AND dead_lettered_at IS NULL`)
		if worked, err := dispatcher.Once(t.Context()); err != nil || !worked {
			t.Fatal("resolver job not dispatched", worked, err)
		}
		var delivered, pending, failed int
		var delay float64
		if err := r.Pool.QueryRow(t.Context(), `SELECT count(*) FILTER (WHERE delivered_at IS NOT NULL AND delivery_receipt IS NOT NULL),count(*) FILTER (WHERE delivered_at IS NULL AND dead_lettered_at IS NULL),count(*) FILTER (WHERE last_error_code IS NOT NULL) FROM outbox_events WHERE event_type='node.resolver_documents.reconcile'`).Scan(&delivered, &pending, &failed); err != nil {
			t.Fatal(err)
		}
		if delivered < 1 || pending != 1 || failed != 0 {
			t.Fatal("resolver chain not confirmed", delivered, pending, failed)
		}
		if err := r.Pool.QueryRow(t.Context(), `SELECT extract(epoch FROM available_at-clock_timestamp()) FROM outbox_events WHERE event_type='node.resolver_documents.reconcile' AND delivered_at IS NULL`).Scan(&delay); err != nil || delay < 0 || delay > 60 {
			t.Fatal("unbounded resolver interval", delay, err)
		}
	}
	// Activation alone is not eligibility. Keep the last issued state; never
	// silently compile an active but unavailable profile as an empty policy.
	cycle()
	before, _ := read(report.NodeId)
	if before.Signed.ConfigGeneration != initial.Signed.ConfigGeneration || !before.Signed.Dns.ResolverDocument.IsNull() {
		t.Fatal("ineligible document advanced state")
	}
	pub, err := dnsstore.NewPublisher(r.Pool, dnsstore.Config{Upstreams: []string{"192.0.2.53:53"}})
	if err != nil {
		t.Fatal(err)
	}
	var event db.ClaimedEvent
	if err = r.Pool.QueryRow(t.Context(), `SELECT event_type,payload FROM outbox_events WHERE event_type='dns.publish' AND payload->>'revision_id'=$1`, profile.RevisionID.String()).Scan(&event.EventType, &event.Payload); err != nil {
		t.Fatal(err)
	}
	if _, err = pub.Handle(t.Context(), event); err != nil {
		t.Fatal(err)
	}
	document, err := r.ResolverDocuments.LatestPublished(t.Context(), profile.DefinitionID)
	if err != nil {
		t.Fatal(err)
	}
	if err = r.ReconcileResolverDocuments(commandtx.WithClaim(t.Context(), db.ClaimedEvent{ID: uuid.New()})); !errors.Is(err, db.ErrLeaseLost) {
		t.Fatal("stale claim compiled", err)
	}
	unchanged, _ := read(report.NodeId)
	if unchanged.Signed.ConfigGeneration != before.Signed.ConfigGeneration {
		t.Fatal("lost claim changed state")
	}
	cycle()
	after, raw := read(report.NodeId)
	if after.Signed.ConfigGeneration != before.Signed.ConfigGeneration+1 || mustValue(t, after.Signed.Dns.ResolverDocument) != document.Reference {
		t.Fatal("eligible document not issued")
	}
	for _, node := range []uuid.UUID{report.NodeId, other.NodeId} {
		state, _ := read(node)
		if mustValue(t, state.Signed.Dns.ResolverDocument) != document.Reference {
			t.Fatal("fleet node missed eligible document")
		}
	}
	w := f.call("GET", "/api/v1/agent/resolver-documents/"+document.Reference.DocumentId.String(), nil, cert, nil)
	if w.Code != http.StatusOK || !bytes.Equal(w.Body.Bytes(), document.Canonical) || w.Header().Get("ETag") != `"sha256:`+document.Reference.ContentSha256+`"` {
		t.Fatal("issued document not authorized", w.Code, w.Body.String())
	}
	cycle()
	_, stable := read(report.NodeId)
	if !bytes.Equal(raw, stable) {
		t.Fatal("unchanged document churned signed envelope")
	}
	newer, _ := publishResolverProfile(t, f, &profile.DefinitionID, 61)
	cycle()
	latest, _ := read(report.NodeId)
	if latest.Signed.ConfigGeneration != after.Signed.ConfigGeneration+1 || mustValue(t, latest.Signed.Dns.ResolverDocument) != newer.Reference {
		t.Fatal("new document generation not delivered")
	}
	decode[api.ErrorEnvelope](t, f.call("GET", "/api/v1/agent/resolver-documents/"+uuid.NewString(), nil, cert, nil), http.StatusNotFound)
}
