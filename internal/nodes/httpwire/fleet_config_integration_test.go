//go:build integration

package httpwire_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/fleetupdate"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const fleetConfigPath = "/api/v1/admin/fleet-config"

func TestD065_FleetConfigHTTPReadWrite(t *testing.T) {
	f := setup(t)
	before := f.call("GET", fleetConfigPath, nil, nil, nil)
	// Both operations are exercised before assertions to preserve the pre-wiring
	// 501 evidence for GET and PUT against the production nodes handler set.
	headers := map[string]string{"If-Match": `"1"`, "Idempotency-Key": uuid.NewString()}
	body := api.FleetConfigUpdate{MaxOfflineLeaseSeconds: 60, Reason: "TEST ONLY AT29 lease window"}
	first := f.call("PUT", fleetConfigPath, body, nil, headers)
	t.Logf("GET %d: %s", before.Code, before.Body.String())
	t.Logf("PUT %d: %s", first.Code, first.Body.String())
	if before.Code != 200 || first.Code != 200 {
		t.Fatalf("fleet configuration GET/PUT status = %d/%d; want 200/200", before.Code, first.Code)
	}
	initial := decode[api.FleetConfig](t, before, 200)
	stored := decode[api.FleetConfig](t, first, 200)
	if initial.MaxOfflineLeaseSeconds != 86400 || initial.DefaultMaxOfflineLeaseSeconds == nil || *initial.DefaultMaxOfflineLeaseSeconds != 86400 || initial.ResourceVersion != 1 || before.Header().Get("ETag") != `"1"` {
		t.Fatal("wrong singleton defaults", initial)
	}
	if stored.MaxOfflineLeaseSeconds != 60 || stored.DefaultMaxOfflineLeaseSeconds == nil || *stored.DefaultMaxOfflineLeaseSeconds != 86400 || stored.ResourceVersion != 2 || first.Header().Get("ETag") != `"2"` || !stored.UpdatedAt.After(initial.UpdatedAt) {
		t.Fatal("write not versioned", stored)
	}
	replay := f.call("PUT", fleetConfigPath, body, nil, headers)
	decode[api.FleetConfig](t, replay, 200)
	if !bytes.Equal(first.Body.Bytes(), replay.Body.Bytes()) || first.Header().Get("ETag") != replay.Header().Get("ETag") {
		t.Fatal("replay changed original resource")
	}
	for _, tc := range []struct {
		name, match, key, code string
		body                   any
		status                 int
	}{
		{"missing precondition", "", uuid.NewString(), "PRECONDITION_REQUIRED", body, 428},
		{"stale precondition", `"1"`, uuid.NewString(), "RESOURCE_VERSION_CONFLICT", body, 412},
		{"weak precondition", `W/"2"`, uuid.NewString(), "VALIDATION_FAILED", body, 422},
		{"changed payload", `"2"`, headers["Idempotency-Key"], "IDEMPOTENCY_CONFLICT", api.FleetConfigUpdate{MaxOfflineLeaseSeconds: 900, Reason: body.Reason}, 409},
		{"below floor", `"2"`, uuid.NewString(), "VALIDATION_FAILED", api.FleetConfigUpdate{MaxOfflineLeaseSeconds: 59, Reason: body.Reason}, 422},
		{"above ceiling", `"2"`, uuid.NewString(), "VALIDATION_FAILED", api.FleetConfigUpdate{MaxOfflineLeaseSeconds: 86401, Reason: body.Reason}, 422},
		{"empty reason", `"2"`, uuid.NewString(), "VALIDATION_FAILED", api.FleetConfigUpdate{MaxOfflineLeaseSeconds: 60}, 422},
		{"long reason", `"2"`, uuid.NewString(), "VALIDATION_FAILED", api.FleetConfigUpdate{MaxOfflineLeaseSeconds: 60, Reason: strings.Repeat("x", 501)}, 422},
		{"unknown field", `"2"`, uuid.NewString(), "VALIDATION_FAILED", map[string]any{"max_offline_lease_seconds": 60, "reason": body.Reason, "generation": "99"}, 422},
		{"missing key", `"2"`, "", "MALFORMED_REQUEST", body, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := f.call("PUT", fleetConfigPath, tc.body, nil, map[string]string{"If-Match": tc.match, "Idempotency-Key": tc.key})
			err := decode[api.ErrorEnvelope](t, w, tc.status)
			if string(err.Error.Code) != tc.code {
				t.Fatal(w.Body.String())
			}
			if tc.status == 412 && err.Error.Details["current_resource_version"] != "2" {
				t.Fatal("missing current version", err)
			}
			t.Logf("%d %s", w.Code, err.Error.Code)
		})
	}
	// Auth/CSRF are supplied by the generated composition, including on replay.
	badCSRF := map[string]string{"If-Match": `"2"`, "Idempotency-Key": headers["Idempotency-Key"], "X-CSRF-Token": "wrong"}
	decode[api.ErrorEnvelope](t, f.call("PUT", fleetConfigPath, body, nil, badCSRF), 403)
	f.p.Role = auth.Member
	decode[api.ErrorEnvelope](t, f.call("GET", fleetConfigPath, nil, nil, nil), 403)
	decode[api.ErrorEnvelope](t, f.call("PUT", fleetConfigPath, body, nil, headers), 403)
	f.p.Role = auth.Admin
	unchanged := f.call("GET", fleetConfigPath, nil, nil, nil)
	decode[api.FleetConfig](t, unchanged, 200)
	if !bytes.Equal(first.Body.Bytes(), unchanged.Body.Bytes()) {
		t.Fatal("rejected requests changed configuration")
	}
	events := fleetConfigAudit(t, f)
	if len(events.Items) != 1 {
		t.Fatalf("accepted write and replay produced %d audit rows", len(events.Items))
	}
	assertFleetConfigAudit(t, f, events.Items[0], 86400, 60, body.Reason, first.Header().Get("X-Request-ID"))
	// A same-value command is an accepted write, with both snapshots present.
	same := f.call("PUT", fleetConfigPath, body, nil, map[string]string{"If-Match": `"2"`, "Idempotency-Key": uuid.NewString()})
	if decode[api.FleetConfig](t, same, 200).ResourceVersion != 3 {
		t.Fatal("same-value write did not advance resource version")
	}
	events = fleetConfigAudit(t, f)
	if len(events.Items) != 2 {
		t.Fatal("same-value audit missing")
	}
	assertFleetConfigAudit(t, f, events.Items[0], 60, 60, body.Reason, same.Header().Get("X-Request-ID"))
	restored := f.call("PUT", fleetConfigPath, api.FleetConfigUpdate{MaxOfflineLeaseSeconds: 86400, Reason: "TEST ONLY restore default"}, nil, map[string]string{"If-Match": `"3"`, "Idempotency-Key": uuid.NewString()})
	decode[api.FleetConfig](t, restored, 200)
	replay = f.call("PUT", fleetConfigPath, body, nil, headers)
	decode[api.FleetConfig](t, replay, 200)
	if !bytes.Equal(first.Body.Bytes(), replay.Body.Bytes()) || replay.Header().Get("ETag") != `"2"` || len(fleetConfigAudit(t, f).Items) != 3 {
		t.Fatal("later writes changed original replay or replay appended audit")
	}
	var operations, outbox, incomplete int
	if err := f.r.Pool.QueryRow(context.Background(), `SELECT (SELECT count(*) FROM operations), (SELECT count(*) FROM outbox_events), (SELECT count(*) FROM idempotency_requests WHERE state='in_progress')`).Scan(&operations, &outbox, &incomplete); err != nil {
		t.Fatal(err)
	}
	if operations != 0 || outbox != 0 || incomplete != 0 {
		t.Fatal("local resource write created pending work", operations, outbox, incomplete)
	}
}

