//go:build integration

package httpwire_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	dnsstore "github.com/TED-H-com/Freedom-VPN/internal/dnspublish/store"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TEST ONLY: author policies and execute their real publication event. No DNS
// projection rows or desired-state rows are fabricated (D-050).
func publishResolverProfile(t *testing.T, f *fixture, definition *uuid.UUID, ttl int) (nodes.ResolverDocument, uuid.UUID) {
	t.Helper()
	store := policies.NewStore(f.r.Pool, policies.WithUpstreamDNS([]string{"192.0.2.53:53"}))
	code := "test-profile"
	raw, err := json.Marshal(api.DnsProfilePayload{ProfileCode: code, DisplayName: api.LocalizedText{ZhTW: "TEST ONLY"}, DomainRuleSetRevisionIds: []uuid.UUID{}, ThreatFeedRevisionIds: []uuid.UUID{}, BlockResponse: api.DnsProfilePayloadBlockResponseNxdomain, BlockedTtlSeconds: ttl})
	if err != nil {
		t.Fatal(err)
	}
	in := policies.CreateRevisionInput{Code: &code, DefinitionID: definition, Type: api.PolicyTypeDnsProfile, SchemaVersion: 1, Reason: "TEST ONLY", ActorID: f.p.ID, Payload: raw}
	if definition != nil {
		in.Code = nil
	}
	rev, err := store.CreateRevision(t.Context(), in)
	if err != nil {
		t.Fatal(err)
	}
	var version string
	if err = f.r.Pool.QueryRow(t.Context(), `SELECT resource_version::text FROM policy_definitions WHERE id=$1`, rev.DefinitionID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	_, err = store.ActivateRevision(t.Context(), policies.ActivateRevisionInput{DefinitionID: rev.DefinitionID, RevisionID: rev.RevisionID, ExpectedVersion: version, Reason: "TEST ONLY", ActorID: f.p.ID})
	if err != nil {
		t.Fatal(err)
	}
	pub, err := dnsstore.NewPublisher(f.r.Pool, dnsstore.Config{Upstreams: []string{"192.0.2.53:53"}})
	if err != nil {
		t.Fatal(err)
	}
	var event db.ClaimedEvent
	if err = f.r.Pool.QueryRow(t.Context(), `SELECT event_type,payload FROM outbox_events WHERE event_type='dns.publish' AND payload->>'revision_id'=$1`, rev.RevisionID.String()).Scan(&event.EventType, &event.Payload); err != nil {
		t.Fatal(err)
	}
	if _, err = pub.Handle(t.Context(), event); err != nil {
		t.Fatal(err)
	}
	if f.r.ResolverDocuments == nil {
		f.r.ResolverDocuments = httpwire.NewResolverDocumentSource(f.r.Pool)
	}
	doc, err := f.r.ResolverDocuments.LatestPublished(t.Context(), rev.DefinitionID)
	if err != nil {
		t.Fatal(err)
	}
	return doc, rev.DefinitionID
}

func TestD056_RuntimeUsesPublishedDocumentReader(t *testing.T) {
	r, _, _ := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
	if r.ResolverDocuments == nil {
		t.Fatal("runtime document source was not composed")
	}
	f := setupRuntime(t, r)
	if _, err := r.ResolverDocuments.DocumentByID(t.Context(), uuid.New()); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("unknown document mapping", err)
	}
	first, definition := publishResolverProfile(t, f, nil, 60)
	second, _ := publishResolverProfile(t, f, &definition, 61)
	if first.Reference.DocumentId == second.Reference.DocumentId || first.Reference.Generation >= second.Reference.Generation {
		t.Fatal("latest publication did not move")
	}
	for _, want := range []nodes.ResolverDocument{first, second} {
		got, err := r.ResolverDocuments.DocumentByID(t.Context(), want.Reference.DocumentId)
		if err != nil || got.Reference != want.Reference || !bytes.Equal(got.Canonical, want.Canonical) {
			t.Fatal("immutable published bytes changed", err)
		}
		if err = nodes.ValidateResolverDocument(got); err != nil {
			t.Fatal(err)
		}
	}
}
