package nodes

import (
	"context"
	"encoding/hex"
	"errors"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"time"
)

// NetBirdBindings persists adapter identities in the existing external_bindings
// table. Namespaced opaque keys preserve adapter lookups without a new schema.
// An advisory SESSION lock serializes deployment writes outside SQL transactions.
type NetBirdBindings struct{ Pool *pgxpool.Pool }

func (s NetBirdBindings) Lock(ctx context.Context, deployment string) (func(), error) {
	conn, e := s.Pool.Acquire(ctx)
	if e != nil {
		return nil, e
	}
	if _, e = conn.Exec(ctx, `SELECT pg_advisory_lock(hashtextextended($1,0))`, "fvpn-netbird:"+deployment); e != nil {
		conn.Release()
		return nil, e
	}
	return func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, e := conn.Exec(c, `SELECT pg_advisory_unlock(hashtextextended($1,0))`, "fvpn-netbird:"+deployment)
		if e != nil {
			_ = conn.Hijack().Close(c)
		} else {
			conn.Release()
		}
	}, nil
}
func bindingType(kind, id string) string { return "netbird:" + kind + ":" + id }
func bindingID(kind, id string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(bindingType(kind, id)))
}
func (s NetBirdBindings) read(ctx context.Context, d, k string, byExternal bool, id string) (netbird.Binding, bool, error) {
	b := netbird.Binding{DeploymentID: d, Kind: k}
	var rt, external, hash string
	var observed *time.Time
	query := `SELECT resource_type,external_id,encode(desired_hash,'hex'),last_observed_at FROM external_bindings WHERE provider='netbird-adapter' AND deployment_id=$1 AND resource_type=$2`
	arg := bindingType(k, id)
	if byExternal {
		query = `SELECT resource_type,external_id,encode(desired_hash,'hex'),last_observed_at FROM external_bindings WHERE provider='netbird-adapter' AND deployment_id=$1 AND external_id=$2`
		arg = k + ":" + id
	}
	e := s.Pool.QueryRow(ctx, query, d, arg).Scan(&rt, &external, &hash, &observed)
	if errors.Is(e, pgx.ErrNoRows) {
		return b, false, nil
	}
	if e != nil {
		return b, false, e
	}
	b.InternalID = strings.TrimPrefix(rt, "netbird:"+k+":")
	b.ExternalID = strings.TrimPrefix(external, k+":")
	if observed != nil {
		b.LastObservedAt = *observed
		b.DesiredHash = hash
	}
	return b, true, nil
}
func (s NetBirdBindings) Get(ctx context.Context, d, k, id string) (netbird.Binding, bool, error) {
	return s.read(ctx, d, k, false, id)
}
func (s NetBirdBindings) FindExternal(ctx context.Context, d, k, id string) (netbird.Binding, bool, error) {
	return s.read(ctx, d, k, true, id)
}
func (s NetBirdBindings) Put(ctx context.Context, b netbird.Binding) error {
	hash := make([]byte, 32)
	var e error
	var observed any
	if b.DesiredHash != "" {
		hash, e = hex.DecodeString(b.DesiredHash)
		if e != nil || len(hash) != 32 {
			return errors.New("invalid binding hash")
		}
		observed = b.LastObservedAt
	}
	tag, e := s.Pool.Exec(ctx, `INSERT INTO external_bindings(resource_type,resource_id,provider,deployment_id,external_id,desired_hash,last_observed_at) VALUES($1,$2,'netbird-adapter',$3,$4,$5,$6) ON CONFLICT(resource_type,resource_id,provider,deployment_id) DO UPDATE SET desired_hash=excluded.desired_hash,last_observed_at=excluded.last_observed_at WHERE external_bindings.external_id=excluded.external_id`, bindingType(b.Kind, b.InternalID), bindingID(b.Kind, b.InternalID), b.DeploymentID, b.Kind+":"+b.ExternalID, hash, observed)
	if e == nil && tag.RowsAffected() != 1 {
		return errors.New("external binding conflict")
	}
	return e
}
func (s NetBirdBindings) Delete(ctx context.Context, d, k, id string) error {
	_, e := s.Pool.Exec(ctx, `DELETE FROM external_bindings WHERE provider='netbird-adapter' AND deployment_id=$1 AND resource_type=$2 AND resource_id=$3`, d, bindingType(k, id), bindingID(k, id))
	return e
}

var _ netbird.BindingStore = NetBirdBindings{}
