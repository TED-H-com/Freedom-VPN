package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

func TestVersionAndUsage(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"version"}, &out, &errb); code != 0 || strings.TrimSpace(out.String()) != updater.Version {
		t.Fatalf("version code %d out %q err %q", code, out.String(), errb.String())
	}
	if code := run(nil, &out, &errb); code != 64 {
		t.Fatalf("usage %d", code)
	}
	if code := run([]string{"nope"}, &out, &errb); code != 64 {
		t.Fatalf("unknown %d", code)
	}
	if code := run([]string{"run"}, &out, &errb); code != 64 {
		t.Fatalf("missing config %d", code)
	}
	if code := run([]string{"apply"}, &out, &errb); code != 64 {
		t.Fatalf("apply usage %d", code)
	}
	if code := run([]string{"apply", "-pending", filepath.Join(t.TempDir(), "pending.json")}, &out, &errb); code != 64 {
		t.Fatalf("apply without config %d", code)
	}
	if code := run([]string{"apply", "-config", filepath.Join(t.TempDir(), "updater.json")}, &out, &errb); code != 64 {
		t.Fatalf("apply without pending %d", code)
	}
	if code := run([]string{"run", "-not-a-flag"}, &out, &errb); code != 64 {
		t.Fatalf("flag %d", code)
	}
	if code := run([]string{"help"}, &out, &errb); code != 0 {
		t.Fatalf("help %d", code)
	}
}

