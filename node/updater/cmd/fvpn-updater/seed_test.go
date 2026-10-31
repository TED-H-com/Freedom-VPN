package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/TED-H-com/Freedom-VPN/packaging/installer"
	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/google/uuid"
)

func TestD089_SeedCLIAndStatus(t *testing.T) {
	for _, flag := range []string{"-h", "--help"} {
		var out, stderr bytes.Buffer
		if code := run([]string{"seed", flag}, &out, &stderr); code != 64 || !strings.Contains(stderr.String(), "-manifest") || !strings.Contains(stderr.String(), "-tree") || strings.Contains(stderr.String(), "unknown command") {
			t.Fatalf("installer probe: %d %s", code, &stderr)
		}
	}
	for _, args := range [][]string{{"seed"}, {"seed", "--config", "unused"}, {"seed", "--config", "unused", "--manifest", "unused"}, {"seed", "--bad-flag"}} {
		var out, stderr bytes.Buffer
		if code := run(args, &out, &stderr); code != 64 {
			t.Fatalf("usage: %v code=%d %s", args, code, &stderr)
		}
	}
	root := t.TempDir()
	fx, err := installer.BuildReleaseFixture(root, "../../../../deploy/node/systemd", uuid.NewString(), "1", map[string][]byte{"fvpn-agent": []byte("TEST ONLY fixture")})
	if err != nil {
		t.Fatal(err)
	}
	lay := updater.Layout{Releases: filepath.Join(root, "releases"), Current: filepath.Join(root, "current"), Identity: filepath.Join(root, "identity"), State: filepath.Join(root, "state"), Journal: filepath.Join(root, "journal"), Lock: filepath.Join(root, "update.lock"), Staging: filepath.Join(root, "staging"), Trust: fx.Trust}
	if err := lay.Ensure(); err != nil {
		t.Fatal(err)
	}
	tree := filepath.Join(lay.Releases, fx.ReleaseID)
	f, err := os.Open(fx.Artifact)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := updater.ExtractArchive(f, tree, updater.ExtractOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("releases/"+fx.ReleaseID, lay.Current); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(fx.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	// The fixture's artifact URL is deliberately unreachable. The staged,
	// hash-bound archive must suffice for a completely offline seed.
	var env struct {
		Signed json.RawMessage `json:"signed"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	m, err := release.ParsedManifest(env.Signed)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := os.ReadFile(fx.Artifact)
	if err != nil {
		t.Fatal(err)
	}
	for _, art := range m.Artifacts {
		if art.Architecture == "amd64" {
			if err := os.WriteFile(filepath.Join(lay.Staging, art.Sha256+".tar.zst"), archive, 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	cfg := filepath.Join(root, "updater.json")
	writeCLIConfig(t, cfg, lay, fx.Trust, filepath.Join(root, "desired.json"), uuid.Nil)
	args := []string{"seed", "--config", cfg, "--manifest", fx.Manifest, "--tree", tree}
	for range 2 {
		var out, stderr bytes.Buffer
		if code := run(args, &out, &stderr); code != 0 {
			t.Fatalf("seed: %d %s", code, &stderr)
		}
	}
	var out, stderr bytes.Buffer
	if code := run([]string{"status", "--config", cfg}, &out, &stderr); code != 0 {
		t.Fatalf("status: %d %s", code, &stderr)
	}
	var snapshot updatejournal.SnapshotState
	if err := json.Unmarshal(out.Bytes(), &snapshot); err != nil || snapshot.SeededBaseline == nil || snapshot.SeededBaseline.ReleaseID != fx.ReleaseID || snapshot.JobID != "" || snapshot.LastGoodRelease != "" {
		t.Fatalf("seed not visible: %s %v", &out, err)
	}
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil || snapshot.SeededBaseline.BootID != strings.TrimSpace(string(boot)) {
		t.Fatalf("seed boot: %+v %v", snapshot.SeededBaseline, err)
	}
	if err := os.WriteFile(filepath.Join(tree, "extra"), []byte("extra"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := run(args, &out, &stderr); code != 1 || !strings.Contains(stderr.String(), "HASH_MISMATCH") {
		t.Fatalf("unverified seed: %d %s", code, &stderr)
	}
	after, err := updatejournal.Snapshot(context.Background(), filepath.Join(lay.Journal, updatejournal.FileName))
	if err != nil || *after.SeededBaseline != *snapshot.SeededBaseline {
		t.Fatal("refusal replaced baseline", err)
	}
}
