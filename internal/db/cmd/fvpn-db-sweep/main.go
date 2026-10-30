// fvpn-db-sweep drops leaked isolated test databases older than one hour.
// It reads FVPN_TEST_DATABASE_URL and never prints that connection string.
// Legacy fvpn_<32 hex> names are included only when FVPN_TEST_DB_SWEEP_LEGACY=1.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
)

func main() { os.Exit(run()) }

func run() int {
	dsn := os.Getenv("FVPN_TEST_DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "FVPN_TEST_DATABASE_URL is required")
		return 2
	}
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(signalCtx, 10*time.Minute)
	defer cancel()
	pool, err := db.Open(ctx, dsn, db.OpenOptions{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "database connection failed")
		return 1
	}
	defer pool.Close()
	if _, err = dbtest.Sweep(ctx, pool, time.Hour); err != nil {
		fmt.Fprintln(os.Stderr, "sweep failed")
		return 1
	}
	return 0
}
