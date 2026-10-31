package updater

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// Umask is process-global. Each child stages and applies a real signed
// candidate, switches current, then seeds the resolved upgraded tree under the
// other caller's umask. Service/health adapters are TEST ONLY; tree verification
// and the updater journal are real. This reproduces F9-02 before PR #307.
func TestF902_SeedUpgradedTreeAcrossCallerUmasks(t *testing.T) {
	mode := os.Getenv("FVPN_TEST_SEED_UPGRADE_UMASK")
	if mode == "" {
		for _, m := range []string{"apply022_seed077", "apply077_seed022"} {
			t.Run(m, func(t *testing.T) {
				cmd := exec.Command(os.Args[0], "-test.run=^TestF902_SeedUpgradedTreeAcrossCallerUmasks$")
				cmd.Env = append(os.Environ(), "FVPN_TEST_SEED_UPGRADE_UMASK="+m)
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("upgraded tree seed: %v\n%s", err, out)
				}
			})
		}
		return
	}
	syscall.Umask(0022)
	w := newWorld(t)
	t.Cleanup(func() { _ = w.eng.Close() })
	quietDesired(w)
	w.releaseSeq = "1"
	old := w.makeRelease("initial", 1, nil)
	manifest := installSeedTree(t, w, old, true)
	if err := w.eng.Seed(t.Context(), manifest, w.layout.releaseDir(old.id.String())); err != nil {
		t.Fatal(err)
	}
	w.releaseSeq = "2"
	next := w.makeRelease("upgraded", 1, nil)
	pending := stageInstaller(t, w, next)
	if mode == "apply077_seed022" {
		syscall.Umask(0077)
	}
	if err := w.eng.ApplyPending(t.Context(), pending); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(next.id)
	if w.oneResult().State != api.UpdateResultStateSucceeded {
		t.Fatal("candidate did not succeed")
	}
	if _, err := os.Stat(w.layout.releaseDir(old.id.String())); err != nil {
		t.Fatal("previous release was not retained", err)
	}
	// Both the sidecar and previous tree remain outside the verified subtree.
	if err := os.WriteFile(filepath.Join(w.layout.State, "TEST-ONLY-sidecar"), []byte("mutable state"), 0600); err != nil {
		t.Fatal(err)
	}
	tree, err := filepath.EvalSymlinks(w.layout.Current)
	if err != nil {
		t.Fatal(err)
	}
	before := treeOrDie(t, tree)
	manifest = filepath.Join(w.layout.Staging, next.id.String()+".manifest.json")
	if err := os.WriteFile(manifest, next.env, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w.layout.Staging, next.amdSHA+".tar.zst"), next.amd, 0600); err != nil {
		t.Fatal(err)
	}
	if mode == "apply022_seed077" {
		syscall.Umask(0077) // installer main sets 0077 before invoking seed
	} else {
		syscall.Umask(0022)
	}
	if err := w.eng.Seed(t.Context(), manifest, tree); err != nil {
		t.Fatal(err)
	}
	if treeOrDie(t, tree) != before {
		t.Fatal("seed repaired the tree")
	}
	seed := seedSnapshot(t, w).SeededBaseline
	if seed == nil || seed.ReleaseID != next.id.String() || seed.TreeHash != before {
		t.Fatal("seed did not verify resolved candidate", seed)
	}
}