func TestD062_StatusShowsCandidateHealthWithoutStartingEngine(t *testing.T) {
	dir := t.TempDir()
	j, err := updatejournal.Open(dir, updatejournal.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	ctx := context.Background()
	if _, _, err := j.ClaimJob(ctx, updatejournal.Job{JobID: "job", Attempt: 1, Generation: 1,
		ReleaseID: "candidate", ManifestSHA256: "aa", IntentJSON: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	if _, err := j.BeginTxn(ctx, updatejournal.Txn{TxnID: "txn", JobID: "job", Attempt: 1,
		Generation: 1, ReleaseID: "candidate", ManifestSHA256: "aa"}); err != nil {
		t.Fatal(err)
	}
	for _, stage := range []updatejournal.Stage{updatejournal.StageBlocked, updatejournal.StageFinalSampled,
		updatejournal.StageStopped, updatejournal.StageMigrated, updatejournal.StageSwitched, updatejournal.StageChecking} {
		if _, err := j.Advance(ctx, "txn", stage, updatejournal.StageUpdate{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := j.RecordHealth(ctx, "txn", map[string]string{"tunnel_probe": "failed"}); err != nil {
		t.Fatal(err)
	}
	if _, err := j.Advance(ctx, "txn", updatejournal.StageRecoveryRequired, updatejournal.StageUpdate{}); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	if code := run([]string{"status", "-journal", filepath.Join(dir, updatejournal.FileName)}, &out, &errb); code != 0 {
		t.Fatalf("status code=%d stderr=%s", code, &errb)
	}
	var got struct {
		State            string            `json:"state"`
		RecoveryRequired bool              `json:"recovery_required"`
		CandidateHealth  map[string]string `json:"candidate_health_json"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || got.State != "failed" || !got.RecoveryRequired || got.CandidateHealth["tunnel_probe"] != "failed" {
		t.Fatalf("status evidence: %s %v", &out, err)
	}
	missing := filepath.Join(dir, "missing", updatejournal.FileName)
	if code := run([]string{"status", "-journal", missing}, &out, &errb); code != 1 {
		t.Fatal("missing journal reported success", code)
	}
	if _, err := os.Stat(filepath.Dir(missing)); !os.IsNotExist(err) {
		t.Fatal("status created a directory", err)
	}
}

func TestD062_StatusShowsWireCandidateHealthForRollback(t *testing.T) {
	for _, checked := range []bool{false, true} {
		t.Run(map[bool]string{false: "not_checked", true: "checked"}[checked], func(t *testing.T) {
			j, err := updatejournal.Open(t.TempDir(), updatejournal.Options{})
			if err != nil {
				t.Fatal(err)
			}
			defer j.Close()
			ctx := context.Background()
			if _, _, err := j.ClaimJob(ctx, updatejournal.Job{JobID: "job", Attempt: 1, Generation: 1,
				ReleaseID: "candidate", ManifestSHA256: "aa", IntentJSON: []byte(`{}`)}); err != nil {
				t.Fatal(err)
			}
			if _, err := j.BeginTxn(ctx, updatejournal.Txn{TxnID: "txn", JobID: "job", Attempt: 1,
				Generation: 1, ReleaseID: "candidate", ManifestSHA256: "aa"}); err != nil {
				t.Fatal(err)
			}
			if checked {
				for _, stage := range []updatejournal.Stage{updatejournal.StageBlocked, updatejournal.StageFinalSampled,
					updatejournal.StageStopped, updatejournal.StageMigrated, updatejournal.StageSwitched, updatejournal.StageChecking} {
					if _, err := j.Advance(ctx, "txn", stage, updatejournal.StageUpdate{}); err != nil {
						t.Fatal(err)
					}
				}
				if err := j.RecordHealth(ctx, "txn", map[string]string{"tunnel_probe": "failed"}); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := j.Advance(ctx, "txn", updatejournal.StageRollingBack, updatejournal.StageUpdate{Destructive: true}); err != nil {
				t.Fatal(err)
			}
			if _, err := j.Advance(ctx, "txn", updatejournal.StageRolledBack, updatejournal.StageUpdate{Health: map[string]string{"tunnel_probe": "passed"}}); err != nil {
				t.Fatal(err)
			}
			var out, errb bytes.Buffer
			if code := run([]string{"status", "-journal", filepath.Join(j.Dir(), updatejournal.FileName)}, &out, &errb); code != 0 {
				t.Fatalf("status code=%d stderr=%s", code, &errb)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(out.Bytes(), &fields); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(fields["candidate_health"], fields["candidate_health_json"]) ||
				bytes.Equal(fields["health"], fields["candidate_health"]) ||
				(string(fields["candidate_health"]) == "null") == checked {
				t.Fatalf("incorrect candidate health in status: %s", &out)
			}
		})
	}
}

func TestF13_FreshNodeExitsZeroAndMalformedDesiredStateFails(t *testing.T) {
	root := t.TempDir()
	trust := filepath.Join(root, "trust.json")
	writeCLITrust(t, trust)
	lay := updater.Layout{
		Releases: filepath.Join(root, "opt", "releases"),
		Current:  filepath.Join(root, "opt", "current"),
		Identity: filepath.Join(root, "var", "identity"),
		State:    filepath.Join(root, "var", "state"),
		Journal:  filepath.Join(root, "var", "journal"),
		Lock:     filepath.Join(root, "var", "update.lock"),
		Staging:  filepath.Join(root, "var", "staging"),
		Trust:    trust,
	}
	desired := filepath.Join(root, "desired-state.json")
	cfgPath := filepath.Join(root, "updater.json")
	writeCLIConfig(t, cfgPath, lay, trust, desired, uuid.Must(uuid.NewV7()))
	var out, errb bytes.Buffer
	if code := run([]string{"run", "-config", cfgPath}, &out, &errb); code != 0 {
		t.Fatalf("fresh node code %d stderr %s", code, &errb)
	}
	if strings.Count(errb.String(), "desired-state file is absent") != 1 {
		t.Fatalf("log %s", &errb)
	}
	if _, err := os.Stat(filepath.Join(lay.Journal, "forwarding.blocked")); !os.IsNotExist(err) {
		t.Fatal("forwarding blocked on a fresh node", err)
	}
	if err := os.WriteFile(desired, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	errb.Reset()
	if code := run([]string{"run", "-config", cfgPath}, &out, &errb); code != 1 {
		t.Fatalf("malformed desired state code %d stderr %s", code, &errb)
	}
	if _, err := os.Stat(filepath.Join(lay.Journal, "forwarding.blocked")); err != nil {
		t.Fatal("malformed desired state left forwarding open", err)
	}
}

func writeCLITrust(t *testing.T, path string) {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	ts, err := envelope.NewNodeTrustStore(envelope.TrustedKey{
		KeyID: "release-test", PublicKey: pub, Purposes: []envelope.Purpose{envelope.PurposeRelease},
		NotBefore: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:  time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(ts)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeCLIConfig(t *testing.T, path string, lay updater.Layout, trust, desired string, boot uuid.UUID) {
	t.Helper()
	cfg := updater.Config{
		Layout: lay, NodeID: uuid.Must(uuid.NewV7()), IncarnationID: uuid.Must(uuid.NewV7()), BootID: boot,
		Platform: updater.Platform{
			OSID: "ubuntu", OSVersion: "24.04", Architecture: "amd64", UpdaterVersion: "1.2.0",
			ControlAPIMajor: 1, PolicySchema: 1, UsageSchema: 1, StateSchema: 1,
		},
		ReleaseTrust: trust, DesiredTrust: trust, DesiredState: desired,
		MeterCheckpoint: filepath.Join(filepath.Dir(path), "meter.json"),
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestD080_ApplyCommandRecordsInstallerFailure(t *testing.T) {
	root := t.TempDir()
	trust := filepath.Join(root, "trust.json")
	writeCLITrust(t, trust)
	lay := updater.Layout{Releases: filepath.Join(root, "releases"), Current: filepath.Join(root, "current"), Identity: filepath.Join(root, "identity"), State: filepath.Join(root, "state"), Journal: filepath.Join(root, "journal"), Lock: filepath.Join(root, "update.lock"), Staging: filepath.Join(root, "staging"), Trust: trust}
	cfg := filepath.Join(root, "updater.json")
	writeCLIConfig(t, cfg, lay, trust, filepath.Join(root, "desired.json"), uuid.Nil)
	pending := filepath.Join(root, "pending.json")
	id := uuid.Must(uuid.NewV7())
	body, err := json.Marshal(map[string]string{"release_id": id.String(), "manifest_sha256": strings.Repeat("a", 64), "artifact_path": filepath.Join(root, "archive.tar.zst")})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pending, body, 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	if code := run([]string{"apply", "--config", cfg, "--pending", pending}, &out, &errb); code != 1 {
		t.Fatalf("code %d stderr %s", code, &errb)
	}
	snap, err := updatejournal.Snapshot(context.Background(), filepath.Join(lay.Journal, "journal.db"))
	if err != nil || snap.Source != "installer" || snap.RequestedRelease != id.String() || snap.State != "failed" {
		t.Fatalf("snapshot %+v %v", snap, err)
	}
	if _, err := os.Stat(filepath.Join(root, "pending.failed.json")); err != nil {
		t.Fatal(err)
	}
}
