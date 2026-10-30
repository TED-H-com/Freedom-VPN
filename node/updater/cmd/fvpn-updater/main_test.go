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
