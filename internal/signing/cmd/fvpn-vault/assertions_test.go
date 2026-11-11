package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

// Every selected record must exist exactly once, so an omitted or duplicated
// record cannot make an assertion inside a loop silently disappear.
func requireOne[T any](t *testing.T, values []T, label string, match func(T) bool) T {
	t.Helper()
	var found T
	count := 0
	for _, value := range values {
		if match(value) {
			count++
			found = value
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one %s, got %d", label, count)
	}
	return found
}
func requirePurpose(t *testing.T, r inspection, purpose signing.Purpose) purposeSummary {
	t.Helper()
	return requireOne(t, r.Purposes, "inspection purpose", func(p purposeSummary) bool { return p.Purpose == purpose })
}
func requireExportPurpose(t *testing.T, r exportReceipt, purpose signing.Purpose) exportedPurpose {
	t.Helper()
	return requireOne(t, r.Purposes, "export purpose", func(p exportedPurpose) bool { return p.Purpose == purpose })
}
func requireRecord(t *testing.T, records []signing.KeyRecord, id string) signing.KeyRecord {
	t.Helper()
	return requireOne(t, records, "snapshot key", func(k signing.KeyRecord) bool { return k.ID == id })
}
func requireInspectionInventory(t *testing.T, r inspection, records []signing.KeyRecord) {
	t.Helper()
	if len(r.Keys) != len(records) || len(r.Purposes) != len(purposes()) {
		t.Fatal("inspection inventory differs from ledger")
	}
	for _, purpose := range purposes() {
		requirePurpose(t, r, purpose)
	}
	for _, record := range records {
		k := requireOne(t, r.Keys, "inspection key", func(k keySummary) bool { return k.ID == record.ID })
		fingerprint := ""
		if len(record.Public) != 0 {
			fingerprint = digest(record.Public)
		}
		if k.Purpose != record.Purpose || k.State != record.State || k.PublicSHA256 != fingerprint || !k.NotBefore.Equal(record.NotBefore) || !k.NotAfter.Equal(record.NotAfter) {
			t.Fatal("inspection public key record differs")
		}
	}
}
func requireExportInventory(t *testing.T, r exportReceipt, records []signing.KeyRecord) {
	t.Helper()
	nodePurposes := []signing.Purpose{signing.DesiredState, signing.Lease, signing.Recovery, signing.Release}
	if len(r.Purposes) != len(nodePurposes) {
		t.Fatal("export purpose inventory incomplete")
	}
	for _, purpose := range nodePurposes {
		requireExportPurpose(t, r, purpose)
	}
	count := 0
	for _, record := range records {
		purpose := envelope.Purpose(record.Purpose)
		if !purpose.Valid() || record.Purpose == signing.LedgerEpoch {
			continue
		}
		count++
		k := requireOne(t, r.Keys, "exported key", func(k exportedKey) bool { return k.ID == record.ID })
		if k.Purpose != record.Purpose || k.PublicSHA256 != digest(record.Public) || !k.NotBefore.Equal(record.NotBefore) || !k.NotAfter.Equal(record.NotAfter) || k.Revoked != (record.State == "revoked") {
			t.Fatal("exported public key record differs")
		}
	}
	if len(r.Keys) != count {
		t.Fatal("export key inventory differs from ledger")
	}
}
func requireReportInventory(t *testing.T, dir string, raw []byte) {
	t.Helper()
	state, err := os.ReadFile(filepath.Join(dir, "state.json"))
	must(t, err)
	var ledger testLedger
	must(t, json.Unmarshal(state, &ledger))
	var header struct {
		Kind string `json:"kind"`
	}
	must(t, json.Unmarshal(raw, &header))
	switch header.Kind {
	case "fvpn-file-vault-inspect-v1":
		var r inspection
		must(t, json.Unmarshal(raw, &r))
		requireInspectionInventory(t, r, ledger.Keys)
	case "fvpn-file-vault-export-trust-v1":
		var r exportReceipt
		must(t, json.Unmarshal(raw, &r))
		requireExportInventory(t, r, ledger.Keys)
	default:
		t.Fatal("unexpected result kind")
	}
}
func requireTrustedRecord(t *testing.T, raw []byte, record signing.KeyRecord) {
	t.Helper()
	// The parser rejects duplicate IDs. Lookup therefore proves exactly one
	// retained record, with all of its expected node-trust attributes.
	trust, err := envelope.ParseNodeTrustStoreJSON(raw)
	must(t, err)
	k, ok := trust.Lookup(record.ID)
	if !ok || len(k.Purposes) != 1 || k.Purposes[0] != envelope.Purpose(record.Purpose) || !bytes.Equal(k.PublicKey, record.Public) || !k.NotBefore.Equal(record.NotBefore) || !k.NotAfter.Equal(record.NotAfter) || k.Revoked != (record.State == "revoked") {
		t.Fatal("retained bundle record missing or changed")
	}
}
