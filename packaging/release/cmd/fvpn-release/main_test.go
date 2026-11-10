package main

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

// TEST ONLY windows; keys are generated at runtime in test scratch.
const testNotBefore = "2030-01-01T00:00:00Z"
const testNotAfter = "2031-01-01T00:00:00Z"

func TestSEC01_KeygenInvalidTrustPreservesOutputs(t *testing.T) {
	cases := []struct {
		name, notBefore, notAfter string
	}{
		{"missing_both", "", ""},
		{"missing_start", "", testNotAfter},
		{"missing_end", testNotBefore, ""},
		{"invalid_start", "not-a-date", testNotAfter},
		{"invalid_end", testNotBefore, "not-a-date"},
		{"equal_window", testNotBefore, testNotBefore},
		{"reversed_window", testNotAfter, testNotBefore},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, existing := range []bool{false, true} {
				name := "absent"
				if existing {
					name = "existing"
				}
				t.Run(name, func(t *testing.T) {
					root := t.TempDir()
					dir := filepath.Join(root, "outputs")
					keyPath, trustPath := filepath.Join(dir, "key.json"), filepath.Join(dir, "trust.json")
					var keyBefore, trustBefore []byte
					if existing {
						var stdout, stderr bytes.Buffer
						if run([]string{"keygen", "-key-id", "test-existing", "-out", keyPath, "-trust-out", trustPath, "-not-before", testNotBefore, "-not-after", testNotAfter}, &stdout, &stderr) != 0 {
							t.Fatal("could not prepare existing test outputs")
						}
						keyBefore, trustBefore = readTestFile(t, keyPath), readTestFile(t, trustPath)
					}
					var stdout, stderr bytes.Buffer
					code := run([]string{"keygen", "-key-id", "test-replacement", "-out", keyPath, "-trust-out", trustPath, "-not-before", tc.notBefore, "-not-after", tc.notAfter}, &stdout, &stderr)
					if code != 1 || stdout.Len() != 0 || stderr.Len() == 0 {
						t.Fatalf("invalid trust input: exit=%d, stdout bytes=%d, stderr bytes=%d", code, stdout.Len(), stderr.Len())
					}
					if !existing {
						entries, err := os.ReadDir(root)
						if err != nil {
							t.Fatal(err)
						}
						if len(entries) != 0 {
							t.Fatal("invalid trust input created an output directory")
						}
						return
					}
					if !bytes.Equal(readTestFile(t, keyPath), keyBefore) {
						t.Error("invalid trust input replaced the existing private key")
					}
					if !bytes.Equal(readTestFile(t, trustPath), trustBefore) {
						t.Error("invalid trust input replaced the existing public trust")
					}
					entries, err := os.ReadDir(dir)
					if err != nil || len(entries) != 2 {
						t.Fatal("invalid trust input left unexpected output files")
					}
				})
			}
		})
	}
}

func TestSEC01_KeygenOutputs(t *testing.T) {
	for _, withTrust := range []bool{false, true} {
		name := "key_only"
		if withTrust {
			name = "with_trust"
		}
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "outputs")
			keyPath, trustPath := filepath.Join(dir, "key.json"), filepath.Join(dir, "trust.json")
			args := []string{"keygen", "-key-id", "test-release", "-out", keyPath}
			if withTrust {
				args = append(args, "-trust-out", trustPath, "-not-before", testNotBefore, "-not-after", testNotAfter)
			}
			var stdout, stderr bytes.Buffer
			if code := run(args, &stdout, &stderr); code != 0 {
				t.Fatalf("keygen exit=%d", code)
			}
			key, err := release.LoadKey(keyPath)
			if err != nil {
				t.Fatal("cannot load generated private key")
			}
			if key.KeyID != "test-release" || stdout.String() != "key_id=test-release\npurpose=release\n" {
				t.Fatal("keygen output identity mismatch")
			}
			for path, want := range map[string]os.FileMode{dir: 0700, keyPath: 0600} {
				info, err := os.Stat(path)
				if err != nil || info.Mode().Perm() != want {
					t.Fatal("incorrect key or directory permissions")
				}
			}
			seed := []byte(base64.StdEncoding.EncodeToString(key.Seed))
			if bytes.Contains(stdout.Bytes(), seed) || bytes.Contains(stderr.Bytes(), seed) {
				t.Fatal("keygen printed private material")
			}
			if !withTrust {
				if _, err := os.Stat(trustPath); !os.IsNotExist(err) {
					t.Fatal("key-only invocation created public trust output")
				}
				return
			}
			raw := readTestFile(t, trustPath)
			if bytes.Contains(raw, seed) {
				t.Fatal("public trust contains private material")
			}
			trust, err := envelope.ParseNodeTrustStoreJSON(raw)
			if err != nil {
				t.Fatal(err)
			}
			public, ok := trust.Lookup(key.KeyID)
			if !ok || trust.Len() != 1 || !bytes.Equal(public.PublicKey, key.Public()) || len(public.Purposes) != 1 || public.Purposes[0] != envelope.PurposeRelease || public.Revoked {
				t.Fatal("public trust does not match generated release key")
			}
			if public.NotBefore.Format(time.RFC3339) != testNotBefore || public.NotAfter.Format(time.RFC3339) != testNotAfter {
				t.Fatal("public trust validity window mismatch")
			}
		})
	}
}

func readTestFile(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
