package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/nodelock"
	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

func kernelBootID(t *testing.T) (uuid.UUID, string) {
	t.Helper()
	b, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		t.Fatal(err)
	}
	raw := strings.TrimSpace(string(b))
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		t.Fatal(err)
	}
	return id, raw
}

func quietDesired(w *world) { w.src.err = ErrNoWork }

func plantTree(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("PLANTED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ONLY-IN-PLANT"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func stageCopy(t *testing.T, w *world, rel builtRel) (art, tree, sentinel, cmd string) {
	t.Helper()
	art = filepath.Join(w.layout.Staging, rel.amdSHA+".tar.zst")
	if err := os.WriteFile(art, rel.amd, 0o644); err != nil {
		t.Fatal(err)
	}
	tree = filepath.Join(w.root, "installer-tree")
	plantTree(t, tree)
	plantTree(t, w.layout.releaseDir(rel.id.String()))
	sentinel = filepath.Join(w.root, "updater-command-sentinel")
	cmd = "/bin/touch " + sentinel
	return art, tree, sentinel, cmd
}

func writePendingFile(t *testing.T, path string, rel builtRel, art, tree, cmd, manifestPath string, inline []byte, extra bool) {
	t.Helper()
	body := map[string]any{
		"release_id":      rel.id.String(),
		"manifest_sha256": rel.sum,
		"artifact_path":   art,
		"tree_path":       tree,
		"updater_command": cmd,
	}
	if manifestPath != "" {
		body["manifest_path"] = manifestPath
	}
	if len(inline) > 0 {
		body["manifest"] = json.RawMessage(inline)
	}
	if extra {
		body["note"] = "ignored"
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertConsumed(t *testing.T, path, state string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("pending file still at %s: %v", path, err)
	}
	dest := strings.TrimSuffix(path, ".json") + "." + state + ".json"
	if _, err := os.Stat(dest); err != nil {
		t.Fatal(dest, err)
	}
}

func installerIntent(t *testing.T, w *world, rel builtRel) claimed {
	t.Helper()
	_, result := lastPayload(t, w)
	job, err := w.eng.Journal.GetJob(context.Background(), result.JobId.String(), 1)
	if err != nil {
		t.Fatal(err)
	}
	c, err := decodeClaimed(job.IntentJSON)
	if err != nil {
		t.Fatal(err)
	}
	if job.Phase != updatejournal.JobDone {
		t.Fatalf("phase %s", job.Phase)
	}
	if job.ReleaseID != rel.id.String() {
		t.Fatal("wrong release", job.ReleaseID)
	}
	return c
}

func lastPayload(t *testing.T, w *world) ([]byte, api.UpdateResult) {
	t.Helper()
	box, err := w.eng.Journal.Outbox(context.Background())
	if err != nil || len(box) == 0 {
		t.Fatalf("outbox %v %v", len(box), err)
	}
	res := w.oneResult()
	return box[len(box)-1].Payload, res
}

func assertInstallerWireSource(t *testing.T, payload []byte) {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(payload, &m); err != nil {
		t.Fatal(err)
	}
	if string(m["source"]) != `"installer"` {
		t.Fatal("wire result lacks installer source")
	}
}

func assertCurrentAbsent(t *testing.T, w *world) {
	t.Helper()
	id, err := w.layout.currentID()
	if err != nil || id != "" {
		t.Fatalf("current %s %v", id, err)
	}
}

func writeUpdaterConfig(t *testing.T, root, trust string, boot uuid.UUID, desired string) string {
	t.Helper()
	if desired == "" {
		desired = filepath.Join(root, "desired-state.json")
	}
	cfg := Config{
		Layout: testLayout(root), NodeID: uuid.Must(uuid.NewV7()), IncarnationID: uuid.Must(uuid.NewV7()),
		BootID: boot, Platform: testPlatform(), ReleaseTrust: trust, DesiredTrust: trust,
		DesiredState: desired, MeterCheckpoint: filepath.Join(root, "meter.json"),
	}
	path := filepath.Join(root, "updater.json")
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if boot == uuid.Nil {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(b, &fields); err != nil {
			t.Fatal(err)
		}
		delete(fields, "boot_id")
		b, err = json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUPD_F13_OneshotAppliesInstallerRelease(t *testing.T) {
	w := newWorld(t)
	quietDesired(w)
	touchPersistent(t, w.layout)
	idPath, usagePath := filepath.Join(w.layout.Identity, "node.key"), filepath.Join(w.layout.State, "usage-outbox")
	idBefore, usageBefore := fileBytes(t, idPath), fileBytes(t, usagePath)
	kernel, kernelRaw := kernelBootID(t)
	configBoot := uuid.Must(uuid.NewV7())
	got, fromConfig, err := resolveBootID(configBoot, "/proc/sys/kernel/random/boot_id")
	if err != nil || fromConfig || got != kernel || got == configBoot {
		t.Fatalf("boot %s fromConfig %v err %v config %s", got, fromConfig, err, configBoot)
	}
	w.node.BootID = got
	w.eng.Node.BootID = got
	if err := w.eng.Journal.SetGate(context.Background(), updatejournal.GateRecord{
		PublicationIntent: string(api.PublicationIntentPublished),
	}); err != nil {
		t.Fatal(err)
	}
	rel := w.makeRelease("staged", 1, []int{1})
	art, tree, sentinel, cmd := stageCopy(t, w, rel)
	manifestPath := filepath.Join(w.root, "manifest.env")
	if err := os.WriteFile(manifestPath, rel.env, 0o600); err != nil {
		t.Fatal(err)
	}
	pending := filepath.Join(w.layout.Staging, PendingReleaseFile)
	writePendingFile(t, pending, rel, art, tree, cmd, manifestPath, nil, true)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(rel.id)
	readme := fileBytes(t, filepath.Join(w.layout.releaseDir(rel.id.String()), "README"))
	if !bytes.Equal(readme, fileBytes(t, filepath.Join(rel.tree, "README"))) {
		t.Fatalf("installed tree %q", readme)
	}
	if _, err := os.Stat(filepath.Join(w.layout.releaseDir(rel.id.String()), "ONLY-IN-PLANT")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("planted release tree was kept", err)
	}
	if !bytes.Equal(fileBytes(t, filepath.Join(tree, "README")), []byte("PLANTED\n")) {
		t.Fatal("tree_path was used")
	}
	if _, err := os.Stat(sentinel); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("updater_command ran", err)
	}
	stops, starts := w.svc.counts()
	if stops < 1 || starts < 1 {
		t.Fatalf("services stop %d start %d", stops, starts)
	}
	assertConsumed(t, pending, "succeeded")
	c := installerIntent(t, w, rel)
	if c.Source != SourceInstaller || c.BootID != kernel || c.Intent.ReleaseId != rel.id || len(c.Serving) != 0 {
		t.Fatalf("intent source %s boot %s serving %v", c.Source, c.BootID, c.Serving)
	}
	payload, res := lastPayload(t, w)
	assertInstallerWireSource(t, payload)
	if res.State != api.UpdateResultStateSucceeded || res.RequestedReleaseId != rel.id || res.MemberForwardingEnabled || res.PublicationIntent != api.PublicationIntentPublished {
		t.Fatalf("result state %s release %s forwarding %v intent %s", res.State, res.RequestedReleaseId, res.MemberForwardingEnabled, res.PublicationIntent)
	}
	if res.BootId.String() != kernel.String() || res.BootId.String() == configBoot.String() || res.Generation.String() != "1" || res.PolicyGeneration.String() != "0" || !strings.EqualFold(res.ManifestSha256, rel.sum) {
		t.Fatalf("boot %s gen %s policy %s manifest %s", res.BootId, res.Generation, res.PolicyGeneration, res.ManifestSha256)
	}
	blocked, _, _ := w.gate.snapshot()
	if !blocked {
		t.Fatal("forwarding opened for a release the desired state has not listed")
	}
	if _, _, ok, err := w.eng.Journal.AcceptedGeneration(context.Background()); err != nil || ok {
		t.Fatalf("accepted generation moved ok=%v %v", ok, err)
	}
	applied, ok, err := w.eng.Journal.AppliedConfig(context.Background())
	if err != nil || !ok || applied.String() != "0" {
		t.Fatalf("applied config %s ok %v %v", applied, ok, err)
	}
	var holder struct {
		BootID string `json:"boot_id"`
	}
	raw, err := os.ReadFile(w.layout.Lock)
	if err != nil || json.Unmarshal(raw, &holder) != nil || !strings.EqualFold(holder.BootID, kernelRaw) {
		t.Fatalf("lock holder %s raw %s err %v", holder.BootID, kernelRaw, err)
	}
	assertSameBytes(t, idPath, idBefore)
	assertSameBytes(t, usagePath, usageBefore)
	if err := w.run(); !errors.Is(err, ErrMembersBlocked) {
		t.Fatalf("second run %v", err)
	}
	if got := w.outcomes(); len(got) != 1 || got[0] != string(api.UpdateResultStateSucceeded) {
		t.Fatalf("second run reapplied: %v", got)
	}
	w.src.err = nil
	next := w.makeRelease("later", 1, []int{1})
	work := w.work(next, "1", api.PublicationIntentPublished)
	work.PolicyGeneration = decstr.MustParse("4")
	work.PolicyKnown = true
	work.Serving = []uuid.UUID{rel.id, next.id}
	w.setWork(work)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(next.id)
	gen, _, ok, err := w.eng.Journal.AcceptedGeneration(context.Background())
	if err != nil || !ok || gen.String() != "1" {
		t.Fatalf("accepted %s ok %v %v", gen, ok, err)
	}
	applied, ok, err = w.eng.Journal.AppliedConfig(context.Background())
	if err != nil || !ok || applied.String() != "4" {
		t.Fatalf("later policy %s ok %v %v", applied, ok, err)
	}
	assertSameBytes(t, idPath, idBefore)
}

func TestUPD_F13_ApplyPendingRollsBackToLastGood(t *testing.T) {
	w := newWorld(t)
	quietDesired(w)
	touchPersistent(t, w.layout)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	if err := w.eng.Journal.SetGate(context.Background(), updatejournal.GateRecord{
		PublicationIntent: string(api.PublicationIntentPublished), TerminalOutcome: updatejournal.OutcomeSucceeded,
	}); err != nil {
		t.Fatal(err)
	}
	w.health.fail[next.id.String()] = true
	art, tree, sentinel, cmd := stageCopy(t, w, next)
	manifestPath := filepath.Join(w.root, "manifest.env")
	if err := os.WriteFile(manifestPath, next.env, 0o600); err != nil {
		t.Fatal(err)
	}
	pending := filepath.Join(w.root, "explicit-pending.json")
	writePendingFile(t, pending, next, art, tree, cmd, manifestPath, nil, false)
	if err := w.eng.ApplyPending(context.Background(), pending); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(old.id)
	if !bytes.Equal(fileBytes(t, filepath.Join(w.layout.releaseDir(next.id.String()), "README")), fileBytes(t, filepath.Join(next.tree, "README"))) {
		t.Fatal("candidate tree was the plant")
	}
	if _, err := os.Stat(sentinel); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("updater_command ran", err)
	}
	stops, starts := w.svc.counts()
	if stops < 1 || starts < 1 {
		t.Fatalf("services stop %d start %d", stops, starts)
	}
	assertConsumed(t, pending, "rolled_back")
	c := installerIntent(t, w, next)
	if c.Source != SourceInstaller || len(c.Serving) != 1 || c.Serving[0] != old.id {
		t.Fatalf("serving %v source %s", c.Serving, c.Source)
	}
	payload, res := lastPayload(t, w)
	assertInstallerWireSource(t, payload)
	if res.State != api.UpdateResultStateRolledBack || res.RequestedReleaseId != next.id || requiredValue(t, res.CurrentReleaseId) != old.id || requiredValue(t, res.PreviousReleaseId) != next.id {
		t.Fatalf("state %s current %v previous %v requested %s", res.State, res.CurrentReleaseId, res.PreviousReleaseId, res.RequestedReleaseId)
	}
	if res.PublicationIntent != api.PublicationIntentPublished || res.MemberForwardingEnabled {
		t.Fatalf("intent %s forwarding %v", res.PublicationIntent, res.MemberForwardingEnabled)
	}
	if _, _, ok, err := w.eng.Journal.AcceptedGeneration(context.Background()); err != nil || ok {
		t.Fatalf("accepted generation moved ok=%v %v", ok, err)
	}
	if err := w.eng.ApplyPending(context.Background(), pending); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("second apply %v", err)
	}
	if err := w.run(); !errors.Is(err, ErrMembersBlocked) {
		t.Fatal(err)
	}
	if got := w.outcomes(); len(got) != 1 || got[0] != string(api.UpdateResultStateRolledBack) {
		t.Fatalf("second run reapplied: %v", got)
	}
}

func TestUPD_F13_WriterRecordWithoutManifestFailsClosed(t *testing.T) {
	w := newWorld(t)
	quietDesired(w)
	rel := w.makeRelease("staged", 1, []int{1})
	art, tree, sentinel, cmd := stageCopy(t, w, rel)
	pending := filepath.Join(w.layout.Staging, PendingReleaseFile)
	writePendingFile(t, pending, rel, art, tree, cmd, "", nil, false)
	err := w.run()
	if asOp(err) == nil || asOp(err).Code != apierr.SignatureInvalid {
		t.Fatal(err)
	}
	assertCurrentAbsent(t, w)
	assertConsumed(t, pending, "failed")
	if _, statErr := os.Stat(sentinel); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatal("updater_command ran", statErr)
	}
	c := installerIntent(t, w, rel)
	if c.Source != SourceInstaller || c.Intent.ReleaseId != rel.id {
		t.Fatalf("source %s release %s", c.Source, c.Intent.ReleaseId)
	}
	payload, res := lastPayload(t, w)
	assertInstallerWireSource(t, payload)
	if res.State != api.UpdateResultStateFailed || requiredValue(t, res.ErrorCode) != api.ErrorCode(apierr.ArtifactInvalid) || requiredValue(t, res.Stage) != api.UpdateStageVerify {
		t.Fatalf("state %s code %v stage %v", res.State, res.ErrorCode, res.Stage)
	}
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	if got := w.outcomes(); len(got) != 1 {
		t.Fatalf("second run %v", got)
	}
}

func TestUPD_F13_MalformedPendingIsLeftInPlace(t *testing.T) {
	w := newWorld(t)
	quietDesired(w)
	pending := filepath.Join(w.layout.Staging, PendingReleaseFile)
	if err := os.WriteFile(pending, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := w.run()
	if err == nil || asOp(err) == nil || asOp(err).Code != apierr.ValidationFailed {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(pending); statErr != nil {
		t.Fatal("malformed pending was consumed", statErr)
	}
	assertCurrentAbsent(t, w)
	open, oerr := w.eng.Journal.OpenJobs(context.Background())
	box, berr := w.eng.Journal.Outbox(context.Background())
	if oerr != nil || berr != nil || len(open) != 0 || len(box) != 0 {
		t.Fatalf("jobs %d outbox %d %v %v", len(open), len(box), oerr, berr)
	}
}

func TestUPD_F13_TamperedArtifactFailsClosed(t *testing.T) {
	w := newWorld(t)
	quietDesired(w)
	rel := w.makeRelease("staged", 1, []int{1})
	body := append([]byte(nil), rel.amd...)
	body[len(body)-1] ^= 0xff
	art := filepath.Join(w.layout.Staging, rel.amdSHA+".tar.zst")
	if err := os.WriteFile(art, body, 0o644); err != nil {
		t.Fatal(err)
	}
	pending := filepath.Join(w.root, "tampered.json")
	writePendingFile(t, pending, rel, art, filepath.Join(w.root, "unused-tree"), "/bin/true", "", rel.env, false)
	err := w.eng.ApplyPending(context.Background(), pending)
	if asOp(err) == nil || asOp(err).Code != apierr.HashMismatch {
		t.Fatal(err)
	}
	assertCurrentAbsent(t, w)
	assertConsumed(t, pending, "failed")
	_, res := lastPayload(t, w)
	if res.State != api.UpdateResultStateFailed || requiredValue(t, res.ErrorCode) != api.ErrorCode(apierr.ArtifactInvalid) {
		t.Fatalf("state %s code %v", res.State, res.ErrorCode)
	}
}

func TestUPD_F13_SymlinkArtifactFailsClosed(t *testing.T) {
	w := newWorld(t)
	quietDesired(w)
	rel := w.makeRelease("staged", 1, []int{1})
	art, _, _, _ := stageCopy(t, w, rel)
	link := art + ".link"
	if err := os.Symlink(art, link); err != nil {
		t.Fatal(err)
	}
	pending := filepath.Join(w.root, "link.json")
	writePendingFile(t, pending, rel, link, filepath.Join(w.root, "unused-tree"), "/bin/true", "", rel.env, false)
	err := w.eng.ApplyPending(context.Background(), pending)
	if asOp(err) == nil || asOp(err).Code != apierr.ArtifactInvalid {
		t.Fatal(err)
	}
	assertCurrentAbsent(t, w)
	assertConsumed(t, pending, "failed")
}

func TestUPD_F13_PendingLockBusy(t *testing.T) {
	if os.Getenv("UPDATER_LOCK_CHILD") == "1" {
		lk, err := nodelock.TryAcquire(os.Getenv("UPDATER_LOCK_PATH"))
		if err != nil {
			os.Exit(1)
		}
		if err := os.WriteFile(os.Getenv("UPDATER_LOCK_READY"), []byte("1"), 0o600); err != nil {
			os.Exit(1)
		}
		_, _ = io.Copy(io.Discard, os.Stdin)
		_ = lk.Release()
		os.Exit(0)
	}
	w := newWorld(t)
	pending := filepath.Join(w.layout.Staging, PendingReleaseFile)
	body := []byte(`{"release_id":"not-used"}`)
	if err := os.WriteFile(pending, body, 0o600); err != nil {
		t.Fatal(err)
	}
	ready := filepath.Join(w.root, "ready")
	cmd := exec.Command(os.Args[0], "-test.run=^TestUPD_F13_PendingLockBusy$")
	cmd.Env = append(os.Environ(), "UPDATER_LOCK_CHILD=1", "UPDATER_LOCK_PATH="+w.layout.Lock, "UPDATER_LOCK_READY="+ready)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, statErr := os.Stat(ready); statErr == nil {
			break
		}
		if time.Now().After(deadline) {
			_ = stdin.Close()
			_ = cmd.Wait()
			t.Fatal("child did not take the lock")
		}
		time.Sleep(20 * time.Millisecond)
	}
	err = w.eng.ApplyPending(context.Background(), pending)
	if asOp(err) == nil || asOp(err).Code != apierr.UpdateBusy {
		_ = stdin.Close()
		_ = cmd.Wait()
		t.Fatal(err)
	}
	got, rerr := os.ReadFile(pending)
	if rerr != nil || !bytes.Equal(got, body) {
		t.Fatalf("pending changed %q %v", got, rerr)
	}
	if err := stdin.Close(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
}

func TestUPD_F13_KernelBootIDAndFallback(t *testing.T) {
	kernel, _ := kernelBootID(t)
	root := t.TempDir()
	trust := filepath.Join(root, "trust.json")
	writeTrust(t, trust)
	configBoot := uuid.Must(uuid.NewV7())
	loaded, err := LoadConfig(writeUpdaterConfig(t, root, trust, configBoot, ""))
	if err != nil || loaded.BootID != kernel || loaded.bootIDFromConfig {
		t.Fatalf("loaded %s fromConfig %v %v", loaded.BootID, loaded.bootIDFromConfig, err)
	}
	eng, err := loaded.Engine()
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	if eng.Node.BootID != kernel || eng.bootIDFromConfig {
		t.Fatal("engine kept the config boot id")
	}
	omit := writeUpdaterConfig(t, root, trust, uuid.Nil, "")
	loaded, err = LoadConfig(omit)
	if err != nil || loaded.BootID != kernel || loaded.bootIDFromConfig {
		t.Fatalf("omitted boot %s %v %v", loaded.BootID, loaded.bootIDFromConfig, err)
	}

	bootIDFile := filepath.Join(t.TempDir(), "missing-boot")
	loaded, err = loadConfigWithBootFile(writeUpdaterConfig(t, t.TempDir(), trust, configBoot, ""), probeAddressSources{}, bootIDFile)
	if err != nil || loaded.BootID != configBoot || !loaded.bootIDFromConfig {
		t.Fatalf("fallback %s fromConfig %v %v", loaded.BootID, loaded.bootIDFromConfig, err)
	}
	eng2, err := loaded.Engine()
	if err != nil {
		t.Fatal(err)
	}
	defer eng2.Close()
	var buf bytes.Buffer
	eng2.UseLog(slog.New(slog.NewTextHandler(&buf, nil)))
	if err := eng2.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := eng2.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Count(buf.String(), "kernel boot id unreadable; using boot_id from updater config") != 1 {
		t.Fatalf("log %s", &buf)
	}
	bad := filepath.Join(t.TempDir(), "boot")
	if err := os.WriteFile(bad, []byte("not-a-uuid\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bootIDFile = bad
	loaded, err = loadConfigWithBootFile(writeUpdaterConfig(t, t.TempDir(), trust, configBoot, ""), probeAddressSources{}, bootIDFile)
	if err != nil || loaded.BootID != configBoot || !loaded.bootIDFromConfig {
		t.Fatalf("non-uuid file %s %v %v", loaded.BootID, loaded.bootIDFromConfig, err)
	}
	if _, err := loadConfigWithBootFile(writeUpdaterConfig(t, t.TempDir(), trust, uuid.Nil, ""), probeAddressSources{}, bootIDFile); err == nil || !strings.Contains(err.Error(), "boot_id is required when the kernel boot id is unreadable") {
		t.Fatal(err)
	}
}

func TestUPD_F13_MissingDesiredStateIsNoWork(t *testing.T) {
	root := t.TempDir()
	trust := filepath.Join(root, "trust.json")
	writeTrust(t, trust)
	desired := filepath.Join(root, "desired-state.json")
	var buf bytes.Buffer
	eng := loadFresh(t, root, trust, desired, &buf)
	defer eng.Close()
	err := eng.Run(context.Background())
	if err != nil || ExitCode(err) != 0 {
		t.Fatal(err)
	}
	if strings.Count(buf.String(), "desired-state file is absent") != 1 {
		t.Fatalf("log %s", &buf)
	}
	if _, statErr := os.Stat(filepath.Join(eng.Layout.Journal, "forwarding.blocked")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatal("forwarding blocked", statErr)
	}
	box, berr := eng.Journal.Outbox(context.Background())
	open, oerr := eng.Journal.OpenJobs(context.Background())
	if berr != nil || oerr != nil || len(box) != 0 || len(open) != 0 {
		t.Fatalf("outbox %d jobs %d %v %v", len(box), len(open), berr, oerr)
	}

	root2 := t.TempDir()
	badDesired := filepath.Join(root2, "desired-state.json")
	if err := os.WriteFile(badDesired, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	engBad := loadFresh(t, root2, trust, badDesired, nil)
	defer engBad.Close()
	err = engBad.Run(context.Background())
	if err == nil || errors.Is(err, ErrNoWork) || ExitCode(err) == 0 {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(filepath.Join(engBad.Layout.Journal, "forwarding.blocked")); statErr != nil {
		t.Fatal("malformed file did not block", statErr)
	}

	root3 := t.TempDir()
	dirDesired := filepath.Join(root3, "desired-state.json")
	if err := os.Mkdir(dirDesired, 0o755); err != nil {
		t.Fatal(err)
	}
	engDir := loadFresh(t, root3, trust, dirDesired, nil)
	defer engDir.Close()
	err = engDir.Run(context.Background())
	if err == nil || errors.Is(err, ErrNoWork) {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(filepath.Join(engDir.Layout.Journal, "forwarding.blocked")); statErr != nil {
		t.Fatal("unreadable desired state did not block", statErr)
	}
}

func loadFresh(t *testing.T, root, trust, desired string, buf *bytes.Buffer) *Engine {
	t.Helper()
	if _, err := os.Stat(trust); err != nil {
		writeTrust(t, trust)
	}
	loaded, err := LoadConfig(writeUpdaterConfig(t, root, trust, uuid.Must(uuid.NewV7()), desired))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := loaded.Engine()
	if err != nil {
		t.Fatal(err)
	}
	if buf == nil {
		buf = &bytes.Buffer{}
	}
	eng.UseLog(slog.New(slog.NewTextHandler(buf, nil)))
	return eng
}

func stageInstaller(t *testing.T, w *world, rel builtRel) string {
	t.Helper()
	art, tree, _, cmd := stageCopy(t, w, rel)
	path := filepath.Join(w.layout.Staging, PendingReleaseFile)
	writePendingFile(t, path, rel, art, tree, cmd, "", rel.env, false)
	return path
}

func TestAT25_InstallerRecoveryKeepsClaimAndConsumesPending(t *testing.T) {
	for _, stage := range []updatejournal.Stage{updatejournal.StagePrepared, updatejournal.StageSwitched, updatejournal.StageChecking} {
		t.Run(string(stage), func(t *testing.T) {
			w := newWorld(t)
			quietDesired(w)
			rel := w.makeRelease("candidate", 1, []int{1})
			pending := stageInstaller(t, w, rel)
			w.eng.Hooks.AfterStage = func(s string) error {
				if s == string(stage) {
					return ErrCrashed
				}
				return nil
			}
			if err := w.eng.ApplyPending(context.Background(), pending); !errors.Is(err, ErrCrashed) {
				t.Fatal(err)
			}
			txn, err := w.eng.Journal.ActiveTxn(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			boot := w.eng.Node.BootID
			if err := w.eng.Close(); err != nil {
				t.Fatal(err)
			}
			w.eng.Journal = reopen(t, w.layout.Journal)
			w.eng.Node.BootID = uuid.Must(uuid.NewV7())
			w.eng.Hooks.AfterStage = nil
			if err := w.eng.ApplyPending(context.Background(), pending); !errors.Is(err, ErrMembersBlocked) {
				t.Fatal(err)
			}
			last, err := w.eng.Journal.LastTxn(context.Background())
			if err != nil || last.TxnID != txn.TxnID {
				t.Fatalf("txn changed: %v %v", last, err)
			}
			assertConsumed(t, pending, "succeeded")
			w.mustCurrent(rel.id)
			_, r := lastPayload(t, w)
			if r.BootId != boot || len(w.outcomes()) != 1 {
				t.Fatalf("claim/result changed: %+v", r)
			}
			snap, err := updatejournal.Snapshot(context.Background(), filepath.Join(w.layout.Journal, "journal.db"))
			if err != nil || snap.Source != SourceInstaller || snap.RequestedRelease != rel.id.String() || snap.State != "succeeded" {
				t.Fatalf("snapshot %+v %v", snap, err)
			}
		})
	}
}

func TestAT26_InstallerCleanupSurvivesSealAndPreservesNewRequest(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		t.Run(fmt.Sprint(replacement), func(t *testing.T) {
			w := newWorld(t)
			quietDesired(w)
			rel := w.makeRelease("candidate", 1, []int{1})
			pending := stageInstaller(t, w, rel)
			dest := strings.TrimSuffix(pending, ".json") + ".succeeded.json"
			if err := os.Mkdir(dest, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := w.eng.ApplyPending(context.Background(), pending); err == nil {
				t.Fatal("rename failure ignored")
			}
			c := installerIntent(t, w, rel)
			jobs, err := w.eng.Journal.InstallerCleanups(context.Background())
			if err != nil || len(jobs) != 1 {
				t.Fatalf("cleanup was not durable: %v %v", jobs, err)
			}
			if err := os.Remove(dest); err != nil {
				t.Fatal(err)
			}
			if replacement {
				stageInstaller(t, w, w.makeRelease("later", 1, []int{1}))
			}
			before := fileBytes(t, pending)
			if err := w.eng.Close(); err != nil {
				t.Fatal(err)
			}
			w.eng.Journal = reopen(t, w.layout.Journal)
			if err := w.eng.Recover(context.Background()); err != nil && !errors.Is(err, ErrMembersBlocked) {
				t.Fatal(err)
			}
			if replacement {
				assertSameBytes(t, pending, before)
			} else {
				assertConsumed(t, pending, "succeeded")
			}
			jobs, err = w.eng.Journal.InstallerCleanups(context.Background())
			if err != nil || len(jobs) != 0 || len(w.outcomes()) != 1 {
				t.Fatalf("cleanup repeated: %v %v", jobs, err)
			}
			if c.Source != SourceInstaller {
				t.Fatal(c.Source)
			}
		})
	}
}

func TestUPD02_InstallerRejectsUntrustedAndRevokedReleases(t *testing.T) {
	for _, mode := range []string{"signature", "node_revoked", "config_revoked", "signed_revoked", "unjournaled_current"} {
		t.Run(mode, func(t *testing.T) {
			w := newWorld(t)
			quietDesired(w)
			rel := w.makeRelease("candidate", 1, []int{1})
			expected := apierr.ReleaseNotApproved
			switch mode {
			case "signature":
				var env envelope.Envelope
				if err := json.Unmarshal(rel.env, &env); err != nil {
					t.Fatal(err)
				}
				env.Signatures[0].SignatureB64 = strings.Repeat("A", 86) + "=="
				raw, err := json.Marshal(env)
				if err != nil {
					t.Fatal(err)
				}
				rel.env = raw
				rel.sum, err = release.EnvelopeSHA256(raw)
				if err != nil {
					t.Fatal(err)
				}
				expected = apierr.SignatureInvalid
			case "node_revoked":
				w.eng.Node.Revoked = true
				expected = apierr.IdentityRevoked
			case "config_revoked":
				w.eng.RevokedReleaseIDs = []uuid.UUID{rel.id}
			case "signed_revoked":
				if _, _, err := w.eng.Journal.AcceptDesired(context.Background(), 1, true, []string{rel.id.String()}); err != nil {
					t.Fatal(err)
				}
			case "unjournaled_current":
				if err := os.Symlink(filepath.Join("releases", uuid.Must(uuid.NewV7()).String()), w.layout.Current); err != nil {
					t.Fatal(err)
				}
				expected = apierr.CheckpointFailed
			}
			pending := stageInstaller(t, w, rel)
			err := w.eng.ApplyPending(context.Background(), pending)
			if asOp(err) == nil || asOp(err).Code != expected {
				t.Fatalf("want %s: %v", expected, err)
			}
			assertConsumed(t, pending, "failed")
			stops, starts := w.svc.counts()
			if stops != 0 || starts != 0 {
				t.Fatal("services changed")
			}
			if mode != "unjournaled_current" {
				assertCurrentAbsent(t, w)
			}
		})
	}
}

func TestUPD02_InstallerSnapshotDoesNotTrustReplacedArtifact(t *testing.T) {
	w := newWorld(t)
	rel := w.makeRelease("candidate", 1, []int{1})
	art, _, _, _ := stageCopy(t, w, rel)
	selected, err := selectArtifact(rel.manifest, "ubuntu", "24.04", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	path, err := w.eng.snapshotStagedArtifact(art, selected)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	if err := os.WriteFile(art, []byte("tampered after copy"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := release.CheckArtifact(rel.manifest, "amd64", path); err != nil {
		t.Fatal(err)
	}
}

func TestAT26_InstallerCorrectedRecordIsNewWork(t *testing.T) {
	w := newWorld(t)
	quietDesired(w)
	rel := w.makeRelease("candidate", 1, []int{1})
	art, tree, _, cmd := stageCopy(t, w, rel)
	pending := filepath.Join(w.layout.Staging, PendingReleaseFile)
	writePendingFile(t, pending, rel, art, tree, cmd, "", nil, false)
	if err := w.run(); err == nil {
		t.Fatal("missing manifest accepted")
	}
	writePendingFile(t, pending, rel, art, tree, cmd, "", rel.env, false)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(rel.id)
	outcomes := w.outcomes()
	if len(outcomes) != 2 || outcomes[0] != "failed" || outcomes[1] != "succeeded" {
		t.Fatal(outcomes)
	}
}

func TestD080_InstallerSnapshotAfterCentralRetry(t *testing.T) {
	w := newWorld(t)
	quietDesired(w)
	rel := w.makeRelease("candidate", 1, []int{1})
	_, _, err := w.eng.Journal.ClaimJob(context.Background(), updatejournal.Job{
		JobID: uuid.Must(uuid.NewV7()).String(), Attempt: 5, Generation: 7, ReleaseID: uuid.Must(uuid.NewV7()).String(), ManifestSHA256: strings.Repeat("a", 64), IntentJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	pending := stageInstaller(t, w, rel)
	if err := w.eng.ApplyPending(context.Background(), pending); err != nil {
		t.Fatal(err)
	}
	snap, err := updatejournal.Snapshot(context.Background(), filepath.Join(w.layout.Journal, "journal.db"))
	if err != nil || snap.Source != SourceInstaller || snap.RequestedRelease != rel.id.String() {
		t.Fatalf("snapshot %+v %v", snap, err)
	}
	gen, _, _, err := w.eng.Journal.AcceptedGeneration(context.Background())
	if err != nil || gen != 7 {
		t.Fatalf("generation %v %v", gen, err)
	}
}

func TestUPD_MissingDesiredTargetFailsClosed(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "desired.json")
	if err := os.Symlink(filepath.Join(root, "missing"), path); err != nil {
		t.Fatal(err)
	}
	gate := FileGate{Path: filepath.Join(root, "forwarding.blocked")}
	source := GuardSource{Inner: FileSource{Path: path}, Gate: gate}
	if _, err := source.Current(context.Background()); err == nil || errors.Is(err, ErrNoWork) {
		t.Fatal(err)
	}
	if !gate.Blocked() {
		t.Fatal("unreadable desired state did not block")
	}
}

func TestAT25_InstallerRecoveredFailureKeepsExitBlocked(t *testing.T) {
	w := newWorld(t)
	quietDesired(w)
	rel := w.makeRelease("candidate", 1, []int{1})
	w.health.fail[rel.id.String()] = true
	pending := stageInstaller(t, w, rel)
	w.eng.Hooks.AfterStage = func(s string) error {
		if s == string(updatejournal.StageChecking) {
			return ErrCrashed
		}
		return nil
	}
	if err := w.eng.ApplyPending(context.Background(), pending); !errors.Is(err, ErrCrashed) {
		t.Fatal(err)
	}
	w.eng.Hooks.AfterStage = nil
	if err := w.eng.ApplyPending(context.Background(), pending); !errors.Is(err, ErrMembersBlocked) || ExitCode(err) != 2 {
		t.Fatalf("recovery was reported successful: %v", err)
	}
	assertConsumed(t, pending, "failed")
	_, r := lastPayload(t, w)
	if r.State != api.UpdateResultStateFailed || !requiredValue(t, r.RecoveryRequired) {
		t.Fatal("missing recovery evidence")
	}
}

func TestAT25_InstallerOrphanSnapshotCleanup(t *testing.T) {
	w := newWorld(t)
	quietDesired(w)
	owned := filepath.Join(w.layout.Staging, ".installer-orphan.tar.zst")
	archive := filepath.Join(w.layout.Staging, "retained.tar.zst")
	for _, path := range []string{owned, archive} {
		if err := os.WriteFile(path, []byte("TEST ONLY"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.eng.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(owned); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("orphan snapshot not removed", err)
	}
	if _, err := os.Stat(archive); err != nil {
		t.Fatal("unrelated archive removed", err)
	}
}
