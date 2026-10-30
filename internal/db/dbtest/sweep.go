package dbtest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgxpool"
)

// sweepLog receives one line per Sweep call. Tests replace it.
var sweepLog io.Writer = os.Stderr

var errDatabaseBusy = errors.New("database has connections")

// Sweep drops leaked isolated test databases visible from admin.
//
// A new-style name is fvpn_t<unix seconds hex>_<32 hex>. It is dropped only
// when its encoded timestamp is strictly older than olderThan and
// pg_stat_activity shows no connections to it. A legacy name fvpn_<32 hex>
// is dropped only when FVPN_TEST_DB_SWEEP_LEGACY=1 and it is idle. The admin
// database, templates, and every other name are left alone.
//
// The zero-connection check and DROP DATABASE … WITH (FORCE) are sent together
// and are not waited on separately. FORCE runs only when that check passes.
// A database that still has a connection is skipped, not forced.
func Sweep(ctx context.Context, admin *pgxpool.Pool, olderThan time.Duration) (dropped []string, err error) {
	return sweep(ctx, admin, olderThan, time.Now(), nil)
}

// sweep is Sweep with an explicit clock. allow, when non-nil, restricts the
// scan to those names so a test on the shared server cannot drop databases it
// did not create. A nil allow considers every eligible name.
func sweep(ctx context.Context, admin *pgxpool.Pool, olderThan time.Duration, now time.Time, allow map[string]struct{}) (dropped []string, err error) {
	skippedBusy := 0
	defer func() {
		fmt.Fprintf(sweepLog, "dbtest sweep: dropped=%d skipped_busy=%d\n", len(dropped), skippedBusy)
	}()
	if admin == nil {
		return nil, errors.New("missing admin connection")
	}
	if olderThan < 0 {
		return nil, errors.New("negative sweep age")
	}
	names, err := listDatabases(ctx, admin)
	if err != nil {
		return nil, err
	}
	legacy := os.Getenv("FVPN_TEST_DB_SWEEP_LEGACY") == "1"
	var targets []string
	for _, name := range names {
		if allow != nil {
			if _, ok := allow[name]; !ok {
				continue
			}
		}
		if sweepCandidate(name, now, olderThan, legacy) {
			targets = append(targets, name)
		}
	}
	if len(targets) == 0 {
		return nil, nil
	}
	acquired, err := admin.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer acquired.Release()
	failed := false
	defer func() {
		if failed {
			acquired.Conn().Close(ctx)
		}
	}()
	pg := acquired.Conn().PgConn()
	for _, name := range targets {
		if err := ctx.Err(); err != nil {
			failed = true
			return dropped, err
		}
		err := dropIfIdle(ctx, pg, name)
		if errors.Is(err, errDatabaseBusy) {
			skippedBusy++
			continue
		}
		if err != nil {
			failed = true
			return dropped, err
		}
		dropped = append(dropped, name)
	}
	return dropped, nil
}

func listDatabases(ctx context.Context, admin *pgxpool.Pool) ([]string, error) {
	rows, err := admin.Query(ctx, `
		SELECT datname
		FROM pg_database
		WHERE NOT datistemplate
		  AND datname <> current_database()
		ORDER BY datname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	slices.Sort(names)
	return names, nil
}

// dropIfIdle sends the connection check and the drop in one flush.
//
// PostgreSQL runs every simple Query string as one transaction and rejects
// DROP DATABASE inside that transaction or as a later pipeline command. The
// check therefore commits on its own statement, and the drop is the next Query
// message in the same flush. When the check finds a connection it aborts its
// transaction, so the already-sent DROP is rejected and the database stays.
// ROLLBACK then returns the session to idle. FORCE applies only to the drop
// that runs after a passed check.
func dropIfIdle(ctx context.Context, pg *pgconn.PgConn, name string) error {
	if !newNamePattern.MatchString(name) && !legacyNamePattern.MatchString(name) {
		return errors.New("refusing to drop database with unexpected name")
	}
	ident := pgx.Identifier{name}.Sanitize()
	literal := "'" + strings.ReplaceAll(name, "'", "''") + "'"
	check := `BEGIN;
DO $sweep$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_stat_activity
    WHERE datname = ` + literal + ` AND pid <> pg_backend_pid()
  ) THEN
    RAISE EXCEPTION 'database has connections' USING ERRCODE = '55006';
  END IF;
END
$sweep$;
COMMIT;`
	drop := `DROP DATABASE ` + ident + ` WITH (FORCE);`
	reset := `ROLLBACK;`

	netConn := pg.Conn()
	deadline := time.Now().Add(30 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err := netConn.SetDeadline(deadline); err != nil {
		return err
	}
	defer netConn.SetDeadline(time.Time{})

	fe := pg.Frontend()
	fe.SendQuery(&pgproto3.Query{String: check})
	fe.SendQuery(&pgproto3.Query{String: drop})
	fe.SendQuery(&pgproto3.Query{String: reset})
	if err := fe.Flush(); err != nil {
		return err
	}
	checkErr, _, err := readQuery(fe)
	if err != nil {
		return err
	}
	dropErr, _, err := readQuery(fe)
	if err != nil {
		return err
	}
	resetErr, resetStatus, err := readQuery(fe)
	if err != nil {
		return err
	}
	if resetStatus != 'I' || resetErr != nil {
		if resetErr != nil {
			return fmt.Errorf("reset admin connection: %w", resetErr)
		}
		return errors.New("admin connection not idle after sweep batch")
	}
	if checkErr == nil && dropErr == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(checkErr, &pgErr) && pgErr.Code == "55006" {
		if dropErr == nil {
			return fmt.Errorf("dropped %s despite active connections", name)
		}
		return errDatabaseBusy
	}
	if checkErr != nil {
		return fmt.Errorf("connection check for %s: %w", name, checkErr)
	}
	return fmt.Errorf("drop %s: %w", name, dropErr)
}

func readQuery(fe *pgproto3.Frontend) (error, byte, error) {
	var first error
	for {
		msg, err := fe.Receive()
		if err != nil {
			return nil, 0, err
		}
		switch m := msg.(type) {
		case *pgproto3.ErrorResponse:
			if first == nil {
				first = pgconn.ErrorResponseToPgError(m)
			}
		case *pgproto3.ReadyForQuery:
			return first, m.TxStatus, nil
		}
	}
}
