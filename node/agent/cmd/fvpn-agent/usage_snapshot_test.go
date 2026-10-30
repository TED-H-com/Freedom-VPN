package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/node/agent"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
)

func TestUsageSnapshotCommand(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeRunConfig(t, dir)
	var stderr bytes.Buffer
	if code := usageCommandTo(nil, &bytes.Buffer{}, &stderr); code != agent.ExitSnapshotArgs {
		t.Fatalf("no args %d %s", code, stderr.String())
	}
	stderr.Reset()
	if code := usageSnapshotTo([]string{"-json"}, &bytes.Buffer{}, &stderr); code != agent.ExitSnapshotArgs {
		t.Fatalf("missing config %d %s", code, stderr.String())
	}
	if strings.Contains(stderr.String(), dir) {
		t.Fatalf("stderr leaked a path: %s", stderr.String())
	}

	stderr.Reset()
	var stdout bytes.Buffer
	code := usageSnapshotTo([]string{"-config", cfgPath, "-json"}, &stdout, &stderr)
	if code != agent.ExitSnapshotAbsent || stdout.Len() != 0 {
		t.Fatalf("absent code %d out %q err %s", code, stdout.String(), stderr.String())
	}
	if stderr.String() != "fvpn-agent: usage store is absent\n" {
		t.Fatalf("stderr %q", stderr.String())
	}

	db := filepath.Join(dir, "state", "usage.db")
	if err := os.MkdirAll(filepath.Dir(db), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(db, []byte("not a database"), 0o600); err != nil {
		t.Fatal(err)
	}
	stderr.Reset()
	stdout.Reset()
	code = usageSnapshotTo([]string{"-config", cfgPath}, &stdout, &stderr)
	if code != agent.ExitSnapshotCorrupt || stderr.String() != "fvpn-agent: usage store is corrupt\n" {
		t.Fatalf("corrupt code %d %q", code, stderr.String())
	}

	if err := os.Remove(db); err != nil {
		t.Fatal(err)
	}
	node := uuid.MustParse("93bc18d9-26c2-4ec4-bf5b-0c70df89b0cb")
	inc := uuid.MustParse("d1c5d642-f6ac-4ca4-bcd6-5460d491a2e0")
	st, err := usagestore.Open(db, node, inc, usagestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	stderr.Reset()
	stdout.Reset()
	if code := usageSnapshotTo([]string{"-config", cfgPath, "-json"}, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("json code %d %s", code, stderr.String())
	}
	var again bytes.Buffer
	if code := usageSnapshotTo([]string{"-json", "-config", cfgPath}, &again, &stderr); code != 0 {
		t.Fatal(code)
	}
	if !bytes.Equal(stdout.Bytes(), again.Bytes()) {
		t.Fatalf("not identical\n%s\n%s", stdout.Bytes(), again.Bytes())
	}
	var snap usagestore.Snapshot
	if err := json.Unmarshal(stdout.Bytes(), &snap); err != nil {
		t.Fatal(err)
	}
	if snap.SchemaVersion != 1 || snap.NodeID != node.String() || snap.LastEventSeq != "0" || snap.AckedThrough != "0" || snap.UnackedCount != 0 {
		t.Fatalf("%+v", snap)
	}
	if snap.LedgerEpoch != nil || len(snap.MeterEpochs) != 0 || len(snap.UnackedEvents) != 0 {
		t.Fatalf("%+v", snap)
	}

	stderr.Reset()
	stdout.Reset()
	if code := usageSnapshotTo([]string{"-config", cfgPath}, &stdout, &stderr); code != 0 {
		t.Fatal(stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("last_event_seq: 0\n")) || !bytes.Contains(stdout.Bytes(), []byte("acked_through: 0\n")) {
		t.Fatalf("text %s", stdout.String())
	}
}

func TestUsageSnapshotProcess(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeRunConfig(t, dir)
	db := filepath.Join(dir, "state", "usage.db")
	if err := os.MkdirAll(filepath.Dir(db), 0o700); err != nil {
		t.Fatal(err)
	}
	node := uuid.MustParse("93bc18d9-26c2-4ec4-bf5b-0c70df89b0cb")
	inc := uuid.MustParse("d1c5d642-f6ac-4ca4-bcd6-5460d491a2e0")
	st, err := usagestore.Open(db, node, inc, usagestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	dev := uuid.MustParse("2df6c7df-e7e8-4bd4-9f2c-a5836dd0d8ea")
	mem := uuid.MustParse("dc01a74b-9e6f-41ac-81cd-03291d0a88c0")
	epoch := uuid.Must(uuid.NewV7())
	zero := usagestore.Counter{
		Name: "u_" + strings.ReplaceAll(dev.String(), "-", ""), DeviceID: dev, Direction: "upload",
		MemberID: mem, BindingRevision: 7,
	}
	if _, err := st.Commit(context.Background(), usagestore.Commit{
		Counters: []usagestore.Counter{zero}, Open: &usagestore.NewEpoch{ID: epoch, Reason: "baseline", BootID: "boot"},
		BootID: "boot", SampledAt: at,
	}); err != nil {
		t.Fatal(err)
	}
	bumped := zero
	bumped.Bytes = 15
	bumped.Packets = 1
	if _, err := st.Commit(context.Background(), usagestore.Commit{
		Epoch: epoch, Counters: []usagestore.Counter{bumped}, BootID: "boot", SampledAt: at.Add(10 * time.Second),
		Event: &usagestore.Draft{
			EpochID: epoch, Start: at, End: at.Add(10 * time.Second), Quality: "complete",
			Records: []usagestore.DraftRecord{{
				MemberID: mem, DeviceID: dev, BindingRevision: 7, PeriodID: "2026-10", Upload: 15,
			}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	run := func() []byte {
		t.Helper()
		cmd := exec.Command(exe, "usage", "snapshot", "-config", cfgPath, "-json")
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	a := run()
	b := run()
	if !bytes.Equal(a, b) {
		t.Fatalf("process snapshots diverged\n%s\n%s", a, b)
	}
	if !bytes.Contains(a, []byte(`"event_seq":"1"`)) || !bytes.Contains(a, []byte(`"upload_bytes":"15"`)) || !bytes.Contains(a, []byte(`"payload_sha256":"`)) {
		t.Fatalf("stdout %s", a)
	}
	if bytes.Contains(a, []byte("BEGIN ")) || bytes.Contains(bytes.ToLower(a), []byte("token")) {
		t.Fatalf("stdout looks like a secret or a trace: %s", a)
	}
}

func writeRunConfig(t *testing.T, dir string) string {
	t.Helper()
	cfg := `{
		"state_dir": "` + filepath.Join(dir, "state") + `",
		"identity_dir": "` + filepath.Join(dir, "identity") + `",
		"journal_dir": "` + filepath.Join(dir, "journal") + `",
		"lock_path": "` + filepath.Join(dir, "update.lock") + `",
		"updater_journal": "` + filepath.Join(dir, "journal", "journal.db") + `",
		"control_url": "https://control.example.invalid",
		"server_ca_file": "` + filepath.Join(dir, "ca.pem") + `",
		"trust_bundle": "` + filepath.Join(dir, "trust.json") + `",
		"datapath": {"member_ingress": "fv-fh", "uplink": "fv-ih", "snat": "192.0.2.37"},
		"agent_release": "0.0.0-test",
		"host_reserve_bps": 1000000,
		"netbird_backend": "kernel"
	}`
	path := filepath.Join(dir, "agent-run.json")
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