func fleetConfigAudit(t *testing.T, f *fixture) api.AuditEventList {
	t.Helper()
	w := f.call("GET", "/api/v1/admin/audit-events?resource_type=fleet_config&resource_id="+releases.FleetID, nil, nil, nil)
	return decode[api.AuditEventList](t, w, 200)
}

func assertFleetConfigAudit(t *testing.T, f *fixture, event api.AuditEvent, before, after int, reason, requestID string) {
	t.Helper()
	b, a := mustValue(t, event.Changes.Before), mustValue(t, event.Changes.After)
	if event.Action != "fleet_config.update" || event.Resource.Type != api.ResourceTypeFleetConfig || event.Resource.Id.String() != releases.FleetID || event.Result != api.AuditEventResultSucceeded || event.Actor.Type != api.AuditEventActorTypeAdmin || mustValue(t, event.Actor.Id) != f.p.ID || !event.OperationId.IsNull() || mustValue(t, event.RequestId).String() != requestID || b["max_offline_lease_seconds"] != float64(before) || a["max_offline_lease_seconds"] != float64(after) || a["reason"] != reason {
		t.Fatalf("wrong audit projection: %+v", event)
	}
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("GET audit-events 200: %s", raw)
}

func TestD065_AT29_FleetConfigOnlyChangesNewLeases(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	vault := f.r.Provider.(*signing.FileProvider)
	if _, err := vault.Rotate(ctx, signing.Lease, 48*time.Hour, time.Hour); err != nil {
		t.Fatal(err)
	}
	_, enrolled, cert := f.enroll(f.create(nil))
	r := api.LeaseRenewalRequest{RenewalRequestId: uuid.New(), NodeId: enrolled.NodeId, NodeIncarnationId: enrolled.NodeIncarnationId, CurrentLeaseId: presence.NewNullNullable[uuid.UUID](), AppliedConfigGeneration: presence.NewNullNullable[decstr.U64]()}
	key := map[string]string{"Idempotency-Key": uuid.NewString()}
	first := f.call("POST", "/api/v1/agent/lease-renewals", r, cert, key)
	issued := decode[api.ServingLeaseEnvelope](t, first, 200)
	if issued.Signed.MaxOfflineSeconds != 86400 || issued.Signed.ExpiresAt.Sub(issued.Signed.IssuedAt) != 24*time.Hour {
		t.Fatal("wrong default lease")
	}
	current := f.call("GET", fleetConfigPath, nil, nil, nil)
	decode[api.FleetConfig](t, current, 200)
	w := f.call("PUT", fleetConfigPath, api.FleetConfigUpdate{MaxOfflineLeaseSeconds: 60, Reason: "TEST ONLY AT29"}, nil, map[string]string{"If-Match": current.Header().Get("ETag"), "Idempotency-Key": uuid.NewString()})
	decode[api.FleetConfig](t, w, 200)
	replay := f.call("POST", "/api/v1/agent/lease-renewals", r, cert, key)
	decode[api.ServingLeaseEnvelope](t, replay, 200)
	if !bytes.Equal(first.Body.Bytes(), replay.Body.Bytes()) {
		t.Fatal("configuration change rewrote issued lease")
	}
	r.RenewalRequestId, r.CurrentLeaseId = uuid.New(), presence.NewNullableWithValue(issued.Signed.LeaseId)
	shortKey := map[string]string{"Idempotency-Key": uuid.NewString()}
	short := f.call("POST", "/api/v1/agent/lease-renewals", r, cert, shortKey)
	renewed := decode[api.ServingLeaseEnvelope](t, short, 200)
	if renewed.Signed.MaxOfflineSeconds != 60 || renewed.Signed.ExpiresAt.Sub(renewed.Signed.IssuedAt) != time.Minute {
		t.Fatal("renewal ignored the new ceiling", renewed.Signed)
	}
	decode[api.FleetConfig](t, f.call("PUT", fleetConfigPath, api.FleetConfigUpdate{MaxOfflineLeaseSeconds: 86400, Reason: "TEST ONLY restore after AT29"}, nil, map[string]string{"If-Match": w.Header().Get("ETag"), "Idempotency-Key": uuid.NewString()}), 200)
	replay = f.call("POST", "/api/v1/agent/lease-renewals", r, cert, shortKey)
	decode[api.ServingLeaseEnvelope](t, replay, 200)
	if !bytes.Equal(short.Body.Bytes(), replay.Body.Bytes()) {
		t.Fatal("restoring default extended the short lease")
	}
	r.RenewalRequestId, r.CurrentLeaseId = uuid.New(), presence.NewNullableWithValue(renewed.Signed.LeaseId)
	last := decode[api.ServingLeaseEnvelope](t, f.call("POST", "/api/v1/agent/lease-renewals", r, cert, nil), 200)
	if last.Signed.MaxOfflineSeconds != 86400 {
		t.Fatal("renewal did not use restored default")
	}
	t.Log("issued 86400 s; PUT 60; original replay unchanged; renewal 60 s; PUT 86400; short replay unchanged; renewal 86400 s")
}

