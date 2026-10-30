//go:build integration

package policies_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/apps/control"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/dnsfeeds"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/policies/policiestest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// D-039: compose only control-core and policies; no unrelated domain fixtures.
func newFeedHTTPStack(t *testing.T, pool *pgxpool.Pool) *httpStack {
	t.Helper()
	cfg := config.Defaults()
	cfg.PublicOrigin = "https://portal.example.invalid"
	cfg.BrowserRate = config.Bucket{PerSecond: 1000, Burst: 1000}
	cfg.IngressRate = cfg.BrowserRate
	cfg.PoliciesConfigFile = policiestest.New(t).Path
	var sets []any
	for _, domain := range control.Domains() {
		if domain.Name != "control-core" && domain.Name != "policies" {
			continue
		}
		built, err := domain.Build(t.Context(), control.Dependencies{Config: cfg, Pool: pool})
		if err != nil {
			t.Fatal(err)
		}
		sets = append(sets, built.Handlers...)
	}
	handlers, err := httpapi.Compose(sets...)
	if err != nil {
		t.Fatal(err)
	}
	admin := auth.Principal{Role: auth.Admin, ID: uuid.New(), MFAVerified: true, CSRFToken: "TEST-ONLY-CSRF", AuthenticatedAt: time.Now().UTC()}
	handler, err := httpapi.New(handlers, httpapi.Options{Config: cfg, Commands: &httpapi.Commands{Pool: pool, Retention: time.Hour},
		Sessions: auth.SessionAuthenticatorFunc(func(*http.Request) (auth.Principal, error) { return admin, nil }),
	})
	if err != nil {
		t.Fatal(err)
	}
	return &httpStack{public: httpapi.Transport(nil, handler), adminP: admin}
}

