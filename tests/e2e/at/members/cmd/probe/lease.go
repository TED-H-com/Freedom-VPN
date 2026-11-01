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

// Read only the local stored lease. Never emit the signed envelope or members.
func storedLease() {
	out := map[string]any{"available": false, "read_at": time.Now().UTC()}
	defer func() { _ = json.NewEncoder(os.Stdout).Encode(out) }()
	db, e := sql.Open("sqlite", "file:"+os.Args[2]+"?mode=ro&_pragma=busy_timeout(3000)")
	if e != nil {
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var id, issued, expires, highest, boot string
	var mono int64
	var envelope []byte
	e = db.QueryRowContext(ctx, "SELECT lease_id,issued_at,expires_at,highest_issued_at,envelope,mono_sent_ns,boot_id FROM lease WHERE id=1").Scan(&id, &issued, &expires, &highest, &envelope, &mono, &boot)
	if e == sql.ErrNoRows {
		out["available"] = true
		out["present"] = false
		return
	}
	if e != nil {
		return
	}
	out = map[string]any{"available": true, "present": true, "read_at": time.Now().UTC(), "lease_id": id, "issued_at": issued, "expires_at": expires, "highest_issued_at": highest, "boot_id": boot, "mono_sent_ns": fmt.Sprint(mono), "envelope_sha256": fmt.Sprintf("%x", sha256.Sum256(envelope))}
}
