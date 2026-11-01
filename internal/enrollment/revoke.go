package enrollment

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
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

func (s *Service) RevokeNode(ctx context.Context, cmd Command, node uuid.UUID, expected *decstr.U64, reason string) (RevocationResult, error) {
	return s.revokeNode(ctx, cmd, node, expected, reason, false)
}
func (s *Service) RetireNode(ctx context.Context, cmd Command, node uuid.UUID, expected *decstr.U64, reason string) (RevocationResult, error) {
	return s.revokeNode(ctx, cmd, node, expected, reason, true)
}
func (s *Service) revokeNode(ctx context.Context, cmd Command, node uuid.UUID, expected *decstr.U64, reason string, retire bool) (out RevocationResult, err error) {
	ctx = requestContext(ctx)
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
		path := "/api/v1/admin/nodes/" + node.String()
		if !retire {
			path += "/revoke"
		}
		record, e := begin(ctx, tx, cmd, path, body)
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
		if state == "retired" || state == "revoked" && !retire {
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
		state = "revoked"
		kind := api.OperationKindNodeRevoke
		if retire {
			state = "retired"
			kind = api.OperationKindNodeRetire
		}
		var generation decstr.U64
		if e = tx.QueryRow(ctx, `UPDATE nodes SET lifecycle_state=$2,retired_at=CASE WHEN $2='retired' THEN clock_timestamp() ELSE retired_at END,desired_config_generation=desired_config_generation+1,resource_version=resource_version+1 WHERE id=$1 RETURNING desired_config_generation::text`, node, state).Scan(&generation); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `UPDATE exits SET publication_intent='disabled',resource_version=resource_version+1 WHERE node_id=$1`, node); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `UPDATE node_incarnations SET revoked_at=coalesce(revoked_at,clock_timestamp()) WHERE node_id=$1`, node); e != nil {
			return e
		}
		if e = nodes.CloseInactiveGenerationHoldsTx(ctx, tx, node, time.Now().UTC()); e != nil {
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
		out.ResourceVersion, e = out.ResourceVersion.Add(1)
		if e != nil {
			return e
		}
		actor, e := adminPrincipal(cmd.Principal)
		if e != nil {
			return e
		}
		steps := map[string]operations.Step{}
		for name, step := range nodes.NewNodeOperationSteps(string(kind), time.Now().UTC()) {
			steps[name] = operations.Step{Required: step.Required, Reason: step.Reason}
		}
		// §5.2: identity withdrawal is required in addition to route withdrawal;
		// a missing node ACK cannot hide unconfirmed setup keys or remote peers.
		steps[nodes.StepNetBirdIdentityCleanup] = operations.Step{Required: true}
		out.OperationID, e = operations.Create(ctx, tx, operations.CreateRequest{Kind: kind, Principal: actor, Resource: api.ResourceRef{Type: api.ResourceTypeNode, Id: node}, RequestID: commandtx.RequestID(ctx), ResourceVersion: &out.ResourceVersion, DesiredGeneration: &generation, Steps: steps})
		if e != nil {
			return e
		}
		payload, e := canonjson.Marshal(map[string]any{"schema_version": 1, "node_id": node, "incarnation_id": incarnation, "operation_id": out.OperationID})
		if e != nil {
			return e
		}
		if e = db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: node, AggregateType: "node", AggregateVersion: out.ResourceVersion.String(), EventType: desiredstate.EventNodePublicationChanged, SchemaVersion: 1, Payload: payload}); e != nil {
			return e
		}
		if e = complete(ctx, tx, record, api.ResourceTypeNode, node, 202, &out.OperationID); e != nil {
			return e
		}
		action := "node.revoked"
		if retire {
			action = "node.retired"
		}
		return appendDomainAudit(ctx, tx, actor, action, api.ResourceRef{Type: api.ResourceTypeNode, Id: node}, &out.OperationID, reason)
	})
	if err != nil {
		return RevocationResult{}, safe(err)
	}
	s.log(ctx, "node identity revoked; withdrawal pending", node)
	return out, nil
}

