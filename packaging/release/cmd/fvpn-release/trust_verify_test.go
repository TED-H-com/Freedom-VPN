package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

func TestUPD02_TrustVerifierRefusalsAndCentreAgreement(t *testing.T) {
	type refusal struct {
		name    string
		prepare func(*trustFixture) []byte
	}
	cases := []refusal{
		{"changed_signed_byte", func(f *trustFixture) []byte {
			return bytes.Replace(f.signed(f.update(f.record("B")), "A"), []byte(`"revision":"10"`), []byte(`"revision":"11"`), 1)
		}},
		{"changed_signature", func(f *trustFixture) []byte {
			var e envelope.Envelope
			if err := json.Unmarshal(f.signed(f.update(f.record("B")), "A"), &e); err != nil {
				t.Fatal(err)
			}
			sig := []byte(e.Signatures[0].SignatureB64)
			if sig[0] == 'A' {
				sig[0] = 'B'
			} else {
				sig[0] = 'A'
			}
			e.Signatures[0].SignatureB64 = string(sig)
			return jsonTrustTest(t, e)
		}},
		{"unknown_signer", func(f *trustFixture) []byte { return f.signed(f.update(f.record("B")), "C") }},
		{"revoked_signer", func(f *trustFixture) []byte {
			k := f.record("A")
			at := trustDate(f.at)
			k.State, k.RevokedAt = "revoked", &at
			f.setRecords(k)
			return f.signed(f.update(f.record("B")), "A")
		}},
		{"not_yet_valid_signer", func(f *trustFixture) []byte {
			k := f.record("A")
			k.NotBefore = trustDate(f.narrowEnd)
			f.setRecords(k)
			return f.signed(f.update(f.record("B")), "A")
		}},
		{"expired_signer", func(f *trustFixture) []byte {
			k := f.record("A")
			k.NotAfter = trustDate(f.start).Add(24 * time.Hour)
			f.setRecords(k)
			return f.signed(f.update(f.record("B")), "A")
		}},
		{"known_revoked_second_signature", func(f *trustFixture) []byte {
			k := f.record("B")
			at := trustDate(f.at)
			k.State, k.RevokedAt = "revoked", &at
			f.setRecords(k)
			e, err := envelope.Sign(f.update(f.record("C")), f.keys["A"].Private(), "A")
			if err != nil {
				t.Fatal(err)
			}
			if err := e.AddSignature(f.keys["B"].Private(), "B"); err != nil {
				t.Fatal(err)
			}
			return jsonTrustTest(t, e)
		}},
		{"wrong_kind", func(f *trustFixture) []byte { u := f.update(f.record("B")); u.Kind = "wrong"; return f.signed(u, "A") }},
		{"duplicate_id", func(f *trustFixture) []byte { return f.signed(f.update(f.record("B"), f.record("B")), "A") }},
		{"substituted_public", func(f *trustFixture) []byte { k := f.record("B"); k.ID = "A"; return f.signed(f.update(k), "A") }},
		{"earlier_start", func(f *trustFixture) []byte {
			k := f.record("A")
			k.NotBefore = trustDate("1999-01-01T00:00:00Z")
			return f.signed(f.update(k), "A")
		}},
		{"later_end", func(f *trustFixture) []byte {
			k := f.record("A")
			k.NotAfter = trustDate("2101-01-01T00:00:00Z")
			return f.signed(f.update(k), "A")
		}},
		{"revoked_to_active", func(f *trustFixture) []byte {
			k := f.record("B")
			at := trustDate(f.at)
			k.State, k.RevokedAt = "revoked", &at
			f.setRecords(k)
			k.State, k.RevokedAt = "active", nil
			return f.signed(f.update(k), "A")
		}},
		{"revoked_to_retired", func(f *trustFixture) []byte {
			k := f.record("B")
			at := trustDate(f.at)
			k.State, k.RevokedAt = "revoked", &at
			f.setRecords(k)
			k.State, k.RevokedAt = "retired", nil
			return f.signed(f.update(k), "A")
		}},
		{"reused_existing_public", func(f *trustFixture) []byte { k := f.record("A"); k.ID = "B"; return f.signed(f.update(k), "A") }},
		{"reused_new_public", func(f *trustFixture) []byte {
			k := f.record("B")
			k.ID = "C"
			return f.signed(f.update(f.record("B"), k), "A")
		}},
		{"empty_id", func(f *trustFixture) []byte { k := f.record("B"); k.ID = ""; return f.signed(f.update(k), "A") }},
		{"long_id", func(f *trustFixture) []byte {
			k := f.record("B")
			k.ID = strings.Repeat("b", 129)
			return f.signed(f.update(k), "A")
		}},
		{"unicode_id_byte_limit", func(f *trustFixture) []byte {
			k := f.record("B")
			k.ID = strings.Repeat("é", 65)
			return f.signed(f.update(k), "A")
		}},
		{"zero_start", func(f *trustFixture) []byte {
			k := f.record("B")
			k.NotBefore = trustDate("0001-01-01T00:00:00Z")
			return f.signed(f.update(k), "A")
		}},
		{"empty_window", func(f *trustFixture) []byte {
			k := f.record("B")
			k.NotAfter = k.NotBefore
			return f.signed(f.update(k), "A")
		}},
		{"reversed_window", func(f *trustFixture) []byte {
			k := f.record("B")
			k.NotBefore, k.NotAfter = k.NotAfter, k.NotBefore
			return f.signed(f.update(k), "A")
		}},
		{"bad_state", func(f *trustFixture) []byte {
			k := f.record("B")
			k.State = "unknown"
			return f.signed(f.update(k), "A")
		}},
		{"revoked_without_time", func(f *trustFixture) []byte {
			k := f.record("B")
			k.State = "revoked"
			return f.signed(f.update(k), "A")
		}},
		{"active_with_revocation_time", func(f *trustFixture) []byte {
			k := f.record("B")
			at := trustDate(f.at)
			k.RevokedAt = &at
			return f.signed(f.update(k), "A")
		}},
		{"bad_public_size", func(f *trustFixture) []byte {
			k := f.record("B")
			k.Public = k.Public[:31]
			return f.signed(f.update(k), "A")
		}},
		{"wrong_purpose", func(f *trustFixture) []byte {
			k := f.record("B")
			k.Purpose = envelope.PurposeLease
			return f.signed(f.update(k), "A")
		}},
		{"zero_revision", func(f *trustFixture) []byte { u := f.update(f.record("B")); u.Revision = 0; return f.signed(u, "A") }},
		{"empty_keys", func(f *trustFixture) []byte { return f.signed(f.update(), "A") }},
		{"unknown_payload_field", func(f *trustFixture) []byte {
			return f.signed(map[string]any{"kind": release.ReleaseTrustUpdateKind, "revision": "10", "keys": []release.ReleaseTrustRecord{f.record("B")}, "extra": "field"}, "A")
		}},
		{"trailing_data", func(f *trustFixture) []byte { return append(f.signed(f.update(f.record("B")), "A"), []byte(" {}")...) }},
		{"empty_file", func(f *trustFixture) []byte { return []byte{} }},
		{"duplicate_json_member", func(f *trustFixture) []byte {
			return bytes.Replace(f.signed(f.update(f.record("B")), "A"), []byte(`"revision":"10"`), []byte(`"revision":"10","revision":"10"`), 1)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newTrustFixture(t)
			raw := tc.prepare(f)
			writeTrustTest(t, f.path("update.json"), raw)
			f.refuse(f.verifyArgs(), 1)
			f.rejectCentre(raw)
		})
	}
	for _, current := range []string{"10", "11"} {
		t.Run("current_revision_"+current, func(t *testing.T) {
			f := newTrustFixture(t)
			u := f.update(f.record("B"))
			raw := f.signed(u, "A")
			writeTrustTest(t, f.path("update.json"), raw)
			f.refuse(f.verifyArgs("-current-revision", current), 1)
			// Supply the same current revision to the real centre.
			if current == "11" {
				u.Revision = 11
			}
			f.apply(f.signed(u, "A"))
			f.rejectCentre(raw)
		})
	}
}

