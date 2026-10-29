package agent

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func TestClassifyExternalReset(t *testing.T) {
	dev := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	liveName := meter.CounterName(dev, meter.DirUpload)
	idleName := meter.CounterName(dev, meter.DirDownload)
	dns := "nu_" + "11111111111141118111111111111111"
	live := map[string]enforce.CounterValue{
		liveName:  {Packets: 4, Bytes: 1000},
		idleName:  {},
		dns:       {Packets: 9, Bytes: 99},
		"ct_diag": {Packets: 1, Bytes: 5},
	}
	idle := map[string]enforce.CounterValue{liveName: {}, idleName: {}}

	broken := enforce.State{Counters: cloneCounters(live)}
	if restore, force, report := classifyExternal(live, broken, true); !restore || !force || !report {
		t.Fatalf("table missing: restore %v force %v report %v", restore, force, report)
	}
	noChain := enforce.State{Table: true, Counters: cloneCounters(live)}
	if _, force, report := classifyExternal(live, noChain, true); !force || !report {
		t.Fatalf("chain missing: force %v report %v", force, report)
	}
	gen0 := enforce.State{Table: true, Chains: map[string]bool{"forward": true}, Counters: cloneCounters(live)}
	if _, force, report := classifyExternal(live, gen0, true); !force || !report {
		t.Fatalf("generation cleared: force %v report %v", force, report)
	}
	if restore, force, report := classifyExternal(idle, gen0, true); !restore || force || report {
		t.Fatalf("idle external: restore %v force %v report %v", restore, force, report)
	}
	if restore, force, report := classifyExternal(live, gen0, false); restore || force || report {
		t.Fatalf("baseline hand-off: restore %v force %v report %v", restore, force, report)
	}
	if _, force, report := classifyExternal(map[string]enforce.CounterValue{}, gen0, true); force || report {
		t.Fatalf("empty previous: force %v report %v", force, report)
	}

	dropped := cloneCounters(live)
	dropped[liveName] = enforce.CounterValue{Packets: 1, Bytes: 10}
	droppedState := enforce.State{Table: true, Chains: map[string]bool{"forward": true}, Counters: dropped}
	if _, force, report := classifyExternal(live, droppedState, true); force || !report {
		t.Fatalf("value drop on a broken ruleset: force %v report %v", force, report)
	}
	intact := enforce.State{
		Table: true, Chains: map[string]bool{"forward": true}, Generation: 1,
		Counters: cloneCounters(live),
	}
	if restore, force, report := classifyExternal(live, intact, true); restore || force || report {
		t.Fatalf("intact ruleset: restore %v force %v report %v", restore, force, report)
	}
	intactDrop := intact
	intactDrop.Counters = dropped
	if _, force, report := classifyExternal(live, intactDrop, true); force || !report {
		t.Fatalf("value drop on an intact ruleset: force %v report %v", force, report)
	}
}

func TestApplyNeedsForce(t *testing.T) {
	dev := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	name := meter.CounterName(dev, meter.DirUpload)
	live := map[string]enforce.CounterValue{name: {Packets: 4, Bytes: 1000}}
	post := cloneCounters(live)
	ext := enforce.DetectMeterResets(live, enforce.State{Counters: post})
	if len(ext) == 0 || !externalDetail(ext) {
		t.Fatalf("expected an external reset, got %+v", ext)
	}
	if force, report := applyNeedsForce(live, ext, post, true); !force || !report {
		t.Fatalf("serving external: force %v report %v", force, report)
	}
	if force, report := applyNeedsForce(live, ext, post, false); force || report {
		t.Fatalf("first apply after baseline: force %v report %v", force, report)
	}
	idle := map[string]enforce.CounterValue{name: {}}
	idleResets := enforce.DetectMeterResets(idle, enforce.State{Counters: idle})
	if force, report := applyNeedsForce(idle, idleResets, idle, true); force || report {
		t.Fatalf("idle: force %v report %v", force, report)
	}
	dropped := map[string]enforce.CounterValue{name: {Packets: 1, Bytes: 10}}
	value := enforce.DetectResets(live, dropped)
	if force, report := applyNeedsForce(live, value, dropped, true); force || !report {
		t.Fatalf("value drop: force %v report %v", force, report)
	}
}

func TestNudgeOneUsageCounter(t *testing.T) {
	dev := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	up := meter.CounterName(dev, meter.DirUpload)
	dn := meter.CounterName(dev, meter.DirDownload)
	dns := "nu_" + "11111111111141118111111111111111"
	in := map[string]enforce.CounterValue{
		dn:  {},
		dns: {Packets: 8, Bytes: 80},
		up:  {Packets: 4, Bytes: 1000},
	}
	out := nudgeOne(in)
	if out[up].Packets != 3 || out[up].Bytes != 1000 {
		t.Fatalf("upload %+v", out[up])
	}
	if out[dn] != in[dn] || out[dns] != in[dns] {
		t.Fatalf("nudged an idle or DNS counter: %+v", out)
	}
	if in[up].Packets != 4 {
		t.Fatal("nudge mutated the input")
	}
	onlyBytes := map[string]enforce.CounterValue{up: {Bytes: 5}}
	got := nudgeOne(onlyBytes)
	if got[up].Bytes != 4 || got[up].Packets != 0 {
		t.Fatalf("byte nudge %+v", got[up])
	}
}