// CompatibilityDomain normalizes an IP endpoint or CIDR into its coarsened
// compatibility domain (IPv4 /24 or IPv6 /48) before server-side hashing (D-072).
// Non-IP opaque domain identifiers pass through trimmed.
func CompatibilityDomain(endpoint string) string {
	s := strings.TrimSpace(endpoint)
	if s == "" {
		return ""
	}
	if ip, _, err := net.ParseCIDR(s); err == nil && ip != nil {
		return coarsenIPDomain(ip)
	}
	host := s
	if h, _, err := net.SplitHostPort(s); err == nil {
		host = h
	} else if len(s) >= 2 && s[0] == '[' && s[len(s)-1] == ']' {
		host = s[1 : len(s)-1]
	}
	if i := strings.IndexByte(host, '%'); i >= 0 {
		host = host[:i]
	}
	if ip := net.ParseIP(host); ip != nil {
		return coarsenIPDomain(ip)
	}
	return s
}

func coarsenIPDomain(ip net.IP) string {
	if v4 := ip.To4(); v4 != nil {
		masked := v4.Mask(net.CIDRMask(24, 32))
		if masked != nil {
			return masked.String() + "/24"
		}
	}
	masked := ip.Mask(net.CIDRMask(48, 128))
	if masked != nil {
		return masked.String() + "/48"
	}
	return ip.String()
}

// ObserveEndpoint receives an opaque, authenticated compatibility-domain value
// from the adapter, never raw headers. Hashes avoid persisting endpoint lists.
// A single switch X->Y inside activeWindow is treated as roaming (X remains
// mapped while Y is recorded as candidate in node_identity_endpoints and
// promoted only after X has been quiet for a full activeWindow). Interleaving
// (X->Y->X within activeWindow, two switches within one window, or a third
// domain Z while X and Y are active) records a conflict and freezes automatic
// mapping forever (§6.2, D-072, D-073).
// Returns whether mapping updates are allowed. It does not grant forwarding.
func (s *Service) ObserveEndpoint(ctx context.Context, cert *x509.Certificate, endpoint string, activeWindow time.Duration) (allowed bool, err error) {
	ctx = requestContext(ctx)
	if !validIdentifier(endpoint) || len(endpoint) > 1024 || activeWindow <= 0 {
		return false, apierr.New(apierr.ValidationFailed, "Endpoint observation invalid.")
	}
	if _, err = s.auth.Authenticate(ctx, cert); err != nil {
		return false, err
	}
	err = commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var e error
		allowed, e = s.ObserveEndpointTx(ctx, tx, cert, endpoint, activeWindow)
		return e
	})
	return allowed, safe(err)
}

