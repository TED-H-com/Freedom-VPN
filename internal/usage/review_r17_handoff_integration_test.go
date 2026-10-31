//go:build integration

package usage_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/internal/usage/usagetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

func handoffEvent(t *testing.T, ev api.UsageEvent, end time.Time, seq decstr.U64, origin uuid.UUID) api.UsageEvent {
	t.Helper()
	ev.IntervalEnd = end
	ev.EventSeq = seq
	ev.NodeIncarnationId = origin
	ev, err := usage.WithComputedHash(ev)
	if err != nil {
		t.Fatal(err)
	}
	return ev
}

func TestReviewR17_HandoffDoesNotRewriteOrExtendTheRetiredStream(t *testing.T) {
	f := newHandoffFixture(t, 10*time.Second, time.Time{})
	req := f.v.Cases[0].Request
	origin := *req.OriginIncarnationId
	first, err := f.svc.IngestBatch(t.Context(), f.id, req)
	if err != nil || first.Results[0].Status != api.UsageEventResultStatusAccepted || first.NodeIncarnationId != origin {
		t.Fatalf("seed %+v %v", first, err)
	}
	samples := handoffCount(t, f, `SELECT count(*) FROM usage_samples`)
	if samples != 2 {
		t.Fatalf("seed samples=%d", samples)
	}

	again, err := f.svc.IngestBatch(t.Context(), f.id, req)
	if err != nil || again.Results[0].Status != api.UsageEventResultStatusDuplicate {
		t.Fatalf("overlapping seq %+v %v", again, err)
	}
	if n := handoffCount(t, f, `SELECT count(*) FROM usage_samples`); n != samples {
		t.Fatalf("duplicate hand-off doubled bytes: %d", n)
	}

	rewritten := req
	rewritten.Events = append([]api.UsageEvent{}, req.Events...)
	rewritten.Events[0].Records = append([]api.UsageRecord{}, req.Events[0].Records...)
	rewritten.Events[0].Records[0].UploadBytes++
	rewritten.Events[0] = handoffEvent(t, rewritten.Events[0], rewritten.Events[0].IntervalEnd, rewritten.Events[0].EventSeq, origin)
	conflict, err := f.svc.IngestBatch(t.Context(), f.id, rewritten)
	if err != nil || conflict.Results[0].Status != api.UsageEventResultStatusConflict {
		t.Fatalf("rewrite %+v %v", conflict, err)
	}
	if n := handoffCount(t, f, `SELECT count(*) FROM usage_samples`); n != samples {
		t.Fatalf("conflicting hand-off doubled bytes: %d", n)
	}

	late := req
	late.Events = []api.UsageEvent{handoffEvent(t, req.Events[0], f.v.Context.RetiredAt.Add(10*time.Second+time.Nanosecond), req.Events[0].EventSeq, origin)}
	rejected, err := f.svc.IngestBatch(t.Context(), f.id, late)
	if err != nil || rejected.Results[0].Status != api.UsageEventResultStatusRejected {
		t.Fatalf("one quantum past the window %+v %v", rejected, err)
	}
	assertHandoffAudit(t, f, apierr.ValidationFailed, "retired_window", &req.Events[0].EventSeq)
	if n := handoffCount(t, f, `SELECT count(*) FROM usage_samples`); n != samples {
		t.Fatalf("late event was counted: %d", n)
	}
}

