package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

func changeAdd(f *trustFixture, change func(map[string]any)) {
	var doc map[string]any
	if err := json.Unmarshal(readTestFile(f.t, f.path("B.trust")), &doc); err != nil {
		f.t.Fatal(err)
	}
	change(doc)
	writeTrustTest(f.t, f.path("B.trust"), jsonTrustTest(f.t, doc))
}
func addKey(doc map[string]any) map[string]any { return doc["keys"].([]any)[0].(map[string]any) }

func TestUPD02_TrustProducerRefusals(t *testing.T) {
	type refusal struct {
		name    string
		code    int
		prepare func(*trustFixture, map[string]string) []byte // optional matching centre refusal
	}
	cases := []refusal{
		{"unknown_signer", 1, func(f *trustFixture, flags map[string]string) []byte {
			flags["key"] = f.path("C.key")
			return f.signed(f.update(f.record("B")), "C")
		}},
		{"mismatched_signer_public", 1, func(f *trustFixture, flags map[string]string) []byte {
			k := f.keys["C"]
			k.KeyID = "A"
			if err := k.Save(f.path("mismatch.key")); err != nil {
				t.Fatal(err)
			}
			f.keys["mismatch"] = k
			flags["key"] = f.path("mismatch.key")
			return f.signed(f.update(f.record("B")), "mismatch")
		}},
		{"revoked_signer", 1, func(f *trustFixture, flags map[string]string) []byte {
			k := f.record("A")
			at := trustDate(trustAt)
			k.State, k.RevokedAt = "revoked", &at
			f.setRecords(k)
			return f.signed(f.update(f.record("B")), "A")
		}},
		{"not_yet_valid_signer", 1, func(f *trustFixture, flags map[string]string) []byte {
			k := f.record("A")
			k.NotBefore = trustDate(trustNarrowEnd)
			f.setRecords(k)
			return f.signed(f.update(f.record("B")), "A")
		}},
		{"expired_signer", 1, func(f *trustFixture, flags map[string]string) []byte {
			k := f.record("A")
			k.NotAfter = trustDate("2001-01-01T00:00:00Z")
			f.setRecords(k)
			return f.signed(f.update(f.record("B")), "A")
		}},
		{"add_existing_id", 1, func(f *trustFixture, flags map[string]string) []byte {
			changeAdd(f, func(d map[string]any) { addKey(d)["key_id"] = "A" })
			k := f.record("B")
			k.ID = "A"
			return f.signed(f.update(k), "A")
		}},
		{"add_existing_same_key", 1, func(f *trustFixture, flags map[string]string) []byte { flags["add"] = f.path("A.trust"); return nil }},
		{"add_existing_public", 1, func(f *trustFixture, flags map[string]string) []byte {
			var d map[string]any
			if err := json.Unmarshal(readTestFile(t, f.path("A.trust")), &d); err != nil {
				t.Fatal(err)
			}
			changeAdd(f, func(v map[string]any) { addKey(v)["public_key_b64"] = addKey(d)["public_key_b64"] })
			k := f.record("A")
			k.ID = "B"
			return f.signed(f.update(k), "A")
		}},
		{"add_empty", 1, func(f *trustFixture, flags map[string]string) []byte {
			changeAdd(f, func(d map[string]any) { d["keys"] = []any{} })
			return nil
		}},
		{"add_two_keys", 1, func(f *trustFixture, flags map[string]string) []byte {
			var other map[string]any
			if err := json.Unmarshal(readTestFile(t, f.path("C.trust")), &other); err != nil {
				t.Fatal(err)
			}
			changeAdd(f, func(d map[string]any) { d["keys"] = append(d["keys"].([]any), addKey(other)) })
			return nil
		}},
		{"add_wrong_purpose", 1, func(f *trustFixture, flags map[string]string) []byte {
			changeAdd(f, func(d map[string]any) { addKey(d)["purposes"] = []string{"lease"} })
			k := f.record("B")
			k.Purpose = envelope.PurposeLease
			return f.signed(f.update(k), "A")
		}},
		{"add_extra_purpose", 1, func(f *trustFixture, flags map[string]string) []byte {
			changeAdd(f, func(d map[string]any) { addKey(d)["purposes"] = []string{"release", "lease"} })
			return nil
		}},
		{"add_revoked", 1, func(f *trustFixture, flags map[string]string) []byte {
			changeAdd(f, func(d map[string]any) { addKey(d)["revoked"] = true })
			return nil
		}},
		{"add_ended", 1, func(f *trustFixture, flags map[string]string) []byte {
			changeAdd(f, func(d map[string]any) { addKey(d)["not_after"] = trustAt })
			return nil
		}},
		{"retire_unknown", 1, func(f *trustFixture, flags map[string]string) []byte {
			delete(flags, "add")
			flags["retire"], flags["not-after"] = "unknown", trustEnd
			return nil
		}},
		{"revoke_unknown", 1, func(f *trustFixture, flags map[string]string) []byte {
			delete(flags, "add")
			flags["revoke"] = "unknown"
			return nil
		}},
		{"revoke_revoked", 1, func(f *trustFixture, flags map[string]string) []byte {
			k := f.record("B")
			at := trustDate(trustAt)
			k.State, k.RevokedAt = "revoked", &at
			f.setRecords(k)
			delete(flags, "add")
			flags["revoke"] = "B"
			return nil
		}},
		{"retire_revoked", 1, func(f *trustFixture, flags map[string]string) []byte {
			k := f.record("B")
			at := trustDate(trustAt)
			k.State, k.RevokedAt = "revoked", &at
			f.setRecords(k)
			delete(flags, "add")
			flags["retire"], flags["not-after"] = "B", trustEnd
			k.State, k.RevokedAt = "retired", nil
			return f.signed(f.update(k), "A")
		}},
		{"retire_missing_end", 2, func(f *trustFixture, flags map[string]string) []byte {
			delete(flags, "add")
			flags["retire"] = "A"
			return nil
		}},
		{"retire_end_expanded", 1, func(f *trustFixture, flags map[string]string) []byte {
			delete(flags, "add")
			flags["retire"], flags["not-after"] = "A", "2101-01-01T00:00:00Z"
			k := f.record("A")
			k.State, k.NotAfter = "retired", trustDate(flags["not-after"])
			return f.signed(f.update(k), "A")
		}},
		{"retire_end_equal_start", 1, func(f *trustFixture, flags map[string]string) []byte {
			delete(flags, "add")
			flags["retire"], flags["not-after"] = "A", trustStart
			k := f.record("A")
			k.State, k.NotAfter = "retired", trustDate(trustStart)
			return f.signed(f.update(k), "A")
		}},
		{"retire_end_before_start", 1, func(f *trustFixture, flags map[string]string) []byte {
			delete(flags, "add")
			flags["retire"], flags["not-after"] = "A", "1999-01-01T00:00:00Z"
			k := f.record("A")
			k.State, k.NotAfter = "retired", trustDate(flags["not-after"])
			return f.signed(f.update(k), "A")
		}},
		{"no_operation", 2, func(f *trustFixture, flags map[string]string) []byte {
			delete(flags, "add")
			return f.signed(f.update(), "A")
		}},
		{"two_operations", 2, func(f *trustFixture, flags map[string]string) []byte { flags["revoke"] = "A"; return nil }},
		{"unrelated_not_after", 2, func(f *trustFixture, flags map[string]string) []byte { flags["not-after"] = trustEnd; return nil }},
		{"unknown_flag", 2, func(f *trustFixture, flags map[string]string) []byte { flags["unknown"] = "true"; return nil }},
		{"invalid_at", 2, func(f *trustFixture, flags map[string]string) []byte { flags["at"] = "bad"; return nil }},
		{"empty_at", 2, func(f *trustFixture, flags map[string]string) []byte { flags["at"] = ""; return nil }},
		{"invalid_not_after", 2, func(f *trustFixture, flags map[string]string) []byte {
			delete(flags, "add")
			flags["retire"], flags["not-after"] = "A", "bad"
			return nil
		}},
		{"private_key_bad_purpose", 1, func(f *trustFixture, flags map[string]string) []byte {
			writeTrustTest(t, f.path("A.key"), []byte(`{"purpose":"lease","key_id":"A","seed_b64":"bad"}`))
			return nil
		}},
		{"private_key_unknown_field", 1, func(f *trustFixture, flags map[string]string) []byte {
			writeTrustTest(t, f.path("A.key"), []byte(`{"unknown":"bad"}`))
			return nil
		}},
		{"no_usable_root", 1, func(f *trustFixture, flags map[string]string) []byte {
			delete(flags, "add")
			flags["revoke"] = "A"
			return nil
		}},
	}
	for _, value := range []string{"0", "-1", "a", "01", "+1", "18446744073709551616", ""} {
		cases = append(cases, refusal{"revision_" + value, 2, func(f *trustFixture, flags map[string]string) []byte {
			flags["revision"] = value
			// Sign the invalid wire revision directly to compare with the centre.
			d := map[string]any{"kind": release.ReleaseTrustUpdateKind, "revision": value, "keys": []release.ReleaseTrustRecord{f.record("B")}}
			return f.signed(d, "A")
		}})
	}
	for _, name := range []string{"key", "trust", "revision", "out"} {
		cases = append(cases, refusal{"missing_" + name, 2, func(f *trustFixture, flags map[string]string) []byte { delete(flags, name); return nil }})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newTrustFixture(t)
			flags := f.produceFlags()
			candidate := tc.prepare(f, flags)
			f.refuse(trustArgs("trust-update", flags), tc.code)
			if _, err := os.Lstat(f.path("update.json")); !os.IsNotExist(err) {
				t.Fatal("refusal left output")
			}
			writeTrustTest(t, f.path("update.json"), []byte("existing output must remain byte-identical\n"))
			f.refuse(trustArgs("trust-update", flags), tc.code)
			if candidate != nil {
				f.rejectCentre(candidate)
			}
		})
	}
}

