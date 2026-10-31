package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/packaging/installer"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

// The umask is process-global: isolate every extraction and seed invocation.
func TestF902_UmaskHelper(t *testing.T) {
	action := os.Getenv("FVPN_TEST_UMASK_ACTION")
	if action == "" {
		return
	}
	mask, err := strconv.ParseUint(os.Getenv("FVPN_TEST_UMASK"), 8, 32)
	if err != nil {
		t.Fatal(err)
	}
	syscall.Umask(int(mask))
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("FVPN_TEST_UMASK_ARGS")), &args); err != nil {
		t.Fatal(err)
	}
	if action == "extract" {
		f, err := os.Open(args[0])
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := updater.ExtractArchive(f, args[1], updater.ExtractOptions{}); err != nil {
			t.Fatal(err)
		}
		return
	}
	var out, stderr bytes.Buffer
	if code := run(args, &out, &stderr); code != 0 {
		t.Fatalf("seed exit %d: %s", code, &stderr)
	}
}

func runUmask(t *testing.T, mask, action string, args []string) {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestF902_UmaskHelper$")
	cmd.Env = append(os.Environ(), "FVPN_TEST_UMASK="+mask, "FVPN_TEST_UMASK_ACTION="+action, "FVPN_TEST_UMASK_ARGS="+string(raw))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s under %s: %v %s", action, mask, err, out)
	}
}

func TestF902_ExtractAndSeedAcrossUmasks(t *testing.T) {
	for _, installMask := range []string{"022", "077"} {
		t.Run(installMask, func(t *testing.T) {
			lay, _, tree, _, args := seedCLIInputs(t)
			reference, err := fsutil.TreeHash(tree)
			if err != nil {
				t.Fatal(err)
			}
			archives, err := filepath.Glob(filepath.Join(lay.Staging, "*.tar.zst"))
			if err != nil || len(archives) != 1 {
				t.Fatal(archives, err)
			}
			if err := os.RemoveAll(tree); err != nil {
				t.Fatal(err)
			}
			runUmask(t, installMask, "extract", []string{archives[0], tree})
			got, err := fsutil.TreeHash(tree)
			if err != nil || got != reference {
				t.Fatal("umask changed tree", err)
			}
			if err := filepath.WalkDir(tree, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				st, err := d.Info()
				if err != nil {
					return err
				}
				if d.IsDir() && st.Mode().Perm() != 0755 {
					t.Errorf("directory %s mode %o", filepath.Base(path), st.Mode().Perm())
				}
				if d.Name() == "fvpn-agent" && st.Mode().Perm() != 0755 {
					t.Errorf("agent mode %o", st.Mode().Perm())
				}
				if strings.HasSuffix(d.Name(), ".service") && st.Mode().Perm() != 0644 {
					t.Errorf("unit mode %o", st.Mode().Perm())
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			for _, seedMask := range []string{"022", "077"} {
				runUmask(t, seedMask, "seed", args)
			}
			// Permission drift still fails; no masking of the live tree's mode bits.
			if err := os.Chmod(filepath.Join(tree, "fvpn-agent"), 0700); err != nil {
				t.Fatal(err)
			}
			var out, stderr bytes.Buffer
			if code := run(args, &out, &stderr); code != 1 || !strings.Contains(stderr.String(), "HASH_MISMATCH") {
				t.Fatalf("mode drift accepted: %d %s", code, &stderr)
			}
		})
	}
}

func TestF903_ApplyCLIReportsSealedCheckpointFailure(t *testing.T) {
	lay, cfgPath, _, baseline, args := seedCLIInputs(t)
	var out, stderr bytes.Buffer
	if code := run(args, &out, &stderr); code != 0 {
		t.Fatalf("seed %d %s", code, &stderr)
	}
	next, err := installer.BuildReleaseFixture(t.TempDir(), "../../../../deploy/node/systemd", uuid.NewString(), "2", map[string][]byte{"fvpn-agent": []byte("TEST ONLY next")})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := updater.LoadConfig(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	var keys []envelope.TrustedKey
	for i, path := range []string{cfg.ReleaseTrust, next.Trust} {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		trust, err := envelope.ParseNodeTrustStoreJSON(b)
		if err != nil {
			t.Fatal(err)
		}
		key, ok := trust.Lookup("release-signing-test-" + strconv.Itoa(i+1))
		if !ok {
			t.Fatal("fixture trust key absent")
		}
		keys = append(keys, key)
	}
	trust, err := envelope.NewNodeTrustStore(keys...)
	if err != nil {
		t.Fatal(err)
	}
	trustRaw, err := json.Marshal(trust)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.ReleaseTrust, trustRaw, 0600); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	pending := filepath.Join(lay.Staging, updater.PendingReleaseFile)
	record, err := json.Marshal(updater.PendingRelease{ReleaseID: next.ReleaseID, ManifestSHA256: next.SHA, ManifestPath: next.Manifest, ArtifactPath: next.Artifact})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := os.WriteFile(pending, record, 0600); err != nil {
			t.Fatal(err)
		}
		stderr.Reset()
		if code := run([]string{"apply", "--config", cfgPath, "--pending", pending}, &out, &stderr); code == 0 || !strings.Contains(stderr.String(), "outcome=failed") || !strings.Contains(stderr.String(), "CHECKPOINT_FAILED") {
			t.Fatalf("failed apply: %d %s", code, &stderr)
		}
		snap, err := updatejournal.Snapshot(t.Context(), filepath.Join(lay.Journal, updatejournal.FileName))
		if err != nil || snap.State != "failed" || snap.Stage != "final_sample" || snap.CurrentRelease != baseline {
			t.Fatalf("failure snapshot: %+v %v", snap, err)
		}
	}
}