func TestReviewR17_OlderRetiredTailUsesThatOriginsWindow(t *testing.T) {
	f := newHandoffFixture(t, 10*time.Second, time.Time{})
	req := f.v.Cases[0].Request
	older := *req.OriginIncarnationId
	retiredAt := f.v.Context.RetiredAt
	enrolledAt := f.v.Context.EnrolledAt
	intermediate := uuid.New()
	mid := retiredAt.Add(time.Microsecond)
	revoked := enrolledAt.Add(-time.Microsecond)
	mustExec(t, f.pool, `INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,os,arch,created_at,revoked_at)
		VALUES($1,$2,decode(repeat('55',32),'hex'),'ubuntu','amd64',$3,$4)`,
		intermediate, req.NodeId, mid, revoked)

	accepted, err := f.svc.IngestBatch(t.Context(), f.id, req)
	if err != nil || accepted.Results[0].Status != api.UsageEventResultStatusAccepted || accepted.NodeIncarnationId != older {
		t.Fatalf("older tail inside its own window %+v %v", accepted, err)
	}

	late := req
	late.Events = []api.UsageEvent{handoffEvent(t, req.Events[0], retiredAt.Add(10*time.Second+time.Nanosecond), decstr.U64(1235), older)}
	got, err := f.svc.IngestBatch(t.Context(), f.id, late)
	if err != nil || got.Results[0].Status != api.UsageEventResultStatusRejected || got.NodeIncarnationId != older {
		t.Fatalf("older window was widened to the intermediate retirement %+v %v", got, err)
	}

	next := req
	next.OriginIncarnationId = &intermediate
	next.Events = []api.UsageEvent{handoffEvent(t, req.Events[0], revoked.Add(10*time.Second), 1, intermediate)}
	got, err = f.svc.IngestBatch(t.Context(), f.id, next)
	if err != nil || got.Results[0].Status != api.UsageEventResultStatusAccepted || got.NodeIncarnationId != intermediate {
		t.Fatalf("intermediate tail %+v %v", got, err)
	}
	outside := next
	outside.Events = []api.UsageEvent{handoffEvent(t, req.Events[0], revoked.Add(10*time.Second+time.Nanosecond), 2, intermediate)}
	got, err = f.svc.IngestBatch(t.Context(), f.id, outside)
	if err != nil || got.Results[0].Status != api.UsageEventResultStatusRejected {
		t.Fatalf("intermediate window %+v %v", got, err)
	}
}

func TestReviewR17_HandoffOriginIdentityAndIdempotency(t *testing.T) {
	f := newHandoffFixture(t, 10*time.Second, time.Time{})
	req := f.v.Cases[0].Request
	same := req
	same.OriginIncarnationId = &req.NodeIncarnationId
	same.Events = []api.UsageEvent{handoffEvent(t, req.Events[0], req.Events[0].IntervalEnd, req.Events[0].EventSeq, req.NodeIncarnationId)}
	_, err := f.svc.IngestBatch(t.Context(), f.id, same)
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != apierr.StateConflict {
		t.Fatalf("origin equal to presenter: %v", err)
	}
	assertHandoffAudit(t, f, apierr.StateConflict, "origin_ineligible", nil)

	live := req
	live.OriginIncarnationId = nil
	_, err = f.svc.IngestBatch(t.Context(), f.id, live)
	if !errors.As(err, &ae) || ae.Code != apierr.IncarnationMismatch {
		t.Fatalf("absent origin selected a retired stream: %v", err)
	}

	key := uuid.NewString()
	h := newHandoffHTTP(t, f, nil)
	first := postHandoff(t, h, req, key, nil)
	if first.Code != http.StatusOK {
		t.Fatalf("first %d %s", first.Code, first.Body.String())
	}
	otherOrigin := uuid.New()
	changed := req
	changed.OriginIncarnationId = &otherOrigin
	second := postHandoff(t, h, changed, key, nil)
	if second.Code != http.StatusConflict || !json.Valid(second.Body.Bytes()) {
		t.Fatalf("same key different origin: %d %s", second.Code, second.Body.String())
	}
	var env api.ErrorEnvelope
	if err = json.Unmarshal(second.Body.Bytes(), &env); err != nil || env.Error.Code != api.ErrorCodeIDEMPOTENCYCONFLICT {
		t.Fatalf("idempotency body %s %v", second.Body.String(), err)
	}
	if n := handoffCount(t, f, `SELECT count(*) FROM usage_samples`); n != 2 {
		t.Fatalf("conflicting key wrote samples: %d", n)
	}
}

