package devices

import (
	"context"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Unknown identities have no member/device resource yet. Attribute the denied
// controller decision to the existing fleet configuration, never invent a member
// UUID or retain a provider ID/subject. Known attempts use their real resources.
func decisionResource(member uuid.UUID) (api.ResourceType, uuid.UUID) {
	if member != uuid.Nil {
		return api.ResourceTypeMember, member
	}
	return api.ResourceTypeFleetConfig, uuid.MustParse("00000000-0000-0000-0000-000000000001")
}

// recordDenial commits independently of the rejected command transaction; an
// audit failure remains an error, never silent successful admission (D-029).
func (s *Service) recordDenial(ctx context.Context, c entitlement.Command, p Peer, member uuid.UUID, cause error) error {
	reason := ""
	switch {
	case errors.Is(cause, ErrLimit):
		reason = "device_limit_reached"
	case errors.Is(cause, ErrEntitlement):
		reason = "entitlement_inactive"
	case errors.Is(cause, ErrUnmapped):
		reason = "identity_unmapped"
	default:
		return nil
	}
	if c.Actor.ID == uuid.Nil || (c.Actor.Role != "admin" && c.Actor.Role != "system") {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		resource, id := decisionResource(member)
		var existing uuid.UUID
		err := tx.QueryRow(ctx, `SELECT id FROM member_devices WHERE deployment_id=$1 AND peer_id=$2`, p.DeploymentID, p.ID).Scan(&existing)
		if err == nil {
			resource, id = api.ResourceTypeMemberDevice, existing
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		return entitlement.AuditResult(ctx, tx, c.Actor, "device.authorize", resource, id, api.AuditEventResultDenied, nil, map[string]any{"reason": reason})
	})
}

func auditIdentity(ctx context.Context, tx pgx.Tx, member, identity uuid.UUID, result string) error {
	action := "identity_binding.rejected"
	status := api.AuditEventResultDenied
	switch result {
	case identityUnchanged:
		return nil
	case identityCreated:
		action = "identity_binding.created"
		status = api.AuditEventResultSucceeded
	case identityUpdated:
		action = "identity_binding.proof_updated"
		status = api.AuditEventResultSucceeded
	}
	resource, id := decisionResource(member)
	fields := map[string]any{"reason": result}
	if identity != uuid.Nil {
		fields["identity_binding_id"] = identity
	}
	return entitlement.AuditResult(ctx, tx, entitlement.Actor{ID: audit.SystemActor.ID, Role: string(audit.SystemActor.Type)}, action, resource, id, status, nil, fields)
}
