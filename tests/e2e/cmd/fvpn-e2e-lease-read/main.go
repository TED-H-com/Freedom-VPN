// Command fvpn-e2e-lease-read reads the live SQLite lease inside the owned node.
// TEST ONLY content-free readback; no store mutation or envelope export.
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db, err := sql.Open("sqlite", "file:/var/lib/vpn-node/state/agent.db?mode=ro&_pragma=query_only(1)&_pragma=busy_timeout(1000)")
	if err != nil {
		os.Exit(1)
	}
	defer db.Close()
	var id string
	var raw []byte
	if db.QueryRowContext(ctx, `SELECT lease_id,envelope FROM lease WHERE id=1`).Scan(&id, &raw) != nil {
		os.Exit(1)
	}
	if json.NewEncoder(os.Stdout).Encode(map[string]any{"lease_id": id, "body_sha256": fmt.Sprintf("%x", sha256.Sum256(raw)), "bytes": len(raw), "read_at": time.Now().UTC()}) != nil {
		os.Exit(1)
	}
}
