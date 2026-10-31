package main

import (
	"database/sql"
	"os"
)

// journalCopy takes a coherent SQLite snapshot, including WAL, through a
// read-only source connection. The copy is private test evidence, not state.
func journalCopy() error {
	return sqliteCopy("/var/lib/vpn-node/update-journal/journal.db", "/var/lib/vpn-node/e2e/journal.snapshot.db")
}

func usageCopy() error {
	return sqliteCopy("/var/lib/vpn-node/state/usage.db", "/var/lib/vpn-node/e2e/usage.snapshot.db")
}

func sqliteCopy(source, dest string) error {
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return err
	}
	db, err := sql.Open("sqlite", "file:"+source+"?mode=ro&_pragma=busy_timeout(2000)")
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec("VACUUM INTO '" + dest + "'")
	if err != nil {
		return err
	}
	return os.Chmod(dest, 0600)
}
