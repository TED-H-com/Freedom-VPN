//go:build integration

package httpwire_test

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/fleetupdate"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TestReviewR16_GenerationAdvanceAfterCacheIsServed is the HTTP-shaped D-099
// proof. The first desired-state response fills the cache. Raising only
// applied_config_generation (the restored-node case: the node retained a
// generation this database's desired pointer does not) must be served as a
// strictly newer signed generation. wire-nodes follow-up 29 owns the fix.
func TestReviewR16_GenerationAdvanceAfterCacheIsServed(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	vault := f.r.Provider.(*signing.FileProvider)
	if _, err := vault.Rotate(ctx, signing.DesiredState, 48*time.Hour, time.Hour); err != nil {
		t.Fatal(err)
	}
	rf := releasetest.New(t)
	f.r.Releases = &releases.Service{Pool: f.r.Pool, Validator: rf.Validator}
	var err error
	f.r.Fleet, err = fleetupdate.New(f.r.Pool, f.r.Releases, fleetupdate.Config{AuthorizationTTL: time.Hour, OnlineWindow: time.Minute, StuckWindow: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	f.r.Config.DesiredStateSeconds = 3600
	f.r.Config.DNSResolverIPv4 = []string{"100.64.0.53"}
	definition, revision := uuid.New(), uuid.New()
	err = db.InTx(ctx, f.r.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		if _, e := tx.Exec(ctx, `INSERT INTO policy_definitions(id,type,code) VALUES($1,'release_support','TEST ONLY r16 support')`, definition); e != nil {
			return e
		}
		hash, _ := canonjson.Hash([]byte(`{}`))
		if _, e := tx.Exec(ctx, `INSERT INTO policy_revisions(id,definition_id,type,revision,schema_version,payload,payload_hash,signing_key_ref,created_by) VALUES($1,$2,'release_support',1,1,'{}',$3,'TEST ONLY key','TEST ONLY actor')`, revision, definition, hash[:]); e != nil {
			return e
		}
		_, e := tx.Exec(ctx, `UPDATE policy_definitions SET active_revision_id=$2,resource_version=resource_version+1 WHERE id=$1`, definition, revision)
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	_, enrolled, cert := f.enroll(f.create(nil))
	first := decode[api.DesiredStateEnvelope](t, f.call("GET", "/api/v1/agent/desired-state", nil, cert, nil), 200)
	applied := uint64(first.Signed.ConfigGeneration) + 5
	if _, err := f.r.Pool.Exec(ctx, `UPDATE nodes SET applied_config_generation=$2::numeric, resource_version=resource_version+1 WHERE id=$1`, enrolled.NodeId, applied); err != nil {
		t.Fatal(err)
	}
	second := decode[api.DesiredStateEnvelope](t, f.call("GET", "/api/v1/agent/desired-state", nil, cert, nil), 200)
	if uint64(second.Signed.ConfigGeneration) <= applied {
		t.Fatalf("TODO(review): D-099 HTTP cache served generation %s after applied_config_generation %d; want a signed generation strictly above the applied generation (first offer was %s)", second.Signed.ConfigGeneration, applied, first.Signed.ConfigGeneration)
	}
}
