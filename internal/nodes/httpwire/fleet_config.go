package httpwire

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/internal/leases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func fleetConfig(row pgx.Row) (api.FleetConfig, error) {
	out := api.FleetConfig{DefaultMaxOfflineLeaseSeconds: ptr(api.FleetConfigDefaultMaxOfflineLeaseSecondsN86400)}
	err := row.Scan(&out.MaxOfflineLeaseSeconds, &out.ResourceVersion, &out.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		// D-065: match the issuer's missing-row default without making a GET write.
		// Version zero and the Unix epoch denote an as-yet unpersisted singleton.
		out.MaxOfflineLeaseSeconds = leases.DefaultMaxOfflineLeaseSeconds
		out.UpdatedAt = time.Unix(0, 0).UTC()
		return out, nil
	}
	out.UpdatedAt = out.UpdatedAt.UTC()
	return out, err
}

func (h *Handlers) GetFleetConfig(ctx context.Context, _ gen.GetFleetConfigRequestObject) (gen.GetFleetConfigResponseObject, error) {
	if err := h.ready(ctx); err != nil {
		return nil, err
	}
	out, err := fleetConfig(h.Runtime.Pool.QueryRow(ctx, `SELECT max_offline_lease_seconds,resource_version::text,updated_at FROM fleet_config WHERE id=$1`, releases.FleetID))
	return gen.GetFleetConfig200JSONResponse{Body: out, Headers: gen.GetFleetConfig200ResponseHeaders{ETag: ptr(httpapi.ETag(out.ResourceVersion))}}, err
}

func (h *Handlers) UpdateFleetConfig(ctx context.Context, r gen.UpdateFleetConfigRequestObject) (gen.UpdateFleetConfigResponseObject, error) {
	if err := h.ready(ctx); err != nil {
		return nil, err
	}
	tx, err := httpapi.Transaction(ctx)
	if err != nil {
		return nil, err
	}
	record, ok := httpapi.CommandRecord(ctx)
	if !ok {
		return nil, errors.New("fleet configuration command receipt unavailable")
	}
	// Missing-row recovery remains fail closed for releases (table defaults).
	// Conflict handling and the row lock serialize with every release-target writer;
	// this path never writes generation or release-support columns (§14.5, D-065).
	if _, err = tx.Exec(ctx, `INSERT INTO fleet_config(id,resource_version) VALUES($1,0) ON CONFLICT(id) DO NOTHING`, releases.FleetID); err != nil {
		return nil, err
	}
	before, err := fleetConfig(tx.QueryRow(ctx, `SELECT max_offline_lease_seconds,resource_version::text,updated_at FROM fleet_config WHERE id=$1 FOR UPDATE`, releases.FleetID))
	if err != nil {
		return nil, err
	}
	if err = httpapi.CheckIfMatch(r.Params.IfMatch, before.ResourceVersion); err != nil {
		return nil, err
	}
	out, err := fleetConfig(tx.QueryRow(ctx, `UPDATE fleet_config SET max_offline_lease_seconds=$2,resource_version=resource_version+1,updated_at=clock_timestamp() WHERE id=$1 RETURNING max_offline_lease_seconds,resource_version::text,updated_at`, releases.FleetID, r.Body.MaxOfflineLeaseSeconds))
	if err != nil {
		return nil, err
	}
	// D-065 requires both snapshots even for a same-value accepted write. Use
	// the core allowlist/redaction separately on each side, rather than dropping
	// unchanged fields. The immutable audit doubles as the nonsecret replay
	// receipt, keyed by the framework reservation ID (never the plaintext key).
	_, beforeJSON, err := audit.Diff(nil, map[string]any{"max_offline_lease_seconds": before.MaxOfflineLeaseSeconds}, "max_offline_lease_seconds")
	if err != nil {
		return nil, err
	}
	_, afterJSON, err := audit.Diff(nil, map[string]any{
		"max_offline_lease_seconds": out.MaxOfflineLeaseSeconds,
		"resource_version":          out.ResourceVersion, "updated_at": out.UpdatedAt, "reason": r.Body.Reason,
	}, "max_offline_lease_seconds", "resource_version", "updated_at", "reason")
	if err != nil {
		return nil, err
	}
	if err = db.AppendAudit(ctx, tx, db.AuditEvent{
		ID: record.ID, Actor: principal(ctx).ID.String(), ActorType: string(api.AuditEventActorTypeAdmin),
		Action: "fleet_config.update", Resource: "fleet_config:" + releases.FleetID,
		Result: string(api.AuditEventResultSucceeded), RequestID: requestID(ctx), SchemaVersion: 1,
		Before: beforeJSON, After: afterJSON,
	}); err != nil {
		return nil, err
	}
	if err = httpapi.SetResult(ctx, db.IdempotencyResult{ResourceType: string(api.ResourceTypeFleetConfig), ResourceID: ptr(uuid.MustParse(releases.FleetID)), ResponseStatus: 200}); err != nil {
		return nil, err
	}
	return gen.UpdateFleetConfig200JSONResponse{Body: out, Headers: gen.UpdateFleetConfig200ResponseHeaders{ETag: ptr(httpapi.ETag(out.ResourceVersion))}}, nil
}

func (h *Handlers) replayFleetConfig(ctx context.Context, ref db.IdempotencyRecord) (any, error) {
	if value(ref.ResourceType) != string(api.ResourceTypeFleetConfig) || value(ref.ResourceID).String() != releases.FleetID || value(ref.ResponseStatus) != 200 || ref.OperationID != nil {
		return nil, errors.New("invalid fleet configuration receipt")
	}
	var raw []byte
	err := h.Runtime.Pool.QueryRow(ctx, `SELECT after_diff FROM audit_events WHERE id=$1 AND actor=$2 AND actor_type='admin' AND action='fleet_config.update' AND resource=$3 AND result='succeeded' AND operation_id IS NULL`, ref.ID, principal(ctx).ID.String(), "fleet_config:"+releases.FleetID).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var out api.FleetConfig
	if err = json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	out.DefaultMaxOfflineLeaseSeconds = ptr(api.FleetConfigDefaultMaxOfflineLeaseSecondsN86400)
	return gen.UpdateFleetConfig200JSONResponse{Body: out, Headers: gen.UpdateFleetConfig200ResponseHeaders{ETag: ptr(httpapi.ETag(out.ResourceVersion))}}, nil
}