func TestUPD02_TrustSchemaFixtures(t *testing.T) {
	files, err := filepath.Glob("../../../../contracts/examples/trust/invalid/*.json")
	if err != nil || len(files) != 6 {
		t.Fatalf("expected six invalid trust fixtures, got %d", len(files))
	}
	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			f := newTrustFixture(t)
			raw := readTestFile(t, path)
			if err := release.ValidateTrustUpdateEnvelope(raw); err == nil {
				t.Fatal("invalid fixture passed schema")
			}
			writeTrustTest(t, f.path("update.json"), raw)
			f.refuse(f.verifyArgs(), 1)
			// Re-sign the unchanged invalid payload with the runtime root so the
			// provider proves the record rule, not just an unknown signer failure.
			var e envelope.Envelope
			if err := json.Unmarshal(raw, &e); err != nil {
				t.Fatal(err)
			}
			candidate := f.signed(e.Signed, "A")
			f.rejectCentre(candidate)
			// Also feed the original bytes to a provider holding the vector's
			// matching root; no unknown-signer shortcut is involved.
			var vector struct {
				SeedHex string `json:"seed_hex"`
				KeyID   string `json:"key_id"`
			}
			if err := json.Unmarshal(readTestFile(t, "../../../../contracts/vectors/release-trust-update-signing.json"), &vector); err != nil {
				t.Fatal(err)
			}
			seed, err := hex.DecodeString(vector.SeedHex)
			if err != nil {
				t.Fatal(err)
			}
			key, err := release.KeyFromSeed(vector.KeyID, seed)
			if err != nil {
				t.Fatal(err)
			}
			provider, err := signing.InitializeFileProvider(f.path("vector-vault"))
			if err != nil {
				t.Fatal(err)
			}
			root := f.trusted("A")
			root.KeyID, root.PublicKey = key.KeyID, key.Public()
			if err := provider.PinReleaseRoot(context.Background(), root); err != nil {
				t.Fatal(err)
			}
			before := readTestFile(t, f.path("vector-vault/state.json"))
			if err := provider.UpdateReleaseTrust(context.Background(), raw); err == nil {
				t.Fatal("centre accepted original invalid fixture")
			}
			if !bytes.Equal(before, readTestFile(t, f.path("vector-vault/state.json"))) {
				t.Fatal("original fixture changed ledger")
			}
		})
	}
}

