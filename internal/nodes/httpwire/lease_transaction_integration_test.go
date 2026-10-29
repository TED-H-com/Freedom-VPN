//go:build integration

package httpwire_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type leaseProbe struct {
	*httpwire.Handlers
	after func(context.Context) error
}

func (h *leaseProbe) RenewServingLease(ctx context.Context, r gen.RenewServingLeaseRequestObject) (gen.RenewServingLeaseResponseObject, error) {
	response, err := h.Handlers.RenewServingLease(ctx, r)
	if err != nil {
		return nil, err
	}
	if err = h.after(ctx); err != nil {
		return nil, err
	}
	return response, nil
}

func TestAT29_HTTPKeyedLeaseHoldsShareLocksAndRollsBack(t *testing.T) {
	f, enrolled, cert, member, _ := servingFixture(t)
	rollback := true
	probe := &leaseProbe{Handlers: &httpwire.Handlers{Runtime: f.r}}
	probe.after = func(ctx context.Context) error {
		tx, err := httpapi.Transaction(ctx)
		if err != nil {
			return err
		}
		var count int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM serving_leases WHERE node_id=$1`, enrolled.NodeId).Scan(&count); err != nil || count != 1 {
			t.Errorf("lease absent inside command tx: %d %v", count, err)
		}
		if err = f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM serving_leases WHERE node_id=$1`, enrolled.NodeId).Scan(&count); err != nil || count != 0 {
			t.Errorf("lease committed ahead of HTTP result: %d %v", count, err)
		}
		// These suspension/revocation updates require NO KEY UPDATE row locks.
		// A foreign-key KEY SHARE alone would not block them; RenewTx's FOR SHARE must.
		for _, attempt := range []struct {
			query string
			id    uuid.UUID
		}{
			{`UPDATE nodes SET lifecycle_state='revoked',resource_version=resource_version+1 WHERE id=$1`, enrolled.NodeId},
			{`UPDATE members SET admin_suspended=true,resource_version=resource_version+1 WHERE id=$1`, member},
			{`UPDATE entitlements SET admin_suspended=true WHERE member_id=$1`, member},
		} {
			other, err := f.r.Pool.Begin(ctx)
			if err != nil {
				return err
			}
			if _, err = other.Exec(ctx, `SET LOCAL lock_timeout='50ms'`); err != nil {
				_ = other.Rollback(ctx)
				return err
			}
			_, err = other.Exec(ctx, attempt.query, attempt.id)
			var pgerr *pgconn.PgError
			if !errors.As(err, &pgerr) || pgerr.Code != "55P03" {
				t.Errorf("concurrent revocation not blocked by lease locks: %v", err)
			}
			_ = other.Rollback(ctx)
		}
		if rollback {
			return errors.New("TEST ONLY fail before command completion")
		}
		return nil
	}
	composite, err := httpapi.Compose(probe)
	if err != nil {
		t.Fatal(err)
	}
	c := config.Defaults()
	h, err := httpapi.New(composite, httpapi.Options{Config: c, Agents: f.r.PKI, AgentListener: true, Commands: &httpapi.Commands{Pool: f.r.Pool, Retention: time.Hour}})
	if err != nil {
		t.Fatal(err)
	}
	f.agent = httpapi.Transport(nil, httpwire.RequestContext(h))
	body := api.LeaseRenewalRequest{RenewalRequestId: uuid.New(), NodeId: enrolled.NodeId, NodeIncarnationId: enrolled.NodeIncarnationId, CurrentLeaseId: presence.NewNullNullable[uuid.UUID](), AppliedConfigGeneration: presence.NewNullNullable[decstr.U64]()}
	headers := map[string]string{"Idempotency-Key": uuid.NewString()}
	if w := f.call("POST", "/api/v1/agent/lease-renewals", body, cert, headers); w.Code != 500 {
		t.Fatalf("injected rollback: %d %s", w.Code, w.Body.String())
	}
	var count int
	for _, query := range []string{`SELECT count(*) FROM serving_leases`, `SELECT count(*) FROM serving_lease_members`, `SELECT count(*) FROM audit_events WHERE action='node.lease_renewed'`, `SELECT count(*) FROM idempotency_requests WHERE route LIKE '%lease-renewals%'`} {
		if err = f.r.Pool.QueryRow(t.Context(), query).Scan(&count); err != nil || count != 0 {
			t.Fatalf("partial lease/receipt commit: %d %v", count, err)
		}
	}
	rollback = false
	out := decode[api.ServingLeaseEnvelope](t, f.call("POST", "/api/v1/agent/lease-renewals", body, cert, headers), 200)
	if len(out.Signed.Members) != 1 || out.Signed.Members[0].MemberId != member {
		t.Fatal("D-030 member grant missing")
	}
	// Once the command commits, revocation proceeds and a fresh lease omits the member.
	f.sql(`UPDATE members SET admin_suspended=true,resource_version=resource_version+1 WHERE id=$1`, member)
	if err = f.r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM idempotency_requests WHERE route LIKE '%lease-renewals%' AND response_status=200 AND operation_id IS NULL`).Scan(&count); err != nil || count != 1 {
		t.Fatal("lease command receipt missing", count, err)
	}
}
