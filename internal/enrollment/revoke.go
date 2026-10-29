package enrollment

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RevocationResult is pending external withdrawal, never completed based on a
// successful local commit (§5.2). Adapters fetch the normal operation resource.
type RevocationResult struct {
	OperationID     uuid.UUID
	ResourceVersion decstr.U64
}

func (s *Service) RevokeNode(ctx context.Context, cmd Command, node uuid.UUID, expected *decstr.U64, reason string) (out RevocationResult, err error) {
	if expected == nil {
		return out, apierr.New(apierr.PreconditionRequired, "If-Match required.")
	}
	if !validReason(reason) {
		return out, apierr.New(apierr.ValidationFailed, "Revocation reason required.")
	}
	body := struct {
		Version decstr.U64 `json:"expected_version"`
		Reason  string     `json:"reason"`
	}{*expected, reason}
	err = commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		record, e := begin(ctx, tx, cmd, "/api/v1/admin/nodes/"+node.String()+"/revoke", body)
		if e != nil {
			return e
		}
		var version string
		var incarnation *uuid.UUID
		var state string
		if e = tx.QueryRow(ctx, `SELECT resource_version::text,current_incarnation_id,lifecycle_state FROM nodes WHERE id=$1 FOR UPDATE`, node).Scan(&version, &incarnation, &state); e != nil {
			return e
		}
		out.ResourceVersion, e = decstr.Parse(version)
		if e != nil {
			return e
		}
		if !record.New {
			if record.OperationID == nil {
				return errors.New("missing revocation operation")
			}
			out.OperationID = *record.OperationID
			return nil
		}
		if out.ResourceVersion != *expected {
			return apierr.New(apierr.ResourceVersionConflict, "Node version changed.").WithDetail("current_resource_version", version)
		}
		if state == "revoked" || state == "retired" {
			return apierr.New(apierr.StateConflict, "Node already revoked or retired.")
		}
		// This is a local, durable revocation journal, not a remote KMS call.
		// Persist before commit so an old DB restore cannot undo revocation.
		// If the DB commit fails, the journal deliberately remains fail closed.
		if incarnation != nil {
			if e = s.fence.DenyIdentity(ctx, incarnation.String()); e != nil {
				return e
			}
		}
		if _, e = tx.Exec(ctx, `UPDATE nodes SET lifecycle_state='revoked',resource_version=resource_version+1 WHERE id=$1`, node); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `UPDATE node_incarnations SET revoked_at=coalesce(revoked_at,clock_timestamp()) WHERE node_id=$1`, node); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `UPDATE node_certificates SET revoked_at=coalesce(revoked_at,clock_timestamp()) WHERE incarnation_id IN (SELECT id FROM node_incarnations WHERE node_id=$1)`, node); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `UPDATE node_enrollments SET state='revoked',revoked_at=clock_timestamp() WHERE node_id=$1 AND state='pending'`, node); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `UPDATE node_bootstrap_secrets SET ciphertext=NULL,encryption_key_id=NULL,erased_at=coalesce(erased_at,clock_timestamp()) WHERE incarnation_id IN (SELECT id FROM node_incarnations WHERE node_id=$1)`, node); e != nil {
			return e
		}
		out.OperationID, e = uuid.NewV7()
		if e != nil {
			return e
		}
		out.ResourceVersion, e = out.ResourceVersion.Add(1)
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO operations(id,kind,principal,resource_type,resource_id,steps) VALUES($1,'node.revoke',$2,'node',$3,'{"route_withdrawal":{"required":true,"status":"pending"},"peer_withdrawal":{"required":true,"status":"pending"}}')`, out.OperationID, cmd.Principal, node); e != nil {
			return e
		}
		for _, event := range []string{"node.routes.withdraw", "node.peer.withdraw"} {
			payload, e := canonjson.Marshal(map[string]any{"schema_version": 1, "node_id": node, "incarnation_id": incarnation, "operation_id": out.OperationID})
			if e != nil {
				return e
			}
			if e = db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: node, AggregateType: "node", AggregateVersion: out.ResourceVersion.String(), EventType: event, SchemaVersion: 1, Payload: payload}); e != nil {
				return e
			}
		}
		if e = complete(ctx, tx, record, "node", node, 202, &out.OperationID); e != nil {
			return e
		}
		return auditReason(ctx, tx, cmd.Principal, "node.revoked", node, &out.OperationID, reason)
	})
	if err != nil {
		return RevocationResult{}, safe(err)
	}
	s.log(ctx, "node identity revoked; withdrawal pending", node)
	return out, nil
}

// ObserveEndpoint receives an opaque, authenticated compatibility-domain value
// from the adapter, never raw headers. Hashes avoid persisting endpoint lists.
// A changed endpoint within activeWindow freezes automatic mapping forever;
// incompatible activity is evidence requiring operator reconciliation (§6.2).
// Returns whether mapping updates are allowed. It does not grant forwarding.
func (s *Service) ObserveEndpoint(ctx context.Context, cert *x509.Certificate, endpoint string, activeWindow time.Duration) (allowed bool, err error) {
	if !validIdentifier(endpoint) || len(endpoint) > 1024 || activeWindow <= 0 {
		return false, apierr.New(apierr.ValidationFailed, "Endpoint observation invalid.")
	}
	id, err := s.auth.Authenticate(ctx, cert)
	if err != nil {
		return false, err
	}
	hash := sha256.Sum256([]byte(endpoint))
	err = commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if _, e := s.auth.LockAuthenticated(ctx, tx, cert); e != nil {
			return e
		}
		var prev []byte
		var active, frozen bool
		e := tx.QueryRow(ctx, `SELECT endpoint_hash,last_seen_at>clock_timestamp()-$2::bigint*interval '1 millisecond',mapping_frozen FROM node_identity_endpoints WHERE incarnation_id=$1 FOR UPDATE`, id.IncarnationID, activeWindow.Milliseconds()).Scan(&prev, &active, &frozen)
		if errors.Is(e, pgx.ErrNoRows) {
			_, e = tx.Exec(ctx, `INSERT INTO node_identity_endpoints(incarnation_id,endpoint_hash,last_seen_at) VALUES($1,$2,clock_timestamp())`, id.IncarnationID, hash[:])
			allowed = true
			return e
		}
		if e != nil {
			return e
		}
		if frozen {
			allowed = false
			return nil
		}
		if active && string(prev) != string(hash[:]) {
			conflict, e := uuid.NewV7()
			if e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO node_identity_conflicts(id,incarnation_id,previous_hash,conflicting_hash,observed_at) VALUES($1,$2,$3,$4,clock_timestamp()) ON CONFLICT DO NOTHING`, conflict, id.IncarnationID, prev, hash[:]); e != nil {
				return e
			}
			_, e = tx.Exec(ctx, `UPDATE node_identity_endpoints SET mapping_frozen=true WHERE incarnation_id=$1`, id.IncarnationID)
			allowed = false
			return e
		}
		_, e = tx.Exec(ctx, `UPDATE node_identity_endpoints SET endpoint_hash=$2,last_seen_at=clock_timestamp() WHERE incarnation_id=$1`, id.IncarnationID, hash[:])
		allowed = true
		return e
	})
	return allowed, safe(err)
}
