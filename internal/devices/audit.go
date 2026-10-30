package devices

import (
	"context"
	"crypto/sha256"
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
// Explicit commands audit every denial. Discovery uses recordDiscoveryDenial.
func (s *Service) recordDenial(ctx context.Context, c entitlement.Command, p Peer, member uuid.UUID, cause error) error {
	return s.writeDenial(ctx, c, p, member, cause, false)
}

// recordDiscoveryDenial emits the same denial once per peer decision transition.
// A later discovery pass with the same reason writes no audit row.
func (s *Service) recordDiscoveryDenial(ctx context.Context, c entitlement.Command, p Peer, member uuid.UUID, cause error) error {
	return s.writeDenial(ctx, c, p, member, cause, true)
}

func (s *Service) writeDenial(ctx context.Context, c entitlement.Command, p Peer, member uuid.UUID, cause error, once bool) error {
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
		if once {
			changed, err := noteDiscoveryDecision(ctx, tx, p.DeploymentID, "peer:"+p.ID, reason)
			if err != nil || !changed {
				return err
			}
		}
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

// discoveryAdmitted replaces a stored denial once the peer or user is no longer
// denied. The same denial reason then audits again if it returns.
const discoveryAdmitted = "admitted"

// noteDiscoveryDecision persists the last denial reason for one discovery key.
// The row lives in external_bindings (kind discovery_decision), scoped by
// deployment. It is adapter bookkeeping, not a contract resource: unmapped
// peers still have no domain UUID. changed is true the first time a key is
// seen and whenever the reason hash changes.
func noteDiscoveryDecision(ctx context.Context, tx pgx.Tx, deployment, key, reason string) (bool, error) {
	sum := sha256.Sum256([]byte(reason))
	id, err := bindingID("discovery_decision", key)
	if err != nil {
		return false, err
	}
	resourceType := bindingType("discovery_decision", key)
	externalID := "discovery_decision:" + key
	tag, err := tx.Exec(ctx, `INSERT INTO external_bindings(resource_type,resource_id,provider,deployment_id,external_id,desired_hash,last_observed_at) VALUES($1,$2,'netbird',$3,$4,$5,clock_timestamp()) ON CONFLICT(resource_type,resource_id,provider,deployment_id) DO UPDATE SET desired_hash=excluded.desired_hash,last_observed_at=clock_timestamp() WHERE external_bindings.external_id=excluded.external_id AND external_bindings.desired_hash IS DISTINCT FROM excluded.desired_hash`, resourceType, id, deployment, externalID, sum[:])
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 1 {
		return true, nil
	}
	_, err = tx.Exec(ctx, `UPDATE external_bindings SET last_observed_at=clock_timestamp() WHERE provider='netbird' AND deployment_id=$1 AND resource_type=$2 AND resource_id=$3 AND external_id=$4`, deployment, resourceType, id, externalID)
	return false, err
}

// relaxDiscoveryDecision records that a key is no longer denied. It does not
// insert a row for a key that was never denied, and it does not write an audit.
func relaxDiscoveryDecision(ctx context.Context, tx pgx.Tx, deployment, key string) error {
	sum := sha256.Sum256([]byte(discoveryAdmitted))
	id, err := bindingID("discovery_decision", key)
	if err != nil {
		return err
	}
	resourceType := bindingType("discovery_decision", key)
	externalID := "discovery_decision:" + key
	_, err = tx.Exec(ctx, `UPDATE external_bindings SET desired_hash=$5,last_observed_at=clock_timestamp() WHERE provider='netbird' AND deployment_id=$1 AND resource_type=$2 AND resource_id=$3 AND external_id=$4 AND desired_hash IS DISTINCT FROM $5`, deployment, resourceType, id, externalID, sum[:])
	return err
}

func identityRejection(result string) bool {
	switch result {
	case identityUnmapped, identityConflict, identityService, identityBlocked:
		return true
	default:
		return false
	}
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
