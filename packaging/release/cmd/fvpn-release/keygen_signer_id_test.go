package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/packaging/release"
)

func TestUPD02_KeygenRefusesUnsignableID(t *testing.T) {
	cases := []struct{ name, id string }{
		{"space", "release B"},
		{"leading_dot", ".B"},
		{"leading_hyphen", "-B"},
		{"leading_underscore", "_B"},
		{"leading_colon", ":B"},
		{"non_ascii", "é"},
		{"too_long", strings.Repeat("a", 129)},
		{"newline", "release\nB"},
		{"empty", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, state := range []struct {
				name                   string
				keyExists, trustExists bool
			}{
				{"absent", false, false},
				{"existing", true, true},
				{"key_only", true, false},
				{"trust_only", false, true},
			} {
				t.Run(state.name, func(t *testing.T) {
					dir := t.TempDir()
					paths := []string{filepath.Join(dir, "key.json"), filepath.Join(dir, "trust.json")}
					exists := []bool{state.keyExists, state.trustExists}
					before := make([]os.FileInfo, len(paths))
					// TEST ONLY sentinel files contain no private material.
					contents := [][]byte{[]byte("existing key sentinel\n"), []byte("existing trust sentinel\n")}
					stamp := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)
					for i, path := range paths {
						if !exists[i] {
							continue
						}
						if err := os.WriteFile(path, contents[i], 0o640); err != nil {
							t.Fatal(err)
						}
						if err := os.Chtimes(path, stamp, stamp); err != nil {
							t.Fatal(err)
						}
						var err error
						before[i], err = os.Stat(path)
						if err != nil {
							t.Fatal(err)
						}
					}
					// A create-and-delete temporary file also changes the parent mtime.
					if err := os.Chtimes(dir, stamp, stamp); err != nil {
						t.Fatal(err)
					}
					var stdout, stderr bytes.Buffer
					code := run([]string{"keygen", "-key-id", tc.id, "-out", paths[0], "-trust-out", paths[1], "-not-before", testNotBefore, "-not-after", testNotAfter}, &stdout, &stderr)
					want := "fvpn-release: -key-id cannot sign: must match " + trustSignerIDPattern + "\n"
					if tc.id == "" {
						want = "fvpn-release: keygen requires -key-id and -out\n"
					}
					if code != 1 || stdout.Len() != 0 || stderr.String() != want {
						t.Errorf("exit=%d stdout=%q stderr=%q; want exit=1, empty stdout, stderr=%q", code, &stdout, &stderr, want)
					}
					wantEntries := 0
					for i, path := range paths {
						after, err := os.Lstat(path)
						if !exists[i] {
							if !os.IsNotExist(err) {
								t.Errorf("absent %s did not stay absent", filepath.Base(path))
							}
							continue
						}
						wantEntries++
						if err != nil {
							t.Fatal(err)
						}
						if !bytes.Equal(readTestFile(t, path), contents[i]) || after.Mode() != before[i].Mode() || !after.ModTime().Equal(before[i].ModTime()) || !os.SameFile(before[i], after) {
							t.Errorf("existing %s changed bytes, mode, mtime or identity", filepath.Base(path))
						}
					}
					entries, err := os.ReadDir(dir)
					if err != nil {
						t.Fatal(err)
					}
					if len(entries) != wantEntries {
						t.Error("refused keygen left unexpected files")
					}
					info, err := os.Stat(dir)
					if err != nil {
						t.Fatal(err)
					}
					if !info.ModTime().Equal(stamp) {
						t.Error("refused keygen touched output directory")
					}
				})
			}
		})
	}
}

func TestUPD02_KeygenSignableIDBoundaries(t *testing.T) {
	for _, id := range []string{"a", "Z", "0", strings.Repeat("a", 128), "a.b", "a-b", "a_b", "a:b", "1._:-"} {
		t.Run(id, func(t *testing.T) {
			dir := t.TempDir()
			keyPath, trustPath := filepath.Join(dir, "key.json"), filepath.Join(dir, "trust.json")
			var stdout, stderr bytes.Buffer
			code := run([]string{"keygen", "-key-id", id, "-out", keyPath, "-trust-out", trustPath, "-not-before", testNotBefore, "-not-after", testNotAfter}, &stdout, &stderr)
			if code != 0 || stdout.String() != "key_id="+id+"\npurpose=release\n" || stderr.String() != "fvpn-release: private key stays on the build machine; do not copy it to nodes\n" {
				t.Fatalf("keygen exit=%d stdout=%q stderr=%q", code, &stdout, &stderr)
			}
			key, err := release.LoadKey(keyPath)
			if err != nil {
				t.Fatal("cannot load generated key")
			}
			if key.KeyID != id {
				t.Fatal("loaded key ID differs")
			}
			// TEST ONLY: the new root signs its own retirement while still usable.
			updatePath := filepath.Join(dir, "update.json")
			stdout.Reset()
			stderr.Reset()
			code = run([]string{"trust-update", "-key", keyPath, "-trust", trustPath, "-revision", "1", "-retire", id, "-not-after", "2030-12-01T00:00:00Z", "-at", "2030-06-01T00:00:00Z", "-out", updatePath}, &stdout, &stderr)
			if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "signer_key_id="+id+"\n") {
				t.Fatalf("trust-update exit=%d stdout=%q stderr=%q", code, &stdout, &stderr)
			}
			stdout.Reset()
			stderr.Reset()
			code = run([]string{"trust-verify", "-update", updatePath, "-trust", trustPath, "-current-revision", "0", "-at", "2030-06-01T00:00:00Z"}, &stdout, &stderr)
			if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "signer_key_ids="+id+"\n") {
				t.Fatalf("trust-verify exit=%d stdout=%q stderr=%q", code, &stdout, &stderr)
			}
		})
	}
}

func TestUPD02_KeygenSignerIDRequiredFlagsFirst(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"keygen", "-key-id", "release B"}, &stdout, &stderr)
	if code != 1 || stdout.Len() != 0 || stderr.String() != "fvpn-release: keygen requires -key-id and -out\n" {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, &stdout, &stderr)
	}
}
