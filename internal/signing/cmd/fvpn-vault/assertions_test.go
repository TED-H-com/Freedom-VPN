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
	must(t, inspectionFieldsError(r, records))
}
func requireExportInventory(t *testing.T, r exportReceipt, records []signing.KeyRecord) {
	t.Helper()
	must(t, exportFieldsError(r, records))
}
func requireReportInventory(t *testing.T, dir string, raw []byte, invocation ...reportInvocation) {
	t.Helper()
	state, err := os.ReadFile(filepath.Join(dir, "state.json"))
	must(t, err)
	var ledger testLedger
	must(t, json.Unmarshal(state, &ledger))
	var expected *reportInvocation
	if len(invocation) != 0 {
		expected = &invocation[0]
	}
	must(t, reportFieldsError(raw, ledger, expected))
	if expected != nil && expected.options.subcommand == "export-trust" {
		published, err := os.ReadFile(expected.options.out)
		must(t, err)
		want, err := fixtureTrustBytes(ledger.Keys)
		must(t, err)
		if !bytes.Equal(published, want) {
			t.Fatal("published bytes differ from fixture trust")
		}
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
