//go:build integration

package billing

import (
	"context"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"testing"
)

func Test15_2_ImmutablePlanRevisionsAndReferences(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	group, plan, revision, fup, dns := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO member_groups(id,code,name) VALUES($1,'test','TEST ONLY')`, group); err != nil {
		t.Fatal(err)
	}
	for id, typ := range map[uuid.UUID]string{fup: "fair_use", dns: "dns_profile"} {
		definition := uuid.New()
		if _, err := pool.Exec(ctx, `INSERT INTO policy_definitions(id,type,code) VALUES($1,$2,$2)`, definition, typ); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO policy_revisions(id,definition_id,type,revision,schema_version,payload,payload_hash,signing_key_ref,created_by) VALUES($1,$2,$3,1,1,'{}',decode(repeat('00',32),'hex'),'test-key','TEST ONLY')`, id, definition, typ); err != nil {
			t.Fatal(err)
		}
	}
	p := DefaultPlanPayload()
	p.DisplayName = api.LocalizedText{ZhTW: "TEST ONLY"}
	p.ExitGroupIds = []uuid.UUID{group}
	p.FairUseRevisionId = fup
	p.DnsProfileRevisionId = dns
	p.Price = api.PlanPrice{AmountMinor: 1000, Currency: "JPY", Provider: "fake", ProviderPriceRef: "test-price", BillingInterval: api.PlanPriceBillingIntervalMonth}
	err := db.InTx(ctx, pool, db.TxOptions{}, func(tx pgx.Tx) error {
		r, e := CreatePlanRevision(ctx, tx, plan, revision, "test", p)
		if e != nil {
			return e
		}
		if r.Revision != 1 || r.Payload.DeviceLimit != 3 {
			t.Fatal(r)
		}
		return ActivatePlanRevision(ctx, tx, plan, revision, 1)
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE plan_revisions SET device_limit=100 WHERE id=$1`, revision); err == nil {
		t.Fatal("revision mutated")
	}
	err = db.InTx(ctx, pool, db.TxOptions{}, func(tx pgx.Tx) error {
		rows, e := ListPlans(ctx, tx, uuid.Nil, 100)
		if e != nil {
			return e
		}
		if len(rows) != 1 || rows[0].ID != revision {
			t.Fatal(rows)
		}
		_, e = ReadPlan(ctx, tx, revision)
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	p.ExitGroupIds = []uuid.UUID{uuid.New()}
	err = db.InTx(ctx, pool, db.TxOptions{}, func(tx pgx.Tx) error { _, e := CreatePlanRevision(ctx, tx, plan, uuid.New(), "test", p); return e })
	if err == nil {
		t.Fatal("unknown group accepted")
	}
}
