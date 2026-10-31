package updater

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func reviewCheckpoint(epoch string, seq uint64) []byte {
	raw, err := json.Marshal(UsageCheckpoint{
		ClosedMeterEpochID: epoch, LastDurableEventSeq: decstr.U64(seq), NewMeterEpochID: "",
	})
	if err != nil {
		panic(err)
	}
	return append(raw, '\n')
}

func TestReviewR17_CraftedCheckpointCannotSealTheBarrier(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "meter.checkpoint")
	m := FileMeter{Path: path, Wait: 200 * time.Millisecond}
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := m.FinalSample(t.Context()); err == nil {
		t.Fatal("truncated checkpoint decoded")
	}
	target := filepath.Join(dir, "real.json")
	if err := os.WriteFile(target, reviewCheckpoint(uuid.NewString(), 1), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if _, err := m.FinalSample(t.Context()); err == nil {
		t.Fatal("symlink checkpoint decoded")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	epoch := uuid.NewString()
	if err := fsutil.WriteFileAtomic(path, reviewCheckpoint(epoch, 9), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := m.FinalSample(t.Context())
	if err != nil || got.ClosedMeterEpochID != epoch || got.NewMeterEpochID != "" {
		t.Fatalf("well-formed foreign epoch was rejected by the decoder: %+v %v", got, err)
	}

	boot := uuid.New()
	gate := filepath.Join(dir, "gate")
	eng := &Engine{Layout: Layout{Journal: dir}, Node: Node{BootID: boot}, Gate: FileGate{Path: gate}}
	txn := updatejournal.Txn{TxnID: "t1"}
	if err := eng.prepareMeterWait(txn, m); err != nil {
		t.Fatal(err)
	}
	_, err = eng.waitMeter(t.Context(), txn, m)
	var op *OpError
	if !errors.As(err, &op) || op.Detail != "meter_checkpoint_uncorrelated" {
		t.Fatalf("TODO(review): crafted checkpoint sealed the barrier: %v", err)
	}
}

func TestReviewR17_UpdaterDoesNotWriteEtcVpnNode(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{"layout.go": true, "probe_addresses.go": true, "host.go": true}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "/etc/vpn-node") && !allowed[name] {
			t.Fatalf("%s references /etc/vpn-node", name)
		}
	}
	if DefaultLayout().Trust != "/etc/vpn-node/release-trust.json" {
		t.Fatal(DefaultLayout().Trust)
	}
	layout := string(mustRead(t, "layout.go"))
	if !strings.Contains(layout, "filepath.Dir(l.Trust)") || strings.Contains(layout, "release-trust.json\",") && strings.Contains(layout, "WriteFile") {
		t.Fatal("trust file is no longer only a parent directory")
	}
	probe := string(mustRead(t, "probe_addresses.go"))
	if strings.Contains(probe, "WriteFile") || strings.Contains(probe, "O_WRONLY") || strings.Contains(probe, "O_RDWR") {
		t.Fatal("probe path writer")
	}
}

func mustRead(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
