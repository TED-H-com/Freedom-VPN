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
		e := h.Runtime.Pool.QueryRow(ctx, `SELECT received_at FROM node_observations WHERE node_id=$1 AND incarnation_id=$2 AND report_id=$3`, p.NodeID, p.IncarnationID, body.ReportId).Scan(&out.ReceivedAt)
		out.ReceivedAt = out.ReceivedAt.UTC()
		return gen.SubmitObservation202JSONResponse{Body: out}, e
	case "SubmitUpdateResult":
		var body api.UpdateResult
		if e := json.Unmarshal(httpapi.RawBody(ctx), &body); e != nil {
			return nil, e
		}
		var raw []byte
		if e := h.Runtime.Pool.QueryRow(ctx, `SELECT ack FROM update_results WHERE node_id=$1 AND incarnation_id=$2 AND result_id=$3`, p.NodeID, p.IncarnationID, body.ResultId).Scan(&raw); e != nil {
			return nil, e
		}
		var out api.UpdateResultAck
		e := json.Unmarshal(raw, &out)
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
