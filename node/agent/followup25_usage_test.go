package agent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func TestD118_ENR05_VerifiedReenrollmentHandsOffFinalSample(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	r.srv.SetUsageUnavailable(1000)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	cfg, node, origin := r.eng.cfg, r.eng.nodeID, r.eng.incID
	r.eng.meter.Close()
	binding := r.eng.appliedModel.Bindings[0]
	name := meter.CounterName(binding.DeviceID, meter.DirUpload)
	src := &barrierSource{obs: meter.Observation{BootID: r.eng.bootID.String(), Bindings: []enforce.Binding{binding}, Counters: map[string]enforce.CounterValue{name: {}}}}
	var err error
	r.eng.meter, err = meter.New(meter.Config{NodeID: node, IncarnationID: origin, Store: r.eng.usage, Source: src})
	if err != nil {
		t.Fatal(err)
	}
	src.obs.Counters[name] = enforce.CounterValue{Packets: 1, Bytes: 4096}
	// Close's real barrier commits the last admitted delta before retirement.
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	old, err := usagestore.Open(cfg.UsageDBPath(), node, origin, usagestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	held, err := old.Unacked(t.Context(), 0, 100, 1<<20)
	if err != nil || len(held) == 0 {
		t.Fatal("missing final sample", err)
	}
	var total uint64
	for _, row := range held {
		var ev api.UsageEvent
		if err := json.Unmarshal(row.Payload, &ev); err != nil {
			t.Fatal(err)
		}
		for _, rec := range ev.Records {
			total += uint64(rec.UploadBytes)
		}
	}
	if total != 4096 {
		t.Fatalf("closing barrier bytes=%d", total)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}
	// TEST ONLY reinstall: retain usage.db, remove this rig's identity and agent
	// state, then obtain a new certificate through the real enrollment client.
	lock, err := nodelock.Acquire(t.Context(), cfg.LockPath, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	for _, path := range []string{cfg.DBPath(), cfg.DBPath() + "-wal", cfg.DBPath() + "-shm"} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	if err := os.RemoveAll(cfg.IdentityDir); err != nil {
		t.Fatal(err)
	}
	r.srv.AddReenrollmentToken("re-enroll-test-only", node)
	next, err := Start(Options{Config: cfg, Clock: r.clock, Plane: r.plane, MeterSource: src, Prober: passProbe{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := next.Enroll(t.Context(), "re-enroll-test-only"); err != nil {
		t.Fatal(err)
	}
	current := next.incID
	if next.nodeID != node || current == origin {
		t.Fatal("not a verified new incarnation")
	}
	// Crash/restart after identity installation, before ensureUsage: adoption
	// must authorize the retained old database on this next boot too.
	if err := next.Close(); err != nil {
		t.Fatal(err)
	}
	next, err = Start(Options{Config: cfg, Clock: r.clock, Plane: r.plane, MeterSource: src, Prober: passProbe{}})
	if err != nil {
		t.Fatal(err)
	}
	r.eng = next
	t.Cleanup(func() { _ = next.Close() })
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	stopUsageLoops(t, r)
	if next.usage == nil || next.usagePersist {
		t.Fatal("production adoption hook failed")
	}
	streams, err := next.usage.RetiredStreams(t.Context())
	if err != nil || len(streams) != 1 || streams[0].IncarnationID != origin || streams[0].Pending != len(held) {
		t.Fatal("retired stream", streams, err)
	}
	retired, err := next.usage.RetiredUnacked(t.Context(), origin, 100, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	for i, row := range retired {
		if !bytes.Equal(row.Payload, held[i].Payload) || row.Hash != held[i].Hash {
			t.Fatal("retirement rewrote final sample")
		}
	}
	head, err := next.usage.Head(t.Context())
	if err != nil || head.NextSeq != 2 || head.AckedThrough != 0 {
		t.Fatal("current stream cursor", head, err)
	}
	// A current meter sample starts at sequence one, independently of origin.
	src.obs.Counters[name] = enforce.CounterValue{Packets: 2, Bytes: 4196}
	if err := next.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	r.srv.SetUsageUnavailable(0)
	// The stopped loop may already have tried a 503. A new uploader on restart
	// consumes its durable pending body; no synthetic ACK or certificate swap.
	if err := next.Close(); err != nil {
		t.Fatal(err)
	}
	next, err = Start(Options{Config: cfg, Clock: r.clock, Plane: r.plane, MeterSource: src, Prober: passProbe{}})
	if err != nil {
		t.Fatal(err)
	}
	r.eng = next
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	stopUsageLoops(t, r)
	for range 3 {
		if err := next.uploader.Once(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	bodies := r.srv.UsageBodies()
	if len(bodies) < 2 {
		t.Fatalf("missing streams: %d", len(bodies))
	}
	var first, second api.UsageBatchRequest
	if err := json.Unmarshal(bodies[0], &first); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(bodies[1], &second); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(first)
	var wire struct {
		Origin string            `json:"origin_incarnation_id"`
		Events []json.RawMessage `json:"events"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Origin != origin.String() || first.NodeIncarnationId != current || second.NodeIncarnationId != current {
		t.Fatal("wrong presenter/order")
	}
	for i, event := range wire.Events {
		if !bytes.Equal(compactJSON(t, event), compactJSON(t, held[i].Payload)) {
			t.Fatal("hand-off rebuilt event")
		}
	}
	if second.Events[0].EventSeq != 1 {
		t.Fatal("new stream does not start at one")
	}
	streams, err = next.usage.RetiredStreams(t.Context())
	if err != nil || len(streams) != 1 || streams[0].Pending != 0 || streams[0].AckedThrough != held[len(held)-1].Seq {
		t.Fatal("origin ACK", streams, err)
	}
	head, err = next.usage.Head(t.Context())
	if err != nil || head.AckedThrough != 2 {
		t.Fatal("current ACK", head, err)
	}
	// Same verified incarnation has now reopened twice; neither retired itself
	// nor rotated the prior stream again. The ledger file stayed in place.
	if _, err := os.Stat(filepath.Join(cfg.StateDir, "usage.db")); err != nil {
		t.Fatal(err)
	}
}

func compactJSON(t *testing.T, raw []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := json.Compact(&out, raw); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
