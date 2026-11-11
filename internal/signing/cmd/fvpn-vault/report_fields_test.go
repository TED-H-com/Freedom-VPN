package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/google/uuid"
)

// Exercise the shared assertions themselves against real command documents.
// A dropped false/zero field, changed valid enum, or omitted optional field must
// fail just as a changed key ID would. Frozen time makes instant mutations exact.
func TestSEC01_ReportAssertionsRejectMissingOrChangedFields(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newVault(t, true)
		_, err := f.p.Rotate(context.Background(), signing.DesiredState, time.Hour, time.Minute)
		must(t, err)
		must(t, f.p.Revoke(context.Background(), f.ids[signing.Lease]))
		must(t, f.p.DenyIdentity(context.Background(), uuid.NewString()))
		state, err := os.ReadFile(filepath.Join(f.dir, "state.json"))
		must(t, err)
		var ledger testLedger
		must(t, json.Unmarshal(state, &ledger))
		covered := make(map[string]bool)
		for _, args := range commands(f.dir, filepath.Join(t.TempDir(), "trust.json")) {
			started := time.Now().UTC()
			got := invoke(t, command{}, f.dir, args...)
			ended := time.Now().UTC()
			o, ok := parse(args)
			if !ok || !started.Equal(ended) {
				t.Fatal("mutation fixture needs valid flags and a frozen instant")
			}
			wantExit := exitResultFail // The fixture has a revoked lease with no replacement.
			prefix := "inspection"
			if o.subcommand == "export-trust" {
				wantExit, prefix = exitOK, "export"
			}
			requireCode(t, got, wantExit)
			expected := reportInvocation{options: o, started: started, ended: ended}
			must(t, reportFieldsError(got.out, ledger, &expected))
			decode := func() map[string]any {
				var doc map[string]any
				must(t, json.Unmarshal(got.out, &doc))
				return doc
			}
			reject := func(doc map[string]any, label string) {
				t.Helper()
				raw, err := json.Marshal(doc)
				must(t, err)
				if reportFieldsError(raw, ledger, &expected) == nil {
					t.Fatalf("shared assertions accepted %s", label)
				}
			}
			// Each selector returns the same object in a fresh decoded document.
			selectors := []struct {
				label        string
				selectObject func(map[string]any) map[string]any
			}{{prefix, func(doc map[string]any) map[string]any { return doc }}}
			for _, collection := range []string{"keys", "purposes"} {
				for i := range decode()[collection].([]any) {
					selectors = append(selectors, struct {
						label        string
						selectObject func(map[string]any) map[string]any
					}{prefix + "." + strings.TrimSuffix(collection, "s"), func(doc map[string]any) map[string]any {
						return doc[collection].([]any)[i].(map[string]any)
					}})
				}
			}
			for _, selector := range selectors {
				for field, value := range selector.selectObject(decode()) {
					label := selector.label + "." + field
					covered[label] = true
					doc := decode()
					delete(selector.selectObject(doc), field)
					reject(doc, "missing "+label)
					doc = decode()
					selector.selectObject(doc)[field] = changedField(t, field, value)
					reject(doc, "changed "+label)
					if field == "not_before" || field == "not_after" || field == "revoked_at" || field == "inspected_at" || field == "exported_at" {
						at, err := time.Parse(time.RFC3339Nano, value.(string))
						must(t, err)
						for _, zone := range []*time.Location{time.FixedZone("TEST ONLY offset", 3600), time.FixedZone("TEST ONLY zero", 0)} {
							doc = decode()
							selector.selectObject(doc)[field] = at.In(zone).Format("2006-01-02T15:04:05.999999999-07:00")
							reject(doc, "non-UTC serialization "+label)
						}
					}
				}
			}
			if prefix == "inspection" {
				for i, value := range decode()["keys"].([]any) {
					key := value.(map[string]any)
					if key["state"] != "revoked" {
						for _, injected := range []any{nil, started.Format(time.RFC3339Nano)} {
							doc := decode()
							doc["keys"].([]any)[i].(map[string]any)["revoked_at"] = injected
							reject(doc, "revoked_at on non-revoked record")
						}
					}
					if key["purpose"] == string(signing.BootstrapSecret) {
						doc := decode()
						doc["keys"].([]any)[i].(map[string]any)["public_sha256"] = ""
						reject(doc, "bootstrap fingerprint presence")
					}
				}
			}
		}
		// Enumerate the public schema independently of the documents: omission of
		// an entire category/optional field cannot silently skip its mutation test.
		for group, fields := range map[string]string{
			"inspection":         "kind provider inspected_at ledger_revision release_revision denied_identity_count key_use_checked keys purposes result",
			"inspection.key":     "key_id purpose state not_before not_after revoked_at public_sha256 window_status key_use_status",
			"inspection.purpose": "purpose metadata_status key_use_status result",
			"export":             "kind provider exported_at ledger_revision release_revision output sha256 key_use_checked private_key_files_read keys purposes",
			"export.key":         "key_id purpose not_before not_after revoked public_sha256",
			"export.purpose":     "purpose current_non_revoked_key",
		} {
			for _, field := range strings.Fields(fields) {
				if !covered[group+"."+field] {
					t.Fatalf("field mutation coverage missing: %s.%s", group, field)
				}
			}
		}
		if len(covered) != 42 {
			t.Fatalf("expected mutations for 42 schema fields, got %d", len(covered))
		}
	})
}

func changedField(t *testing.T, field string, value any) any {
	t.Helper()
	if field == "not_before" || field == "not_after" || field == "revoked_at" || field == "inspected_at" || field == "exported_at" {
		at, err := time.Parse(time.RFC3339Nano, value.(string))
		must(t, err)
		return at.Add(time.Nanosecond).Format(time.RFC3339Nano)
	}
	if field == "public_sha256" || field == "sha256" {
		s := value.(string)
		if s[0] == '0' {
			return "1" + s[1:]
		}
		return "0" + s[1:]
	}
	if alternatives, ok := map[string][2]string{
		"result": {"pass", "fail"}, "window_status": {"current", "expired"},
		"key_use_status": {"passed", "failed"}, "metadata_status": {"pass", "expired"},
		"purpose": {"lease", "recovery"}, "state": {"active", "retired"},
		"ledger_revision": {"0", "1"}, "release_revision": {"0", "1"},
	}[field]; ok {
		if value == alternatives[0] {
			return alternatives[1]
		}
		return alternatives[0]
	}
	switch v := value.(type) {
	case string:
		return "TEST-ONLY-changed-" + v
	case bool:
		return !v
	case float64:
		return v + 1
	case []any:
		return []any{}
	default:
		t.Fatalf("unhandled field mutation %s", field)
		return nil
	}
}
