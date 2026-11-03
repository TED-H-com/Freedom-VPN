package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/google/uuid"
)

type f36StopSource struct {
	obs  meter.Observation
	hold bool
}

func (s *f36StopSource) Read(context.Context) (meter.Observation, error) {
	if s.hold {
		fmt.Println("checkpoint pending")
		select {} // TEST ONLY: parent terminates this child before checkpoint commit.
	}
	return s.obs, nil
}

type f36DurableState struct {
	Config      Config
	Node, Inc   uuid.UUID
	Observation meter.Observation
	Meta        map[string]string
	Desired     agentstore.Desired
	Lease       agentstore.Lease
	Usage       usagestore.Snapshot
	Heartbeat   []byte
}

func f36ReadState(t *testing.T, cfg Config, node, inc uuid.UUID, obs meter.Observation) f36DurableState {
	t.Helper()
	disk, err := agentstore.Open(cfg.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	defer disk.Close()
	s := f36DurableState{Config: cfg, Node: node, Inc: inc, Observation: obs, Meta: map[string]string{}}
	for _, key := range []string{"retained_applied", agentstore.AppliedSnapshotKey, pendingRevocationsKey, conntrackAttemptsKey, conntrackBlockedKey} {
		value, _, err := disk.Meta(t.Context(), key)
		if err != nil {
			t.Fatal(err)
		}
		s.Meta[key] = value
	}
	s.Desired, _, err = disk.Desired(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	s.Lease, _, err = disk.Lease(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	s.Usage, err = usagestore.ReadSnapshot(t.Context(), cfg.UsageDBPath())
	if err != nil {
		t.Fatal(err)
	}
	s.Heartbeat, err = os.ReadFile(cfg.HeartbeatPath)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAT13_F36_InterruptedCloseDurability(t *testing.T) {
	if path := os.Getenv("FVPN_TEST_ONLY_DURABLE_CHILD"); path != "" {
		r := newRig(t)
		alignControlClock(r)
		r.srv.SetUsageUnavailable(1000)
		serveBoundGeneration(t, r)
		r.manualUsage(t)
		r.eng.meter.Close()
		binding := r.eng.appliedModel.Bindings[0]
		name := meter.CounterName(binding.DeviceID, meter.DirUpload)
		source := &f36StopSource{obs: meter.Observation{BootID: r.eng.bootID.String(), Bindings: []enforce.Binding{binding}, Counters: map[string]enforce.CounterValue{name: {}}}}
		var err error
		r.eng.meter, err = meter.New(meter.Config{NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: source})
		if err != nil {
			t.Fatal(err)
		}
		source.obs.Counters[name] = enforce.CounterValue{Packets: 1, Bytes: 4096}
		if err := r.eng.meter.Sample(t.Context()); err != nil {
			t.Fatal(err)
		}
		// Reserve exactly as #388 does before its first Apply instruction.
		m := *r.eng.appliedModel
		m.Bindings = nil
		m.Revoke = []netip.Addr{binding.AllowedV4[0].Addr()}
		if err := r.eng.reserveConntrack(t.Context(), m); err != nil {
			t.Fatal(err)
		}
		source.obs.Counters[name] = enforce.CounterValue{Packets: 2, Bytes: 8192}
		s := f36ReadState(t, r.eng.cfg, r.eng.nodeID, r.eng.incID, source.obs)
		raw, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		source.hold = true
		_ = r.eng.Close()
		os.Exit(92) // Parent must actually interrupt Close, not observe normal exit.
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAT13_F36_InterruptedCloseDurability$")
	// Every child TempDir is inside this parent's disk-backed cleanup scope.
	cmd.Env = append(os.Environ(), "FVPN_TEST_ONLY_DURABLE_CHILD="+path, "TMPDIR="+dir)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	scan := bufio.NewScanner(out)
	if !scan.Scan() || scan.Text() != "checkpoint pending" {
		t.Fatalf("child did not enter Close: %q %v", scan.Text(), cmd.Wait())
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	err = cmd.Wait()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.Sys().(syscall.WaitStatus).Signal() != syscall.SIGTERM {
		t.Fatalf("expected signal 15: %v %s", err, stderr.String())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var before f36DurableState
	if err := json.Unmarshal(raw, &before); err != nil {
		t.Fatal(err)
	}
	after := f36ReadState(t, before.Config, before.Node, before.Inc, before.Observation)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("interrupted Close tore durable metadata, lease, usage or heartbeat")
	}
	if after.Meta[conntrackAttemptsKey] != "1" || after.Meta[pendingRevocationsKey] == "[]" {
		t.Fatal("reservation lost")
	}
	if _, err := os.Stat(before.Config.MeterCheckpoint); !os.IsNotExist(err) {
		t.Fatal("interrupted restart fabricated a completed withdrawal checkpoint", err)
	}
	// A retained heartbeat is not a stop receipt. The unsampled delta remains
	// in same-boot kernel counters; a successor samples it exactly once.
	store, err := usagestore.Open(before.Config.UsageDBPath(), before.Node, before.Inc, usagestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	m, err := meter.New(meter.Config{NodeID: before.Node, IncarnationID: before.Inc, Store: store, Source: &f36StopSource{obs: before.Observation}})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	snap, err := usagestore.ReadSnapshot(t.Context(), before.Config.UsageDBPath())
	if err != nil {
		t.Fatal(err)
	}
	var total uint64
	for _, ev := range snap.UnackedEvents {
		for _, rec := range ev.Records {
			n, err := strconv.ParseUint(rec.UploadBytes, 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			total += n
		}
	}
	if total != 8192 {
		t.Fatalf("same-boot recovered bytes=%d want=8192", total)
	}
	for i, ev := range before.Usage.UnackedEvents {
		if !reflect.DeepEqual(ev, snap.UnackedEvents[i]) {
			t.Fatal("recovery rewrote a committed event")
		}
	}
}