func TestUPD02_TrustLedgerOnlyDecisions(t *testing.T) {
	for _, kind := range []string{"stored_revision", "retired_to_active", "changed_revoked_at"} {
		t.Run(kind, func(t *testing.T) {
			f := newTrustFixture(t)
			k := f.record("B")
			at := trustDate(f.at)
			switch kind {
			case "retired_to_active":
				k.State = "retired"
			case "changed_revoked_at":
				k.State, k.RevokedAt = "revoked", &at
			}
			f.setRecords(k)
			u := f.update(k)
			switch kind {
			case "stored_revision":
				u.Revision = 1
			case "retired_to_active":
				u.Keys[0].State = "active"
			case "changed_revoked_at":
				changed := at.Add(time.Second)
				u.Keys[0].RevokedAt = &changed
			}
			raw := f.signed(u, "A")
			writeTrustTest(t, f.path("update.json"), raw)
			f.success(f.verifyArgs()) // Public bundle cannot reveal these facts.
			f.rejectCentre(raw)
		})
	}
}

func TestUPD02_TrustSignerReportingAndBoundaries(t *testing.T) {
	f := newTrustFixture(t)
	f.setRecords(f.record("B"))
	e, err := envelope.Sign(f.update(f.record("C")), f.keys["A"].Private(), "A")
	if err != nil {
		t.Fatal(err)
	}
	if err := e.AddSignature(f.keys["B"].Private(), "B"); err != nil {
		t.Fatal(err)
	}
	if err := e.AddSignature(f.keys["C"].Private(), "C"); err != nil {
		t.Fatal(err)
	}
	raw := jsonTrustTest(t, e)
	writeTrustTest(t, f.path("update.json"), raw)
	out := f.success(f.verifyArgs())
	if !strings.Contains(out, "signer_key_ids=A,B\n") {
		t.Fatal(out)
	}
	f.apply(raw)
	// at == not_before passes; at == not_after fails (half-open window).
	for _, at := range []string{f.start, f.end} {
		args := []string{"trust-verify", "-update", f.path("update.json"), "-trust", f.path("current.trust"), "-at", at}
		if at == f.start {
			f.success(args)
		} else {
			f.refuse(args, 1)
		}
	}
	// Validation remains specific to the new schema; legacy sign/verify cannot
	// be used as generic signing or verification escape hatches.
	if err := release.ValidateEnvelope(raw); err == nil {
		t.Fatal("trust update passed manifest schema")
	}
	if _, _, err := release.Sign(e.Signed, f.keys["A"]); err == nil {
		t.Fatal("manifest Sign signed trust update")
	}
	ts := f.export()
	if _, _, err := release.Verify(raw, ts, trustDate(f.at)); err == nil {
		t.Fatal("manifest Verify accepted trust update")
	}
}

