package devices

import (
	"context"
	"errors"
	"log/slog"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	issuerform "github.com/TED-H-com/Freedom-VPN/internal/members/issuer"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	identityCreated   = "created"
	identityUnchanged = "unchanged"
	identityUpdated   = "proof_updated"
	identityUnmapped  = "identity_unmapped"
	identityConflict  = "identity_conflict"
	identityService   = "service_user"
	identityBlocked   = "blocked_user"
)

// IdentityDiscovery implements D-052: the configured shared issuer uses public
// subjects for both OAuth clients (§2 [S36], §15.2). An observed NetBird user ID
// must equal the verified portal subject. No email/name fallback is permitted.
type IdentityDiscovery struct {
	issuer  string
	logger  *slog.Logger
	results *prometheus.CounterVec
}

// NewIdentityDiscovery takes the central oidc_issuer, never a separate issuer
// setting. The worker registers this collector on its existing /metrics registry.
func NewIdentityDiscovery(issuer string, registry prometheus.Registerer, logger *slog.Logger) (*IdentityDiscovery, error) {
	canonical, err := issuerform.Canonical(issuer)
	if err != nil {
		return nil, err
	}
	issuer = canonical
	if logger == nil {
		logger = slog.Default()
	}
	d := &IdentityDiscovery{issuer: issuer, logger: logger, results: prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fvpn_netbird_identity_discovery_total",
		Help: "NetBird identity discovery observations by outcome; retries count again, identities are not labels.",
	}, []string{"result"})}
	if registry != nil {
		if err := registry.Register(d.results); err != nil {
			return nil, err
		}
	}
	return d, nil
}

func (d *IdentityDiscovery) proof() string { return "shared_issuer_public_subject:" + d.issuer }

func (d *IdentityDiscovery) record(ctx context.Context, result string) {
	d.results.WithLabelValues(result).Inc()
	if result == identityUnmapped || result == identityConflict {
		// Bounded reason only: never expose subjects, emails, names or tokens.
		d.logger.WarnContext(ctx, "NetBird identity not mapped", "reason", result)
	}
}

// bindUser runs after the NetBird user read has completed. Member locking and
// binding writes share one transaction; no provider I/O runs under that lock.
func (a *NetBirdAdapter) bindUser(ctx context.Context, external string) (string, error) {
	result := identityUnmapped
	err := db.InTx(ctx, a.Pool, db.TxOptions{}, func(tx pgx.Tx) (retErr error) {
		var member, identity uuid.UUID
		defer func() {
			if retErr == nil {
				retErr = auditIdentity(ctx, tx, member, identity, result)
			}
		}()
		type candidate struct{ identity, member uuid.UUID }
		rows, err := tx.Query(ctx, `SELECT i.id,i.member_id FROM identity_bindings i WHERE i.issuer=$1 AND i.subject=$2 AND i.stable_claim_ref='sub' AND i.binding_method='oidc' AND i.verified_at<=clock_timestamp() ORDER BY i.id LIMIT 2`, a.Identity.issuer, external)
		if err != nil {
			return err
		}
		var matches []candidate
		for rows.Next() {
			var c candidate
			if err = rows.Scan(&c.identity, &c.member); err != nil {
				rows.Close()
				return err
			}
			matches = append(matches, c)
		}
		err = rows.Err()
		rows.Close()
		if err != nil || len(matches) == 0 {
			return err
		}
		// Multiple client-scoped portal identities must not select an arbitrary
		// member, even though the deployment promises public subjects.
		if len(matches) != 1 {
			result = identityConflict
			return nil
		}
		c := matches[0]
		member, identity = c.member, c.identity
		var open bool
		if err = tx.QueryRow(ctx, `SELECT status<>'closed' FROM members WHERE id=$1 FOR UPDATE`, c.member).Scan(&open); err != nil || !open {
			return err
		}
		// Recheck verification after obtaining the member lock; hold the identity
		// row through the write so mutable verification metadata cannot race it.
		var verified uuid.UUID
		err = tx.QueryRow(ctx, `SELECT id FROM identity_bindings WHERE id=$1 AND member_id=$2 AND issuer=$3 AND subject=$4 AND stable_claim_ref='sub' AND binding_method='oidc' AND verified_at<=clock_timestamp() FOR SHARE`, c.identity, c.member, a.Identity.issuer, external).Scan(&verified)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `INSERT INTO netbird_user_bindings(id,member_id,deployment_id,external_user_id,identity_binding_id,mapping_proof,revision,verified_at) VALUES($1,$2,$3,$4,$5,$6,1,clock_timestamp()) ON CONFLICT(deployment_id,external_user_id) DO NOTHING`, id, c.member, a.DeploymentID, external, c.identity, a.Identity.proof())
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 1 {
			result = identityCreated
			return nil
		}
		var existing candidate
		var proof string
		var revision decstr.U64
		var future bool
		err = tx.QueryRow(ctx, `SELECT identity_binding_id,member_id,mapping_proof,revision::text,verified_at>clock_timestamp() FROM netbird_user_bindings WHERE deployment_id=$1 AND external_user_id=$2 FOR UPDATE`, a.DeploymentID, external).Scan(&existing.identity, &existing.member, &proof, &revision, &future)
		if err != nil {
			return err
		}
		// §15.2 and the immutable-field trigger forbid reassignment. Record the
		// anomaly and keep every byte of the conflicting binding unchanged.
		if existing != c {
			result = identityConflict
			return nil
		}
		if proof == a.Identity.proof() && revision > 0 && !future {
			result = identityUnchanged
			return nil
		}
		_, err = tx.Exec(ctx, `UPDATE netbird_user_bindings SET mapping_proof=$3,revision=revision+1,verified_at=clock_timestamp() WHERE deployment_id=$1 AND external_user_id=$2`, a.DeploymentID, external, a.Identity.proof())
		result = identityUpdated
		return err
	})
	return result, err
}

func (a *NetBirdAdapter) verifiedIdentity(ctx context.Context, external string) (bool, error) {
	var verified bool
	err := a.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM netbird_user_bindings n JOIN identity_bindings i ON i.id=n.identity_binding_id AND i.member_id=n.member_id WHERE n.deployment_id=$1 AND n.external_user_id=$2 AND n.mapping_proof=$3 AND n.revision>0 AND n.verified_at<=clock_timestamp() AND i.issuer=$4 AND i.subject=n.external_user_id AND i.stable_claim_ref='sub' AND i.binding_method='oidc' AND i.verified_at<=clock_timestamp() AND NOT EXISTS(SELECT 1 FROM identity_bindings other WHERE other.issuer=i.issuer AND other.subject=i.subject AND other.id<>i.id))`, a.DeploymentID, external, a.Identity.proof(), a.Identity.issuer).Scan(&verified)
	return verified, err
}