func TestUPD02_TrustOutputRefusals(t *testing.T) {
	for _, kind := range []string{"file", "directory", "symlink", "dangling-symlink", "missing-parent"} {
		t.Run(kind, func(t *testing.T) {
			f := newTrustFixture(t)
			flags := f.produceFlags()
			out := flags["out"]
			switch kind {
			case "file":
				writeTrustTest(t, out, []byte("existing\n"))
			case "directory":
				if err := os.Mkdir(out, 0700); err != nil {
					t.Fatal(err)
				}
				writeTrustTest(t, filepath.Join(out, "kept"), []byte("contents\n"))
			case "symlink", "dangling-symlink":
				if kind == "symlink" {
					writeTrustTest(t, f.path("target"), []byte("target\n"))
				}
				if err := os.Symlink("target", out); err != nil {
					t.Fatal(err)
				}
			case "missing-parent":
				flags["out"] = f.path("absent/update.json")
			}
			f.refuse(trustArgs("trust-update", flags), 1)
		})
	}
}

func TestUPD02_TrustUsageRefusals(t *testing.T) {
	f := newTrustFixture(t)
	flags := f.produceFlags()
	// All flag types, both spelling forms, equal values, booleans and '--'.
	for name, value := range flags {
		t.Run("repeat_"+name, func(t *testing.T) { f.refuse(append(trustArgs("trust-update", flags), "--"+name, value), 2) })
	}
	for _, args := range [][]string{
		append(trustArgs("trust-update", flags), "-allow-no-usable-root", "-allow-no-usable-root=false"),
		append(trustArgs("trust-update", flags), "stray"),
		append(trustArgs("trust-update", flags), "--", "stray"),
		append(trustArgs("trust-update", flags), "-key"),
		append(trustArgs("trust-update", flags), "-allow-no-usable-root=invalid"),
		{"trust-verify"},
		f.verifyArgs("stray"),
		f.verifyArgs("-unknown", "x"),
		f.verifyArgs("-at", trustAt),
		f.verifyArgs("-current-revision=0", "--current-revision=0"),
		f.verifyArgs("-current-revision=01"),
		f.verifyArgs("-current-revision="),
		f.verifyArgs("-current-revision=-1"),
		f.verifyArgs("-current-revision=18446744073709551616"),
	} {
		f.refuse(args, 2)
	}
	for _, command := range []string{"trust-update", "trust-verify"} {
		var out, diagnostic strings.Builder
		if code := run([]string{command, "-h"}, &out, &diagnostic); code != 0 || out.Len() != 0 || !strings.Contains(diagnostic.String(), "centre decides against its own ledger") {
			t.Fatalf("help exit %d: %s", code, &diagnostic)
		}
	}
}