func TestSEC01_TrustPrivateKeyErrorsDoNotEchoSeed(t *testing.T) {
	f := newTrustFixture(t)
	// The loader reports unknown field names; CLI must redact even a malicious
	// file that puts a private seed in a field name or purpose value.
	keyRaw := readTestFile(t, f.path("A.key"))
	var doc map[string]any
	if err := json.Unmarshal(keyRaw, &doc); err != nil {
		t.Fatal(err)
	}
	seed := doc["seed_b64"].(string)
	doc[seed] = "unknown"
	writeTrustTest(t, f.path("A.key"), jsonTrustTest(t, doc))
	f.refuse(trustArgs("trust-update", f.produceFlags()), 1)
	delete(doc, seed)
	doc["purpose"] = seed
	writeTrustTest(t, f.path("A.key"), jsonTrustTest(t, doc))
	f.refuse(trustArgs("trust-update", f.produceFlags()), 1)
	if _, err := os.Stat(f.path("update.json")); !os.IsNotExist(err) {
		t.Fatal("key error left output")
	}
}

func TestUPD02_TrustSignatureStructure(t *testing.T) {
	for _, kind := range []string{"duplicate_signer", "unknown_algorithm", "malformed_unknown_signature", "bad_known_second_signature"} {
		t.Run(kind, func(t *testing.T) {
			f := newTrustFixture(t)
			f.setRecords(f.record("B"))
			e, err := envelope.Sign(f.update(f.record("C")), f.keys["A"].Private(), "A")
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "duplicate_signer":
				e.Signatures = append(e.Signatures, e.Signatures[0])
			case "unknown_algorithm":
				e.Signatures[0].Algorithm = "unknown"
			case "malformed_unknown_signature":
				e.Signatures = append(e.Signatures, envelope.Signature{KeyID: "unknown", Algorithm: envelope.AlgorithmEd25519, SignatureB64: "invalid"})
			case "bad_known_second_signature":
				if err := e.AddSignature(f.keys["C"].Private(), "B"); err != nil {
					t.Fatal(err)
				}
			}
			raw := jsonTrustTest(t, e)
			writeTrustTest(t, f.path("update.json"), raw)
			f.refuse(f.verifyArgs(), 1)
			f.rejectCentre(raw)
		})
	}
}

func TestUPD02_TrustMixedBundleAndFutureAddition(t *testing.T) {
	f := newTrustFixture(t)
	// Model a full node bundle export. Other purposes never count as roots.
	a, lease := f.trusted("A"), f.trusted("C")
	lease.Purposes = []envelope.Purpose{envelope.PurposeLease}
	ts, err := envelope.NewNodeTrustStore(a, lease)
	if err != nil {
		t.Fatal(err)
	}
	writeTrustTest(t, f.path("current.trust"), jsonTrustTest(t, ts))
	flags := f.produceFlags()
	changeAdd(f, func(d map[string]any) { addKey(d)["not_before"] = f.narrowEnd })
	f.success(trustArgs("trust-update", flags))
	f.success(f.verifyArgs())
	f.apply(readTestFile(t, f.path("update.json")))
	if err := os.Remove(f.path("update.json")); err != nil {
		t.Fatal(err)
	}
	// An unchanged end is a valid explicit retirement, and still usable.
	delete(flags, "add")
	flags["retire"], flags["not-after"] = "A", f.end
	f.success(trustArgs("trust-update", flags))
	if out := f.success(f.verifyArgs()); !strings.Contains(out, `"window_narrowed":false`) || !strings.Contains(out, "usable_root_after=true\n") {
		t.Fatal(out)
	}
	if err := os.Remove(f.path("update.json")); err != nil {
		t.Fatal(err)
	}
	delete(flags, "retire")
	delete(flags, "not-after")
	flags["revoke"] = "A"
	f.refuse(trustArgs("trust-update", flags), 1)
	// A lease key is also unavailable as a release signer or target.
	flags["key"], flags["revoke"] = f.path("C.key"), "A"
	f.refuse(trustArgs("trust-update", flags), 1)
	flags["key"], flags["revoke"] = f.path("A.key"), "C"
	f.refuse(trustArgs("trust-update", flags), 1)
}
