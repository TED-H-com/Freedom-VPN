package installer

import (
	"context"
	"os"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/google/uuid"
)

func TestF905_DowngradeRefusedBeforeStaging(t *testing.T) {
	for _, tc := range []struct {
		name, seq string
		sameID    bool
	}{{"lower", "9", false}, {"equal_different_id", "10", false}, {"lower_same_id", "9", true}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			env := baseEnv(t, dir)
			current := mustRelease(t, dir, uuid.NewString(), "10")
			if _, err := Install(t.Context(), env, mustReq(t, dir, current)); err != nil {
				t.Fatal(err)
			}
			id := uuid.NewString()
			if tc.sameID {
				id = current.ReleaseID
			}
			lower := mustRelease(t, dir, id, tc.seq)
			// Preserve an existing handoff as well as the installed release.
			if err := os.WriteFile(env.Layout.pendingUpgrade(), []byte("TEST ONLY existing handoff\n"), 0600); err != nil {
				t.Fatal(err)
			}
			before := upgradeTrees(t, env.Layout)
			calls := joinedCalls(env)
			res, err := Upgrade(t.Context(), env, lower.Manifest, lower.Artifact, lower.Trust)
			if res != nil || err == nil || AsError(err).Exit != ExitFailed || AsError(err).Code != "DOWNGRADE" {
				t.Fatalf("downgrade: %+v %v", res, err)
			}
			if after := upgradeTrees(t, env.Layout); after != before || joinedCalls(env) != calls {
				t.Fatal("refusal changed staging/releases or ran host/service commands")
			}
			if _, err := os.Stat(env.Layout.releaseDir(lower.ReleaseID)); !tc.sameID && !os.IsNotExist(err) {
				t.Fatalf("candidate was extracted: %v", err)
			}
		})
	}
}

func TestF905_SharedLockPrecedesInstalledSequenceRead(t *testing.T) {
	dir := t.TempDir()
	env := baseEnv(t, dir)
	current := mustRelease(t, dir, uuid.NewString(), "2")
	if _, err := Install(t.Context(), env, mustReq(t, dir, current)); err != nil {
		t.Fatal(err)
	}
	lower := mustRelease(t, dir, uuid.NewString(), "1")
	lk, err := nodelock.TryAcquire(env.Layout.LockPath)
	if err != nil {
		t.Fatal(err)
	}
	defer lk.Release()
	// An unreadable record must not be reached while another writer holds it.
	if err := os.Remove(env.Layout.installedFile()); err != nil {
		t.Fatal(err)
	}
	before := upgradeTrees(t, env.Layout)
	_, err = Upgrade(context.Background(), env, lower.Manifest, lower.Artifact, lower.Trust)
	if err == nil || AsError(err).Exit != ExitBusy || upgradeTrees(t, env.Layout) != before {
		t.Fatalf("locked upgrade: %v", err)
	}
}

func upgradeTrees(t *testing.T, l Layout) [3]string {
	t.Helper()
	var sums [3]string
	for i, path := range []string{l.StagingDir, l.ReleasesDir} {
		var err error
		sums[i], err = fsutil.TreeHash(path)
		if err != nil {
			t.Fatal(err)
		}
	}
	var err error
	sums[2], err = os.Readlink(l.CurrentLink)
	if err != nil {
		t.Fatal(err)
	}
	return sums
}