func TestD065_FleetConfigReleaseWriterInterleaving(t *testing.T) {
	f := setup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rf := releasetest.New(t)
	f.r.Releases = &releases.Service{Pool: f.r.Pool, Validator: rf.Validator}
	var err error
	f.r.Fleet, err = fleetupdate.New(f.r.Pool, f.r.Releases, fleetupdate.Config{AuthorizationTTL: time.Hour, OnlineWindow: time.Minute, StuckWindow: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	var registration api.ReleaseRegisterRequest
	if err = json.Unmarshal(rf.Request(t, rf.Manifest), &registration); err != nil {
		t.Fatal(err)
	}
	decode[api.Release](t, f.call("POST", "/api/v1/admin/releases", registration, nil, map[string]string{"Idempotency-Key": uuid.NewString()}), 201)
	support := api.ReleaseSupportPayload{ApprovedReleaseIds: []uuid.UUID{rf.Manifest.ReleaseId}, RevokedReleaseIds: []uuid.UUID{}}
	support.MinimumServingVersion.ServingReleaseIds = support.ApprovedReleaseIds
	if err = db.InTx(ctx, f.r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return f.r.Releases.ApplySupportTx(ctx, tx, f.p.ID, support, "TEST ONLY support")
	}); err != nil {
		t.Fatal(err)
	}
	// Read-only snapshots compare every non-lease field, including release policy
	// and both generation counters, without maintaining a hand-written field list.
	snapshot := func() (string, decstr.U64) {
		t.Helper()
		var raw string
		var generation decstr.U64
		if err := f.r.Pool.QueryRow(ctx, `SELECT (to_jsonb(f)-'max_offline_lease_seconds'-'resource_version'-'updated_at')::text,generation::text FROM fleet_config f`).Scan(&raw, &generation); err != nil {
			t.Fatal(err)
		}
		return raw, generation
	}
	initial := decode[api.FleetConfig](t, f.call("GET", fleetConfigPath, nil, nil, nil), 200)
	_, generation := snapshot()
	tx, err := f.r.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var pid uint32
	if err = tx.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	batchRequest := api.UpdateBatchCreateRequest{ReleaseId: rf.Manifest.ReleaseId, Scope: "all_nodes", ExpectedGeneration: generation, SetFleetDefault: true, Reason: "TEST ONLY interleaving"}
	if _, err = f.r.Fleet.CreateBatch(commandtx.WithTransaction(ctx, tx), releases.Command{ActorID: f.p.ID, IdempotencyKey: uuid.New()}, batchRequest); err != nil {
		t.Fatal(err)
	}
	done := make(chan *httptest.ResponseRecorder, 1)
	body := api.FleetConfigUpdate{MaxOfflineLeaseSeconds: 60, Reason: "TEST ONLY concurrent config"}
	headers := map[string]string{"If-Match": `"` + initial.ResourceVersion.String() + `"`, "Idempotency-Key": uuid.NewString()}
	go func() { done <- f.call("PUT", fleetConfigPath, body, nil, headers) }()
	// Observe the actual lock wait: the PUT must compare its ETag only after the
	// real CreateBatch writer commits, rather than overwrite a concurrent version.
	for {
		var blocked bool
		if err = f.r.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND $1=ANY(pg_blocking_pids(pid)))`, pid).Scan(&blocked); err != nil {
			t.Fatal(err)
		}
		if blocked {
			break
		}
		select {
		case w := <-done:
			t.Fatalf("PUT escaped the fleet row lock: %d %s", w.Code, w.Body.String())
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var rejected *httptest.ResponseRecorder
	select {
	case rejected = <-done:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	conflict := decode[api.ErrorEnvelope](t, rejected, 412)
	if conflict.Error.Details["current_resource_version"] != (initial.ResourceVersion+1).String() || len(fleetConfigAudit(t, f).Items) != 0 {
		t.Fatal("concurrent stale PUT wrote audit or returned stale version", conflict)
	}
	afterBatch, nextGeneration := snapshot()
	if nextGeneration != generation+1 {
		t.Fatal("batch generation did not advance")
	}
	headers["If-Match"] = `"` + (initial.ResourceVersion + 1).String() + `"`
	w := f.call("PUT", fleetConfigPath, body, nil, headers)
	updated := decode[api.FleetConfig](t, w, 200)
	if updated.ResourceVersion != initial.ResourceVersion+2 || updated.MaxOfflineLeaseSeconds != 60 {
		t.Fatal("config lost fleet version", updated)
	}
	if afterConfig, _ := snapshot(); afterConfig != afterBatch {
		t.Fatal("config changed a release field", afterConfig, afterBatch)
	}
	// The reverse order also preserves the config: a subsequent real fleet write
	// advances generation/version, leaving the lease ceiling set by PUT intact.
	batchRequest.ExpectedGeneration = nextGeneration
	decode[api.UpdateBatchAccepted](t, f.call("POST", "/api/v1/admin/update-batches", batchRequest, nil, map[string]string{"Idempotency-Key": uuid.NewString()}), 202)
	latest := decode[api.FleetConfig](t, f.call("GET", fleetConfigPath, nil, nil, nil), 200)
	if latest.MaxOfflineLeaseSeconds != 60 || latest.ResourceVersion != initial.ResourceVersion+3 {
		t.Fatal("release writer lost lease setting or version", latest)
	}
	replay := f.call("PUT", fleetConfigPath, body, nil, headers)
	decode[api.FleetConfig](t, replay, 200)
	if !bytes.Equal(w.Body.Bytes(), replay.Body.Bytes()) || len(fleetConfigAudit(t, f).Items) != 1 {
		t.Fatal("release write changed original receipt")
	}
	t.Logf("blocked PUT observed CreateBatch commit: 412; retry version %s; next batch version %s; release fields and 60 s setting preserved", updated.ResourceVersion, latest.ResourceVersion)
}
