//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type commandFixture struct {
	pool   *pgxpool.Pool
	calls  atomic.Int32
	record atomic.Pointer[db.IdempotencyRecord]
}

func (f *commandFixture) UpdateNode(ctx context.Context, r gen.UpdateNodeRequestObject) (gen.UpdateNodeResponseObject, error) {
	tx, e := Transaction(ctx)
	if e != nil {
		return nil, e
	}
	record, ok := CommandRecord(ctx)
	if !ok {
		return nil, errors.New("command record unavailable")
	}
	f.record.Store(&record)
	if e = CheckIfMatch(r.Params.IfMatch, 7); e != nil {
		return nil, e
	}
	p, _ := auth.FromContext(ctx)
	rid, _ := apierr.RequestIDFromContext(ctx)
	id, e := operations.Create(ctx, tx, operations.CreateRequest{Kind: "node.update", Principal: p, Resource: api.ResourceRef{Type: "node", Id: r.NodeId}, RequestID: uuid.MustParse(rid), Steps: map[string]operations.Step{"readback": {Required: true}}})
	if e != nil {
		return nil, e
	}
	f.calls.Add(1)
	e = db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: r.NodeId, AggregateType: "node", AggregateVersion: "7", EventType: "test.readback", SchemaVersion: 1, Payload: []byte(`{}`)})
	if e != nil {
		return nil, e
	}
	if r.Body.Name != nil && *r.Body.Name == "rollback" {
		return gen.UpdateNode202JSONResponse{OperationAcceptedJSONResponse: operations.Accepted(api.Operation{OperationId: id, Status: "pending",
			CompletedAt:       presence.NewNullNullable[time.Time](),
			DesiredGeneration: presence.NewNullNullable[decstr.U64](),
			Error: presence.NewNullNullable[struct {
				Code    api.ErrorCode `json:"code"`
				Message string        `json:"message"`
			}](),
			ObservedGeneration: presence.NewNullNullable[decstr.U64](),
			ResourceVersion:    presence.NewNullNullable[decstr.U64](),
		}, p.Role)}, nil
	}
	e = SetResult(ctx, db.IdempotencyResult{ResourceType: "node", ResourceID: &r.NodeId, OperationID: &id, ResponseStatus: 202})
	if e != nil {
		return nil, e
	}
	return gen.UpdateNode202JSONResponse{OperationAcceptedJSONResponse: operations.Accepted(api.Operation{OperationId: id, Status: "pending",
		CompletedAt:       presence.NewNullNullable[time.Time](),
		DesiredGeneration: presence.NewNullNullable[decstr.U64](),
		Error: presence.NewNullNullable[struct {
			Code    api.ErrorCode `json:"code"`
			Message string        `json:"message"`
		}](),
		ObservedGeneration: presence.NewNullNullable[decstr.U64](),
		ResourceVersion:    presence.NewNullNullable[decstr.U64](),
	}, p.Role)}, nil
}
func (f *commandFixture) Replay(ctx context.Context, _ string, r db.IdempotencyRecord) (any, error) {
	if _, ok := CommandRecord(ctx); ok {
		return nil, errors.New("replay unexpectedly has an active command record")
	}
	p, _ := auth.FromContext(ctx)
	op, e := (operations.Store{Pool: f.pool}).Get(ctx, *r.OperationID, p)
	if e != nil {
		return nil, e
	}
	return gen.UpdateNode202JSONResponse{OperationAcceptedJSONResponse: operations.Accepted(op, p.Role)}, nil
}
func TestAPI01_IdempotencyReplayConflictConcurrentExpiry(t *testing.T) {
	pool := dbtest.New(t)
	p := principal(auth.Admin)
	f := &commandFixture{pool: pool}
	c := config.Defaults()
	c.PublicOrigin = "https://portal.example.invalid"
	c.BrowserRate = config.Bucket{PerSecond: 1000, Burst: 1000}
	handlers, e := Compose(f)
	if e != nil {
		t.Fatal(e)
	}
	apiHandler, e := New(handlers, Options{Config: c, Sessions: auth.SessionAuthenticatorFunc(func(*http.Request) (auth.Principal, error) { return p, nil }), Commands: &Commands{Pool: pool, Retention: time.Hour}})
	if e != nil {
		t.Fatal(e)
	}
	h := Transport(nil, apiHandler)
	key := uuid.NewString()
	path := "/api/v1/admin/nodes/" + uuid.NewString()
	invoke := func(body string) *httptest.ResponseRecorder {
		r := request("PATCH", path, body, p)
		r.Header.Set("Idempotency-Key", key)
		r.Header.Set("If-Match", `"7"`)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	var wg sync.WaitGroup
	responses := make(chan *httptest.ResponseRecorder, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); responses <- invoke(`{"name":"new"}`) }()
	}
	wg.Wait()
	close(responses)
	var id uuid.UUID
	for w := range responses {
		if w.Code != 202 {
			t.Fatalf("concurrent response %d %s", w.Code, w.Body.String())
		}
		var op api.Operation
		if e = json.Unmarshal(w.Body.Bytes(), &op); e != nil {
			t.Fatal(e)
		}
		if id == uuid.Nil {
			id = op.OperationId
		}
		if id != op.OperationId {
			t.Fatal("different operations")
		}
		if w.Header().Get("Location") == "" {
			t.Fatal("missing Location")
		}
	}
	if f.calls.Load() != 1 {
		t.Fatal("duplicate mutation", f.calls.Load())
	}
	checkRecord := func() db.IdempotencyRecord {
		t.Helper()
		record := f.record.Load()
		if record == nil || !record.New || record.ID == uuid.Nil || record.State != "in_progress" {
			t.Fatalf("missing active reservation: %+v", record)
		}
		var stored db.IdempotencyRecord
		var route string
		if e := pool.QueryRow(t.Context(), `SELECT id,created_at,expiry,route FROM idempotency_requests WHERE id=$1`, record.ID).Scan(&stored.ID, &stored.CreatedAt, &stored.Expiry, &route); e != nil {
			t.Fatal(e)
		}
		routeSuffix := "|node_id=" + strings.TrimPrefix(path, "/api/v1/admin/nodes/")
		if stored.ID != record.ID || !stored.CreatedAt.Equal(record.CreatedAt) || !stored.Expiry.Equal(record.Expiry) || !strings.HasSuffix(route, routeSuffix) {
			t.Fatalf("accessor differs from scoped reservation: got %+v, stored %+v, route %q", *record, stored, route)
		}
		return *record
	}
	firstRecord := checkRecord()
	if w := invoke(`{ "name" : "new" }`); w.Code != 202 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := invoke(`{"name":"different"}`); w.Code != 409 {
		t.Fatal("conflict", w.Code, w.Body.String())
	}
	// Replay re-reads the current operation, rather than a saved plaintext body.
	if _, e = pool.Exec(t.Context(), `UPDATE operations SET status='running' WHERE id=$1`, id); e != nil {
		t.Fatal(e)
	}
	w := invoke(`{"name":"new"}`)
	var op api.Operation
	_ = json.Unmarshal(w.Body.Bytes(), &op)
	if op.Status != "running" {
		t.Fatal("replayed stale operation")
	}
	if _, e = pool.Exec(t.Context(), `UPDATE idempotency_requests SET created_at=clock_timestamp()-interval '2 hours',expiry=clock_timestamp()-interval '1 hour'`); e != nil {
		t.Fatal(e)
	}
	w = invoke(`{"name":"new"}`)
	if w.Code != 202 || f.calls.Load() != 2 {
		t.Fatal("expiry", w.Code, w.Body.String(), f.calls.Load())
	}
	renewedRecord := checkRecord()
	if renewedRecord.ID == firstRecord.ID || !renewedRecord.Expiry.After(firstRecord.Expiry) {
		t.Fatal("expired reservation reused", firstRecord, renewedRecord)
	}
	r := request("PATCH", path, `{"name":"conflict"}`, p)
	r.Header.Set("If-Match", `"6"`)
	assertStatus(t, h, r, 412)
	r = request("PATCH", path, `{"name":"rollback"}`, p)
	r.Header.Set("If-Match", `"7"`)
	assertStatus(t, h, r, 500)
	var records, outbox int
	if e = pool.QueryRow(t.Context(), `SELECT count(*) FROM operations`).Scan(&records); e != nil {
		t.Fatal(e)
	}
	if e = pool.QueryRow(t.Context(), `SELECT count(*) FROM outbox_events`).Scan(&outbox); e != nil {
		t.Fatal(e)
	}
	if records != 2 || outbox != 2 {
		t.Fatal("non-atomic effects", records, outbox)
	}
	var incomplete, completed int
	if e = pool.QueryRow(t.Context(), `SELECT count(*) FILTER (WHERE state<>'completed' OR response_status IS NULL OR (resource_id IS NULL AND operation_id IS NULL)),count(*) FROM idempotency_requests`).Scan(&incomplete, &completed); e != nil || incomplete != 0 || completed != 1 {
		t.Fatal("command reservations must commit complete; failed commands must roll back", incomplete, completed, e)
	}
	// Same principal, method and key on a different resource must expose that
	// route's reservation, not the record found by an incomplete scope lookup.
	path = "/api/v1/admin/nodes/" + uuid.NewString()
	if w := invoke(`{"name":"new"}`); w.Code != 202 {
		t.Fatal(w.Code, w.Body.String())
	}
	if otherRecord := checkRecord(); otherRecord.ID == renewedRecord.ID {
		t.Fatal("different route reused command record", otherRecord)
	}
}

func TestAT28_MemberOperationHTTPIsolation(t *testing.T) {
	pool := dbtest.New(t)
	owner, other := principal(auth.Member), principal(auth.Member)
	var id uuid.UUID
	e := db.InTx(t.Context(), pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var e error
		id, e = operations.Create(t.Context(), tx, operations.CreateRequest{Kind: "member.update", Principal: owner, Resource: api.ResourceRef{Type: "member", Id: owner.ID}, RequestID: uuid.New(), Steps: map[string]operations.Step{"policy": {Required: true}}})
		return e
	})
	if e != nil {
		t.Fatal(e)
	}
	store := operations.Store{Pool: pool}
	path := "/api/v1/member/operations/" + id.String()
	assertStatus(t, fixture(t, owner, false, nil, store), request("GET", path, "", owner), 200)
	assertStatus(t, fixture(t, other, false, nil, store), request("GET", path, "", other), 404)
}
