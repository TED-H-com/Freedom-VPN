package enrollment

import (
	"context"
	"crypto/x509"
	"errors"
	"log/slog"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type BootstrapDelivery struct {
	ExternalKeyID string
	SetupKey      string
	ExpiresAt     time.Time
}

func (BootstrapDelivery) LogValue() slog.Value {
	return slog.StringValue("[REDACTED bootstrap delivery]")
}

func bootstrapAAD(id pki.NodeIdentity, operation uuid.UUID, external string, expiry time.Time) ([]byte, error) {
	return canonjson.Marshal(map[string]any{"purpose": "netbird-bootstrap-v1", "node_id": id.NodeID, "incarnation_id": id.IncarnationID, "operation_id": operation, "external_key_id": external, "expires_at": expiry.UTC().Format(time.RFC3339Nano)})
}

// StoreBootstrap is a worker-only API, called after external read-back. Replays
// with the same operation/external ID are inert, including after delivery. A
// different external key is a conflict, never an invitation to duplicate it.
func (s *Service) StoreBootstrap(ctx context.Context, id pki.NodeIdentity, operation uuid.UUID, external, secret string, expires time.Time) error {
	if operation == uuid.Nil || !validIdentifier(external) || len(external) > 512 || secret == "" || len(secret) > 8192 {
		return apierr.New(apierr.ValidationFailed, "Invalid bootstrap result.")
	}
	if err := s.auth.CheckFence(ctx, id); err != nil {
		return err
	}
	expires = expires.UTC().Truncate(time.Microsecond) // PostgreSQL timestamp precision, also in AAD.
	aad, err := bootstrapAAD(id, operation, external, expires)
	if err != nil {
		return safe(err)
	}
	sealed, err := s.secrets.Seal(ctx, []byte(secret), aad)
	if err != nil {
		return safe(err)
	}
	err = db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var allowed bool
		e := tx.QueryRow(ctx, `SELECT n.lifecycle_state IN ('enrolling','registered') AND n.current_incarnation_id=$2 AND i.revoked_at IS NULL FROM nodes n JOIN node_incarnations i ON i.node_id=n.id AND i.id=$2 WHERE n.id=$1 FOR UPDATE OF n`, id.NodeID, id.IncarnationID).Scan(&allowed)
		if e != nil {
			return e
		}
		if !allowed {
			return apierr.New(apierr.IdentityRevoked, "Node identity revoked.")
		}
		if e := s.auth.CheckFence(ctx, id); e != nil {
			return e
		}
		var savedExternal string
		var savedOperation uuid.UUID
		e = tx.QueryRow(ctx, `SELECT external_key_id,operation_id FROM node_bootstrap_secrets WHERE incarnation_id=$1`, id.IncarnationID).Scan(&savedExternal, &savedOperation)
		if e == nil {
			if savedExternal != external || savedOperation != operation {
				return apierr.New(apierr.StateConflict, "Bootstrap operation already bound.")
			}
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		var valid bool
		e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM node_enrollments e JOIN operations o ON o.id=e.operation_id WHERE e.node_id=$1 AND e.incarnation_id=$2 AND e.operation_id=$3 AND o.status IN ('pending','running') AND o.kind='node.provision') AND $4::timestamptz>clock_timestamp() AND $4::timestamptz<=clock_timestamp()+$5::bigint*interval '1 millisecond'`, id.NodeID, id.IncarnationID, operation, expires, s.cfg.BootstrapValidity.Milliseconds()).Scan(&valid)
		if e != nil {
			return e
		}
		if !valid {
			return apierr.New(apierr.StateConflict, "Invalid bootstrap operation or deadline.")
		}
		_, e = tx.Exec(ctx, `INSERT INTO node_bootstrap_secrets(incarnation_id,operation_id,external_key_id,encryption_key_id,ciphertext,expires_at) VALUES($1,$2,$3,$4,$5,$6)`, id.IncarnationID, operation, external, sealed.KeyID, sealed.Ciphertext, expires)
		return e
	})
	return safe(err)
}

// TakeBootstrap accepts the peer certificate from verified TLS, not a claimed
// NodeIdentity. Commit erases ciphertext before plaintext leaves the service.
// A lost delivery response cannot be replayed; reconcile the existing external
// setup key/peer through the worker, never automatically create a second key.
func (s *Service) TakeBootstrap(ctx context.Context, cert *x509.Certificate) (out BootstrapDelivery, err error) {
	id, err := s.auth.Authenticate(ctx, cert)
	if err != nil {
		return out, err
	}
	var sealed signing.Sealed
	var op uuid.UUID
	var unavailable bool
	err = db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		unavailable = false
		sealed = signing.Sealed{}
		out = BootstrapDelivery{}
		if _, e := s.auth.LockAuthenticated(ctx, tx, cert); e != nil {
			return e
		}
		var keyID *string
		var expired bool
		e := tx.QueryRow(ctx, `SELECT operation_id,external_key_id,encryption_key_id,ciphertext,expires_at,expires_at<=clock_timestamp() FROM node_bootstrap_secrets WHERE incarnation_id=$1 FOR UPDATE`, id.IncarnationID).Scan(&op, &out.ExternalKeyID, &keyID, &sealed.Ciphertext, &out.ExpiresAt, &expired)
		if errors.Is(e, pgx.ErrNoRows) {
			unavailable = true
			return nil
		}
		if e != nil {
			return e
		}
		if keyID == nil {
			unavailable = true
			return nil
		}
		sealed.KeyID = *keyID
		_, e = tx.Exec(ctx, `UPDATE node_bootstrap_secrets SET ciphertext=NULL,encryption_key_id=NULL,erased_at=clock_timestamp(),delivered_at=CASE WHEN $2 THEN NULL ELSE clock_timestamp() END WHERE incarnation_id=$1`, id.IncarnationID, expired)
		if e != nil {
			return e
		}
		if expired {
			unavailable = true
			return nil
		}
		return audit(ctx, tx, "node:"+id.NodeID.String(), "bootstrap.claimed", id.IncarnationID, &op)
	})
	if err != nil {
		return BootstrapDelivery{}, safe(err)
	}
	if unavailable {
		return BootstrapDelivery{}, apierr.New(apierr.StateConflict, "Bootstrap is absent, expired or already delivered.")
	}
	aad, err := bootstrapAAD(id, op, out.ExternalKeyID, out.ExpiresAt)
	if err != nil {
		return BootstrapDelivery{}, safe(err)
	}
	plain, err := s.secrets.Open(ctx, sealed, aad)
	if err != nil {
		return BootstrapDelivery{}, safe(err)
	}
	if !time.Now().Before(out.ExpiresAt) {
		clear(plain)
		return BootstrapDelivery{}, apierr.New(apierr.StateConflict, "Bootstrap delivery expired.")
	}
	out.SetupKey = string(plain)
	clear(plain)
	out.ExpiresAt = out.ExpiresAt.UTC()
	s.log(ctx, "bootstrap claimed", id.IncarnationID)
	return out, nil
}

// ExpireBootstrap is a periodic worker sweep; retrieval also erases expired
// material. Only the external ID and processing metadata remain after erasure.
func (s *Service) ExpireBootstrap(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `UPDATE node_bootstrap_secrets SET ciphertext=NULL,encryption_key_id=NULL,erased_at=clock_timestamp() WHERE ciphertext IS NOT NULL AND expires_at<=clock_timestamp()`)
	return safe(err)
}
