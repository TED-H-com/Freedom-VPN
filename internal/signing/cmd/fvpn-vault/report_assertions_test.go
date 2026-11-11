package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

type reportInvocation struct {
	options        options
	started, ended time.Time
}

func fixtureSHA256(raw []byte) string { return fmt.Sprintf("%x", sha256.Sum256(raw)) }

func fixturePurposes() []signing.Purpose {
	return []signing.Purpose{signing.NodeCA, signing.BootstrapSecret, signing.DesiredState, signing.Lease, signing.Recovery, signing.LedgerEpoch, signing.Release}
}
func fixtureWindow(k signing.KeyRecord, at time.Time) string {
	if at.Before(k.NotBefore) {
		return "not_yet_valid"
	}
	if !at.Before(k.NotAfter) {
		return "expired"
	}
	return "current"
}
func fixtureMetadata(records []signing.KeyRecord, purpose signing.Purpose, at time.Time) string {
	states := map[string]bool{}
	for _, k := range records {
		if k.Purpose != purpose {
			continue
		}
		if k.State == "revoked" {
			states["revoked"] = true
			continue
		}
		if purpose != signing.Release && k.State != "active" {
			continue
		}
		states[fixtureWindow(k, at)] = true
	}
	if states["current"] {
		return "pass"
	}
	for _, status := range []string{"not_yet_valid", "expired", "revoked"} {
		if states[status] {
			return status
		}
	}
	if purpose == signing.Release {
		return "no_key"
	}
	return "no_active_key"
}
func inspectionFieldsError(r inspection, records []signing.KeyRecord) error {
	if r.Kind != "fvpn-file-vault-inspect-v1" || r.Provider != "file" || r.InspectedAt.IsZero() || r.InspectedAt.Location() != time.UTC {
		return fmt.Errorf("inspection header differs")
	}
	if len(r.Keys) != len(records) || len(r.Purposes) != len(fixturePurposes()) {
		return fmt.Errorf("inspection inventory differs")
	}
	keys := make(map[string]keySummary)
	for i, k := range r.Keys {
		if i > 0 && r.Keys[i-1].ID >= k.ID {
			return fmt.Errorf("inspection key order/uniqueness differs")
		}
		keys[k.ID] = k
	}
	uses := make(map[signing.Purpose]string)
	for _, record := range records {
		k, ok := keys[record.ID]
		fingerprint := ""
		if len(record.Public) != 0 {
			fingerprint = fixtureSHA256(record.Public)
		}
		if !ok || k.Purpose != record.Purpose || k.State != record.State || k.PublicSHA256 != fingerprint || !k.NotBefore.Equal(record.NotBefore) || !k.NotAfter.Equal(record.NotAfter) || k.NotBefore.Location() != time.UTC || k.NotAfter.Location() != time.UTC {
			return fmt.Errorf("inspection key metadata differs")
		}
		if record.State == "revoked" {
			if record.RevokedAt == nil || k.RevokedAt == nil || !k.RevokedAt.Equal(*record.RevokedAt) || k.RevokedAt.Location() != time.UTC {
				return fmt.Errorf("revoked_at differs from ledger instant or UTC")
			}
		} else if k.RevokedAt != nil {
			return fmt.Errorf("revoked_at present on non-revoked record")
		}
		window := fixtureWindow(record, r.InspectedAt)
		if k.WindowStatus != window {
			return fmt.Errorf("window_status differs from fixture")
		}
		use := "not_checked"
		if r.KeyUseChecked {
			use = "not_applicable"
			if record.Purpose != signing.Release && record.State == "active" && window == "current" {
				if k.KeyUseStatus != "passed" && k.KeyUseStatus != "failed" {
					return fmt.Errorf("eligible key use missing")
				}
				use = k.KeyUseStatus // Exact success/failure is also asserted by each probe fixture.
				uses[record.Purpose] = use
			}
		}
		if k.KeyUseStatus != use {
			return fmt.Errorf("ineligible or unchecked key claimed use")
		}
	}
	result := "pass"
	for i, purpose := range fixturePurposes() {
		p := r.Purposes[i]
		metadata := fixtureMetadata(records, purpose, r.InspectedAt)
		use := "not_checked"
		if r.KeyUseChecked {
			use = "not_applicable"
			if v, ok := uses[purpose]; ok {
				use = v
			}
		}
		outcome := "pass"
		if metadata != "pass" || (r.KeyUseChecked && purpose != signing.Release && use != "passed") {
			outcome, result = "fail", "fail"
		}
		if p.Purpose != purpose || p.MetadataStatus != metadata || p.KeyUseStatus != use || p.Result != outcome {
			return fmt.Errorf("purpose summary differs from fixture/checked keys")
		}
	}
	if r.Result != result {
		return fmt.Errorf("overall inspection result differs")
	}
	return nil
}
func fixtureTrustBytes(records []signing.KeyRecord) ([]byte, error) {
	var keys []envelope.TrustedKey
	for _, k := range records {
		p := envelope.Purpose(k.Purpose)
		if !p.Valid() || k.Purpose == signing.LedgerEpoch {
			continue
		}
		keys = append(keys, envelope.TrustedKey{KeyID: k.ID, PublicKey: k.Public, Purposes: []envelope.Purpose{p}, NotBefore: k.NotBefore, NotAfter: k.NotAfter, Revoked: k.State == "revoked"})
	}
	trust, err := envelope.NewNodeTrustStore(keys...)
	if err != nil {
		return nil, err
	}
	raw, err := json.MarshalIndent(trust, "", "  ")
	return append(raw, '\n'), err
}
func exportFieldsError(r exportReceipt, records []signing.KeyRecord) error {
	if r.Kind != "fvpn-file-vault-export-trust-v1" || r.Provider != "file" || r.ExportedAt.IsZero() || r.ExportedAt.Location() != time.UTC || r.Output == "" || r.Output != filepath.Base(r.Output) || r.KeyUseChecked || r.PrivateKeyFilesRead {
		return fmt.Errorf("export header differs")
	}
	nodePurposes := []signing.Purpose{signing.DesiredState, signing.Lease, signing.Recovery, signing.Release}
	if len(r.Purposes) != len(nodePurposes) {
		return fmt.Errorf("export purpose inventory differs")
	}
	keys := make(map[string]exportedKey)
	for i, k := range r.Keys {
		if i > 0 && r.Keys[i-1].ID >= k.ID {
			return fmt.Errorf("export key order/uniqueness differs")
		}
		keys[k.ID] = k
	}
	count := 0
	current := make(map[signing.Purpose]bool)
	for _, record := range records {
		p := envelope.Purpose(record.Purpose)
		if !p.Valid() || record.Purpose == signing.LedgerEpoch {
			continue
		}
		count++
		k, ok := keys[record.ID]
		if !ok || k.Purpose != record.Purpose || k.PublicSHA256 != fixtureSHA256(record.Public) || !k.NotBefore.Equal(record.NotBefore) || !k.NotAfter.Equal(record.NotAfter) || k.NotBefore.Location() != time.UTC || k.NotAfter.Location() != time.UTC || k.Revoked != (record.State == "revoked") {
			return fmt.Errorf("exported key differs from ledger")
		}
		if record.State != "revoked" && fixtureWindow(record, r.ExportedAt) == "current" {
			current[record.Purpose] = true
		}
	}
	if len(keys) != count {
		return fmt.Errorf("export key inventory differs")
	}
	for i, purpose := range nodePurposes {
		if r.Purposes[i].Purpose != purpose || r.Purposes[i].CurrentNonRevokedKey != current[purpose] {
			return fmt.Errorf("export purpose/window statement differs")
		}
	}
	raw, err := fixtureTrustBytes(records)
	if err != nil {
		return err
	}
	if r.SHA256 != fixtureSHA256(raw) {
		return fmt.Errorf("export hash differs from fixture serialization")
	}
	return nil
}

