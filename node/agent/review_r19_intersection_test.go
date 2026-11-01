package agent

import (
	"bytes"
	"context"
	"net/netip"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// TestReviewR19_Q1_BoundaryIsBeforeSecondSyncLease shows where option (c)
// actually closes, and that the design's interception point is still before
// the second syncLease on the ordinary path.
func TestReviewR19_Q1_BoundaryIsBeforeSecondSyncLease(t *testing.T) {
	r := newRig(t)
	_, drop := servePair(t, r)
	dropped := drop.AllowedIpv4[0]
	renewals := r.srv.LeaseRenewals()
	baselines := r.plane.baselines
	r.eng.leaseAttempted = false
	r.srv.SetFixture(fixtureWithBindings(4, threeBindings(r.clock.Wall())[:1]))
	if err := r.eng.syncDesired(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !r.eng.serving || !modelHasPrefix(r.plane.last, dropped) {
		t.Fatal("acceptance already withdrew")
	}
	if r.eng.needRenew() || r.srv.LeaseRenewals() != renewals {
		t.Fatalf("ordinary acceptance became renewal-due: need=%v renewals=%d", r.eng.needRenew(), r.srv.LeaseRenewals())
	}
	if err := r.eng.syncLease(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.plane.baselines != baselines || !r.eng.serving || r.srv.LeaseRenewals() != renewals {
		t.Fatalf("ordinary syncLease withdrew or renewed: baselines %d->%d serving=%v renewals=%d",
			baselines, r.plane.baselines, r.eng.serving, r.srv.LeaseRenewals())
	}
	r.eng.evaluate(context.Background())
	if err := r.eng.enforce(context.Background(), r.eng.decision); err != nil {
		t.Fatal(err)
	}
	if r.eng.serving || r.plane.baselines <= baselines || modelHasPrefix(r.plane.last, dropped) {
		t.Fatalf("enforce did not withdraw: serving=%v baselines %d->%d", r.eng.serving, baselines, r.plane.baselines)
	}
}

// TestReviewR19_Q1_RetainedErrorRenewsBeforeWithdrawal shows the path on
// which Step's second syncLease performs a lease HTTP before enforce.
func TestReviewR19_Q1_RetainedErrorRenewsBeforeWithdrawal(t *testing.T) {
	r := newRig(t)
	keep, drop := servePair(t, r)
	r.eng.leaseAttempted = false
	r.srv.SetFixture(fixtureWithBindings(4, []api.DeviceBinding{keep}))
	if err := r.eng.syncDesired(context.Background()); err != nil {
		t.Fatal(err)
	}
	r.eng.applyError = &api.NodeError{
		Code: api.ErrorCodeHEALTHCHECKFAILED, Message: "apply failed",
		ObservedAt: r.clock.Wall().UTC(), Stage: "apply",
	}
	if !r.eng.needRenew() || !r.eng.leaseBehindDesired() {
		t.Fatal("retained error did not become renewal-due before enforce")
	}
	renewals := r.srv.LeaseRenewals()
	baselines := r.plane.baselines
	if err := r.eng.syncLease(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.srv.LeaseRenewals() <= renewals || r.plane.baselines != baselines || !modelHasPrefix(r.plane.last, drop.AllowedIpv4[0]) {
		t.Fatalf("lease HTTP did not precede withdrawal: renewals %d->%d baselines %d->%d",
			renewals, r.srv.LeaseRenewals(), baselines, r.plane.baselines)
	}
}

func TestReviewR19_Q1_CorruptDesiredDoesNotInstall(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	gen, bindings := r.eng.appliedGen, len(r.plane.last.Bindings)
	r.srv.CorruptSignatures(true)
	r.srv.SetFixture(generationFixture(gen+1, oneBinding(r.clock.Wall())))
	if err := r.eng.syncDesired(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !r.eng.serving || r.eng.appliedGen != gen || uint64(r.eng.desired.ConfigGeneration) != gen || len(r.plane.last.Bindings) != bindings {
		t.Fatalf("corrupt desired installed: serving=%v applied=%d desired=%s bindings=%d",
			r.eng.serving, r.eng.appliedGen, r.eng.desired.ConfigGeneration, len(r.plane.last.Bindings))
	}
}

func TestReviewR19_Q1_DeadlineReserveBoundary(t *testing.T) {
	exclude := f33Fixture()
	edge := 2*time.Second + enforce.DeadlineInstallBound
	exclude.Installed.Deadline = map[string]time.Duration{"a": edge, "b": edge}
	got := f33Intersection(exclude)
	if !got.Baseline || got.Reason != "empty_subset" {
		t.Fatalf("remaining == 100ms admitted: baseline=%v reason=%s devices=%s", got.Baseline, got.Reason, f33Devices(got))
	}
	nsec := f33Fixture()
	nsec.Installed.Deadline = map[string]time.Duration{
		"a": edge + time.Nanosecond, "b": edge + time.Nanosecond,
	}
	got = f33Intersection(nsec)
	if !got.Baseline || got.Reason != "empty_subset" {
		t.Fatalf("remaining == 100ms+1ns admitted: baseline=%v reason=%s", got.Baseline, got.Reason)
	}
	include := f33Fixture()
	include.Installed.Deadline = map[string]time.Duration{
		"a": edge + enforce.DeadlineGranularity, "b": edge + enforce.DeadlineGranularity,
	}
	got = f33Intersection(include)
	if got.Baseline || f33Devices(got) != "ab" {
		t.Fatalf("remaining == 101ms baselined: %s devices=%s", got.Reason, f33Devices(got))
	}
}

func TestReviewR19_Q2_ReferenceIdentityExcludesChangedDevice(t *testing.T) {
	cases := []struct {
		name string
		edit func(*f33Input)
		want string
	}{
		{"peer", func(in *f33Input) { in.New.Bindings[0].Peer = "other" }, "b"},
		{"address", func(in *f33Input) {
			in.New.Bindings[0].Addresses = []netip.Prefix{netip.MustParsePrefix("100.64.9.9/32")}
		}, "b"},
		{"device", func(in *f33Input) { in.New.Bindings[0].Device = "other" }, "b"},
		{"member", func(in *f33Input) { in.New.Bindings[0].Member = "other" }, "b"},
		{"revision", func(in *f33Input) { in.New.Bindings[0].Revision = 10 }, "b"},
		{"added_device", func(in *f33Input) {
			extra := in.New.Bindings[0]
			extra.Device, extra.Peer = "added", "peer-added"
			extra.Addresses = []netip.Prefix{netip.MustParsePrefix("100.64.9.9/32")}
			in.New.Bindings = append(in.New.Bindings, extra)
		}, "ab"},
		{"same_set_revision_removed", func(in *f33Input) {
			in.New.BindingSetRevision = in.Old.BindingSetRevision
			in.New.Bindings = in.New.Bindings[1:]
		}, "baseline"},
		{"decreased_set_revision", func(in *f33Input) {
			in.New.BindingSetRevision = in.Old.BindingSetRevision - 1
			in.New.Bindings = in.New.Bindings[1:]
		}, "baseline"},
		{"zero_ceil", func(in *f33Input) {
			in.New.NodeCeil = f33Ceil{}
		}, "baseline"},
		{"unverified", func(in *f33Input) { in.New.Verified = false }, "baseline"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := f33Fixture()
			tc.edit(&in)
			if got := f33Devices(f33Intersection(in)); got != tc.want {
				t.Fatalf("devices=%s want %s", got, tc.want)
			}
		})
	}
}

func TestReviewR19_Q2_DesiredStateHasNoWitness(t *testing.T) {
	if _, ok := reflect.TypeOf(api.DesiredState{}).FieldByName("IntersectionConstraints"); ok {
		t.Fatal("DesiredState grew an intersection witness; the review assumption is stale")
	}
	if _, ok := reflect.TypeOf(api.MemberLease{}).FieldByName("EntitlementRevision"); !ok {
		t.Fatal("MemberLease lost EntitlementRevision")
	}
	src, err := os.ReadFile("../../internal/desiredstate/compiler.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(src, []byte("BindingRevision:          nodes.ParseU64(maxBindingRev),")) || !bytes.Contains(src, []byte("if b.BindingRevision > maxBindingRev")) {
		t.Fatal("compiler no longer stores the max per-binding revision")
	}
}

func TestReviewR19_Q3_ReportStampsLiveDesiredRefs(t *testing.T) {
	r := newRig(t)
	_, drop := servePair(t, r)
	applied := r.eng.appliedGen
	wantRefs, wantBind := r.eng.desired.PolicyRefs, r.eng.desired.BindingRevision
	r.eng.leaseAttempted = false
	r.srv.SetFixture(fixtureWithBindings(applied+1, threeBindings(r.clock.Wall())[:1]))
	if err := r.eng.syncDesired(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.eng.appliedGen != applied || uint64(r.eng.desired.ConfigGeneration) != applied+1 {
		t.Fatalf("setup applied=%d desired=%s", r.eng.appliedGen, r.eng.desired.ConfigGeneration)
	}
	r.eng.desired.PolicyRefs.DnsGeneration = 999
	r.eng.desired.BindingRevision = 4242
	rep, err := r.eng.report(context.Background(), r.clock.Wall().UTC())
	if err != nil {
		t.Fatal(err)
	}
	gen, err := rep.AppliedConfigGeneration.Get()
	if err != nil || uint64(gen) != applied {
		t.Fatalf("applied generation %s err=%v", gen, err)
	}
	refs, err := rep.AppliedPolicyRefs.Get()
	bind, berr := rep.AppliedBindingRevision.Get()
	if err != nil || berr != nil || refs != wantRefs || bind != wantBind || !rep.MemberForwardingEnabled || !modelHasPrefix(r.plane.last, drop.AllowedIpv4[0]) {
		t.Fatalf("report did not retain the applied document: dns=%s bind=%s forwarding=%v err=%v %v",
			refs.DnsGeneration, bind, rep.MemberForwardingEnabled, err, berr)
	}

}

func TestReviewR19_Q4_D133TextRequiresCeilings(t *testing.T) {
	note, err := os.ReadFile("../../docs/node/followup33-intersection-serving-design-2026-11-01.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{
		"Any higher or missing",
		"strictly lower restrictive p90",
		"No return to I.",
	} {
		if !bytes.Contains(note, []byte(s)) {
			t.Fatalf("design note lost %q", s)
		}
	}
}

func TestReviewR19_Q5_CurrentBootAdoptsVerifiedCheckpoint(t *testing.T) {
	r := newRig(t)
	servePair(t, r)
	gen, baselines := r.eng.appliedGen, r.plane.baselines
	restartEngine(t, r, r.plane)
	if !r.eng.serving || r.eng.appliedGen != gen || r.plane.baselines != baselines {
		t.Fatalf("same-boot adoption failed: serving=%v gen=%d baselines %d->%d", r.eng.serving, r.eng.appliedGen, baselines, r.plane.baselines)
	}
}

func TestReviewR19_Q5_PreserveServingOnStopSucceeds(t *testing.T) {
	r := newRig(t)
	servePair(t, r)
	baselines := r.plane.baselines
	if !r.eng.preserveServingOnStop(context.Background()) || !r.eng.serving || r.plane.baselines != baselines {
		t.Fatalf("preserveServingOnStop=%v serving=%v baselines %d->%d", r.eng.serving, r.eng.serving, baselines, r.plane.baselines)
	}
}

func TestReviewR19_Q5_ConntrackCountSurvivesRestartAndClearResetsIt(t *testing.T) {
	r := newRig(t)
	servePair(t, r)
	r.eng.conntrackAttempts = 11
	r.eng.conntrackBlocked = true
	r.eng.persistConntrack(context.Background())
	restartEngine(t, r, r.plane)
	if r.eng.conntrackAttempts != 11 || !r.eng.conntrackBlocked {
		t.Fatalf("restart reset conntrack state: attempts=%d blocked=%v", r.eng.conntrackAttempts, r.eng.conntrackBlocked)
	}
	r.eng.clearConntrackBlock(context.Background())
	if r.eng.conntrackAttempts != 0 || r.eng.conntrackBlocked {
		t.Fatalf("clear did not reset: attempts=%d blocked=%v", r.eng.conntrackAttempts, r.eng.conntrackBlocked)
	}
	raw, ok, err := r.eng.store.Meta(context.Background(), conntrackAttemptsKey)
	if err != nil || !ok || raw != "0" {
		t.Fatalf("persisted attempts %q ok=%v err=%v", raw, ok, err)
	}
}

func TestReviewR19_Q5_InFlightObservationBlocksAdoption(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	r.eng.lastObs = time.Time{}
	if err := r.eng.observe(context.Background()); err != nil {
		t.Fatal(err)
	}
	rows, err := r.eng.store.Pending(context.Background())
	if err != nil || len(rows) != 1 {
		t.Fatalf("pending %d err=%v", len(rows), err)
	}
	if err := r.eng.store.BumpAttempt(context.Background(), rows[0].ReportID, "in_flight"); err != nil {
		t.Fatal(err)
	}
	restartEngine(t, r, r.plane)
	if r.eng.serving || !r.eng.freezeUncertain {
		t.Fatalf("in-flight observation adopted: serving=%v uncertain=%v", r.eng.serving, r.eng.freezeUncertain)
	}
}

// The executor runs the real Apply delete loop, then interrupts it before any
// post-Apply metadata can be written. Each fresh engine loads only durable state.
func TestReviewR19_Q4_ConntrackAttemptIsCountedAfterTheDelete(t *testing.T) {
	r := newRig(t)
	servePair(t, r)
	ctx := context.Background()
	r.srv.SetFixture(fixtureWithBindings(4, threeBindings(r.clock.Wall())[:1]))
	if err := r.eng.syncDesired(ctx); err != nil {
		t.Fatal(err)
	}
	r.eng.leaseRefreshPending, r.eng.leaseAttempted = true, false
	if err := r.eng.syncLease(ctx); err != nil {
		t.Fatal(err)
	}
	r.eng.evaluate(ctx)
	if !r.eng.decision.Serve {
		t.Fatal(r.eng.decision.Reasons)
	}
	model := r.eng.decision.Model
	expires := r.eng.decision.PolicyExpires
	crashes := 0
	for attempt := 1; attempt <= maxConntrackDeleteAttempts; attempt++ {
		p := &reviewR19CrashPlane{memPlane: r.plane}
		p.ex.failDelete = true
		p.ex.crash = func() { crashes++; panic("TEST ONLY crash after conntrack -D") }
		r.eng.plane = p
		func() {
			defer func() {
				if got := recover(); got != "TEST ONLY crash after conntrack -D" {
					t.Fatalf("crash = %v apply=%d log=%s", got, p.applied, r.log.String())
				}
			}()
			_ = r.eng.enforce(ctx, Decision{Serve: true, Model: model, PolicyExpires: expires})
		}()
		// Open a second database connection, with no Close/checkpoint of the failed
		// engine. The next engine must not reconstruct its count from RAM.
		disk, err := agentstore.Open(r.eng.cfg.DBPath())
		if err != nil {
			t.Fatal(err)
		}
		raw, ok, err := disk.Meta(ctx, conntrackAttemptsKey)
		count, parseErr := strconv.Atoi(raw)
		if err != nil || !ok || parseErr != nil || count < crashes {
			_ = disk.Close()
			t.Fatalf("stored count=%q attempts=%d ok=%v err=%v parse=%v", raw, crashes, ok, err, parseErr)
		}
		next := &Engine{store: disk}
		if err := next.loadConntrackState(ctx); err != nil {
			t.Fatal(err)
		}
		if err := next.loadRevocations(ctx); err != nil {
			t.Fatal(err)
		}
		if len(next.pendingRevoke) == 0 {
			t.Fatal("crash lost pending addresses")
		}
		// Reload durable accounting into the otherwise unchanged test engine;
		// no post-Apply continuation from the crashed call is run.
		r.eng.conntrackAttempts, r.eng.conntrackBlocked = next.conntrackAttempts, next.conntrackBlocked
		r.eng.pendingRevoke = next.pendingRevoke
		if err := disk.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if crashes != maxConntrackDeleteAttempts || !r.eng.conntrackBlocked {
		t.Fatalf("crashes=%d blocked=%v", crashes, r.eng.conntrackBlocked)
	}
	p := &reviewR19CrashPlane{memPlane: r.plane}
	r.eng.plane = p
	_ = r.eng.enforce(ctx, Decision{Serve: true, Model: model, PolicyExpires: expires})
	if r.eng.serving || p.applied != 0 {
		t.Fatal("exhausted crash budget admitted another Apply")
	}
}

type reviewR19CrashExecutor struct {
	renderedExecutor
	crash func()
}

func (ex *reviewR19CrashExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	raw, err := ex.renderedExecutor.Output(ctx, name, args...)
	if name == "conntrack" && len(args) > 0 && args[0] == "-D" && ex.crash != nil {
		ex.crash()
	}
	return raw, err
}

type reviewR19CrashPlane struct {
	*memPlane
	ex      reviewR19CrashExecutor
	applied int
}

func (p *reviewR19CrashPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	p.applied++
	m = kernelClockModel(m)
	p.ex.model = m
	return enforce.Apply(ctx, &p.ex, m, prev)
}
