package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/installerjob"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

// Emulate initial installation's filesystem only: no manufactured journal
// release, pointer or tree hash. Seed must derive all evidence itself.
func installSeedTree(t *testing.T, w *world, rel builtRel, cache bool) string {
	t.Helper()
	copyTree(t, rel.tree, w.layout.releaseDir(rel.id.String()))
	link, err := w.layout.relativeRelease(rel.id.String())
	if err != nil {
		t.Fatal(err)
	}
	if err := fsutil.SwitchSymlink(w.layout.Current, link); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(w.layout.Staging, rel.id.String()+".manifest.json")
	if err := os.WriteFile(manifest, rel.env, 0o600); err != nil {
		t.Fatal(err)
	}
	if cache {
		if err := os.WriteFile(filepath.Join(w.layout.Staging, rel.amdSHA+".tar.zst"), rel.amd, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return manifest
}

func seedSnapshot(t *testing.T, w *world) updatejournal.SnapshotState {
	t.Helper()
	s, err := updatejournal.Snapshot(context.Background(), filepath.Join(w.layout.Journal, updatejournal.FileName))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestD089_SeedThenInstallerApply(t *testing.T) {
	ctx := context.Background()
	w := newWorld(t)
	quietDesired(w)
	w.releaseSeq = "1"
	old := w.makeRelease("installed", 1, []int{1})
	manifest := installSeedTree(t, w, old, false) // real signed URL download
	touchPersistent(t, w.layout)
	identity := treeOrDie(t, w.layout.Identity)
	state := treeOrDie(t, w.layout.State)
	if err := w.eng.Seed(ctx, manifest, w.layout.releaseDir(old.id.String())); err != nil {
		t.Fatal(err)
	}
	first := seedSnapshot(t, w)
	if first.SeededBaseline == nil || first.CurrentRelease != old.id.String() || first.JobID != "" || first.LastGoodRelease != "" || first.Health != nil {
		t.Fatalf("baseline claims job or health: %+v", first)
	}
	b := first.SeededBaseline
	if b.ReleaseID != old.id.String() || b.ManifestSHA256 != old.sum || b.TreeHash != treeOrDie(t, w.layout.releaseDir(old.id.String())) || b.BootID != w.node.BootID.String() || !b.SeededAt.Equal(w.now) {
		t.Fatalf("baseline evidence: %+v", b)
	}
	// Reopen SQLite and replay from a later boot. Neither timestamp nor row
	// count changes, and seed never starts a service or produces a result.
	if err := w.eng.Journal.Close(); err != nil {
		t.Fatal(err)
	}
	w.eng.Journal = openJournal(t, w.layout.Journal)
	w.eng.Node.BootID = uuid.New()
	w.now = w.now.Add(time.Hour)
	if err := w.eng.Seed(ctx, manifest, w.layout.releaseDir(old.id.String())); err != nil {
		t.Fatal(err)
	}
	if got := seedSnapshot(t, w); !reflect.DeepEqual(first, got) {
		t.Fatalf("replay changed seed: %+v", got)
	}
	releases, err := w.eng.Journal.ListReleases(ctx)
	if err != nil || len(releases) != 1 {
		t.Fatalf("release rows: %v %v", releases, err)
	}
	if box, err := w.eng.Journal.Outbox(ctx); err != nil || len(box) != 0 {
		t.Fatalf("seed produced results: %+v %v", box, err)
	}
	if stops, starts := w.svc.counts(); stops != 0 || starts != 0 {
		t.Fatal("seed touched services")
	}
	w.releaseSeq = "2"
	next := w.makeRelease("upgrade", 1, []int{1})
	art, tree, _, cmd := stageCopy(t, w, next)
	nextManifest := filepath.Join(w.layout.Staging, next.id.String()+".manifest.json")
	if err := os.WriteFile(nextManifest, next.env, 0o600); err != nil {
		t.Fatal(err)
	}
	pending := filepath.Join(w.layout.Staging, PendingReleaseFile)
	writePendingFile(t, pending, next, art, tree, cmd, nextManifest, nil, false)
	var record map[string]any
	if err := json.Unmarshal(fileBytes(t, pending), &record); err != nil {
		t.Fatal(err)
	}
	record["request_id"] = uuid.NewString()
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pending, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(next.id)
	payload, result := lastPayload(t, w)
	assertInstallerWireSource(t, payload)
	wantJob := installerjob.ID(next.id, next.sum, w.node.Incarnation)
	if result.JobId != wantJob || result.Attempt != 1 || result.State != api.UpdateResultStateSucceeded || requiredValue(t, result.CurrentReleaseId) != next.id || requiredValue(t, result.Health).Processes != api.HealthCheckStatusPassed || result.MemberForwardingEnabled {
		t.Fatalf("apply result: %+v", result)
	}
	if stops, starts := w.svc.counts(); stops < 1 || starts < 1 {
		t.Fatal("apply skipped service switch")
	}
	if treeOrDie(t, w.layout.Identity) != identity || treeOrDie(t, w.layout.State) != state {
		t.Fatal("identity or usage state changed")
	}
	assertConsumed(t, pending, "succeeded")
	if err := w.run(); err != nil && !errors.Is(err, ErrMembersBlocked) {
		t.Fatal(err)
	}
	if box, _ := w.eng.Journal.Outbox(ctx); len(box) != 1 {
		t.Fatal("replay produced another result")
	}
}

func TestD089_SeedRefusesUnverifiedTree(t *testing.T) {
	for _, change := range []string{"bytes", "missing", "extra", "extra directory", "mode", "symlink", "signature", "archive", "current", "revoked"} {
		t.Run(change, func(t *testing.T) {
			w := newWorld(t)
			r := w.makeRelease("installed", 1, []int{1})
			manifest := installSeedTree(t, w, r, true)
			tree := w.layout.releaseDir(r.id.String())
			file := filepath.Join(tree, "README")
			var err error
			switch change {
			case "bytes":
				err = os.WriteFile(file, []byte("modified"), 0o644)
			case "missing":
				err = os.Remove(file)
			case "extra":
				err = os.WriteFile(filepath.Join(tree, "extra"), []byte("extra"), 0o644)
			case "extra directory":
				err = os.Mkdir(filepath.Join(tree, "extra"), 0o755)
			case "mode":
				err = os.Chmod(file, 0o755)
			case "symlink":
				err = os.Symlink("README", filepath.Join(tree, "extra"))
			case "signature":
				err = os.WriteFile(manifest, bytes.Replace(r.env, []byte(`"1.2.0"`), []byte(`"9.9.9"`), 1), 0o600)
			case "archive":
				err = os.WriteFile(filepath.Join(w.layout.Staging, r.amdSHA+".tar.zst"), make([]byte, len(r.amd)), 0o600)
			case "current":
				err = fsutil.SwitchSymlink(w.layout.Current, r.tree)
			case "revoked":
				w.eng.RevokedReleaseIDs = []uuid.UUID{r.id}
			}
			if err != nil {
				t.Fatal(err)
			}
			before := seedSnapshot(t, w)
			link, _ := os.Readlink(w.layout.Current)
			if err := w.eng.Seed(context.Background(), manifest, tree); err == nil {
				t.Fatal("unverified seed accepted")
			}
			if after := seedSnapshot(t, w); !reflect.DeepEqual(before, after) {
				t.Fatal("failed seed changed journal")
			}
			if got, _ := os.Readlink(w.layout.Current); got != link {
				t.Fatal("failed seed moved current")
			}
			if releases, _ := w.eng.Journal.ListReleases(context.Background()); len(releases) != 0 {
				t.Fatal("failed seed recorded release")
			}
			if stops, starts := w.svc.counts(); stops != 0 || starts != 0 {
				t.Fatal("failed seed started recovery")
			}
		})
	}
}

func TestD089_SeedRefusesDifferentJournal(t *testing.T) {
	for _, change := range []string{"current", "tree", "manifest"} {
		t.Run(change, func(t *testing.T) {
			ctx := context.Background()
			w := newWorld(t)
			r := w.makeRelease("installed", 1, []int{1})
			manifest := installSeedTree(t, w, r, true)
			evidence := updatejournal.Release{ReleaseID: r.id.String(), ManifestSHA256: r.sum, ManifestEnvelope: r.env, ArtifactSHA256: r.amdSHA, TreeHash: treeOrDie(t, r.tree), StateSchemaTo: 1}
			switch change {
			case "current":
				evidence.ReleaseID = uuid.NewString()
			case "tree":
				evidence.TreeHash = strings.Repeat("0", 64)
			case "manifest":
				evidence.ManifestSHA256 = strings.Repeat("0", 64)
			}
			if err := w.eng.Journal.PutRelease(ctx, evidence); err != nil {
				t.Fatal(err)
			}
			if err := w.eng.Journal.SetPointers(ctx, updatejournal.Pointers{CurrentRelease: evidence.ReleaseID}); err != nil {
				t.Fatal(err)
			}
			before := seedSnapshot(t, w)
			stored, _ := w.eng.Journal.GetRelease(ctx, evidence.ReleaseID)
			if err := w.eng.Seed(ctx, manifest, w.layout.releaseDir(r.id.String())); err == nil || !strings.Contains(err.Error(), "differs") {
				t.Fatalf("conflict: %v", err)
			}
			got, _ := w.eng.Journal.GetRelease(ctx, evidence.ReleaseID)
			if !reflect.DeepEqual(stored, got) || !reflect.DeepEqual(before, seedSnapshot(t, w)) {
				t.Fatal("refusal overwrote existing evidence")
			}
		})
	}
}

func TestD089_SeedTakesSharedLock(t *testing.T) {
	w := newWorld(t)
	r := w.makeRelease("installed", 1, []int{1})
	manifest := installSeedTree(t, w, r, true)
	lk, err := nodelock.TryAcquire(w.layout.Lock)
	if err != nil {
		t.Fatal(err)
	}
	defer lk.Release()
	err = w.eng.Seed(context.Background(), manifest, w.layout.releaseDir(r.id.String()))
	var op *OpError
	if !errors.As(err, &op) || op.Code != apierr.UpdateBusy || seedSnapshot(t, w).SeededBaseline != nil {
		t.Fatalf("shared lock not respected: %v", err)
	}
}
