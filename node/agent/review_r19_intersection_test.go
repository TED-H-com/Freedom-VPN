package agent

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"reflect"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/controlclient"
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

func TestReviewR19_Q1_ReferenceReplayExtendsDeadline(t *testing.T) {
	in := f33Fixture()
	base := f33Intersection(in)
	if base.Baseline || len(base.Candidate.Bindings) != 2 {
		t.Fatalf("fixture: baseline=%v reason=%s devices=%s", base.Baseline, base.Reason, f33Devices(base))
	}
	later := in
	later.Lease.Anchor = 10 * time.Second
	later.Now = 11 * time.Second
	later.Installed.Deadline = map[string]time.Duration{"a": 110 * time.Second, "b": 110 * time.Second}
	got := f33Intersection(later)
	if got.Baseline || len(got.Candidate.Bindings) != 2 {
		t.Fatalf("replay baselined: %s", got.Reason)
	}
	if got.Candidate.Bindings[0].Deadline <= base.Candidate.Bindings[0].Deadline {
		t.Fatalf("replay did not move D: base=%s later=%s", base.Candidate.Bindings[0].Deadline, got.Candidate.Bindings[0].Deadline)
	}
	t.Fatalf("TODO(review): recomputing S with Anchor 1s->10s and a rebuilt installed deadline moved D from %s to %s. The reference does not pin L",
		base.Candidate.Bindings[0].Deadline, got.Candidate.Bindings[0].Deadline)
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

func TestReviewR19_Q2_ReferenceAdmitsUnboundedCeil(t *testing.T) {
	in := f33Fixture()
	huge := uint64(math.MaxUint64)
	in.Old.NodeCeil = f33Ceil{huge, huge}
	in.New.NodeCeil = f33Ceil{huge, huge}
	in.Installed.NodeCeil = f33Ceil{huge, huge}
	got := f33Intersection(in)
	if got.Baseline || got.Candidate.NodeCeil.Up != huge {
		t.Fatalf("MaxUint64 was not admitted: baseline=%v reason=%s ceil=%d", got.Baseline, got.Reason, got.Candidate.NodeCeil.Up)
	}
	t.Fatalf("TODO(review): f33Intersection admits node ceil %d. f33CeilOK only requires Up>0 and Down>0. A missing direction that nodeCapacityRatePair encodes as math.MaxInt64 is a real ceiling, not the node cap", huge)
}

func TestReviewR19_Q2_ReferenceKeepsDeniedResolver(t *testing.T) {
	in := f33Fixture()
	in.New.Denied = append(append([]netip.Prefix{}, in.New.Denied...), netip.MustParsePrefix("192.0.2.53/32"))
	got := f33Intersection(in)
	if got.Baseline || !slices.Contains(got.Candidate.Resolvers, netip.MustParseAddr("192.0.2.53")) {
		t.Fatalf("resolver handling changed: baseline=%v reason=%s resolvers=%v", got.Baseline, got.Reason, got.Candidate.Resolvers)
	}
	var denied bool
	for _, p := range got.Candidate.Denied {
		if p.String() == "192.0.2.53/32" {
			denied = true
		}
	}
	if !denied {
		t.Fatal("denial union dropped 192.0.2.53/32")
	}
	rt := reflect.TypeOf(f33Policy{})
	if _, ok := rt.FieldByName("ResolverPorts"); ok {
		t.Fatal("review assumption stale: resolver ports are modeled")
	}
	t.Fatalf("TODO(review): Baseline=false with resolver 192.0.2.53 also present in Denied. The result has no port. An all-ports accept of Candidate.Resolvers opens more than TCP/UDP 53")
}

func TestReviewR19_Q2_ReferenceIgnoresRateModeAndWitnessSnapshot(t *testing.T) {
	in := f33Fixture()
	got := f33Intersection(in)
	if f33Devices(got) != "ab" {
		t.Fatalf("fixture devices=%s", f33Devices(got))
	}
	for _, name := range []string{"Mode", "Rate", "Period", "Snapshot", "WitnessHash"} {
		if _, ok := reflect.TypeOf(f33Policy{}).FieldByName(name); ok {
			t.Fatalf("review assumption stale: f33Policy.%s exists", name)
		}
		if _, ok := reflect.TypeOf(f33Member{}).FieldByName(name); ok {
			t.Fatalf("review assumption stale: f33Member.%s exists", name)
		}
	}
	if _, ok := reflect.TypeOf(f33Input{}).FieldByName("AtomicPolicy"); !ok || !in.AtomicPolicy {
		t.Fatal("AtomicPolicy flag missing")
	}
	t.Fatalf("TODO(review): unchanged ceils return devices %s with AtomicPolicy=%v. f33Policy has no mode, rate guarantee, selection interval, or witness snapshot hash. A shadow-to-enforce tightening, a second rate interval, or a stale witness that reuses entitlement revision 11 is admitted",
		f33Devices(got), in.AtomicPolicy)
}

func TestReviewR19_Q2_RequiresTCKeepsCandidate(t *testing.T) {
	in := f33Fixture()
	in.New.NodeCeil = f33Ceil{80, 150}
	got := f33Intersection(in)
	if !got.Baseline || got.Reason != "requires_tc" || len(got.Candidate.Bindings) == 0 {
		t.Fatalf("setup baseline=%v reason=%s bindings=%d", got.Baseline, got.Reason, len(got.Candidate.Bindings))
	}
	t.Fatalf("TODO(review): requires_tc returns Baseline=true and still retains %d candidate bindings at node ceil %d/%d. A caller that installs Candidate without reading Baseline forwards",
		len(got.Candidate.Bindings), got.Candidate.NodeCeil.Up, got.Candidate.NodeCeil.Down)
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
	if err != nil || berr != nil || refs.DnsGeneration != 999 || bind != 4242 || !rep.MemberForwardingEnabled || !modelHasPrefix(r.plane.last, drop.AllowedIpv4[0]) {
		t.Fatalf("report did not stamp the live desired document: dns=%s bind=%s forwarding=%v err=%v %v",
			refs.DnsGeneration, bind, rep.MemberForwardingEnabled, err, berr)
	}
	t.Fatalf("TODO(review): after accepting generation %d and before withdrawal, report ACK applied=%d stamps desired DNS generation 999 and binding revision 4242 with member_forwarding_enabled=true. Engine.report reads e.desired, not the retained applied snapshot",
		applied+1, applied)
}

func TestReviewR19_Q3_UnsupportedObservationDoesNotBaseline(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	var posts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		posts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"VALIDATION_FAILED","message":"unknown forwarding_block_reason","request_id":"r19","retryable":false}}`))
	}))
	t.Cleanup(srv.Close)
	client, err := controlclient.New(controlclient.Config{BaseURL: srv.URL, RootCAs: x509.NewCertPool()})
	if err != nil {
		t.Fatal(err)
	}
	orig := r.eng.client
	r.eng.client = client
	t.Cleanup(func() {
		r.eng.client = orig
		client.Close()
	})
	rep, err := r.eng.report(context.Background(), r.clock.Wall().UTC())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.eng.store.Enqueue(context.Background(), rep.ReportId, raw, r.clock.Wall().UTC()); err != nil {
		t.Fatal(err)
	}
	r.eng.observationRetryAt = 0
	baselines, serving := r.plane.baselines, r.eng.serving
	if err := r.eng.flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	pending, err := r.eng.store.Pending(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if posts.Load() != 1 || r.plane.baselines != baselines || r.eng.serving != serving || len(pending) != 0 {
		t.Fatalf("422 changed dataplane or left the report pending: posts=%d baselines %d->%d serving %v->%v pending=%d",
			posts.Load(), baselines, r.plane.baselines, serving, r.eng.serving, len(pending))
	}
	t.Fatalf("TODO(review): HTTP 422 was MarkConflict and the serving ruleset stayed up (posts=%d baselines=%d serving=%v). observe.go flush does not abandon an intersection or install the baseline", posts.Load(), baselines, serving)
}

func TestReviewR19_Q4_QualifyingShareOnMergedEvidence(t *testing.T) {
	type population struct {
		Transitions []struct {
			BaselineSequences []string `json:"baseline_sequences"`
			Metrics           struct {
				End string `json:"baseline_to_apply_end_ns"`
			} `json:"metrics"`
		} `json:"transitions"`
	}
	load := func(path string) (n, baselined int, p50, p99 int64) {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var doc population
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		if len(doc.Transitions) != 40 {
			t.Fatalf("%s population %d", path, len(doc.Transitions))
		}
		ns := make([]int64, 0, len(doc.Transitions))
		for _, tr := range doc.Transitions {
			if len(tr.BaselineSequences) > 0 {
				baselined++
			}
			var v int64
			if _, err := fmt.Sscanf(tr.Metrics.End, "%d", &v); err != nil || v <= 0 {
				t.Fatalf("%s duration %q: %v", path, tr.Metrics.End, err)
			}
			ns = append(ns, v)
		}
		slices.Sort(ns)
		return len(doc.Transitions), baselined, ns[nearestRank(len(ns), 0.50)-1], ns[nearestRank(len(ns), 0.99)-1]
	}
	finalN, finalBase, p50, p99 := load("../../docs/node/followup31-evidence/generations-final/analysis.json")
	earlyN, earlyBase, _, _ := load("../../docs/node/followup31-evidence/generations/analysis.json")
	if finalN+earlyN != 80 || finalBase != finalN || earlyBase != earlyN {
		t.Fatalf("populations final=%d/%d early=%d/%d", finalN, finalBase, earlyN, earlyBase)
	}
	members, err := os.ReadFile("../../docs/e2e/at/members/summary.md")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(members, []byte("58 baseline calls")) || !bytes.Contains(members, []byte("215.189032 ms")) {
		t.Fatal("members summary no longer records the AT02 baseline count or the 215.189032 ms transition")
	}
	if _, ok := reflect.TypeOf(api.DesiredState{}).FieldByName("IntersectionConstraints"); ok {
		t.Fatal("witness exists; share estimate is stale")
	}
	t.Fatalf("TODO(review): qualifying share for S on today's documents is 0 of 80 follow-up 31 transitions (40+40, every one baselined). generations-final nearest-rank baseline-to-apply p50=%d ns p99=%d ns. Members run 258ba504 records 58 AT02 baselines and one 215.189032 ms baseline-to-Apply. DesiredState has no intersection witness, and compiler BindingRevision is the max device revision",
		p50, p99)
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

func TestReviewR19_Q5_RollbackSourceDoesNotBaseline(t *testing.T) {
	src, err := os.ReadFile("../updater/apply.go")
	if err != nil {
		t.Fatal(err)
	}
	const sig = "func (e *Engine) stepRollback("
	i := bytes.Index(src, []byte(sig))
	if i < 0 {
		t.Fatal("stepRollback missing")
	}
	rest := src[i+len(sig):]
	j := bytes.Index(rest, []byte("\nfunc "))
	if j < 0 {
		t.Fatal("stepRollback end missing")
	}
	body := rest[:j]
	if bytes.Contains(body, []byte(".Baseline(")) || bytes.Contains(body, []byte("ApplyBaseline")) || bytes.Contains(body, []byte("nft ")) {
		t.Fatal("stepRollback now touches the dataplane; the review assumption is stale")
	}
	if !bytes.Contains(body, []byte("e.Services.Stop(ctx)")) || !bytes.Contains(body, []byte("e.Services.Start(ctx, e.Layout.releaseDir(target))")) {
		t.Fatal("stepRollback no longer stops and starts the restored release")
	}
	t.Fatal("TODO(review): stepRollback stops services, switches the symlink, and starts the restored release. It does not install a baseline or rewrite the heartbeat before the old binary runs")
}

func TestReviewR19_Q4_ConntrackAttemptIsCountedAfterTheDelete(t *testing.T) {
	eng, err := os.ReadFile("engine.go")
	if err != nil {
		t.Fatal(err)
	}
	applySrc, err := os.ReadFile("../enforce/apply.go")
	if err != nil {
		t.Fatal(err)
	}
	del := bytes.Index(applySrc, []byte(`"conntrack", "-D"`))
	inc := bytes.Index(eng, []byte("e.conntrackAttempts++"))
	persist := bytes.Index(eng, []byte("e.persistConntrack(ctx)"))
	if del < 0 || inc < 0 || persist < inc {
		t.Fatalf("order missing delete=%d increment=%d persist=%d", del, inc, persist)
	}
	if bytes.Contains(applySrc, []byte("conntrackAttempts")) {
		t.Fatal("Apply learned the attempt counter; the review assumption is stale")
	}
	t.Fatal("TODO(review): conntrack -D runs inside Apply and engine.go increments conntrackAttempts only after Apply returns, then persistConntrack. A crash between the delete and the meta write under-counts the 12-attempt bound. No reservation is persisted before the batch")
}

func nearestRank(n int, p float64) int {
	rank := int(math.Ceil(p * float64(n)))
	if rank < 1 {
		return 1
	}
	if rank > n {
		return n
	}
	return rank
}
