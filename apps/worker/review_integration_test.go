//go:build integration

package worker

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/dnsfeeds"
	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	dnspublishstore "github.com/TED-H-com/Freedom-VPN/internal/dnspublish/store"
	"github.com/TED-H-com/Freedom-VPN/internal/fup"
	policystore "github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestReview_F04_PolicyActivateOperationUntruthfulOnDNSPublishOrFeedFailureAndDropsObservedRefs(t *testing.T) {
	pool := dbtest.New(t)
	now := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)

	feedBody := "malware-1.example.com\nmalware-2.example.com\n"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, feedBody)
	}))
	defer server.Close()

	client := server.Client()
	tr := client.Transport.(*http.Transport).Clone()
	tr.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	client.Transport = tr

	feedSvc, err := dnsfeeds.New(pool, dnsfeeds.Options{Client: client, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("dnsfeeds.New: %v", err)
	}
	pubStore, err := dnspublishstore.NewPublisher(pool, dnspublishstore.Config{Upstreams: []string{"192.0.2.53:53"}})
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}

	jobs := PolicyJobs()
	jobs.Register(dnspublish.EventDNSPublish, pubStore.Handle)
	jobs.Register(dnsfeeds.EventRefresh, feedSvc.Handle)

	polServices := &PoliciesServices{
		Pool:      pool,
		FUPWorker: &fup.Worker{},
	}
	ctx := WithPolicies(context.Background(), polServices)

	dispatcher, err := New(pool, jobs, Config{
		Concurrency: 1,
		Attempts:    3,
		Lease:       time.Minute,
		JobTimeout:  30 * time.Second,
		Poll:        time.Second,
		Backoff:     time.Second,
		MaxBackoff:  time.Minute,
	}, nil, nil)
	if err != nil {
		t.Fatalf("worker.New: %v", err)
	}
	drainAll := func(t *testing.T) {
		t.Helper()
		for range 20 {
			worked, err := dispatcher.Once(ctx)
			if err != nil {
				t.Fatalf("dispatcher.Once: %v", err)
			}
			if !worked {
				break
			}
		}
	}

	ps := policystore.NewStore(pool, policystore.WithClock(func() time.Time { return now }))
	actor := uuid.New()

	t.Run("threat_feed_activation_marked_succeeded_when_bootstrap_refresh_fails", func(t *testing.T) {
		// Upstream returns feedBody, but ContentSha256 pins a different digest -> content_mismatch.
		badPayload := api.ThreatFeedPayload{
			Provider:              "TEST ONLY",
			FeedCode:              "mismatch-feed",
			LicenseName:           "TEST ONLY",
			LicenseUrl:            "https://example.com/license",
			CommercialUseBasis:    "TEST ONLY fixture",
			Attribution:           presence.NewNullableWithValue("TEST ONLY attribution"),
			SourceUrl:             "https://example.com/feed",
			Format:                api.ThreatFeedPayloadFormatDomainList,
			Categories:            []api.ThreatFeedPayloadCategories{api.ThreatFeedPayloadCategoriesMalware},
			ContentSha256:         fmt.Sprintf("%x", sha256.Sum256([]byte("different-pinned-body.example.com\n"))),
			MaxSizeBytes:          4096,
			ValidFrom:             now.Add(-time.Hour),
			ValidUntil:            now.Add(24 * time.Hour),
			UpdateIntervalSeconds: 300,
		}
		raw, _ := json.Marshal(badPayload)
		code := "mismatch-feed"
		rev, err := ps.CreateRevision(ctx, policystore.CreateRevisionInput{
			Code:          &code,
			Type:          api.PolicyTypeThreatFeed,
			SchemaVersion: 1,
			Payload:       raw,
			Reason:        "TEST ONLY create mismatch feed",
			ActorID:       actor,
		})
		if err != nil {
			t.Fatalf("CreateRevision: %v", err)
		}
		actRes, err := ps.ActivateRevision(ctx, policystore.ActivateRevisionInput{
			DefinitionID:    rev.DefinitionID,
			RevisionID:      rev.RevisionID,
			ExpectedVersion: "1",
			Reason:          "TEST ONLY activate mismatch feed",
			ActorID:         actor,
		})
		if err != nil {
			t.Fatalf("ActivateRevision: %v", err)
		}

		drainAll(t)

		opID := uuid.UUID(actRes.Operation.OperationId)
		var opStatus string
		var lastErrorCode string
		if err := pool.QueryRow(ctx, `SELECT status FROM operations WHERE id = $1`, opID).Scan(&opStatus); err != nil {
			t.Fatalf("query operation: %v", err)
		}
		if err := pool.QueryRow(ctx, `SELECT last_error_code FROM dns_feed_refresh_state WHERE feed_revision_id = $1`, rev.RevisionID).Scan(&lastErrorCode); err != nil {
			t.Fatalf("query feed state: %v", err)
		}
		if opStatus == "succeeded" && lastErrorCode != "" {
			t.Errorf("TODO(review) [F-04]: policy.activate operation %s reported status=%q even though threat_feed bootstrap failed with last_error_code=%q",
				opID, opStatus, lastErrorCode)
		}
	})

	t.Run("dns_profile_activation_marked_succeeded_when_publish_fails_and_drops_observed_refs_when_publish_succeeds", func(t *testing.T) {
		// 1. Activate a dns_profile referencing the unbootstrapped mismatch-feed -> dns.publish fails with feed_last_good_missing.
		var mismatchRevID uuid.UUID
		if err := pool.QueryRow(ctx, `SELECT active_revision_id FROM policy_definitions WHERE code = 'mismatch-feed'`).Scan(&mismatchRevID); err != nil {
			t.Fatalf("query mismatchRevID: %v", err)
		}

		en := "Broken Profile"
		profCode := "broken-profile"
		profPayload := api.DnsProfilePayload{
			ProfileCode:              "broken-profile",
			DisplayName:              api.LocalizedText{ZhTW: "故障設定", En: &en},
			DomainRuleSetRevisionIds: []openapi_types.UUID{},
			ThreatFeedRevisionIds:    []openapi_types.UUID{openapi_types.UUID(mismatchRevID)},
			BlockResponse:            api.DnsProfilePayloadBlockResponseNxdomain,
			BlockedTtlSeconds:        60,
		}
		rawProf, _ := json.Marshal(profPayload)
		profRev, err := ps.CreateRevision(ctx, policystore.CreateRevisionInput{
			Code:          &profCode,
			Type:          api.PolicyTypeDnsProfile,
			SchemaVersion: 1,
			Payload:       rawProf,
			Reason:        "TEST ONLY broken profile",
			ActorID:       actor,
		})
		if err != nil {
			t.Fatalf("CreateRevision broken profile: %v", err)
		}
		brokenAct, err := ps.ActivateRevision(ctx, policystore.ActivateRevisionInput{
			DefinitionID:    profRev.DefinitionID,
			RevisionID:      profRev.RevisionID,
			ExpectedVersion: "1",
			Reason:          "TEST ONLY activate broken profile",
			ActorID:         actor,
		})
		if err != nil {
			t.Fatalf("ActivateRevision broken profile: %v", err)
		}

		drainAll(t)

		brokenOpID := uuid.UUID(brokenAct.Operation.OperationId)
		var brokenOpStatus string
		var publishedDocs int
		var publishAudits int
		_ = pool.QueryRow(ctx, `SELECT status FROM operations WHERE id = $1`, brokenOpID).Scan(&brokenOpStatus)
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM dns_resolver_documents WHERE profile_revision_id = $1`, profRev.RevisionID).Scan(&publishedDocs)
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action LIKE 'dns.publish%' AND resource_id = $1`, profRev.RevisionID).Scan(&publishAudits)

		if brokenOpStatus == "succeeded" && publishedDocs == 0 {
			t.Errorf("TODO(review) [F-04]: policy.activate operation %s for dns_profile reported status=%q even though dns.publish failed and published 0 resolver documents (publish audit count=%d)",
				brokenOpID, brokenOpStatus, publishAudits)
		}

		// 2. Now bootstrap mismatch-feed so the failed dns.publish retry (or a concurrent dns.publish
		// that finishes after handlePolicyActivated) runs noteOperation after operations.status is already 'succeeded'.
		feedBody = "different-pinned-body.example.com\n"
		if _, err := feedSvc.Handle(ctx, db.ClaimedEvent{
			OutboxEvent: db.OutboxEvent{
				ID:            uuid.New(),
				EventType:     dnsfeeds.EventRefresh,
				SchemaVersion: 1,
				Payload:       []byte(fmt.Sprintf(`{"revision_id":%q}`, mismatchRevID.String())),
			},
		}); err != nil {
			t.Fatalf("bootstrap mismatchRevID after fix: %v", err)
		}
		_, _ = pool.Exec(ctx, `UPDATE outbox_events SET available_at = clock_timestamp() - interval '1 second', lease_until = NULL WHERE event_type = 'dns.publish' AND delivered_at IS NULL`)
		drainAll(t)

		var observedRefs map[string]any
		if err := pool.QueryRow(ctx, `SELECT observed_refs FROM operations WHERE id = $1`, brokenOpID).Scan(&observedRefs); err != nil {
			t.Fatalf("query observed_refs: %v", err)
		}
		if _, ok := observedRefs["dns_document"]; !ok {
			t.Errorf("TODO(review) [F-04]: policy.activate operation %s lost observed_refs.dns_document (got observed_refs=%v) because handlePolicyActivated set status='succeeded' before dns.publish ran noteOperation",
				brokenOpID, observedRefs)
		}
	})
}
