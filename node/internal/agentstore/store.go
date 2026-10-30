package agentstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite (AGENTS.md §3)
)

// DefaultPath is the agent database (§4 node paths).
const DefaultPath = "/var/lib/vpn-node/state/agent.db"

// ErrStaleGeneration means a desired-state generation is older than the one
// already accepted (§13.4, §14.7). The stored envelope is left unchanged.
var ErrStaleGeneration = errors.New("agentstore: stale desired-state generation")

// Store is one open agent database. It is safe for one process; the node
// lock stops a second writer from migrating the schema.
type Store struct {
	db   *sql.DB
	path string
}

// Open creates or opens the database at path. The parent directory is 0700
// and the file is 0600. A schema newer than this binary is refused.
func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("agentstore: empty path")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	_, statErr := os.Stat(path)
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=synchronous(FULL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db, path: path}
	if err := s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	if errors.Is(statErr, os.ErrNotExist) {
		if err := os.Chmod(path, 0o600); err != nil {
			_ = db.Close()
			return nil, err
		}
		if err := syncDir(dir); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return s, nil
}

// Close closes the database.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Path returns the database file path.
func (s *Store) Path() string { return s.path }

func (s *Store) tx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

const schemaVersion = 2

func (s *Store) migrate(ctx context.Context) error {
	var v int
	if err := s.db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&v); err != nil {
		return fmt.Errorf("agentstore: schema version: %w", err)
	}
	if v > schemaVersion {
		return fmt.Errorf("agentstore: schema %d is newer than supported %d", v, schemaVersion)
	}
	if v == schemaVersion {
		return nil
	}
	stmts := []string{
		`CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL)`,
		`CREATE TABLE meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL)`,
		`CREATE TABLE enrollment (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			enrollment_request_id TEXT NOT NULL,
			node_incarnation_id TEXT NOT NULL,
			csr_pem TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			node_id TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			enrolled_at TEXT NOT NULL DEFAULT '')`,
		`CREATE TABLE identity (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			node_id TEXT NOT NULL,
			node_incarnation_id TEXT NOT NULL,
			certificate_serial TEXT NOT NULL,
			certificate_not_after TEXT NOT NULL,
			public_key_sha256 TEXT NOT NULL)`,
		`CREATE TABLE desired (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			accepted_generation INTEGER NOT NULL,
			applied_generation INTEGER,
			etag TEXT NOT NULL,
			envelope BLOB NOT NULL,
			issued_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			publication_intent TEXT NOT NULL)`,
		`CREATE TABLE lease (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			lease_id TEXT NOT NULL,
			issued_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			highest_issued_at TEXT NOT NULL,
			envelope BLOB NOT NULL,
			mono_sent_ns INTEGER NOT NULL,
			boot_id TEXT NOT NULL)`,
		`CREATE TABLE anchor (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			boot_id TEXT NOT NULL,
			server_unix_ns INTEGER NOT NULL,
			mono_ns INTEGER NOT NULL,
			wall_unix_ns INTEGER NOT NULL,
			trusted INTEGER NOT NULL)`,
		`CREATE TABLE qos_assignment (
			member_id TEXT PRIMARY KEY,
			minor INTEGER NOT NULL)`,
		`CREATE TABLE counter_reading (
			name TEXT PRIMARY KEY,
			packets INTEGER NOT NULL,
			bytes INTEGER NOT NULL)`,
		`CREATE TABLE applied_address (
			addr TEXT PRIMARY KEY)`,
		`CREATE TABLE outbox (
			report_id TEXT PRIMARY KEY,
			payload BLOB NOT NULL,
			status TEXT NOT NULL,
			attempts INTEGER NOT NULL DEFAULT 0,
			last_error TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			accepted_at TEXT NOT NULL DEFAULT '')`,
	}
	applied := fmtTime(time.Now().UTC())
	return s.tx(ctx, func(tx *sql.Tx) error {
		if v < 1 {
			for _, stmt := range stmts {
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("agentstore: migrate: %w", err)
				}
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, name, applied_at) VALUES (1, 'agent_core', ?)`, applied); err != nil {
				return err
			}
		}
		if v < 2 {
			if _, err := tx.ExecContext(ctx, `CREATE TABLE resolver_cache (
				document_id TEXT PRIMARY KEY,
				content_sha256 TEXT NOT NULL,
				body BLOB NOT NULL)`); err != nil {
				return fmt.Errorf("agentstore: migrate: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, name, applied_at) VALUES (2, 'resolver_cache', ?)`, applied); err != nil {
				return err
			}
		}
		_, err := tx.ExecContext(ctx, `PRAGMA user_version = 2`)
		return err
	})
}
