//go:build integration

package handover_test

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
)

func TestHandover_SQLStatementsExecuteAgainstMigratedSchema(t *testing.T) {
	root := repoRoot(t)
	blocks := extractSQLBlocks(t, root)
	if len(blocks) == 0 {
		t.Fatal("expected SQL blocks in owned handover docs")
	}

	pool := dbtest.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Assert that the applied migration versions in the freshly migrated database
	// equal the files in db/migrations/*.sql (verifying the fvpn-migrate status
	// expectation without hardcoding literal migration IDs in the docs).
	wantMigrationIDs := repoMigrationIDs(t, root)
	migRows, err := pool.Query(ctx, `
		SELECT to_char(version_id, 'FM00000')
		FROM goose_db_version
		WHERE version_id > 0 AND is_applied = true
		ORDER BY version_id ASC
	`)
	if err != nil {
		t.Fatalf("query goose_db_version: %v", err)
	}
	var appliedIDs []string
	for migRows.Next() {
		var id string
		if err := migRows.Scan(&id); err != nil {
			migRows.Close()
			t.Fatalf("scan goose_db_version: %v", err)
		}
		appliedIDs = append(appliedIDs, id)
	}
	if err := migRows.Err(); err != nil {
		migRows.Close()
		t.Fatalf("iterate goose_db_version: %v", err)
	}
	migRows.Close()
	if !slices.Equal(appliedIDs, wantMigrationIDs) {
		t.Fatalf("applied migrations in goose_db_version = %v, want files from db/migrations/*.sql = %v", appliedIDs, wantMigrationIDs)
	}

	executedBlocks := 0
	executedStmts := 0
	for _, b := range blocks {
		if b.ClusterRoleDDL {
			t.Logf("[non-mechanical: cluster-role-ddl] %s SQL block #%d skipped cluster-level role DDL (D-039: schema only)", b.Doc, b.Index)
			if strings.Contains(b.Raw, "purge_retained(text, timestamptz, text)") {
				var regproc *string
				if err := pool.QueryRow(ctx, "SELECT to_regprocedure('purge_retained(text, timestamptz, text)')::text").Scan(&regproc); err != nil {
					t.Fatalf("%s SQL block #%d: query pg_proc for purge_retained: %v", b.Doc, b.Index, err)
				}
				if regproc == nil || *regproc == "" {
					t.Errorf("%s SQL block #%d: referenced function purge_retained(text, timestamptz, text) does not exist in migrated schema", b.Doc, b.Index)
				}
			}
			continue
		}

		stmts := splitSQLStatements(b.Raw)
		if len(stmts) == 0 {
			t.Errorf("%s SQL block #%d produced 0 executable SQL statements", b.Doc, b.Index)
			continue
		}
		executedBlocks++
		for j, stmt := range stmts {
			if _, err := pool.Exec(ctx, stmt); err != nil {
				t.Errorf("%s SQL block #%d statement #%d failed against migrated schema:\nSQL:\n%s\nerror: %v", b.Doc, b.Index, j+1, stmt, err)
			}
			executedStmts++
		}
	}

	if executedBlocks == 0 || executedStmts == 0 {
		t.Fatalf("expected to execute schema SQL statements; got blocks=%d stmts=%d", executedBlocks, executedStmts)
	}

	// Verify every backticked `<table>.<column>` reference in owned docs against
	// information_schema.columns when `<table>` is a public PostgreSQL table.
	tables := map[string]map[string]bool{}
	rows, err := pool.Query(ctx, `
		SELECT table_name, column_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
	`)
	if err != nil {
		t.Fatalf("query information_schema.columns: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tbl, col string
		if err := rows.Scan(&tbl, &col); err != nil {
			t.Fatalf("scan information_schema.columns: %v", err)
		}
		if tables[tbl] == nil {
			tables[tbl] = map[string]bool{}
		}
		tables[tbl][col] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate information_schema.columns: %v", err)
	}

	tableColRE := regexp.MustCompile("`([a-z][a-z0-9_]*)\\.([a-z][a-z0-9_]*)`")
	nonColumnDotted := map[string]string{
		"members.json":        "config-file-name",
		"members.reconcile":   "worker-periodic-event-type",
		"fleet_config.update": "audit-action-name",
	}
	checkedTableCols := 0
	for _, doc := range ownedDocs {
		content := readOwnedDoc(t, root, doc)
		for _, m := range tableColRE.FindAllStringSubmatch(content, -1) {
			full, tbl, col := m[1]+"."+m[2], m[1], m[2]
			if kind, ok := nonColumnDotted[full]; ok {
				t.Logf("[non-mechanical: %s] %s -> %s", kind, doc, full)
				continue
			}
			cols, isPublicTable := tables[tbl]
			if !isPublicTable {
				continue
			}
			if !cols[col] {
				t.Errorf("%s: backticked schema reference `%s.%s` names a non-existent column on table %q", doc, tbl, col, tbl)
			}
			checkedTableCols++
		}
	}
	if checkedTableCols == 0 {
		t.Fatal("expected to verify at least one `<table>.<column>` reference against information_schema.columns")
	}
}

func splitSQLStatements(raw string) []string {
	var cleanLines []string
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		cleanLines = append(cleanLines, line)
	}
	joined := strings.Join(cleanLines, "\n")
	parts := strings.Split(joined, ";")
	var out []string
	for _, p := range parts {
		stmt := strings.TrimSpace(p)
		if stmt != "" {
			out = append(out, stmt)
		}
	}
	return out
}
