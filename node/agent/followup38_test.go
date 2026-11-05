package agent

// TEST ONLY regression fixtures. Every counter belongs to this in-memory
// plane; no host networking is read or modified.
import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

type r21ReadFunc func(context.Context) (meter.Observation, error)

func (f r21ReadFunc) Read(ctx context.Context) (meter.Observation, error) { return f(ctx) }

func TestR21_H1_MixedBindingsCrashWindows(t *testing.T) {
	for _, mode := range []string{"one_crash", "two_crashes_pending", "boot_changed"} {
		t.Run(mode, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			r.srv.SetUsageUnavailable(1000)
			serveBoundGeneration(t, r)
			r.manualUsage(t)
			b := oneBinding(r.clock.Wall())
			stable := b
			stable.DeviceId = uuid.Must(uuid.NewV7())
			stable.PeerId = "peer-stable"
			stable.BindingRevision = 7
			stable.AllowedIpv4 = []string{"100.64.1.2/32"}
			f := generationFixture(3, b)
			f.Bindings = append(f.Bindings, stable)
			r.srv.SetFixture(f)
			stepOK(t, r)
			changedCounter := meter.CounterName(b.DeviceId, meter.DirUpload)
			stableCounter := meter.CounterName(stable.DeviceId, meter.DirUpload)
			src := &r20DynamicSource{boot: r.eng.bootID.String(), bindings: r.eng.meterBindings, counters: map[string]enforce.CounterValue{
				changedCounter: {Packets: 1, Bytes: 4096}, stableCounter: {Packets: 1, Bytes: 1000},
			}}
			r.eng.meter.Close()
			var err error
			r.eng.meter, err = meter.New(meter.Config{NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: src})
			if err != nil {
				t.Fatal(err)
			}
			if err := r.eng.meter.Sample(t.Context()); err != nil {
				t.Fatal(err)
			}

			rebound := b
			rebound.MemberId = uuid.Must(uuid.NewV7())
			rebound.BindingRevision = 2
			f.Generation, f.Bindings = 4, []api.DeviceBinding{rebound, stable}
			r.srv.SetFixture(f)
			if err := r.eng.syncDesired(t.Context()); err != nil {
				t.Fatal(err)
			}
			// Apply is allowed to install the accepted desired generation before the
			// retained model write. Kill at that exact boundary, with no new sample.
			model := *r.eng.appliedModel
			model.Generation = 4
			model.Bindings = append([]enforce.Binding(nil), model.Bindings...)
			for i := range model.Bindings {
				if model.Bindings[i].DeviceID == b.DeviceId {
					model.Bindings[i].MemberID = rebound.MemberId
					model.Bindings[i].BindingRevision = 2
				}
			}
			grant := model.Leases[0]
			grant.MemberID = rebound.MemberId
			model.Leases = append(append([]enforce.MemberLease(nil), model.Leases...), grant)
			qos := model.QoS.Members[0]
			qos.MemberID = rebound.MemberId
			model.QoS.Members = append(append([]enforce.MemberQoS(nil), model.QoS.Members...), qos)
			for i := range model.QoS.Members {
				model.QoS.Members[i].Up.RateBPS /= 2
				model.QoS.Members[i].Down.RateBPS /= 2
			}
			model.QoS.Assignments = nil
			// TEST ONLY legacy crash image: this direct plane call bypasses the
			// agent's new pre-Apply intent and settlement boundary. Retain the
			// r21 upgrade guard without claiming this is a new production path.
			if err := r.eng.store.SetMeta(t.Context(), usageIntentKey, ""); err != nil {
				t.Fatal(err)
			}
			if _, err := r.plane.Apply(t.Context(), model, r.eng.counters); err != nil {
				t.Fatal(err)
			}
			r20Crash(r.eng)
			src.setCounter(changedCounter, enforce.CounterValue{Packets: 2, Bytes: 8192})
			src.setCounter(stableCounter, enforce.CounterValue{Packets: 2, Bytes: 3000})
			cfg := r.eng.cfg
			if mode == "two_crashes_pending" {
				calls := 0
				fail := r21ReadFunc(func(ctx context.Context) (meter.Observation, error) {
					calls++
					if calls == 2 {
						return meter.Observation{}, errors.New("TEST ONLY settlement source failure")
					}
					return src.Read(ctx)
				})
				p2, err := Start(Options{Config: cfg, Clock: r.clock, Plane: r.plane, MeterSource: fail, Prober: passProbe{}})
				if err != nil {
					t.Fatal(err)
				}
				src.setBindings(p2.meterBindings)
				if err := p2.Boot(t.Context()); err != nil {
					t.Fatal(err)
				}
				if !p2.retainedPending {
					t.Fatal("precondition: settlement did not stay pending")
				}
				r20Crash(p2)
			}
			if mode == "boot_changed" {
				src.boot = uuid.Must(uuid.NewV7()).String()
				if err := os.WriteFile(cfg.BootIDFile, []byte(src.boot+"\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			next := r20Successor(t, r, cfg, src, r.plane)
			if err := next.Boot(t.Context()); err != nil {
				t.Fatal(err)
			}
			u := r21ReadLedger(t, cfg, next.nodeID, next.incID)
			wantStable := uint64(3000)
			if mode == "boot_changed" {
				wantStable = 1000
			}
			if u.byRevision[1] != 4096 || u.byRevision[2] != 0 || u.byRevision[7] != wantStable || len(u.gaps) != 1 {
				t.Fatalf("H-1 %s: misattribution/lost unchanged bytes: revisions=%v gaps=%v", mode, u.byRevision, u.gaps)
			}
			if mode == "boot_changed" && u.gaps[0].Gap.Reason != api.UsageGapReasonPowerLoss {
				t.Fatal("missing power-loss provenance")
			}
			r20Crash(next)
			again := r20Successor(t, r, cfg, src, r.plane)
			if err := again.Boot(t.Context()); err != nil {
				t.Fatal(err)
			}
			u2 := r21ReadLedger(t, cfg, again.nodeID, again.incID)
			if u2.byRevision[1] != 4096 || u2.byRevision[7] != wantStable || len(u2.gaps) != 1 {
				t.Fatal("settlement was not exactly once")
			}
		})
	}
}

func TestR21_M2_CanceledBootFailureHeartbeat(t *testing.T) {
	for _, stage := range []string{"runtime", "baseline"} {
		t.Run(stage, func(t *testing.T) {
			r, src, _ := r20ServedPendingDelta(t)
			cfg := r20ChangedSets(r)
			next := r20Successor(t, r, cfg, src, r.plane)
			r.eng = next
			if stage == "runtime" {
				if err := next.store.SetMeta(t.Context(), "retained_applied", "{"); err != nil {
					t.Fatal(err)
				}
			} else {
				next.plane = &r21FailBaselinePlane{memPlane: r.plane}
			}
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			err := next.Run(ctx)
			if !errors.Is(err, ErrBoot) {
				t.Fatalf("%s: want Boot failure, got %v", stage, err)
			}
			beat := mustBeat(t, r)
			if beat.Serving || beat.BaselineOK != (stage == "runtime") || beat.Reason == "serving" {
				t.Fatalf("untruthful failure beat: %+v", beat)
			}
		})
	}
}

func TestR21_M2_OpenUsageLockFollowsCaller(t *testing.T) {
	r, src, _ := r20ServedPendingDelta(t)
	next := r20Successor(t, r, r.eng.cfg, src, r.plane)
	// A new enrolled incarnation needs the node lock to retire the old ledger.
	next.incID = uuid.Must(uuid.NewV7())
	lock, err := nodelock.TryAcquire(next.cfg.LockPath)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	caller, cancel := context.WithCancel(t.Context())
	cancel()
	ctx, done := bootLocal(caller, time.Second)
	defer done()
	start := time.Now()
	st, err := next.openUsage(ctx)
	if st != nil {
		_ = st.Close()
		t.Fatal("opened new incarnation without lock")
	}
	if !errors.Is(err, context.Canceled) || time.Since(start) > 80*time.Millisecond {
		t.Fatalf("lock ignored caller: %v (%s)", err, time.Since(start))
	}
}

// A scaled deadline runs the identical Boot wrapper. Every blocking stage
// keeps the original deadline, including nested adoption cleanup. The actual
// configured allowance is checked separately and Close retains its real 3s.
type r21BudgetPlane struct {
	*memPlane
	stage   string
	seen    []time.Time
	release <-chan struct{}
}

func (p *r21BudgetPlane) block(ctx context.Context) {
	d, _ := ctx.Deadline()
	p.seen = append(p.seen, d)
	if p.release != nil {
		<-p.release
	} else {
		<-ctx.Done()
	}
}
func (p *r21BudgetPlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
	if p.stage == "baseline" || p.stage == "adoption_cleanup" || p.stage == "adoption_verify" || p.stage == "adoption_probe" || p.stage == "adoption_lock" {
		p.block(ctx)
		return ctx.Err()
	}
	return p.memPlane.Baseline(ctx, dp)
}
func (p *r21BudgetPlane) MemberPath(ctx context.Context, m enforce.Model, expected *MemberPath) (MemberPath, error) {
	if p.stage == "adoption_cleanup" {
		return MemberPath{}, errMemberPath
	}
	return p.memPlane.MemberPath(ctx, m, expected)
}
func (p *r21BudgetPlane) RestartGeneration(ctx context.Context) (uint64, error) {
	if p.stage == "generation" {
		p.block(ctx)
		return 0, ctx.Err()
	}
	return p.memPlane.RestartGeneration(ctx)
}

func (p *r21BudgetPlane) VerifyServing(ctx context.Context, m enforce.Model, a map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	if p.stage == "adoption_verify" {
		p.block(ctx)
		return enforce.ServingEvidence{}, ctx.Err()
	}
	return p.memPlane.VerifyServing(ctx, m, a)
}

type r21TermProber struct {
	cancel  context.CancelFunc
	called  bool
	stopped bool
}

func (p *r21TermProber) Probe(ctx context.Context, _ ProbeRequest) ProbeResult {
	p.called = true
	p.cancel()
	p.stopped = errors.Is(ctx.Err(), context.Canceled)
	return ProbeResult{}
}

func TestR21_M2_OneBootBudgetAllStages(t *testing.T) {
	if bootLocalBudget+closeLocalBudget != 13*time.Second {
		t.Fatal("stop allowances changed")
	}
	for _, stage := range []string{"runtime", "recover", "generation", "baseline", "adoption_cleanup", "adoption_verify", "adoption_probe", "adoption_lock", "meter_open", "settlement", "uncooperative"} {
		t.Run(stage, func(t *testing.T) {
			r, src, _ := r20ServedPendingDelta(t)
			cfg := r.eng.cfg
			if stage != "adoption_cleanup" && stage != "adoption_verify" && stage != "adoption_probe" && stage != "adoption_lock" {
				cfg = r20ChangedSets(r)
			}
			plane := &r21BudgetPlane{memPlane: r.plane, stage: stage}
			next := r20Successor(t, r, cfg, src, plane)
			var db *sql.DB
			var tx *sql.Tx
			if stage == "runtime" {
				// loadRuntime migrates this real pre-D-100 path. Another SQLite writer
				// blocks the migration; the Boot wrapper still owns a single deadline.
				if err := next.store.SetMeta(t.Context(), "member_path", `{"generation":"0","interface":"fv-member"}`); err != nil {
					t.Fatal(err)
				}
				var err error
				db, err = sql.Open("sqlite", "file:"+cfg.DBPath())
				if err != nil {
					t.Fatal(err)
				}
				tx, err = db.Begin()
				if err != nil {
					t.Fatal(err)
				}
				if _, err := tx.Exec("UPDATE meta SET value=value WHERE key='member_path'"); err != nil {
					t.Fatal(err)
				}
			}
			if stage == "recover" {
				// A real cancellable child, no host state or networking.
				next.cfg.UpdaterRecover = "/bin/sleep"
				// sleep rejects recover, so a TEST ONLY script execs a long sleep instead.
				path := filepath.Join(t.TempDir(), "recover")
				if err := os.WriteFile(path, []byte("#!/bin/sh\nexec sleep 60\n"), 0700); err != nil {
					t.Fatal(err)
				}
				next.cfg.UpdaterRecover, next.cfg.UpdaterConfig = path, "test.json"
			}
			var release chan struct{}
			if stage == "uncooperative" {
				release = make(chan struct{})
				plane.stage = "baseline"
				plane.release = release
			}
			if stage == "meter_open" || stage == "settlement" {
				calls := 0
				next.meterSource = r21ReadFunc(func(ctx context.Context) (meter.Observation, error) {
					calls++
					if stage == "meter_open" || calls == 2 {
						plane.block(ctx)
						return meter.Observation{}, ctx.Err()
					}
					return src.Read(ctx)
				})
			}
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			// Adoption cleanup needs a live caller until it enters member-path refusal.
			var probe *r21TermProber
			var lock *nodelock.Lock
			if stage == "adoption_cleanup" || stage == "adoption_verify" || stage == "adoption_probe" || stage == "adoption_lock" {
				ctx, cancel = context.WithCancel(t.Context())
				defer cancel()
				if stage == "adoption_probe" {
					probe = &r21TermProber{cancel: cancel}
					next.prober = probe
				}
				if stage == "adoption_lock" {
					var err error
					lock, err = nodelock.TryAcquire(cfg.LockPath)
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			started := time.Now()
			_ = next.bootWithBudget(ctx, 120*time.Millisecond)
			elapsed := time.Since(started)
			if elapsed > 400*time.Millisecond {
				t.Errorf("%s stacked local budgets: %s", stage, elapsed)
			}
			if release != nil {
				close(release)
			}
			if tx != nil {
				_ = tx.Rollback()
				_ = db.Close()
			}
			<-next.bootDone
			if lock != nil {
				lock.Release()
			}
			if probe != nil && (!probe.called || !probe.stopped) {
				t.Fatal("probe did not follow TERM")
			}
			for _, d := range plane.seen {
				if d.After(started.Add(125 * time.Millisecond)) {
					t.Errorf("cleanup extended Boot deadline: %s", d.Sub(started))
				}
			}
			if err := next.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestR21_M2_CloseWaitsForTimedOutBootWithinStopBudget(t *testing.T) {
	r, src, _ := r20ServedPendingDelta(t)
	release := make(chan struct{})
	plane := &r21BudgetPlane{memPlane: r.plane, stage: "baseline", release: release}
	next := r20Successor(t, r, r20ChangedSets(r), src, plane)
	started := time.Now()
	err := next.Boot(t.Context())
	if !errors.Is(err, ErrBoot) {
		close(release)
		t.Fatalf("unbounded Boot: %v", err)
	}
	err = next.Close()
	elapsed := time.Since(started)
	close(release)
	<-next.closeDone
	if !errors.Is(err, context.DeadlineExceeded) || elapsed > 13300*time.Millisecond {
		t.Fatalf("Boot+Close exceeded combined allowance: %s %v", elapsed, err)
	}
}

func TestR21_M1_FinalClockGapClosesEpoch(t *testing.T) {
	r, src, _ := r20ServedPendingDelta(t)
	st, err := usagestore.Open(r.eng.cfg.UsageDBPath(), r.eng.nodeID, r.eng.incID, usagestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	m, err := meter.New(meter.Config{NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: st, Source: src, Now: func() time.Time { return time.Now().Add(-time.Hour) }})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.FinalSample(t.Context()); err != nil {
		t.Fatal(err)
	}
	head, err := st.Head(t.Context())
	if err != nil || head.OpenEpoch != uuid.Nil {
		t.Fatalf("final clock gap left epoch open: %v %v", head, err)
	}
}

func TestR21_M2_UploadCancellationDuringAdoption(t *testing.T) {
	r, src, _ := r20ServedPendingDelta(t)
	r.srv.SetUsageUnavailable(0)
	before := r21ServerUsageEvents(r)
	caller, cancel := context.WithCancel(t.Context())
	defer cancel()
	next, err := Start(Options{Config: r.eng.cfg, Clock: r.clock, Plane: r.plane, MeterSource: src, Prober: &r20CancelProber{cancel: cancel}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })
	src.setBindings(next.meterBindings)
	if err := next.Boot(caller); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if after := r21ServerUsageEvents(r); after != before {
		t.Fatalf("M-2: canceled adoption started %d post-TERM uploads", after-before)
	}
}

func TestR21_M2_TermAfterBaselineKeepsNormalStopOnLedgerTimeout(t *testing.T) {
	r, src, _ := r20ServedPendingDelta(t)
	next := r20Successor(t, r, r20ChangedSets(r), src, r.plane)
	release := make(chan struct{})
	entered := make(chan f53DeadlineRead, 1)
	next.meterSource = r21ReadFunc(func(ctx context.Context) (meter.Observation, error) {
		entered <- f53DeadlineRead{ctx: ctx, errOnEntry: ctx.Err()}
		<-release
		return src.Read(ctx)
	})
	caller, cancel := context.WithCancel(t.Context())
	cancel()
	err := next.bootWithBudget(caller, bootLocalBudget)
	close(release)
	<-next.bootDone
	f53RequireExpiredLedgerRead(t, entered)
	if !errors.Is(err, context.Canceled) || errors.Is(err, ErrBoot) || !next.baselineOK {
		t.Fatalf("completed baseline lost normal TERM status: %v", err)
	}
}

type r21AfterBootAuditPlane struct {
	*memPlane
	calls  int
	block  bool
	cancel context.CancelFunc
}

func (p *r21AfterBootAuditPlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
	err := p.memPlane.Baseline(ctx, dp)
	if p.cancel != nil {
		p.cancel()
	}
	return err
}

func (p *r21AfterBootAuditPlane) VerifyServing(ctx context.Context, m enforce.Model, a map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	if p.block {
		p.calls++
		<-ctx.Done()
		return enforce.ServingEvidence{}, ctx.Err()
	}
	return p.memPlane.VerifyServing(ctx, m, a)
}

func TestR21_M2_TermCompletedBootDoesNotStartAnotherStepBudget(t *testing.T) {
	r, src, _ := r20ServedPendingDelta(t)
	p := &r21AfterBootAuditPlane{memPlane: r.plane, block: true}
	next := r20Successor(t, r, r20ChangedSets(r), src, p)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	p.cancel = cancel
	err := next.Run(ctx)
	p.block = false
	defer next.Close()
	if !errors.Is(err, context.Canceled) || !next.baselineOK {
		t.Fatalf("TERM did not complete its baseline normally: %v baseline=%v", err, next.baselineOK)
	}
	if p.calls != 0 {
		t.Fatalf("TERM after Boot started %d additional Step audits outside Boot+Close budget", p.calls)
	}
}
