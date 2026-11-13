//go:build integration

package db_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Test14_1_OpenScansTimestamptzUTC(t *testing.T) {
	local := time.Local
	time.Local = time.FixedZone("UTC-5", -5*60*60)
	t.Cleanup(func() { time.Local = local })

	isolated, dsn := dbtest.NewUnmigratedWithDSN(t)
	ctx := t.Context()
	want := time.Date(2026, time.October, 28, 9, 6, 30, 123456000, time.Local)
	if _, err := isolated.Exec(ctx, `CREATE TABLE utc_times (occurred_at timestamptz NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := isolated.Exec(ctx, `INSERT INTO utc_times (occurred_at) VALUES ($1)`, want); err != nil {
		t.Fatal(err)
	}
	pool, err := db.Open(ctx, dsn, db.OpenOptions{MaxConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	t.Run("new_connections", func(t *testing.T) {
		// Hold both acquisitions until cleanup to exercise distinct connections.
		for i := range 2 {
			conn, err := pool.Acquire(ctx)
			if err != nil {
				t.Fatal("acquire isolated connection")
			}
			t.Cleanup(conn.Release)
			var got time.Time
			if err := conn.QueryRow(ctx, `SELECT occurred_at FROM utc_times`, pgx.QueryResultFormats{pgx.BinaryFormatCode}).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if got.Location() != time.UTC {
				t.Errorf("connection %d: scanned timestamptz location = %q, want time.UTC", i, got.Location())
			}
			if !got.Equal(want) {
				t.Errorf("connection %d: scanned instant = %s, want %s", i, got.Format(time.RFC3339Nano), want.Format(time.RFC3339Nano))
			}
		}
	})

	t.Run("existing_after_connect", func(t *testing.T) {
		callbackZone := time.FixedZone("callback+3", 3*60*60)
		for _, tc := range []struct {
			name         string
			wantErr      error
			wantLocation *time.Location
		}{
			{name: "success", wantLocation: time.UTC},
			{name: "error", wantErr: errors.New("TEST ONLY connection setup failure"), wantLocation: callbackZone},
		} {
			t.Run(tc.name, func(t *testing.T) {
				// A fresh, unpooled connection starts with pgx's default codec and
				// cannot pass codec mutations to another case or pool borrower.
				conn, err := pgx.ConnectConfig(ctx, pool.Config().ConnConfig.Copy())
				if err != nil {
					t.Fatal("connect isolated callback connection")
				}
				t.Cleanup(func() {
					closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					if err := conn.Close(closeCtx); err != nil {
						t.Error("close isolated callback connection")
					}
				})
				calls := 0
				cfg := &pgxpool.Config{AfterConnect: func(gotCtx context.Context, gotConn *pgx.Conn) error {
					calls++
					if gotCtx != ctx || gotConn != conn {
						t.Error("existing AfterConnect received different context or connection")
					}
					gotConn.TypeMap().RegisterType(&pgtype.Type{
						Name:  "timestamptz",
						OID:   pgtype.TimestamptzOID,
						Codec: &pgtype.TimestamptzCodec{ScanLocation: callbackZone},
					})
					return tc.wantErr
				}}
				db.ConfigureUTC(cfg)
				if err := cfg.AfterConnect(ctx, conn); !errors.Is(err, tc.wantErr) || calls != 1 {
					t.Errorf("existing AfterConnect: calls=%d error=%v, want calls=1 error=%v", calls, err, tc.wantErr)
				}
				var got time.Time
				if err := conn.QueryRow(ctx, `SELECT occurred_at FROM utc_times`, pgx.QueryResultFormats{pgx.BinaryFormatCode}).Scan(&got); err != nil {
					t.Fatal(err)
				}
				if got.Location() != tc.wantLocation {
					t.Errorf("after existing AfterConnect: scanned timestamptz location = %q, want %q", got.Location(), tc.wantLocation)
				}
				if !got.Equal(want) {
					t.Errorf("after existing AfterConnect: scanned instant = %s, want %s", got.Format(time.RFC3339Nano), want.Format(time.RFC3339Nano))
				}
			})
		}
	})
}
