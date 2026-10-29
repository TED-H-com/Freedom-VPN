//go:build integration

package httpapi

import (
	"context"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type receiptFixture struct {
	pool           *pgxpool.Pool
	calls, replays int
}

func (f *receiptFixture) save(ctx context.Context) (uuid.UUID, error) {
	tx, e := Transaction(ctx)
	if e != nil {
		return uuid.Nil, e
	}
	id := uuid.New()
	f.calls++
	// A durable test receipt; real domains store observations/results in their
	// own tables and use the same transaction and reference-only replay contract.
	e = db.EnqueueOutbox(ctx, tx, db.OutboxEvent{ID: id, AggregateType: "node", AggregateID: id, AggregateVersion: "1", EventType: "test.receipt", SchemaVersion: 1, Payload: []byte(`{}`)})
	if e != nil {
		return id, e
	}
	return id, SetResult(ctx, db.IdempotencyResult{ResourceType: "node", ResourceID: &id, ResponseStatus: 202})
}
func (f *receiptFixture) SubmitObservation(ctx context.Context, _ gen.SubmitObservationRequestObject) (gen.SubmitObservationResponseObject, error) {
	id, e := f.save(ctx)
	return gen.SubmitObservation202JSONResponse{Body: api.ObservationAccepted{Accepted: true, ReportId: id}}, e
}
func (f *receiptFixture) SubmitUpdateResult(ctx context.Context, _ gen.SubmitUpdateResultRequestObject) (gen.SubmitUpdateResultResponseObject, error) {
	id, e := f.save(ctx)
	return gen.SubmitUpdateResult202JSONResponse{Body: api.UpdateResultAck{Accepted: true, ResultId: id}}, e
}
func (f *receiptFixture) Replay(ctx context.Context, route string, record db.IdempotencyRecord) (any, error) {
	f.replays++
	var id uuid.UUID
	if e := f.pool.QueryRow(ctx, `SELECT id FROM outbox_events WHERE id=$1`, record.ResourceID).Scan(&id); e != nil {
		return nil, e
	}
	if route == "SubmitObservation" {
		return gen.SubmitObservation202JSONResponse{Body: api.ObservationAccepted{Accepted: true, ReportId: id}}, nil
	}
	return gen.SubmitUpdateResult202JSONResponse{Body: api.UpdateResultAck{Accepted: true, ResultId: id}}, nil
}
func TestAPI01_AgentReceiptKeyedReplay(t *testing.T) {
	pool := dbtest.New(t)
	f := &receiptFixture{pool: pool}
	composite, e := Compose(f)
	if e != nil {
		t.Fatal(e)
	}
	commands := Commands{Pool: pool, Retention: time.Hour}
	for _, route := range []string{"SubmitObservation", "SubmitUpdateResult"} {
		key := uuid.NewString()
		next := func(ctx context.Context, _ http.ResponseWriter, _ *http.Request, _ any) (any, error) {
			if route == "SubmitObservation" {
				return f.SubmitObservation(ctx, gen.SubmitObservationRequestObject{})
			}
			return f.SubmitUpdateResult(ctx, gen.SubmitUpdateResultRequestObject{})
		}
		h := commands.middleware(composite)(next, route)
		p := principal(auth.Agent)
		ctx := context.WithValue(auth.WithPrincipal(t.Context(), p), metaKey{}, &requestMeta{raw: []byte(`{}`), scope: route})
		var prior uuid.UUID
		for i := 0; i < 2; i++ {
			r := httptest.NewRequest("POST", "/api/v1/agent/observations", nil).WithContext(ctx)
			r.Header.Set("Idempotency-Key", key)
			response, e := h(ctx, httptest.NewRecorder(), r, nil)
			if e != nil {
				t.Fatal(route, e)
			}
			var id uuid.UUID
			switch v := response.(type) {
			case gen.SubmitObservation202JSONResponse:
				id = v.Body.ReportId
			case gen.SubmitUpdateResult202JSONResponse:
				id = v.Body.ResultId
			default:
				t.Fatalf("wrong receipt %T", response)
			}
			if i == 1 && id != prior {
				t.Fatal("replay changed receipt")
			}
			prior = id
		}
		// The exception also requires the certificate principal, not only a route ID.
		operator := principal(auth.Admin)
		ctx = context.WithValue(auth.WithPrincipal(t.Context(), operator), metaKey{}, &requestMeta{raw: []byte(`{}`), scope: route})
		r := httptest.NewRequest("POST", "/api/v1/agent/observations", nil).WithContext(ctx)
		r.Header.Set("Idempotency-Key", uuid.NewString())
		if _, e = h(ctx, httptest.NewRecorder(), r, nil); e == nil {
			t.Fatal("operator got receipt exemption")
		}
	}
	if f.replays != 2 {
		t.Fatal("not replayed", f.replays)
	}
	var records, events int
	if e = pool.QueryRow(t.Context(), `SELECT count(*) FROM idempotency_requests WHERE response_status=202 AND resource_id IS NOT NULL AND operation_id IS NULL AND state='completed'`).Scan(&records); e != nil {
		t.Fatal(e)
	}
	if e = pool.QueryRow(t.Context(), `SELECT count(*) FROM outbox_events`).Scan(&events); e != nil || records != 2 || events != 2 {
		t.Fatal("receipt transaction not atomic", records, events, e)
	}
}
