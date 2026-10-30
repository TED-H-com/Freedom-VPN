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
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/dnsfeeds"
	policystore "github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
)

func TestD054_DNSFeedDispatcherAndActivationFence(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	now := time.Now().UTC()
	const body = "malware.example.com\n"
	var calls atomic.Int32
	started, release := make(chan struct{}), make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 2 {
			close(started)
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
		}
		_, _ = fmt.Fprint(w, body)
	}))
	defer server.Close()
	client := server.Client()
	tr := client.Transport.(*http.Transport).Clone()
	tr.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	client.Transport = tr
	payload := api.ThreatFeedPayload{Provider: "TEST ONLY", FeedCode: "local-feed", LicenseName: "TEST ONLY", LicenseUrl: "https://example.com/license", CommercialUseBasis: "TEST ONLY fixture approval", Attribution: presence.NewNullNullable[string](), SourceUrl: "https://example.com/feed", Format: api.ThreatFeedPayloadFormatDomainList, Categories: []api.ThreatFeedPayloadCategories{api.ThreatFeedPayloadCategoriesMalware}, ContentSha256: fmt.Sprintf("%x", sha256.Sum256([]byte(body))), MaxSizeBytes: 4096, ValidFrom: now.Add(-time.Hour), ValidUntil: now.Add(48 * time.Hour), UpdateIntervalSeconds: 300}
	raw, _ := json.Marshal(payload)
	code := "local-feed"
	actor := uuid.New()
	ps := policystore.NewStore(pool)
	rev, err := ps.CreateRevision(ctx, policystore.CreateRevisionInput{Code: &code, Type: api.PolicyTypeThreatFeed, SchemaVersion: 1, Payload: raw, Reason: "TEST ONLY bootstrap", ActorID: actor})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ps.ActivateRevision(ctx, policystore.ActivateRevisionInput{DefinitionID: rev.DefinitionID, RevisionID: rev.RevisionID, ExpectedVersion: "1", Reason: "TEST ONLY approval", ActorID: actor}); err != nil {
		t.Fatal(err)
	}
	// This isolated dispatcher owns only DNS. Other domains' event is not part of
	// this test; acknowledge it in the fixture so the real feed claim runs first.
	if _, err := pool.Exec(ctx, `UPDATE outbox_events SET delivered_at=clock_timestamp() WHERE event_type='policy.activated'`); err != nil {
		t.Fatal(err)
	}
	svc, err := dnsfeeds.New(pool, dnsfeeds.Options{Client: client, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	jobs := Registry{}
	jobs.Register(dnsfeeds.EventRefresh, svc.Handle)
	dispatcher, err := New(pool, jobs, Config{Concurrency: 1, Attempts: 3, Lease: time.Minute, JobTimeout: 30 * time.Second, Poll: time.Second, Backoff: time.Second, MaxBackoff: time.Minute}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if worked, err := dispatcher.Once(ctx); err != nil || !worked {
		t.Fatalf("dispatch: %v %v", worked, err)
	}
	var receipt string
	if err := pool.QueryRow(ctx, `SELECT delivery_receipt FROM outbox_events WHERE event_type='dns.feed.refresh' AND delivered_at IS NOT NULL`).Scan(&receipt); err != nil || !strings.HasPrefix(receipt, "dns-feed:refreshed:") {
		t.Fatalf("read-back receipt %s %v", receipt, err)
	}
	if worked, err := dispatcher.Once(ctx); err != nil || worked {
		t.Fatalf("unexpected duplicate work: %v %v", worked, err)
	}
	if calls.Load() != 1 {
		t.Fatal("duplicate HTTP fetch")
	}
	// A refresh is due; replace the active revision while HTTP is in flight.
	now = now.Add(6 * time.Hour)
	if err := svc.Schedule(ctx); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := dispatcher.Once(ctx); done <- err }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("refresh did not start")
	}
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	next, err := ps.CreateRevision(ctx, policystore.CreateRevisionInput{DefinitionID: &rev.DefinitionID, Type: api.PolicyTypeThreatFeed, SchemaVersion: 1, Payload: raw, Reason: "TEST ONLY replacement", ActorID: actor})
	if err != nil {
		t.Fatal(err)
	}
	var version string
	if err := pool.QueryRow(ctx, `SELECT resource_version::text FROM policy_definitions WHERE id=$1`, rev.DefinitionID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if _, err := ps.ActivateRevision(ctx, policystore.ActivateRevisionInput{DefinitionID: rev.DefinitionID, RevisionID: next.RevisionID, ExpectedVersion: version, Reason: "TEST ONLY supersede", ActorID: actor}); err != nil {
		t.Fatal(err)
	}
	close(release)
	released = true
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='dns.feed.refreshed'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("superseded fetch committed: %d %v", count, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM dns_feed_contents`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("superseded content committed: %d %v", count, err)
	}
}
