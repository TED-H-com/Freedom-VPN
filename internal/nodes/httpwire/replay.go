package httpwire

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/internal/leases"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/jackc/pgx/v5"
)

// HTTP replay reads the original immutable receipt; it never re-runs a domain
// command or restarts a lease's validity window (§14.5, §16.3).
func (h *Handlers) replayAgentReceipt(ctx context.Context, name string, ref db.IdempotencyRecord) (any, error) {
	p := principal(ctx)
	if ref.ResourceID == nil || *ref.ResourceID != p.NodeID || value(ref.ResourceType) != "node" {
		return nil, errors.New("invalid node receipt reference")
	}
	switch name {
	case "SubmitObservation":
		var body api.ObservationReport
		if e := json.Unmarshal(httpapi.RawBody(ctx), &body); e != nil {
			return nil, e
		}
		out := api.ObservationAccepted{Accepted: api.ObservationAcceptedAcceptedTrue, ReportId: body.ReportId}
		// D-088: replay the immutable receipt with the live freeze annotation.
		// Read both in one transaction/snapshot; never replay endpoint effects.
		e := db.InTx(ctx, h.Runtime.Pool, db.TxOptions{AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
			var frozen bool
			if err := tx.QueryRow(ctx, `SELECT o.received_at, coalesce(e.mapping_frozen,false)
				FROM node_observations o JOIN nodes n ON n.id=o.node_id
				LEFT JOIN node_identity_endpoints e ON e.incarnation_id=o.incarnation_id AND e.incarnation_id=n.current_incarnation_id
				WHERE o.node_id=$1 AND o.incarnation_id=$2 AND o.report_id=$3`, p.NodeID, p.IncarnationID, body.ReportId).Scan(&out.ReceivedAt, &frozen); err != nil {
				return err
			}
			out.IdentityFrozen = nil
			if frozen {
				out.IdentityFrozen = &frozen
			}
			return nil
		})
		out.ReceivedAt = out.ReceivedAt.UTC()
		return gen.SubmitObservation202JSONResponse{Body: out}, e
	case "SubmitUpdateResult":
		var body api.UpdateResult
		if e := json.Unmarshal(httpapi.RawBody(ctx), &body); e != nil {
			return nil, e
		}
		var raw []byte
		var e error
		if body.Source != nil && *body.Source == api.UpdateResultSourceInstaller {
			// The command middleware binds the original request hash. Installer
			// receipts are keyed by job, so also bind that receipt's result/hash;
			// replay must not select different evidence for the same node.
			e = h.Runtime.Pool.QueryRow(ctx, `SELECT ack FROM installer_update_results
				WHERE node_id=$1 AND incarnation_id=$2 AND job_id=$3
				AND (payload->>'result_id')::uuid=$4 AND payload_hash=decode($5,'hex')`,
				p.NodeID, p.IncarnationID, body.JobId, body.ResultId, body.PayloadHash).Scan(&raw)
		} else {
			e = h.Runtime.Pool.QueryRow(ctx, `SELECT ack FROM update_results WHERE node_id=$1 AND incarnation_id=$2 AND result_id=$3`, p.NodeID, p.IncarnationID, body.ResultId).Scan(&raw)
		}
		if e != nil {
			return nil, e
		}
		var out api.UpdateResultAck
		e = json.Unmarshal(raw, &out)
		return gen.SubmitUpdateResult202JSONResponse{Body: out}, e
	case "RenewServingLease":
		var body api.LeaseRenewalRequest
		if e := json.Unmarshal(httpapi.RawBody(ctx), &body); e != nil {
			return nil, e
		}
		out, e := leases.NewService(h.Runtime.Pool, nil).ReadRenewal(ctx, p.NodeID, p.IncarnationID, body.RenewalRequestId)
		return gen.RenewServingLease200JSONResponse{Body: out}, e
	}
	return nil, errors.New("unsupported node receipt")
}
