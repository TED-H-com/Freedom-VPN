package installer

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

// §6.3: copy the chosen input bytes without merging stores or rewriting JSON.
func TestPKG01_DesiredTrustCopies(t *testing.T) {
	clock := time.Now().UTC()
	now := func() time.Time { return clock }
	release := installerTrustBytes(t, envelope.PurposeRelease, now())
	desired := installerTrustBytes(t, envelope.PurposeDesiredState, now())
	control := installerTrustBytes(t, envelope.PurposeDesiredState, now())
	for _, explicit := range []bool{false, true} {
		name := "omitted_copies_bundle"
		if explicit {
			name = "explicit_copies_desired"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			layout := DefaultLayout().WithRoot(filepath.Join(root, "node"))
			env := &Env{Layout: layout, Now: now}
			journal, err := openJournal(env)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(layout.EtcDir, 0o755); err != nil {
				t.Fatal(err)
			}
			input := func(name string, raw []byte) string {
				t.Helper()
				path := filepath.Join(root, name)
				if err := os.WriteFile(path, raw, 0o600); err != nil {
					t.Fatal(err)
				}
				return path
			}
			req := InstallRequest{
				ServerCA:    input("ca.pem", []byte("TEST ONLY public CA input\n")),
				TrustPath:   input("release.json", release),
				TrustBundle: input("control.json", control),
			}
			want := control
			if explicit {
				req.DesiredTrustPath = input("desired.json", desired)
				want = desired
			}
			if err := writePublic(journal, layout, req); err != nil {
				t.Fatal(err)
			}
			for name, want := range map[string][]byte{
				"desired-trust.json": want,
				"release-trust.json": release,
				"trust.json":         control,
			} {
				got, err := os.ReadFile(filepath.Join(layout.EtcDir, name))
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got, want) {
					t.Errorf("%s did not preserve the selected input bytes", name)
				}
			}
		})
	}
}

