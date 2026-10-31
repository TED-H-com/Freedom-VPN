package main

import (
	"database/sql"
	"os"
)

// journalCopy takes a coherent SQLite snapshot, including WAL, through a
// read-only source connection. The copy is private test evidence, not state.
func journalCopy() error {
	const dest = "/var/lib/vpn-node/e2e/journal.snapshot.db"
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return err
	}
	db, err := sql.Open("sqlite", "file:/var/lib/vpn-node/update-journal/journal.db?mode=ro&_pragma=busy_timeout(2000)")
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
