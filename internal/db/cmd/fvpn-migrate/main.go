// fvpn-migrate uses FVPN_DATABASE_URL and never prints connection credentials.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
)

func main() { os.Exit(run()) }
func run() int {
	if len(os.Args) != 2 || (os.Args[1] != "up" && os.Args[1] != "status") {
		fmt.Fprintln(os.Stderr, "usage: fvpn-migrate up|status (FVPN_DATABASE_URL required)")
		return 2
	}
	dsn := os.Getenv("FVPN_DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "FVPN_DATABASE_URL is required")
		return 2
	}
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(signalCtx, 10*time.Minute)
	defer cancel()
	pool, err := db.Open(ctx, dsn, db.OpenOptions{StatementTimeout: 5 * time.Minute, LockTimeout: time.Minute})
	if err != nil {
		fmt.Fprintln(os.Stderr, "database connection failed")
		return 1
	}
	defer pool.Close()
	var database string
	if err = pool.QueryRow(ctx, "SELECT current_database()").Scan(&database); err != nil {
		fmt.Fprintln(os.Stderr, "database identity query failed")
		return 1
	}
	fmt.Printf("current_database=%q\n", database)
	if os.Args[1] == "up" {
		if err = db.Migrate(ctx, pool); err != nil {
			fmt.Fprintln(os.Stderr, "migration failed; inspect database diagnostics using authorized tooling")
			return 1
		}
		fmt.Println("migrations applied")
		return 0
	}
	status, err := db.MigrationStatus(ctx, pool)
	if err != nil {
		fmt.Fprintln(os.Stderr, "migration status failed")
		return 1
	}
	for _, s := range status {
		fmt.Printf("%05d %s\n", s.Source.Version, s.State)
	}
	return 0
}
