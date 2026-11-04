// leaseprobe reads only the accepted lease from the test node's local store.
// It emits the signed public fields and monotonic request stamp, never identity
// keys, envelopes' signatures, database errors or filesystem paths.
package main

import (
	"database/sql"
	"encoding/json"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	if !read() {
		os.Exit(1)
	}
}

func read() bool {
	db, err := sql.Open("sqlite", "file:/var/lib/vpn-node/state/agent.db?mode=ro")
	if err != nil {
		return false
	}
	defer db.Close()
	var raw []byte
	var sent int64
	if db.QueryRow(`SELECT envelope, mono_sent_ns FROM lease WHERE id=1`).Scan(&raw, &sent) != nil {
		return false
	}
	var envelope struct {
		Signed json.RawMessage `json:"signed"`
	}
	if json.Unmarshal(raw, &envelope) != nil || !json.Valid(envelope.Signed) || sent <= 0 {
		return false
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		Signed json.RawMessage `json:"signed"`
		Sent   int64           `json:"request_monotonic_ns,string"`
	}{envelope.Signed, sent}) == nil
}
