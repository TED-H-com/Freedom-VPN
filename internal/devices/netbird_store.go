package devices

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NetBirdBindings persists the adapter's string keys in the existing polymorphic
// external_bindings table. Namespaced external IDs avoid cross-kind collisions.
// ResourceType retains the reversible key; ResourceID is its JCS SHA-256 UUID.
// Every central adapter for this deployment must use this same lock/key scheme.
type NetBirdBindings struct{ Pool *pgxpool.Pool }

func bindingType(kind, key string) string {
	return "netbird:" + base64.RawURLEncoding.EncodeToString([]byte(kind)) + ":" + base64.RawURLEncoding.EncodeToString([]byte(key))
}
func bindingID(kind, key string) (uuid.UUID, error) {
	raw, e := canonjson.Marshal([]string{kind, key})
	if e != nil {
		return uuid.Nil, e
	}
	hash, e := canonjson.Hash(raw)
	if e != nil {
		return uuid.Nil, e
	}
	id, _ := uuid.FromBytes(hash[:16])
	id[6] = id[6]&15 | 0x80
	id[8] = id[8]&63 | 0x80
	return id, nil
}
func (s NetBirdBindings) Lock(ctx context.Context, deployment string) (func(), error) {
	conn, e := s.Pool.Acquire(ctx)
	if e != nil {
		return nil, e
	}
	if _, e = conn.Exec(ctx, `SELECT pg_advisory_lock(hashtextextended($1,119))`, deployment); e != nil {
		// A canceled lock acquisition can have uncertain session state; discard it.
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = conn.Conn().Close(cleanup)
		conn.Release()
		return nil, e
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if _, e := conn.Exec(cleanup, `SELECT pg_advisory_unlock(hashtextextended($1,119))`, deployment); e != nil {
				_ = conn.Conn().Close(cleanup)
			}
			conn.Release()
		})
	}, nil
}
func scanBinding(row pgx.Row, deployment, kind string) (netbird.Binding, bool, error) {
	b := netbird.Binding{DeploymentID: deployment, Kind: kind}
	var typ, external string
	var hash []byte
	var observed *time.Time
	e := row.Scan(&typ, &external, &hash, &observed)
	if errors.Is(e, pgx.ErrNoRows) {
		return b, false, nil
	}
	if e != nil {
		return b, false, e
	}
	parts := strings.Split(typ, ":")
	if len(parts) != 3 {
		return b, false, entitlement.ErrInvalid
	}
	key, e := base64.RawURLEncoding.DecodeString(parts[2])
	if e != nil {
		return b, false, e
	}
	b.InternalID = string(key)
	b.ExternalID = strings.TrimPrefix(external, kind+":")
	// NULL observation denotes the adapter's unconfirmed write marker.
	b.DesiredHash = ""
	if observed != nil {
		b.DesiredHash = hex.EncodeToString(hash)
		b.LastObservedAt = observed.UTC()
	}
	return b, true, nil
}
func (s NetBirdBindings) Get(ctx context.Context, deployment, kind, key string) (netbird.Binding, bool, error) {
	return scanBinding(s.Pool.QueryRow(ctx, `SELECT resource_type,external_id,desired_hash,last_observed_at FROM external_bindings WHERE provider='netbird' AND deployment_id=$1 AND resource_type=$2`, deployment, bindingType(kind, key)), deployment, kind)
}
func (s NetBirdBindings) FindExternal(ctx context.Context, deployment, kind, external string) (netbird.Binding, bool, error) {
	return scanBinding(s.Pool.QueryRow(ctx, `SELECT resource_type,external_id,desired_hash,last_observed_at FROM external_bindings WHERE provider='netbird' AND deployment_id=$1 AND external_id=$2`, deployment, kind+":"+external), deployment, kind)
}

// VerifyMemberIdentity supports upgrading the pre-F-03 adapter hash. The base
// trigger forbids changing this row's member, deployment, user or identity ID;
// discovery may only advance its proof revision. Re-prove the exact tuple and
// current revision rather than trusting an opaque external binding hash.
func (s NetBirdBindings) VerifyMemberIdentity(ctx context.Context, deployment string, d netbird.MemberDevice) (bool, error) {
	var verified bool
	err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM netbird_user_bindings n
 JOIN identity_bindings i ON i.id=n.identity_binding_id AND i.member_id=n.member_id
 WHERE n.deployment_id=$1 AND n.member_id=$2 AND n.identity_binding_id=$3 AND n.external_user_id=$4
 AND n.revision=$5::numeric AND n.revision>0 AND n.verified_at<=clock_timestamp()
 AND i.subject=n.external_user_id AND i.stable_claim_ref='sub' AND i.binding_method='oidc'
 AND i.verified_at<=clock_timestamp() AND n.mapping_proof='shared_issuer_public_subject:'||i.issuer)`, deployment, d.MemberID, d.IdentityBindingID, d.UserID, d.MappingRevision).Scan(&verified)
	return verified, err
}

func (s NetBirdBindings) Put(ctx context.Context, b netbird.Binding) error {
	id, e := bindingID(b.Kind, b.InternalID)
	if e != nil {
		return e
	}
	hash, e := hex.DecodeString(b.DesiredHash)
	if b.DesiredHash == "" && b.LastObservedAt.IsZero() {
		hash = make([]byte, 32)
	}
	if e != nil || len(hash) != 32 || b.DesiredHash == "" && !b.LastObservedAt.IsZero() {
		return entitlement.ErrInvalid
	}
	var observed *time.Time
	if !b.LastObservedAt.IsZero() {
		v := b.LastObservedAt.UTC()
		observed = &v
	}
	tag, e := s.Pool.Exec(ctx, `INSERT INTO external_bindings(resource_type,resource_id,provider,deployment_id,external_id,desired_hash,last_observed_at) VALUES($1,$2,'netbird',$3,$4,$5,$6) ON CONFLICT(resource_type,resource_id,provider,deployment_id) DO UPDATE SET desired_hash=excluded.desired_hash,last_observed_at=excluded.last_observed_at WHERE external_bindings.external_id=excluded.external_id`, bindingType(b.Kind, b.InternalID), id, b.DeploymentID, b.Kind+":"+b.ExternalID, hash, observed)
	if e == nil && tag.RowsAffected() != 1 {
		return entitlement.ErrConflict
	}
	return e
}
func (s NetBirdBindings) Delete(ctx context.Context, deployment, kind, key string) error {
	_, e := s.Pool.Exec(ctx, `DELETE FROM external_bindings WHERE provider='netbird' AND deployment_id=$1 AND resource_type=$2`, deployment, bindingType(kind, key))
	return e
}

var _ netbird.BindingStore = NetBirdBindings{}
