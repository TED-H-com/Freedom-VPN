package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// Reproduce the durable shape in devstack 18 run 09cb1198, with generated
// fixture identities: failed/final_sample/rolled_back, null health and usage,
// retained seeded current, no last_good and no candidate switch. Every command
// opens that same disk journal afresh; no devstack fixture is changed.
func TestF903_Devstack18SealedServingFailureCLI(t *testing.T) {
	lay, config, baseline, pending, record := sealedFailureCLIInputs(t)
	apply := []string{"apply", "--config", config, "--pending", pending}
	var out, stderr bytes.Buffer
	if err := os.WriteFile(pending, record, 0600); err != nil {
		t.Fatal(err)
	}
	if code := run(apply, &out, &stderr); code != 1 || !strings.Contains(stderr.String(), "outcome=failed") || !strings.Contains(stderr.String(), "CHECKPOINT_FAILED") {
		t.Fatalf("initial failure exit=%d stderr=%s", code, &stderr)
	}
	readSeal := func() []byte {
		t.Helper()
		j, err := updatejournal.Open(lay.Journal, updatejournal.Options{})
		if err != nil {
			t.Fatal(err)
		}
		defer j.Close()
		box, err := j.Outbox(t.Context())
		if err != nil || len(box) != 1 {
			t.Fatal("missing seal", err)
		}
		var res api.UpdateResult
		if err := json.Unmarshal(box[0].Payload, &res); err != nil || !res.Health.IsNull() || !res.UsageCheckpoint.IsNull() || res.MemberForwardingEnabled {
			t.Fatalf("unexpected failure evidence: %s %v", box[0].Payload, err)
		}
		return box[0].Payload
	}
	sealed := readSeal()
	// Feed the already sealed journal to apply: it must report the stored
	// outcome, even though replay performs no work and raises no new error.
	if err := os.WriteFile(pending, record, 0600); err != nil {
		t.Fatal(err)
	}
	stderr.Reset()
	if code := run(apply, &out, &stderr); code != 1 || !strings.Contains(stderr.String(), "outcome=failed") || !strings.Contains(stderr.String(), "CHECKPOINT_FAILED") {
		t.Fatalf("sealed apply exit=%d stderr=%s", code, &stderr)
	}
	stderr.Reset()
	if code := run([]string{"recover", "--config", config}, &out, &stderr); code != 2 {
		t.Fatalf("blocked recovery exit=%d stderr=%s", code, &stderr)
	}
	out.Reset()
	if code := run([]string{"status", "--journal", filepath.Join(lay.Journal, updatejournal.FileName)}, &out, &stderr); code != 0 {
		t.Fatalf("status exit=%d stderr=%s", code, &stderr)
	}
	var snap updatejournal.SnapshotState
	if err := json.Unmarshal(out.Bytes(), &snap); err != nil || snap.State != "failed" || snap.Stage != "final_sample" || snap.JournalStage != "rolled_back" || snap.Source != "installer" || snap.CurrentRelease != baseline || snap.LastGoodRelease != "" || snap.SeededBaseline == nil || snap.RecoveryRequired || string(snap.Health) != "null" || string(snap.CandidateHealth) != "null" {
		t.Fatalf("status differs from serving failure: %s %v", &out, err)
	}
	if got, err := filepath.EvalSymlinks(lay.Current); err != nil || got != filepath.Join(lay.Releases, baseline) {
		t.Fatal("candidate switched", got, err)
	}
	if _, err := os.Stat(filepath.Join(lay.State, "meter.checkpoint")); !os.IsNotExist(err) {
		t.Fatal("checkpoint appeared", err)
	}
	if _, err := os.Stat(filepath.Join(lay.Journal, "forwarding.blocked")); err != nil {
		t.Fatal("failure no longer blocked", err)
	}
	if !bytes.Equal(sealed, readSeal()) {
		t.Fatal("CLI replay changed sealed result")
	}
}