func installerTrustBytes(t *testing.T, purpose envelope.Purpose, now time.Time) []byte {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	trust, err := envelope.NewNodeTrustStore(envelope.TrustedKey{
		KeyID: "test-" + string(purpose), PublicKey: pub, Purposes: []envelope.Purpose{purpose},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(trust, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(raw, '\n')
}

// §6.3, §16.2: reject bad trust before any host inspection or effect, including
// on a rerun that would otherwise report already_current.
func TestPKG01_DesiredTrustPreflight(t *testing.T) {
	inputs := t.TempDir()
	fx := mustRelease(t, inputs, "019a0000-0000-7000-8000-0000000000c1", "1")
	for _, source := range []string{"desired-trust", "trust-bundle"} {
		for _, state := range []string{"fresh", "already_current"} {
			for _, condition := range []string{"unreadable", "malformed", "ledger_epoch", "empty", "release_only", "revoked", "expired", "not_yet_valid"} {
				t.Run(source+"/"+state+"/"+condition, func(t *testing.T) {
					root := t.TempDir()
					env := baseEnv(t, root)
					req := mustReq(t, t.TempDir(), fx)
					if state == "already_current" {
						for _, want := range []string{statusInstalled, statusAlreadyCurrent} {
							res, err := Install(t.Context(), env, req)
							if err != nil || res.Status != want {
								t.Fatalf("setup install: %+v %v, want %s", res, err, want)
							}
						}
					}
					path := filepath.Join(t.TempDir(), "trust.json")
					reason := " must contain a non-revoked desired-state key valid at the installer clock (not_before <= now < not_after)"
					var raw []byte
					switch condition {
					case "unreadable":
						reason = " must be a readable node trust store"
					case "malformed":
						raw = []byte(`{"TEST ONLY file contents must not appear in errors":true}`)
						reason = " must be a node trust store"
					case "empty":
						raw = []byte(`{"keys":[]}`)
					case "release_only":
						path = fx.Trust
					default:
						raw = installerTrustCase(t, env.now(), condition)
						if condition == "ledger_epoch" {
							reason = " must be a node trust store"
						}
					}
					if raw != nil {
						if err := os.WriteFile(path, raw, 0o600); err != nil {
							t.Fatal(err)
						}
					}
					if source == "desired-trust" {
						req.DesiredTrustPath = path
					} else {
						req.TrustBundle = path
					}
					assertDesiredTrustRefusal(t, root, env, req, "--"+source+reason)
				})
			}
		}
	}
}

func TestPKG01_DesiredTrustBeforeIdentify(t *testing.T) {
	root := t.TempDir()
	env := baseEnv(t, root)
	inputs := t.TempDir()
	fx := mustRelease(t, inputs, "019a0000-0000-7000-8000-0000000000c1", "1")
	req := mustReq(t, inputs, fx)
	req.DesiredTrustPath = fx.Trust
	if err := os.Remove(env.Layout.OSRelease); err != nil {
		t.Fatal(err)
	}
	assertDesiredTrustRefusal(t, root, env, req, "--desired-trust must contain a non-revoked desired-state key valid at the installer clock (not_before <= now < not_after)")
}

func TestPKG01_DesiredTrustAtNotBefore(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(fmt.Sprintf("explicit_%t", explicit), func(t *testing.T) {
			root := t.TempDir()
			env := baseEnv(t, root)
			inputs := t.TempDir()
			fx := mustRelease(t, inputs, "019a0000-0000-7000-8000-0000000000c1", "1")
			req := mustReq(t, inputs, fx)
			raw := installerTrustCase(t, env.now(), "at_not_before")
			path := filepath.Join(inputs, "at-not-before.json")
			if err := os.WriteFile(path, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			if explicit {
				req.DesiredTrustPath = path
				// An explicit desired root is checked instead of the bundle.
				req.TrustBundle = fx.Trust
			} else {
				req.TrustBundle = path
			}
			res, err := Install(t.Context(), env, req)
			if err != nil || res.Status != statusInstalled {
				t.Fatalf("not_before boundary: %+v %v", res, err)
			}
			got, err := os.ReadFile(env.Layout.desiredTrust())
			if err != nil || !bytes.Equal(got, raw) {
				t.Fatalf("installed desired trust differs: %v", err)
			}
		})
	}
}

func TestPKG01_DesiredTrustLegacyRerun(t *testing.T) {
	root := t.TempDir()
	env := baseEnv(t, root)
	inputs := t.TempDir()
	fx := mustRelease(t, inputs, "019a0000-0000-7000-8000-0000000000c1", "1")
	req := mustReq(t, inputs, fx)
	if _, err := Install(t.Context(), env, req); err != nil {
		t.Fatal(err)
	}
	// Reproduce the previous installer's release-only default on disk.
	legacy, err := os.ReadFile(fx.Trust)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env.Layout.desiredTrust(), legacy, 0o644); err != nil {
		t.Fatal(err)
	}
	before := map[string]string{}
	for _, path := range []string{env.Layout.desiredTrust(), env.Layout.releaseTrust(), env.Layout.trustBundle()} {
		before[path] = hashFile(t, path)
	}
	res, err := Install(t.Context(), env, req)
	if err != nil || res.Status != statusAlreadyCurrent {
		t.Fatalf("legacy rerun: %+v %v", res, err)
	}
	for path, hash := range before {
		if hashFile(t, path) != hash {
			t.Errorf("already_current rerun rewrote %s", filepath.Base(path))
		}
	}
}

func installerTrustCase(t *testing.T, now time.Time, condition string) []byte {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	key := envelope.TrustedKey{
		KeyID: "TEST ONLY key ID must not appear in errors", PublicKey: pub,
		Purposes:  []envelope.Purpose{envelope.PurposeDesiredState},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
	}
	switch condition {
	case "ledger_epoch":
		key.Purposes = []envelope.Purpose{envelope.PurposeLedgerEpoch}
	case "revoked":
		key.Revoked = true
	case "expired":
		key.NotAfter = now
	case "not_yet_valid":
		key.NotBefore = now.Add(time.Nanosecond)
	case "at_not_before":
		key.NotBefore = now
	default:
		t.Fatalf("unknown trust case %q", condition)
	}
	trust, err := envelope.NewTrustStore(key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(trust)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func assertDesiredTrustRefusal(t *testing.T, root string, env *Env, req InstallRequest, want string) {
	t.Helper()
	before := trustHostSnapshot(t, root)
	services := env.Services.(*MemServices)
	active, enabled, reloads := maps.Clone(services.active), maps.Clone(services.enabled), services.Reloads
	*env.Stderr.(callSink).calls = nil
	var looked []string
	look := env.LookPath
	env.LookPath = func(name string) (string, error) {
		looked = append(looked, name)
		return look(name)
	}
	res, err := Install(t.Context(), env, req)
	if err == nil || AsError(err).Exit != ExitUsage || err.Error() != "usage: "+want || res != nil {
		t.Errorf("refusal: result=%+v error=%v; want exit 64 and %q", res, err, "usage: "+want)
	}
	if len(looked) != 0 || joinedCalls(env) != "" {
		t.Errorf("refused install touched host tools: lookups=%v commands=%q", looked, joinedCalls(env))
	}
	if !reflect.DeepEqual(before, trustHostSnapshot(t, root)) {
		t.Error("refused install changed host paths, bytes, modes or modification times")
	}
	if !reflect.DeepEqual(active, services.active) || !reflect.DeepEqual(enabled, services.enabled) || reloads != services.Reloads {
		t.Error("refused install changed host services")
	}
}

func trustHostSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		value := fmt.Sprintf("%s %d", info.Mode(), info.ModTime().UnixNano())
		if info.Mode().IsRegular() {
			value += " " + hashFile(t, path)
		} else if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			value += " " + target
		}
		snapshot[path] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}
