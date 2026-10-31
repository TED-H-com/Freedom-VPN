package usage

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// batchStream separates the authenticated presenter from the immutable stream.
// Only explicit origin_incarnation_id selects a retired stream (D-105).
type batchStream struct {
	identity         NodeIdentity
	presenter        NodeIdentity
	intervalEndLimit *time.Time
}

func (s *Service) selectBatchStream(ctx context.Context, presenter NodeIdentity, batch api.UsageBatchRequest) (batchStream, error) {
	reject := func(code apierr.Code, reason, message string) (batchStream, error) {
		// This transaction must commit even when the HTTP command transaction
		// rolls back its rejected request. No usage ledger rows are written.
		if err := db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
			return auditIngestRejectionInTx(ctx, tx, presenter, batch.OriginIncarnationId, code, reason, nil)
		}); err != nil {
			return batchStream{}, apierr.Wrap(err, apierr.DependencyUnavailable, "usage rejection audit unavailable")
		}
		return batchStream{}, apierr.New(code, message)
	}
	if batch.NodeId != presenter.NodeID || batch.NodeIncarnationId != presenter.NodeIncarnationID {
		return reject(apierr.IncarnationMismatch, "batch_identity", "batch identity does not match mTLS certificate identity")
	}
	if err := s.checkNodeIncarnation(ctx, presenter.NodeID, presenter.NodeIncarnationID); err != nil {
		var ae *apierr.Error
		if errors.As(err, &ae) && (ae.Code == apierr.IncarnationMismatch || ae.Code == apierr.IdentityRevoked || ae.Code == apierr.Forbidden) {
			return reject(ae.Code, "presenter_inactive", "presenting incarnation is not current and active")
		}
		return batchStream{}, err
	}
	stream := batchStream{identity: presenter, presenter: presenter}
	if batch.OriginIncarnationId != nil {
		var retiredAt time.Time
		err := s.pool.QueryRow(ctx, `
			SELECT origin.revoked_at
			FROM nodes n
			JOIN node_incarnations presenting
			  ON presenting.node_id = n.id AND presenting.id = n.current_incarnation_id
			JOIN node_incarnations origin ON origin.node_id = n.id AND origin.id = $3
			WHERE n.id = $1 AND presenting.id = $2
			  AND n.lifecycle_state NOT IN ('revoked', 'retired')
			  AND presenting.revoked_at IS NULL AND origin.revoked_at IS NOT NULL
			  AND presenting.created_at > origin.revoked_at
		`, presenter.NodeID, presenter.NodeIncarnationID, *batch.OriginIncarnationId).Scan(&retiredAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return reject(apierr.StateConflict, "origin_ineligible", "origin incarnation is not eligible for retired-stream hand-off")
		}
		if err != nil {
			return batchStream{}, apierr.Wrap(err, apierr.DependencyUnavailable, "usage incarnation records unavailable")
		}
		limit := retiredAt.Add(s.samplingIntervalBound)
		stream.identity.NodeIncarnationID = *batch.OriginIncarnationId
		stream.intervalEndLimit = &limit
	}
	for _, ev := range batch.Events {
		if ev.NodeId != stream.identity.NodeID || ev.NodeIncarnationId != stream.identity.NodeIncarnationID {
			return reject(apierr.IncarnationMismatch, "event_identity", "event identity does not match the selected usage stream")
		}
	}
	return stream, nil
}

// auditIngestRejectionInTx retains bounded metadata only, never an event body.
func auditIngestRejectionInTx(ctx context.Context, tx pgx.Tx, presenter NodeIdentity, origin *uuid.UUID, code apierr.Code, reason string, seq *decstr.U64) error {
	after := map[string]any{
		"presenting_incarnation_id": presenter.NodeIncarnationID.String(),
		"error_code":                string(code),
		"reason":                    reason,
	}
	if origin != nil {
		after["origin_incarnation_id"] = origin.String()
	}
	if seq != nil {
		after["event_seq"] = seq.String()
	}
	var requestID *uuid.UUID
	if raw, ok := apierr.RequestIDFromContext(ctx); ok {
		if id, err := uuid.Parse(raw); err == nil && id != uuid.Nil {
			requestID = &id
		}
	}
	event, err := audit.NewEvent(audit.Entry{
		Actor:     audit.Actor{Type: api.AuditEventActorTypeNode, ID: presenter.NodeID},
		Action:    "usage.ingest_rejected",
		Resource:  api.ResourceRef{Type: api.ResourceTypeNode, Id: presenter.NodeID},
		Result:    api.AuditEventResultFailed,
		RequestID: requestID,
		After:     after,
		Fields:    []string{"presenting_incarnation_id", "origin_incarnation_id", "error_code", "reason", "event_seq"},
	})
	if err != nil {
		return err
	}
	return audit.AppendEvent(ctx, tx, event)
}
