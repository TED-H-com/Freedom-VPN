//go:build integration

package dnsfeeds

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// All projection rows below are written by the real refresh service, never SQL
// fixtures (D-050). Only a local TLS feed and injected offline failures are used.
func TestD058_StatusProjectionFromRefreshJob(t *testing.T) {
	pool := dbtest.New(t)
	ctx := t.Context()
	now := time.Date(2026, 1, 1, 6, 0, 0, 0, time.UTC)
	body := "a.example.com\n"
	client := localFeed(t, &body)
	ps := policies.NewStore(pool, policies.WithClock(func() time.Time { return now }))
	payload := testPayload(body, now)
	payload.Attribution.SetNull()
	payload.UpdateIntervalSeconds = 0 // D-058d omitted default
	raw, _ := json.Marshal(payload)
	code, actor := "status-feed", uuid.New()
	rev, err := ps.CreateRevision(ctx, policies.CreateRevisionInput{Code: &code, Type: api.PolicyTypeThreatFeed, SchemaVersion: 1, Payload: raw, Reason: "TEST ONLY", ActorID: actor})
	if err != nil {
		t.Fatal(err)
	}
	list, err := ps.ListDefinitions(ctx, policies.ListDefinitionsFilter{})
	if err != nil || len(list.Items) != 1 || !list.Items[0].ThreatFeedStatus.IsNull() {
		t.Fatalf("draft has refresh status: %v", err)
	}
	if _, err := ps.ActivateRevision(ctx, policies.ActivateRevisionInput{DefinitionID: rev.DefinitionID, RevisionID: rev.RevisionID, ExpectedVersion: "1", Reason: "TEST ONLY", ActorID: actor}); err != nil {
		t.Fatal(err)
	}
	status := func() api.ThreatFeedStatus {
		t.Helper()
		list, err := ps.ListDefinitions(ctx, policies.ListDefinitionsFilter{})
		if err != nil || len(list.Items) != 1 {
			t.Fatalf("list: %v", err)
		}
		got, err := list.Items[0].ThreatFeedStatus.Get()
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	fixture := func(name string) api.ThreatFeedStatus {
		t.Helper()
		raw, err := os.ReadFile("../../contracts/examples/valid/PolicyDefinitionList." + name + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var list api.PolicyDefinitionList
		if err := json.Unmarshal(raw, &list); err != nil {
			t.Fatal(err)
		}
		got, err := list.Items[0].ThreatFeedStatus.Get()
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	assertStatus := func(want api.ThreatFeedStatus) {
		t.Helper()
		got, err := json.Marshal(status())
		if err != nil {
			t.Fatal(err)
		}
		expected, err := json.Marshal(want)
		if err != nil || string(got) != string(expected) {
			t.Fatalf("status JSON\ngot  %s\nwant %s (%v)", got, expected, err)
		}
	}
	want := fixture("feed-bootstrap")
	want.ApprovedContentSha256 = payload.ContentSha256
	assertStatus(want)
	s, err := New(pool, Options{Clock: func() time.Time { return now }, Client: client, UpdateCadence: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	event := func() db.ClaimedEvent {
		raw, _ := json.Marshal(map[string]string{"revision_id": rev.RevisionID.String()})
		return db.ClaimedEvent{OutboxEvent: db.OutboxEvent{ID: uuid.New(), EventType: EventRefresh, SchemaVersion: 1, Payload: raw}}
	}
	refresh := func() {
		t.Helper()
		if _, err := s.Handle(ctx, event()); err != nil {
			t.Fatal(err)
		}
	}
	// Starting an attempt does not claim a successful completion.
	e := event()
	a, err := s.begin(ctx, rev.RevisionID, e.ID, now)
	if err != nil || a == nil {
		t.Fatalf("begin: %v", err)
	}
	assertStatus(want) // lease start is not yet a completed observation
	c, reason := Fetch(ctx, client, a.payload, nil, now, 50)
	if ok, err := s.finish(ctx, a, e, c, reason, now); err != nil || !ok {
		t.Fatalf("finish: %v", err)
	}
	want.LastAttemptAt.Set(now)
	want.LastResult = api.ThreatFeedStatusLastResultSuccess
	want.LastGoodContentSha256.Set(payload.ContentSha256)
	want.LastGoodRuleCount.Set(1)
	want.LastGoodAt.Set(now)
	want.LastGoodAgeSeconds.Set(0)
	want.NextDueAt.Set(now.Add(6 * time.Hour)) // revision default > product cadence
	assertStatus(want)
	// The same active revision accepts a later hash. A newer draft does not
	// replace the active revision's approved hash in the status projection.
	now = now.Add(6 * time.Hour)
	body = "a.example.com\n# refreshed\n"
	refresh()
	goodHash := digest(body)
	payload.ContentSha256 = digest("draft.example.com\n")
	draftRaw, _ := json.Marshal(payload)
	draft, err := ps.CreateRevision(ctx, policies.CreateRevisionInput{DefinitionID: &rev.DefinitionID, Type: api.PolicyTypeThreatFeed, SchemaVersion: 1, Payload: draftRaw, Reason: "TEST ONLY draft", ActorID: actor})
	if err != nil {
		t.Fatal(err)
	}
	s.opts.Client = &http.Client{Transport: failedFeedTransport{}}
	for _, hour := range []int{0, 6, 12} {
		now = time.Date(2026, 1, 2, hour, 0, 0, 0, time.UTC)
		refresh()
	}
	want = fixture("feed-last-good")
	want.ApprovedContentSha256 = a.payload.ContentSha256
	want.LastGoodContentSha256.Set(goodHash)
	assertStatus(want)
	// A new attempt retains the previous completed failure until success.
	now = now.Add(6 * time.Hour)
	e = event()
	a, err = s.begin(ctx, rev.RevisionID, e.ID, now)
	if err != nil || a == nil {
		t.Fatalf("begin recovery: %v", err)
	}
	want.LastGoodAgeSeconds.Set(decstr.U64(30 * 3600))
	assertStatus(want)
	c, reason = Fetch(ctx, client, a.payload, a.previous, now, 50)
	if ok, err := s.finish(ctx, a, e, c, reason, now); err != nil || !ok {
		t.Fatalf("finish recovery: %v", err)
	}
	got := status()
	if got.LastResult != api.ThreatFeedStatusLastResultSuccess || got.ConsecutiveFailures != 0 || !got.LastErrorCode.IsNull() || !got.LastErrorMessage.IsNull() {
		t.Fatalf("recovery retained error: %+v", got)
	}
	// Activating the draft resets public status to its own bootstrap state.
	def, err := ps.GetDefinition(ctx, rev.DefinitionID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ps.ActivateRevision(ctx, policies.ActivateRevisionInput{DefinitionID: rev.DefinitionID, RevisionID: draft.RevisionID, ExpectedVersion: def.ResourceVersion.String(), Reason: "TEST ONLY replacement", ActorID: actor}); err != nil {
		t.Fatal(err)
	}
	want = fixture("feed-bootstrap")
	want.ApprovedContentSha256 = payload.ContentSha256
	assertStatus(want)
}

type failedFeedTransport struct{}

func (failedFeedTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("TEST ONLY offline https://user:TEST-ONLY@example.invalid/feed body.example.com")
}

func TestD058_RefreshUsesMaximumOfProductAndMaterializedRevisionCadence(t *testing.T) {
	pool := dbtest.New(t)
	now := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)
	body := "a.example.com\n"
	client := localFeed(t, &body)
	ps := policies.NewStore(pool, policies.WithClock(func() time.Time { return now }))
	for _, tt := range []struct {
		name          string
		revision      int
		product, want time.Duration
	}{
		{"omitted", 0, 5 * time.Minute, 6 * time.Hour},
		{"product", 300, 12 * time.Hour, 12 * time.Hour},
		{"revision", 86400, 6 * time.Hour, 24 * time.Hour},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := testPayload(body, now)
			p.Attribution.SetNull()
			p.UpdateIntervalSeconds = tt.revision
			raw, _ := json.Marshal(p)
			actor := uuid.New()
			rev, err := ps.CreateRevision(t.Context(), policies.CreateRevisionInput{Code: &tt.name, Type: api.PolicyTypeThreatFeed, SchemaVersion: 1, Payload: raw, Reason: "TEST ONLY", ActorID: actor})
			if err != nil {
				t.Fatal(err)
			}
			wantInterval := tt.revision
			if wantInterval == 0 {
				wantInterval = 21600
			}
			var stored api.ThreatFeedPayload
			if err := json.Unmarshal(rev.Payload, &stored); err != nil || stored.UpdateIntervalSeconds != wantInterval || !stored.Attribution.IsNull() {
				t.Fatalf("stored cadence/null: %+v %v", stored, err)
			}
			if _, err := ps.ActivateRevision(t.Context(), policies.ActivateRevisionInput{DefinitionID: rev.DefinitionID, RevisionID: rev.RevisionID, ExpectedVersion: "1", Reason: "TEST ONLY", ActorID: actor}); err != nil {
				t.Fatal(err)
			}
			s, err := New(pool, Options{Client: client, Clock: func() time.Time { return now }, UpdateCadence: tt.product})
			if err != nil {
				t.Fatal(err)
			}
			b, _ := json.Marshal(map[string]string{"revision_id": rev.RevisionID.String()})
			if _, err := s.Handle(t.Context(), db.ClaimedEvent{OutboxEvent: db.OutboxEvent{ID: uuid.New(), EventType: EventRefresh, SchemaVersion: 1, Payload: b}}); err != nil {
				t.Fatal(err)
			}
			var next time.Time
			if err := pool.QueryRow(t.Context(), `SELECT next_attempt_at FROM dns_feed_refresh_state WHERE feed_revision_id=$1`, rev.RevisionID).Scan(&next); err != nil || !next.Equal(now.Add(tt.want)) {
				t.Fatalf("next %v want %v: %v", next, now.Add(tt.want), err)
			}
		})
	}
}
