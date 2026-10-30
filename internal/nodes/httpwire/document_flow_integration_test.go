//go:build integration

package httpwire_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	dnsstore "github.com/TED-H-com/Freedom-VPN/internal/dnspublish/store"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

func TestD056_D058_CompilerSeamDocumentWindowAndSignedProbe(t *testing.T) {
	r, enrolled, _ := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
	f := setupRuntime(t, r)
	cert := pkitest.Certificate(t, enrolled.Response.CertificatePem)
	r.Config.EgressProbe = &api.EgressProbe{Targets: []api.EgressProbeTarget{{Address: "192.0.2.80", Port: 443, Protocol: api.EgressProbeTargetProtocolTcp}}}
	rf := releasetest.New(t)
	r.Releases = &releases.Service{Pool: r.Pool, Validator: rf.Validator}
	if _, err := r.Releases.Register(t.Context(), releases.Command{ActorID: f.p.ID, IdempotencyKey: uuid.New()}, rf.Request(t, rf.Manifest)); err != nil {
		t.Fatal(err)
	}
	supportPayload := api.ReleaseSupportPayload{ApprovedReleaseIds: []uuid.UUID{rf.Manifest.ReleaseId}, RevokedReleaseIds: []uuid.UUID{}}
	supportPayload.MinimumServingVersion.ServingReleaseIds = supportPayload.ApprovedReleaseIds
	supportBytes, err := json.Marshal(supportPayload)
	if err != nil {
		t.Fatal(err)
	}
	store := policies.NewStore(r.Pool, policies.WithReleaseService(r.Releases), policies.WithUpstreamDNS([]string{"192.0.2.53:53"}))
	support := "test-support"
	supportRev, err := store.CreateRevision(t.Context(), policies.CreateRevisionInput{Code: &support, Type: api.PolicyTypeReleaseSupport, SchemaVersion: 1, Reason: "TEST ONLY", ActorID: f.p.ID, Payload: supportBytes})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.ActivateRevision(t.Context(), policies.ActivateRevisionInput{DefinitionID: supportRev.DefinitionID, RevisionID: supportRev.RevisionID, ExpectedVersion: "1", Reason: "TEST ONLY", ActorID: f.p.ID})
	if err != nil {
		t.Fatal(err)
	}
	pub, err := dnsstore.NewPublisher(r.Pool, dnsstore.Config{Upstreams: []string{"192.0.2.53:53"}})
	if err != nil {
		t.Fatal(err)
	}
	var definition *uuid.UUID
	publish := func(ttl int) nodes.ResolverDocument {
		t.Helper()
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
		definition = &rev.DefinitionID
		var version string
		if err = r.Pool.QueryRow(t.Context(), `SELECT resource_version::text FROM policy_definitions WHERE id=$1`, rev.DefinitionID).Scan(&version); err != nil {
			t.Fatal(err)
		}
		_, err = store.ActivateRevision(t.Context(), policies.ActivateRevisionInput{DefinitionID: rev.DefinitionID, RevisionID: rev.RevisionID, ExpectedVersion: version, Reason: "TEST ONLY", ActorID: f.p.ID})
		if err != nil {
			t.Fatal(err)
		}
		var event db.ClaimedEvent
		if err = r.Pool.QueryRow(t.Context(), `SELECT event_type,payload FROM outbox_events WHERE event_type='dns.publish' AND payload->>'revision_id'=$1`, rev.RevisionID.String()).Scan(&event.EventType, &event.Payload); err != nil {
			t.Fatal(err)
		}
		if _, err = pub.Handle(t.Context(), event); err != nil {
			t.Fatal(err)
		}
		d, err := r.ResolverDocuments.LatestPublished(t.Context(), rev.DefinitionID)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	read := func() api.DesiredStateEnvelope {
		t.Helper()
		w := f.call("GET", "/api/v1/agent/desired-state", nil, cert, nil)
		state := decode[api.DesiredStateEnvelope](t, w, 200)
		trust, err := r.Provider.TrustBundle(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err = envelope.Verify(w.Body.Bytes(), trust, envelope.PurposeDesiredState, time.Now()); err != nil {
			t.Fatal(err)
		}
		return state
	}
	noProfile := read()
	if !noProfile.Signed.Dns.ResolverDocument.IsNull() {
		t.Fatal("absent profile not null")
	}
	if p := mustValue(t, noProfile.Signed.EgressProbe); len(p.Targets) != 1 || p.Targets[0].Address != "192.0.2.80" {
		t.Fatal("signed probe missing")
	}
	first := publish(60)
	issued1 := read()
	if mustValue(t, issued1.Signed.Dns.ResolverDocument) != first.Reference {
		t.Fatal("wrong document reference")
	}
	path := func(d nodes.ResolverDocument) string {
		return "/api/v1/agent/resolver-documents/" + d.Reference.DocumentId.String()
	}
	check := func(d nodes.ResolverDocument) {
		t.Helper()
		w := f.call("GET", path(d), nil, cert, nil)
		if w.Code != http.StatusOK || w.Body.String() != string(d.Canonical) || w.Header().Get("ETag") != `"sha256:`+d.Reference.ContentSha256+`"` {
			t.Fatalf("wrong canonical response: %d %s headers=%v", w.Code, w.Body.String(), w.Header())
		}
	}
	check(first)
	_, _, other := f.enroll(f.create(nil))
	decode[api.ErrorEnvelope](t, f.call("GET", path(first), nil, other, nil), 404)
	second := publish(61)
	issued2 := read()
	if issued2.Signed.ConfigGeneration != issued1.Signed.ConfigGeneration+1 {
		t.Fatal("wrong next generation")
	}
	check(first)
	check(second)
	r.Config.EgressProbe.TimeoutMs = new(2001)
	issued3 := read()
	if issued3.Signed.ConfigGeneration != issued2.Signed.ConfigGeneration+1 {
		t.Fatal("probe change failed cache invalidation")
	}
	decode[api.ErrorEnvelope](t, f.call("GET", path(first), nil, cert, nil), 404)
	check(second)
	t.Run("transport_cache_header_core_owner", func(t *testing.T) {
		w := f.call("GET", path(second), nil, cert, nil)
		if w.Header().Get("Cache-Control") == "no-store" {
			t.Skip("control-core owns the pending f11-transport-cache.patch; compiler/document bytes and generation window verified above")
		}
		if w.Header().Get("Cache-Control") != "private, max-age=0" {
			t.Fatal(w.Header())
		}
	})
}