func TestReviewR17_HandoffBoundIsTheLiveServiceConfig(t *testing.T) {
	f := newHandoffFixture(t, 10*time.Second, time.Time{})
	req := f.v.Cases[0].Request
	origin := *req.OriginIncarnationId
	req.Events = []api.UsageEvent{handoffEvent(t, req.Events[0], f.v.Context.RetiredAt.Add(11*time.Second), req.Events[0].EventSeq, origin)}
	got, err := f.svc.IngestBatch(t.Context(), f.id, req)
	if err != nil || got.Results[0].Status != api.UsageEventResultStatusRejected {
		t.Fatalf("11s tail on a 10s service %+v %v", got, err)
	}
	u := usagetest.New(t)
	signer := usage.NewProviderSigner(u.Provider)
	wider, err := usage.NewService(usage.Config{
		Pool: f.pool, Clock: func() time.Time { return f.v.Cases[0].Response.ReceivedAt },
		Signer: signer, LedgerEpochSigner: signer, SamplingIntervalBound: 30 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err = wider.IngestBatch(t.Context(), f.id, req)
	if err != nil || got.Results[0].Status != api.UsageEventResultStatusAccepted || got.NodeIncarnationId != origin {
		t.Fatalf("TODO(review): raising the bound between uploads did not accept the same tail (%+v %v)", got, err)
	}
	if _, err = usage.NewService(usage.Config{Pool: f.pool, SamplingIntervalBound: -time.Second}); err == nil {
		t.Fatal("negative bound was accepted by the service")
	}
	zero, err := usage.NewService(usage.Config{Pool: f.pool, SamplingIntervalBound: 0, Signer: signer, LedgerEpochSigner: signer, Clock: func() time.Time { return f.v.Cases[0].Response.ReceivedAt }})
	if err != nil {
		t.Fatal(err)
	}
	early := req
	early.Events = []api.UsageEvent{handoffEvent(t, req.Events[0], f.v.Context.RetiredAt.Add(10*time.Second+time.Nanosecond), decstr.U64(1235), origin)}
	got, err = zero.IngestBatch(t.Context(), f.id, early)
	if err != nil || got.Results[0].Status != api.UsageEventResultStatusRejected {
		t.Fatalf("zero bound did not select the 10s default %+v %v", got, err)
	}
}

func TestReviewR17_HandoffRejectionsAreAuditedAndBounded(t *testing.T) {
	f := newHandoffFixture(t, 10*time.Second, time.Time{})
	req := f.v.Cases[0].Request
	origin := *req.OriginIncarnationId
	end := f.v.Context.RetiredAt.Add(10*time.Second + time.Nanosecond)
	req.Events = nil
	for seq := decstr.U64(1234); seq <= 1236; seq++ {
		req.Events = append(req.Events, handoffEvent(t, f.v.Cases[0].Request.Events[0], end, seq, origin))
	}
	requestID := uuid.New()
	ctx := apierr.ContextWithRequestID(t.Context(), requestID.String())
	got, err := f.svc.IngestBatch(ctx, f.id, req)
	if err != nil || len(got.Results) != 3 {
		t.Fatal(got, err)
	}
	for _, result := range got.Results {
		if result.Status != api.UsageEventResultStatusRejected {
			t.Fatal(result)
		}
	}
	var audits int
	if err = f.pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_events WHERE action='usage.ingest_rejected' AND request_id=$1`, requestID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 3 {
		t.Fatalf("audits=%d want one per rejected event", audits)
	}
	var after []byte
	if err = f.pool.QueryRow(t.Context(), `SELECT after_diff FROM audit_events WHERE action='usage.ingest_rejected' AND request_id=$1 LIMIT 1`, requestID).Scan(&after); err != nil {
		t.Fatal(err)
	}
	var metadata map[string]any
	if err = json.Unmarshal(after, &metadata); err != nil {
		t.Fatal(err)
	}
	for key := range metadata {
		switch key {
		case "error_code", "reason", "event_seq", "presenting_incarnation_id", "origin_incarnation_id":
		default:
			t.Fatalf("audit retained %s: %s", key, after)
		}
	}
	if n := handoffCount(t, f, `SELECT count(*) FROM usage_samples`); n != 0 {
		t.Fatalf("rejected batch wrote samples: %d", n)
	}
}
