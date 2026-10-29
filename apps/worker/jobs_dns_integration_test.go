//go:build integration

package worker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func Test12_5_DNSDomainRegisteredWithConfigFile(t *testing.T) {
	pool := dbtest.New(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "dns.json")
	// TEST ONLY documentation prefix. The handler does not dial it.
	if err := os.WriteFile(path, []byte(`{"upstreams":["192.0.2.53:53"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c := config.Defaults()
	c.DatabaseURLFile = "TEST-ONLY"
	c.DNSConfigFile = path
	app, err := Open(t.Context(), Dependencies{Config: c, Pool: pool}, dns())
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	err = db.InTx(t.Context(), pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return db.EnqueueOutbox(t.Context(), tx, db.OutboxEvent{
			ID: id, AggregateID: uuid.New(), AggregateType: "policy_definition",
			AggregateVersion: "1", EventType: dnspublish.EventDNSPublish, SchemaVersion: 1,
			Payload: []byte(`{"type":"plan"}`),
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	worked, err := app.Once(t.Context())
	if err != nil || !worked {
		t.Fatalf("dispatch worked=%v err=%v", worked, err)
	}
	var receipt *string
	if err := pool.QueryRow(t.Context(), `SELECT delivery_receipt FROM outbox_events WHERE id=$1`, id).Scan(&receipt); err != nil || receipt == nil || *receipt != "dns-publish:ignored:plan" {
		t.Fatalf("receipt %v err %v", receipt, err)
	}
}