// Exact raw field presence matters: decoding into bool/int fields alone would
// accept a dropped false/zero field, and pointers alone would accept null.
func objectFields(raw []byte, names ...string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	if len(fields) != len(names) {
		return nil, fmt.Errorf("object field count differs")
	}
	for _, name := range names {
		v, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return nil, fmt.Errorf("missing/null %s", name)
		}
	}
	return fields, nil
}
func utcField(raw json.RawMessage) error {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return err
	}
	if !strings.HasSuffix(s, "Z") {
		return fmt.Errorf("time is not serialized in UTC with Z")
	}
	_, err := time.Parse(time.RFC3339Nano, s)
	return err
}
func arrayFields(raw []byte) ([]json.RawMessage, error) {
	if !bytes.HasPrefix(bytes.TrimSpace(raw), []byte("[")) {
		return nil, fmt.Errorf("expected non-null array")
	}
	var values []json.RawMessage
	err := json.Unmarshal(raw, &values)
	return values, err
}
func reportFieldsError(raw []byte, ledger testLedger, expected *reportInvocation) error {
	var header struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	var at time.Time
	var err error
	switch header.Kind {
	case "fvpn-file-vault-inspect-v1":
		fields, err = objectFields(raw, "kind", "provider", "inspected_at", "ledger_revision", "release_revision", "denied_identity_count", "key_use_checked", "keys", "purposes", "result")
		if err != nil {
			return err
		}
		var r inspection
		if err = json.Unmarshal(raw, &r); err != nil {
			return err
		}
		if err = inspectionFieldsError(r, ledger.Keys); err != nil {
			return err
		}
		if r.LedgerRevision != ledger.Revision || r.ReleaseRevision != ledger.ReleaseRevision || r.DeniedIdentityCount != len(ledger.DeniedIdentities) {
			return fmt.Errorf("inspection ledger metadata differs")
		}
		if expected != nil && (expected.options.subcommand != "inspect" || r.KeyUseChecked != expected.options.checkKeys) {
			return fmt.Errorf("inspection flags differ")
		}
		if err = utcField(fields["inspected_at"]); err != nil {
			return err
		}
		at = r.InspectedAt
	case "fvpn-file-vault-export-trust-v1":
		fields, err = objectFields(raw, "kind", "provider", "exported_at", "ledger_revision", "release_revision", "output", "sha256", "key_use_checked", "private_key_files_read", "keys", "purposes")
		if err != nil {
			return err
		}
		var r exportReceipt
		if err = json.Unmarshal(raw, &r); err != nil {
			return err
		}
		if err = exportFieldsError(r, ledger.Keys); err != nil {
			return err
		}
		if r.LedgerRevision != ledger.Revision || r.ReleaseRevision != ledger.ReleaseRevision {
			return fmt.Errorf("export revisions differ")
		}
		if expected != nil && (expected.options.subcommand != "export-trust" || r.Output != filepath.Base(expected.options.out)) {
			return fmt.Errorf("export output basename differs")
		}
		if err = utcField(fields["exported_at"]); err != nil {
			return err
		}
		at = r.ExportedAt
	default:
		return fmt.Errorf("unexpected report kind")
	}
	if expected != nil && (at.Before(expected.started) || at.After(expected.ended)) {
		return fmt.Errorf("report instant outside invocation")
	}
	keys, err := arrayFields(fields["keys"])
	if err != nil {
		return err
	}
	for _, rawKey := range keys {
		var key struct {
			ID string `json:"key_id"`
		}
		if err = json.Unmarshal(rawKey, &key); err != nil {
			return err
		}
		var record *signing.KeyRecord
		for i := range ledger.Keys {
			if ledger.Keys[i].ID == key.ID {
				record = &ledger.Keys[i]
				break
			}
		}
		if record == nil {
			return fmt.Errorf("unknown report key")
		}
		names := []string{"key_id", "purpose", "not_before", "not_after", "public_sha256", "revoked"}
		if header.Kind == "fvpn-file-vault-inspect-v1" {
			names = []string{"key_id", "purpose", "state", "not_before", "not_after", "window_status", "key_use_status"}
			if len(record.Public) != 0 {
				names = append(names, "public_sha256")
			}
			if record.State == "revoked" {
				names = append(names, "revoked_at")
			}
		}
		k, err := objectFields(rawKey, names...)
		if err != nil {
			return err
		}
		for _, field := range []string{"not_before", "not_after", "revoked_at"} {
			if value, ok := k[field]; ok {
				if err = utcField(value); err != nil {
					return err
				}
			}
		}
	}
	summaries, err := arrayFields(fields["purposes"])
	if err != nil {
		return err
	}
	for _, summary := range summaries {
		names := []string{"purpose", "current_non_revoked_key"}
		if header.Kind == "fvpn-file-vault-inspect-v1" {
			names = []string{"purpose", "metadata_status", "key_use_status", "result"}
		}
		if _, err = objectFields(summary, names...); err != nil {
			return err
		}
	}
	return nil
}