// ObserveEndpointTx records an authenticated compatibility-domain observation
// inside the caller's existing database transaction (D-072, D-073). Candidate
// state is persisted on node_identity_endpoints and serialised per incarnation
// with SELECT ... FOR UPDATE so multi-replica control planes and restarts
// evaluate roaming and interleaving identically.
func (s *Service) ObserveEndpointTx(ctx context.Context, tx pgx.Tx, cert *x509.Certificate, endpoint string, activeWindow time.Duration) (allowed bool, err error) {
	ctx = requestContext(ctx)
	if activeWindow <= 0 {
		activeWindow = s.ActiveWindow()
	}
	domain := CompatibilityDomain(endpoint)
	if !validIdentifier(domain) || len(domain) > 1024 || activeWindow <= 0 {
		return false, apierr.New(apierr.ValidationFailed, "Endpoint observation invalid.")
	}
	id, err := s.auth.LockAuthenticated(ctx, tx, cert)
	if err != nil {
		return false, err
	}
	hash := sha256.Sum256([]byte(domain))

	var dbNow time.Time
	if err = tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&dbNow); err != nil {
		return false, safe(err)
	}
	now := dbNow.UTC()
	s.mu.Lock()
	clock := s.clock
	s.mu.Unlock()
	if clock != nil {
		now = clock().UTC()
	}

	insertTag, err := tx.Exec(ctx, `
		INSERT INTO node_identity_endpoints(incarnation_id, endpoint_hash, last_seen_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (incarnation_id) DO NOTHING
	`, id.IncarnationID, hash[:], now)
	if err != nil {
		return false, safe(err)
	}
	if insertTag.RowsAffected() == 1 {
		if _, err = tx.Exec(ctx, `UPDATE nodes SET resource_version=resource_version+1 WHERE id=$1`, id.NodeID); err != nil {
			return false, safe(err)
		}
		return true, nil
	}

	var prev []byte
	var lastSeen time.Time
	var frozen bool
	var candHash []byte
	var candFirstSeen, candLastSeen *time.Time
	if err = tx.QueryRow(ctx, `
		SELECT endpoint_hash, last_seen_at, mapping_frozen,
		       candidate_hash, candidate_first_seen_at, candidate_last_seen_at
		FROM node_identity_endpoints
		WHERE incarnation_id = $1
		FOR UPDATE
	`, id.IncarnationID).Scan(&prev, &lastSeen, &frozen, &candHash, &candFirstSeen, &candLastSeen); err != nil {
		return false, safe(err)
	}
	if frozen {
		return false, nil
	}

	xActive := lastSeen.UTC().After(now.Add(-activeWindow))
	candActive := len(candHash) > 0 && candLastSeen != nil && candLastSeen.UTC().After(now.Add(-activeWindow))

	if bytes.Equal(prev, hash[:]) {
		if candActive {
			// Interleaving X -> Y -> X while Y is still inside activeWindow.
			if err = s.recordIdentityConflict(ctx, tx, id, prev, candHash, now); err != nil {
				return false, safe(err)
			}
			return false, nil
		}
		if _, err = tx.Exec(ctx, `UPDATE node_identity_endpoints SET last_seen_at=$2 WHERE incarnation_id=$1`, id.IncarnationID, now); err != nil {
			return false, safe(err)
		}
		return true, nil
	}

	if !xActive {
		// X has been quiet for at least a full activeWindow; promote the new
		// endpoint (whether the roaming candidate Y or a fresh endpoint).
		// If the promoted endpoint was a roaming candidate that overlapped with X,
		// retain displaced X as an inactive prior-pair marker (lastSeen <= now - activeWindow)
		// so a second {Y, X} window still triggers a conflict.
		hadOverlap := bytes.Equal(candHash, hash[:]) && candFirstSeen != nil && !candFirstSeen.UTC().Before(lastSeen.UTC())
		var nextCandHash []byte
		var nextCandFirst, nextCandLast *time.Time
		if hadOverlap {
			nextCandHash = prev
			ts := lastSeen.UTC()
			nextCandFirst = &ts
			nextCandLast = &ts
		}
		if _, err = tx.Exec(ctx, `
			UPDATE node_identity_endpoints
			SET endpoint_hash = $2,
			    last_seen_at = $3,
			    candidate_hash = $4,
			    candidate_first_seen_at = $5,
			    candidate_last_seen_at = $6
			WHERE incarnation_id = $1
		`, id.IncarnationID, hash[:], now, nextCandHash, nextCandFirst, nextCandLast); err != nil {
			return false, safe(err)
		}
		return true, nil
	}

	// X is still inside activeWindow and incoming hash != prev (X).
	if candActive {
		if !bytes.Equal(candHash, hash[:]) {
			// A third incompatible domain Z observed while X and Y are inside the window.
			if err = s.recordIdentityConflict(ctx, tx, id, prev, hash[:], now); err != nil {
				return false, safe(err)
			}
			return false, nil
		}
		// Y observed again while waiting for X to remain quiet for activeWindow.
		if _, err = tx.Exec(ctx, `
			UPDATE node_identity_endpoints
			SET candidate_last_seen_at = $2
			WHERE incarnation_id = $1
		`, id.IncarnationID, now); err != nil {
			return false, safe(err)
		}
		return true, nil
	}

	if bytes.Equal(candHash, hash[:]) {
		// Y and X have both been seen within one window twice.
		if err = s.recordIdentityConflict(ctx, tx, id, prev, hash[:], now); err != nil {
			return false, safe(err)
		}
		return false, nil
	}

	// First switch X -> Y inside activeWindow: roaming candidate. Keep X mapped
	// in node_identity_endpoints and record Y as candidate until X has been quiet
	// for a full activeWindow.
	if _, err = tx.Exec(ctx, `
		UPDATE node_identity_endpoints
		SET candidate_hash = $2,
		    candidate_first_seen_at = $3,
		    candidate_last_seen_at = $3
		WHERE incarnation_id = $1
	`, id.IncarnationID, hash[:], now); err != nil {
		return false, safe(err)
	}
	return true, nil
}

