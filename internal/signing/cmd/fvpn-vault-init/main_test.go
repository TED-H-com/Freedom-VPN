package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

// TEST ONLY: all private keys are generated in memory or disk test scratch.
func fixture(t *testing.T) (config, string, ed25519.PrivateKey) {
	t.Helper()
	root := t.TempDir()
	pub, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	trust, err := envelope.NewNodeTrustStore(envelope.TrustedKey{KeyID: "test-release", PublicKey: pub, Purposes: []envelope.Purpose{envelope.PurposeRelease}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(trust)
	if err != nil {
		t.Fatal(err)
	}
	trustPath := filepath.Join(root, "release-trust.json")
	if err = os.WriteFile(trustPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	c := config{Provider: "file", ReleaseTrustFile: trustPath, ReleaseKeyID: "test-release", CA: caConfig{CommonName: "TEST ONLY staging CA", Validity: "30m"}, Purposes: make(map[signing.Purpose]window)}
	for _, p := range purposes() {
		c.Purposes[p] = window{Validity: "1h", Overlap: "0s"}
	}
	return c, filepath.Join(root, "ceremony"), private
}
func configBytes(t *testing.T, c config) []byte {
	t.Helper()
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func preparedFixture(t *testing.T) (prepared, string) {
	t.Helper()
	c, out, _ := fixture(t)
	p, err := prepare(configBytes(t, c))
	if err != nil {
		t.Fatal(err)
	}
	return p, out
}
func read(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		value := info.Mode().String()
		if info.Mode().IsRegular() {
			value += " " + digest(read(t, path))
		}
		out[rel] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func TestSEC01_InitCLIArtifacts(t *testing.T) {
	c, out, releasePrivate := fixture(t)
	configPath := filepath.Join(filepath.Dir(out), "config.json")
	if err := os.WriteFile(configPath, configBytes(t, c), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-config", configPath, "-out", out}, &stdout, &stderr); code != 0 {
		t.Fatalf("initialization failed: exit=%d diagnostic=%s", code, stderr.String())
	}
	var r receipt
	if err := json.Unmarshal(read(t, filepath.Join(out, "receipt.json")), &r); err != nil {
		t.Fatal(err)
	}
	if r.Kind != "fvpn-file-vault-init-v1" || r.Provider != "file" || len(r.Keys) != 7 || len(r.Checks) != 4 || r.LedgerRevision != 7 {
		t.Fatal("invalid completion metadata")
	}
	for _, path := range []string{out, filepath.Join(out, "vault")} {
		i, err := os.Stat(path)
		if err != nil || i.Mode().Perm() != 0700 {
			t.Fatal("wrong directory permissions")
		}
	}
	if err := filepath.WalkDir(out, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			i, err := d.Info()
			if err != nil {
				return err
			}
			if i.Mode().Perm() != 0600 {
				t.Error("wrong output permissions")
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	trustRaw := read(t, filepath.Join(out, r.TrustBundle))
	trust, err := envelope.ParseNodeTrustStoreJSON(trustRaw)
	if err != nil {
		t.Fatal(err)
	}
	if trust.Len() != 4 || bytes.Contains(trustRaw, []byte("ledger-epoch")) || digest(trustRaw) != r.TrustBundleSHA256 {
		t.Fatal("incorrect node trust bundle")
	}
	signed, err := envelope.Sign(map[string]any{"kind": "test-release"}, releasePrivate, c.ReleaseKeyID)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(signed)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = envelope.Verify(encoded, trust, envelope.PurposeRelease, time.Now()); err != nil {
		t.Fatal(err)
	}
	provider, err := signing.OpenFileProvider(filepath.Join(out, r.Vault))
	if err != nil {
		t.Fatal(err)
	}
	block, rest := pem.Decode(read(t, filepath.Join(out, r.CACertificate)))
	if block == nil || len(rest) != 0 {
		t.Fatal("invalid CA PEM")
	}
	ca, err := pki.NewCA(t.Context(), provider, r.CAKeyID, block.Bytes, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if cert.Subject.CommonName != c.CA.CommonName || digest(block.Bytes) != r.CADERSHA256 || !r.CANotAfter.Equal(cert.NotAfter) {
		t.Fatal("CA metadata mismatch")
	}
	for _, k := range r.Keys {
		if k.Purpose == signing.NodeCA && cert.NotAfter.After(k.NotAfter) {
			t.Fatal("CA certificate outlives provider key")
		}
	}
	nodeKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{SignatureAlgorithm: x509.ECDSAWithSHA256}, nodeKey)
	if err != nil {
		t.Fatal(err)
	}
	identity := pki.NodeIdentity{NodeID: uuid.New(), IncarnationID: uuid.New()}
	issued, err := ca.Issue(t.Context(), identity, string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr})))
	if err != nil {
		t.Fatal(err)
	}
	got, err := ca.Verify(issued.Certificate)
	if err != nil || got != identity {
		t.Fatal("generated CA cannot issue/verify existing node identity")
	}
	public := bytes.Join([][]byte{stdout.Bytes(), stderr.Bytes(), read(t, filepath.Join(out, "receipt.json")), trustRaw, read(t, filepath.Join(out, "agent-ca.pem"))}, nil)
	files, err := filepath.Glob(filepath.Join(out, "vault", "*.key"))
	if err != nil || len(files) != 6 {
		t.Fatal("wrong private key inventory")
	}
	secrets := [][]byte{releasePrivate.Seed()}
	for _, file := range files {
		raw := read(t, file)
		secrets = append(secrets, raw)
		k, err := x509.ParsePKCS8PrivateKey(raw)
		if err == nil {
			if ed, ok := k.(ed25519.PrivateKey); ok {
				secrets = append(secrets, ed.Seed())
			}
		}
	}
	for _, secret := range secrets {
		if bytes.Contains(public, secret) || bytes.Contains(public, []byte(base64.StdEncoding.EncodeToString(secret))) {
			t.Fatal("private material in public output")
		}
	}
	before := snapshot(t, out)
	stdout.Reset()
	stderr.Reset()
	if run([]string{"-config", configPath, "-out", out}, &stdout, &stderr) != 1 || stdout.Len() != 0 {
		t.Fatal("repeated invocation succeeded")
	}
	if !reflect.DeepEqual(before, snapshot(t, out)) {
		t.Fatal("repeated invocation mutated existing ceremony")
	}
}

func TestSEC01_InitRejectsConfigBeforeWrites(t *testing.T) {
	cases := map[string]func(*config){
		"missing_provider": func(c *config) { c.Provider = "" }, "other_provider": func(c *config) { c.Provider = "kms" },
		"missing_purpose": func(c *config) { delete(c.Purposes, signing.Lease) },
		"unknown_purpose": func(c *config) {
			delete(c.Purposes, signing.Lease)
			c.Purposes["unknown"] = window{Validity: "1h", Overlap: "0s"}
		},
		"missing_overlap":  func(c *config) { c.Purposes[signing.Lease] = window{Validity: "1h"} },
		"missing_validity": func(c *config) { c.Purposes[signing.Lease] = window{Overlap: "0s"} },
		"overflow":         func(c *config) { c.Purposes[signing.Lease] = window{Validity: "999999999999999h", Overlap: "0s"} },
		"negative":         func(c *config) { c.Purposes[signing.Lease] = window{Validity: "1h", Overlap: "-1s"} },
		"zero":             func(c *config) { c.Purposes[signing.Lease] = window{Validity: "0s", Overlap: "0s"} },
		"fractional":       func(c *config) { c.Purposes[signing.Lease] = window{Validity: "1.1s", Overlap: "0s"} },
		"overlap_too_long": func(c *config) { c.Purposes[signing.Lease] = window{Validity: "1h", Overlap: "1h"} },
		"ca_too_long":      func(c *config) { c.CA.Validity = "2h" }, "missing_ca": func(c *config) { c.CA.CommonName = "" },
		"control_in_ca": func(c *config) { c.CA.CommonName = "bad\nname" }, "missing_ca_validity": func(c *config) { c.CA.Validity = "" },
		"wrong_release_id": func(c *config) { c.ReleaseKeyID = "missing" }, "long_release_id": func(c *config) { c.ReleaseKeyID = strings.Repeat("x", 129) },
		"relative_trust": func(c *config) { c.ReleaseTrustFile = "relative.json" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) { c, out, _ := fixture(t); change(&c); runInvalid(t, configBytes(t, c), out) })
	}
	for _, name := range []string{"unknown", "duplicate", "trailing", "null"} {
		t.Run(name, func(t *testing.T) {
			c, out, _ := fixture(t)
			raw := configBytes(t, c)
			switch name {
			case "unknown":
				raw = append([]byte(`{"unknown":true,`), raw[1:]...)
			case "duplicate":
				raw = append([]byte(`{"provider":"file",`), raw[1:]...)
			case "trailing":
				raw = append(raw, []byte(` {}`)...)
			case "null":
				raw = []byte(`null`)
			}
			runInvalid(t, raw, out)
		})
	}
}
func runInvalid(t *testing.T, raw []byte, out string) {
	t.Helper()
	path := filepath.Join(filepath.Dir(out), "invalid.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, filepath.Dir(out))
	var stdout, stderr bytes.Buffer
	if run([]string{"-config", path, "-out", out}, &stdout, &stderr) != 1 || stdout.Len() != 0 || stderr.Len() == 0 {
		t.Fatal("invalid config was not rejected")
	}
	if !reflect.DeepEqual(before, snapshot(t, filepath.Dir(out))) {
		t.Fatal("invalid config changed filesystem")
	}
}
func TestSEC01_InitRejectsReleaseTrust(t *testing.T) {
	for _, name := range []string{"private", "duplicate", "unknown", "expired", "future", "revoked", "wrong_purpose", "extra_root", "long_id"} {
		t.Run(name, func(t *testing.T) {
			c, out, _ := fixture(t)
			raw := read(t, c.ReleaseTrustFile)
			var trust map[string]any
			if err := json.Unmarshal(raw, &trust); err != nil {
				t.Fatal(err)
			}
			rows := trust["keys"].([]any)
			k := rows[0].(map[string]any)
			switch name {
			case "private":
				trust = map[string]any{"purpose": "release", "key_id": "test-release", "seed_b64": "TEST ONLY - NOT A SECRET"}
			case "expired":
				k["not_before"] = time.Now().Add(-2 * time.Hour)
				k["not_after"] = time.Now().Add(-time.Hour)
			case "future":
				k["not_before"] = time.Now().Add(time.Hour)
				k["not_after"] = time.Now().Add(2 * time.Hour)
			case "revoked":
				k["revoked"] = true
			case "wrong_purpose":
				k["purposes"] = []string{"lease"}
			case "extra_root":
				pub, _, err := ed25519.GenerateKey(rand.Reader)
				if err != nil {
					t.Fatal(err)
				}
				other := map[string]any{}
				for key, value := range k {
					other[key] = value
				}
				other["key_id"] = "other"
				other["public_key_b64"] = base64.StdEncoding.EncodeToString(pub)
				trust["keys"] = append(rows, other)
			case "long_id":
				c.ReleaseKeyID = strings.Repeat("x", 129)
				k["key_id"] = c.ReleaseKeyID
			case "unknown":
				k["secret"] = "TEST ONLY - NOT A SECRET"
			}
			raw, err := json.Marshal(trust)
			if err != nil {
				t.Fatal(err)
			}
			if name == "duplicate" {
				raw = append([]byte(`{"keys":[],`), raw[1:]...)
			}
			if err = os.WriteFile(c.ReleaseTrustFile, raw, 0600); err != nil {
				t.Fatal(err)
			}
			runInvalid(t, configBytes(t, c), out)
		})
	}
}
func TestSEC01_InitExistingDestinations(t *testing.T) {
	for _, kind := range []string{"directory", "file", "symlink", "dangling_symlink"} {
		t.Run(kind, func(t *testing.T) {
			p, out := preparedFixture(t)
			target := filepath.Join(filepath.Dir(out), "existing")
			if err := os.Mkdir(target, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(target, "sentinel"), []byte("TEST ONLY preserve"), 0600); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "directory":
				if err := os.Mkdir(out, 0700); err != nil {
					t.Fatal(err)
				}
			case "file":
				if err := os.WriteFile(out, []byte("TEST ONLY preserve"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(target, out); err != nil {
					t.Fatal(err)
				}
			case "dangling_symlink":
				if err := os.Symlink(filepath.Join(target, "missing"), out); err != nil {
					t.Fatal(err)
				}
			}
			before := snapshot(t, filepath.Dir(out))
			if err := initialize(t.Context(), p, out, writeExclusive); err == nil {
				t.Fatal("existing destination accepted")
			}
			if !reflect.DeepEqual(before, snapshot(t, filepath.Dir(out))) {
				t.Fatal("existing destination changed")
			}
		})
	}
}
func TestSEC01_InitConcurrentAndPartialFailure(t *testing.T) {
	t.Run("concurrent", func(t *testing.T) {
		p, out := preparedFixture(t)
		var wg sync.WaitGroup
		results := make(chan error, 2)
		for range 2 {
			wg.Add(1)
			go func() { defer wg.Done(); results <- initialize(t.Context(), p, out, writeExclusive) }()
		}
		wg.Wait()
		close(results)
		success := 0
		for err := range results {
			if err == nil {
				success++
			}
		}
		if success != 1 {
			t.Fatal("expected exactly one successful initializer")
		}
		var r receipt
		if err := json.Unmarshal(read(t, filepath.Join(out, "receipt.json")), &r); err != nil || len(r.Keys) != 7 || r.LedgerRevision != 7 {
			t.Fatal("concurrent initialization changed initial key set")
		}
	})
	for _, name := range []string{"agent-ca.pem", "trust.json", "receipt.json"} {
		t.Run(name, func(t *testing.T) {
			p, out := preparedFixture(t)
			failure := func(path string, raw []byte, mode os.FileMode) error {
				if filepath.Base(path) == name {
					return errors.New("TEST ONLY injected write failure")
				}
				return writeExclusive(path, raw, mode)
			}
			if err := initialize(t.Context(), p, out, failure); err == nil {
				t.Fatal("write failure reported success")
			}
			if _, err := os.Stat(filepath.Join(out, "vault", "state.json")); err != nil {
				t.Fatal("partial vault not retained")
			}
			if _, err := os.Stat(filepath.Join(out, "receipt.json")); !os.IsNotExist(err) {
				t.Fatal("partial initialization published receipt")
			}
			before := snapshot(t, out)
			if err := initialize(t.Context(), p, out, writeExclusive); err == nil {
				t.Fatal("partial initialization automatically resumed")
			}
			if !reflect.DeepEqual(before, snapshot(t, out)) {
				t.Fatal("retry mutated partial initialization")
			}
		})
	}
	t.Run("canceled", func(t *testing.T) {
		p, out := preparedFixture(t)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		if initialize(ctx, p, out, writeExclusive) == nil {
			t.Fatal("canceled initialization succeeded")
		}
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Fatal("canceled initialization wrote output")
		}
	})
}
func TestSEC01_ExclusivePublication(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "receipt.json")
	if err := writeExclusive(path, []byte("complete"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeExclusive(path, []byte("replacement"), 0600); err == nil {
		t.Fatal("receipt replaced")
	}
	if string(read(t, path)) != "complete" {
		t.Fatal("receipt bytes changed")
	}
	names, err := os.ReadDir(dir)
	if err != nil || len(names) != 1 {
		t.Fatal("temporary publication files remain")
	}
}
func TestSEC01_InitCLIArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"-config"}, {"-config", "absent", "-out", "target", "unexpected"}, {"-unknown", "value"}, {"-config", "absent", "-out", "target"}} {
		var stdout, stderr bytes.Buffer
		if run(args, &stdout, &stderr) != 1 || stdout.Len() != 0 || stderr.Len() == 0 {
			t.Fatal("bad arguments accepted")
		}
	}
}

func TestSEC01_InitActualCertificateWindow(t *testing.T) {
	// TEST ONLY fixed time models clock movement without sleeps or changing clocks.
	now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, name := range []string{"valid", "clock_before_ca", "expired_ca", "key_shorter_than_ca", "missing_ca_key", "future_key", "retired_key"} {
		t.Run(name, func(t *testing.T) {
			cert := &x509.Certificate{NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour)}
			keys := []signing.KeyRecord{{ID: "test-ca", Purpose: signing.NodeCA, State: "active", NotBefore: now.Add(-time.Minute), NotAfter: now.Add(2 * time.Hour)}}
			switch name {
			case "clock_before_ca":
				cert.NotBefore = now.Add(time.Second)
			case "expired_ca":
				cert.NotAfter = now
			case "key_shorter_than_ca":
				keys[0].NotAfter = now.Add(time.Minute)
			case "missing_ca_key":
				keys[0].ID = "another-key"
			case "future_key":
				keys[0].NotBefore = now.Add(time.Second)
			case "retired_key":
				keys[0].State = "retired"
			}
			err := validateWindows(now, cert, keys, "test-ca")
			if (err == nil) != (name == "valid") {
				t.Fatal("incorrect actual-window decision")
			}
		})
	}
}