func TestD058_HTTPFeedDefaultNullAttributionAndRefreshStatus(t *testing.T) {
	pool := dbtest.New(t)
	st := newFeedHTTPStack(t, pool)
	raw, err := os.ReadFile("../../contracts/examples/valid/PolicyRevisionCreate.feed-default-null-attribution.json")
	if err != nil {
		t.Fatal(err)
	}
	var request map[string]json.RawMessage
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(request["payload"], &fields); err != nil {
		t.Fatal(err)
	}
	body := "a.example.com\n"
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(body)))
	fields["content_sha256"], _ = json.Marshal(digest)
	fields["source_url"] = json.RawMessage(`"https://example.com/feed"`)
	now := time.Now().UTC().Truncate(time.Second)
	fields["valid_from"], _ = json.Marshal(now.Add(-time.Hour))
	fields["valid_until"], _ = json.Marshal(now.Add(7 * 24 * time.Hour))
	encode := func() string {
		t.Helper()
		request["payload"], _ = json.Marshal(fields)
		encoded, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		return string(encoded)
	}
	fields["update_interval_seconds"] = json.RawMessage("0")
	w := st.doAdmin(t, http.MethodPost, "/api/v1/admin/policies", encode(), "", "")
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("explicit zero: %d %s", w.Code, w.Body.String())
	}
	var count int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM policy_revisions`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("zero persisted: %d %v", count, err)
	}
	delete(fields, "update_interval_seconds")
	key := uuid.NewString()
	w = st.doAdmin(t, http.MethodPost, "/api/v1/admin/policies", encode(), key, "")
	if w.Code != http.StatusCreated {
		t.Fatalf("omitted interval: %d %s", w.Code, w.Body.String())
	}
	rev, def := extractRevisionAndDefinitionIDs(t, w.Body.Bytes())
	var response struct {
		Payload api.ThreatFeedPayload `json:"payload"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || response.Payload.UpdateIntervalSeconds != 21600 || !response.Payload.Attribution.IsNull() {
		t.Fatalf("response default/null: %+v %v", response.Payload, err)
	}
	replay := st.doAdmin(t, http.MethodPost, "/api/v1/admin/policies", encode(), key, "")
	firstJSON, firstErr := canonjson.Canonicalize(w.Body.Bytes())
	replayJSON, replayErr := canonjson.Canonicalize(replay.Body.Bytes())
	if replay.Code != http.StatusCreated || firstErr != nil || replayErr != nil || string(replayJSON) != string(firstJSON) {
		t.Fatalf("idempotent replay: %d %s", replay.Code, replay.Body.String())
	}
	var persisted []byte
	var persistedHash string
	if err := pool.QueryRow(t.Context(), `SELECT payload,encode(payload_hash,'hex') FROM policy_revisions WHERE id=$1`, rev).Scan(&persisted, &persistedHash); err != nil {
		t.Fatal(err)
	}
	var stored api.ThreatFeedPayload
	if err := json.Unmarshal(persisted, &stored); err != nil || stored.UpdateIntervalSeconds != 21600 || !stored.Attribution.IsNull() {
		t.Fatalf("persisted default/null: %+v %v", stored, err)
	}
	if hash, err := canonjson.HashHex(persisted); err != nil || hash != persistedHash {
		t.Fatalf("persisted hash: %v", err)
	}
	get := func() api.PolicyDefinition {
		t.Helper()
		w := st.doAdmin(t, http.MethodGet, "/api/v1/admin/policies?type=threat_feed", "", "", "")
		if w.Code != http.StatusOK {
			t.Fatalf("list: %d %s", w.Code, w.Body.String())
		}
		var list api.PolicyDefinitionList
		if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil || len(list.Items) != 1 {
			t.Fatalf("list: %v %s", err, w.Body.String())
		}
		return list.Items[0]
	}
	if !get().ThreatFeedStatus.IsNull() {
		t.Fatal("draft status should be null")
	}
	activate := fmt.Sprintf(`{"revision_id":%q,"reason":"TEST ONLY"}`, rev.String())
	w = st.doAdmin(t, http.MethodPost, "/api/v1/admin/policies/"+def.String()+"/activate", activate, "", `"1"`)
	if w.Code != http.StatusAccepted {
		t.Fatalf("activate: %d %s", w.Code, w.Body.String())
	}
	getStatus := func() api.ThreatFeedStatus {
		t.Helper()
		s, err := get().ThreatFeedStatus.Get()
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	status := getStatus()
	if status.ApprovedContentSha256 != digest || status.LastResult != api.ThreatFeedStatusLastResultNever || !status.LastGoodAgeSeconds.IsNull() || !status.LastGoodAt.IsNull() || !status.LastGoodContentSha256.IsNull() || !status.NextDueAt.IsNull() {
		t.Fatalf("bootstrap: %+v", status)
	}
	// D-050: run the product job; never seed refresh observations with SQL.
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, body) }))
	t.Cleanup(server.Close)
	client := server.Client()
	transport := client.Transport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	client.Transport = transport
	service, err := dnsfeeds.New(pool, dnsfeeds.Options{Client: client, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	refresh := func() {
		t.Helper()
		b, _ := json.Marshal(map[string]string{"revision_id": rev.String()})
		if _, err := service.Handle(t.Context(), db.ClaimedEvent{OutboxEvent: db.OutboxEvent{ID: uuid.New(), EventType: dnsfeeds.EventRefresh, SchemaVersion: 1, Payload: b}}); err != nil {
			t.Fatal(err)
		}
	}
	body = "wrong.example.com\n"
	refresh()
	status = getStatus()
	code, _ := status.LastErrorCode.Get()
	if status.LastResult != api.ThreatFeedStatusLastResultFailed || status.ConsecutiveFailures != 1 || code != api.ErrorCodeHASHMISMATCH || !status.LastGoodRuleCount.IsNull() || !status.LastGoodAgeSeconds.IsNull() {
		t.Fatalf("failed bootstrap: %+v", status)
	}
	now = now.Add(6 * time.Hour)
	body = "a.example.com\n"
	refresh()
	status = getStatus()
	good, _ := status.LastGoodContentSha256.Get()
	rules, _ := status.LastGoodRuleCount.Get()
	if status.LastResult != api.ThreatFeedStatusLastResultSuccess || good != digest || rules != 1 || status.LastGoodAgeSeconds.IsNull() || !status.LastErrorCode.IsNull() || !status.LastErrorMessage.IsNull() || status.ConsecutiveFailures != 0 {
		t.Fatalf("last-good: %+v", status)
	}
}