func (s *Service) recordIdentityConflict(ctx context.Context, tx pgx.Tx, id pki.NodeIdentity, prevHash, conflictingHash []byte, now time.Time) error {
	conflictID, err := uuid.NewV7()
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `INSERT INTO node_identity_conflicts(id,incarnation_id,previous_hash,conflicting_hash,observed_at) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`, conflictID, id.IncarnationID, prevHash, conflictingHash, now)
	if err != nil {
		return err
	}
	freezeTag, err := tx.Exec(ctx, `
		UPDATE node_identity_endpoints
		SET mapping_frozen = true,
		    candidate_hash = NULL,
		    candidate_first_seen_at = NULL,
		    candidate_last_seen_at = NULL
		WHERE incarnation_id = $1 AND NOT mapping_frozen
	`, id.IncarnationID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 || freezeTag.RowsAffected() > 0 {
		if err := nodes.CloseInactiveGenerationHoldsTx(ctx, tx, id.NodeID, now); err != nil {
			return err
		}
		// D-084: the detecting renewal must see a not-ready incarnation in
		// this transaction, before the next observation arrives.
		if _, err = tx.Exec(ctx, `UPDATE nodes SET provisioning_state='failed', safety_status='failed', resource_version=resource_version+1 WHERE id=$1`, id.NodeID); err != nil {
			return err
		}
		// D-088: the row is the authoritative central deadline. Preserve the
		// original signed envelope for replay; agent notification is separate.
		if _, err = tx.Exec(ctx, `
			UPDATE serving_leases
			SET expires_at = least(expires_at, now() + interval '60 seconds')
			WHERE node_id = $1 AND incarnation_id = $2
			  AND expires_at > now() + interval '60 seconds'
		`, id.NodeID, id.IncarnationID); err != nil {
			return err
		}
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	var conflictCount int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM node_identity_conflicts WHERE incarnation_id=$1`, id.IncarnationID).Scan(&conflictCount); err != nil {
		return err
	}
	requestID := commandtx.RequestID(ctx)
	after := map[string]any{
		"incarnation_id": id.IncarnationID.String(),
		"conflict_count": conflictCount,
		"observed_at":    now.UTC().Format(time.RFC3339),
	}
	event, err := audit.NewEvent(audit.Entry{
		Actor:     audit.Actor{Type: api.AuditEventActorTypeNode, ID: id.NodeID},
		Action:    "node.identity.conflict",
		Resource:  api.ResourceRef{Type: api.ResourceTypeNode, Id: id.NodeID},
		Result:    api.AuditEventResultSucceeded,
		RequestID: &requestID,
		After:     after,
		Fields:    []string{"incarnation_id", "conflict_count", "observed_at"},
	})
	if err != nil {
		return err
	}
	return audit.AppendEvent(ctx, tx, event)
}