func TestUPD02_TrustUsableRootGuard(t *testing.T) {
	for _, operation := range []string{"revoke", "retire"} {
		t.Run(operation, func(t *testing.T) {
			f := newTrustFixture(t)
			flags := f.produceFlags()
			delete(flags, "add")
			flags[operation] = "A"
			if operation == "retire" {
				flags["not-after"] = trustAt
			}
			f.refuse(trustArgs("trust-update", flags), 1)
			flags["allow-no-usable-root"] = "true"
			f.success(trustArgs("trust-update", flags))
			out := f.success(f.verifyArgs())
			if !strings.Contains(out, "usable_root_after=false\n") {
				t.Fatal(out)
			}
			f.apply(readTestFile(t, f.path("update.json"))) // Centre has no usable-root guard.
		})
	}
	// A future key can be added; it is not a usable replacement yet.
	f := newTrustFixture(t)
	k := f.record("B")
	k.NotBefore = trustDate(trustNarrowEnd)
	f.setRecords(k)
	flags := f.produceFlags()
	delete(flags, "add")
	flags["revoke"] = "A"
	f.refuse(trustArgs("trust-update", flags), 1)
}

func TestUPD02_TrustRemainingRepeatedFlags(t *testing.T) {
	f := newTrustFixture(t)
	for _, command := range []string{"trust-update", "trust-verify"} {
		flags := f.produceFlags()
		names := []string{"retire", "revoke", "not-after", "allow-no-usable-root"}
		if command == "trust-verify" {
			flags = map[string]string{"update": f.path("update.json"), "trust": f.path("current.trust"), "at": trustAt, "current-revision": "0"}
			names = []string{"update", "trust", "at", "current-revision"}
		}
		for _, name := range names {
			args := trustArgs(command, flags)
			if _, exists := flags[name]; !exists {
				args = append(args, "-"+name+"=true")
			}
			args = append(args, "--"+name+"=true")
			f.refuse(args, 2)
		}
	}
}