func TestExternalResetOpensOneEpoch(t *testing.T) {
	ctx := context.Background()
	node := uuid.MustParse("019a0000-0000-7000-8000-0000000000a1")
	inc := uuid.MustParse("019a0000-0000-7000-8000-0000000000a2")
	dev := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	mem := uuid.MustParse("019a0000-0000-7000-8000-0000000000e1")
	name := meter.CounterName(dev, meter.DirUpload)
	counters := map[string]enforce.CounterValue{name: {Packets: 5, Bytes: 1000}}
	cur := enforce.State{
		Table: true, Chains: map[string]bool{"forward": true}, Generation: 1,
		Counters: counters,
	}
	var serving atomic.Bool
	ks := &kernelSource{
		boot:   "boot-1",
		expect: func() bool { return serving.Load() },
		read: func(context.Context) (enforce.State, error) {
			return cur, nil
		},
		bindings: func() []enforce.Binding {
			return []enforce.Binding{{DeviceID: dev, MemberID: mem, BindingRevision: 1}}
		},
	}
	at := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	st, err := usagestore.Open(filepath.Join(t.TempDir(), "usage.db"), node, inc, usagestore.Options{
		Now: func() time.Time { return at },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	m, err := meter.New(meter.Config{
		NodeID: node, IncarnationID: inc, Store: st, Source: ks,
		Interval: 10 * time.Second, Now: func() time.Time { return at },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	head, err := st.Head(ctx)
	if err != nil || head.OpenEpoch == uuid.Nil {
		t.Fatalf("open %+v %v", head, err)
	}
	first := head.OpenEpoch

	serving.Store(true)
	cur.Generation = 0
	at = at.Add(10 * time.Second)
	if err := m.Sample(ctx); err != nil {
		t.Fatal(err)
	}
	head, err = st.Head(ctx)
	if err != nil || head.OpenEpoch == first || head.OpenEpoch == uuid.Nil {
		t.Fatalf("epoch did not change: %+v", head)
	}
	opened := head.OpenEpoch
	var gotPackets, gotBytes uint64
	for _, c := range head.Counters {
		if c.Name == name {
			gotPackets, gotBytes = c.Packets, c.Bytes
		}
	}
	if gotPackets != 4 || gotBytes != 1000 {
		t.Fatalf("checkpoint packets %d bytes %d", gotPackets, gotBytes)
	}
	msg, ok := ks.takeReport()
	if !ok || msg != msgExternalReset || len(msg) > nodeErrorLimit {
		t.Fatalf("report %q %v", msg, ok)
	}
	if strings.Contains(msg, name) {
		t.Fatalf("report names a counter: %s", msg)
	}
	gap, err := st.Event(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	var ev api.UsageEvent
	if err := json.Unmarshal(gap.Payload, &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Quality != api.UsageQualityGap || ev.Gap == nil || ev.Gap.Reason != api.UsageGapReasonCounterDiscontinuity {
		t.Fatalf("gap %+v", ev)
	}
	if billed(ev) != 0 {
		t.Fatalf("discontinuity billed %d", billed(ev))
	}

	cur.Generation = 1
	at = at.Add(10 * time.Second)
	if err := m.Sample(ctx); err != nil {
		t.Fatal(err)
	}
	head, err = st.Head(ctx)
	if err != nil || head.OpenEpoch != opened {
		t.Fatalf("quiet sample opened another epoch: %+v", head)
	}
	if _, err := st.Event(ctx, 3); err == nil {
		t.Fatal("quiet sample wrote another event")
	}
}

func TestDecideIsolationBaselineCode(t *testing.T) {
	in := servingInput()
	in.BaselineOK = false
	d := Decide(in)
	if d.Serve {
		t.Fatal("served without a baseline")
	}
	if !hasCode(d.Errors, api.ErrorCodeISOLATIONBASELINEMISSING) || hasCode(d.Errors, api.ErrorCodeHEALTHCHECKFAILED) {
		t.Fatalf("errors %+v", d.Errors)
	}
	if !hasMessage(d.Errors, "ISOLATION_BASELINE_MISSING") {
		t.Fatalf("message %+v", d.Errors)
	}
	var saw bool
	for _, c := range d.Checks {
		if c.Name != api.NodeCheckNameInternetOnly {
			continue
		}
		saw = true
		code, err := c.ErrorCode.Get()
		if c.Status != api.HealthCheckStatusFailed || err != nil || code != api.ErrorCodeISOLATIONBASELINEMISSING {
			t.Fatalf("internet check %+v %v", c, err)
		}
	}
	if !saw {
		t.Fatal("missing internet check")
	}
}

func TestNoteApplyBaselineCode(t *testing.T) {
	e := &Engine{log: slog.New(slog.DiscardHandler), clock: systemClock{}}
	e.noteApply(&enforce.CheckError{Reason: enforce.ReasonIsolationBaselineMissing})
	if !hasCode(e.decision.Errors, api.ErrorCodeISOLATIONBASELINEMISSING) || hasCode(e.decision.Errors, api.ErrorCodeHEALTHCHECKFAILED) {
		t.Fatalf("%+v", e.decision.Errors)
	}
	e.decision.Errors = nil
	e.noteApply(errors.New("boom"))
	if !hasCode(e.decision.Errors, api.ErrorCodeHEALTHCHECKFAILED) || !hasMessage(e.decision.Errors, "apply failed") {
		t.Fatalf("%+v", e.decision.Errors)
	}
}

func billed(ev api.UsageEvent) uint64 {
	var n uint64
	for _, r := range ev.Records {
		n += uint64(r.UploadBytes) + uint64(r.DownloadBytes)
	}
	return n
}
