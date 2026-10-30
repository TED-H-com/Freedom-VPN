package store

import (
	"context"
	"errors"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	dnsoperation "github.com/TED-H-com/Freedom-VPN/internal/dnspublish/operation"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var errFeedUnavailable = errors.New("feed revision is inactive or outside validity")

func (p *Publisher) recordFailure(ctx context.Context, def, rev uuid.UUID, generation decstr.U64, operation string, cause error) error {
	return db.InTx(ctx, p.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(544055)`); err != nil {
			return err
		}
		var current bool
		if err := tx.QueryRow(ctx, `SELECT coalesce(active_revision_id=$2 AND generation=$3::numeric,false) FROM policy_definitions WHERE id=$1 FOR SHARE`, def, rev, generation.String()).Scan(&current); err != nil {
			return err
		}
		if !current {
			return nil
		}
		// A transient refresh failure retains last-good; an invalid approval or
		// expired feed cannot remain advertised as a valid publication.
		if errors.Is(cause, errFeedUnavailable) {
			if _, err := tx.Exec(ctx, `UPDATE dns_publication_heads SET status='withdrawn',feed_invalidated=true,resource_version=resource_version+1,updated_at=clock_timestamp() WHERE definition_id=$1 AND status='active'`, def); err != nil {
				return err
			}
		}
		reason, code := "invalid_config", api.ErrorCodeDEPENDENCYUNAVAILABLE
		var compileError *dnspublish.Error
		if errors.As(cause, &compileError) && compileError.Code == "document_too_large" {
			reason, code = "content_too_large", api.ErrorCodePAYLOADTOOLARGE
		}
		if errors.Is(cause, errFeedUnavailable) {
			reason, code = "outside_validity", api.ErrorCodeVALIDATIONFAILED
		}
		var opID uuid.UUID
		var requestID *uuid.UUID
		if operation != "" {
			var err error
			opID, err = uuid.Parse(operation)
			if err != nil {
				return err
			}
			if err := tx.QueryRow(ctx, `SELECT request_id FROM operations WHERE id=$1`, opID).Scan(&requestID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		}
		if err := dnsoperation.Record(ctx, tx, opID, dnsoperation.DocumentStep, "", &code, p.cfg.Clock().UTC()); err != nil {
			return err
		}
		for _, resource := range []api.ResourceRef{{Type: api.ResourceTypePolicyDefinition, Id: def}, {Type: api.ResourceTypePolicyRevision, Id: rev}} {
			entry := audit.Entry{Actor: audit.SystemActor, Action: "dns.publish_failed", Resource: resource, Result: api.AuditEventResultFailed, RequestID: requestID, After: map[string]any{"reason_code": reason}, Fields: []string{"reason_code"}}
			if opID != uuid.Nil {
				entry.OperationID = &opID
			}
			event, err := audit.NewEvent(entry)
			if err != nil {
				return err
			}
			if err := audit.AppendEvent(ctx, tx, event); err != nil {
				return err
			}
		}
		return nil
	})
}
