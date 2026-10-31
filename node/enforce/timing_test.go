package enforce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"
	"testing"
	"time"
)

// Independent samples inside the executor prove that exported brackets enclose
// the actual call, rather than a later read/log/rollback interval.
type precisionTimingExec struct {
	*fakeExec
	fail  string
	spans map[string]CommandTiming
}

func (f *precisionTimingExec) record(kind string, run func() error) error {
	s := CommandTiming{Start: timingSample()}
	var err error
	if kind == f.fail {
		err = fmt.Errorf("TEST ONLY private executor diagnostic")
	} else {
		err = run()
	}
	s.End, s.OK = timingSample(), err == nil
	f.spans[kind] = s
	return err
}

func (f *precisionTimingExec) NftBatch(ctx context.Context, script string) error {
	kind := "rollback"
	if strings.Contains(script, "add chain inet fvpn "+deadlineGuard) {
		kind = "install"
	} else if script == "delete chain inet fvpn "+deadlineGuard+"\n" {
		kind = "release"
	}
	return f.record(kind, func() error { return f.fakeExec.NftBatch(ctx, script) })
}

func (f *precisionTimingExec) Output(ctx context.Context, name string, args ...string) (raw []byte, err error) {
	if name != "nft" || strings.Join(args, " ") != "list set inet fvpn allowed_v4" {
		return f.fakeExec.Output(ctx, name, args...)
	}
	err = f.record("precision", func() error {
		raw, err = f.fakeExec.Output(ctx, name, args...)
		if f.fail == "invalid_precision" {
			raw = []byte("TEST ONLY invalid listing")
		}
		return err
	})
	return raw, err
}

func TestD104_ISOF603_TimingSurvivesFailure(t *testing.T) {
	for _, fail := range []string{"", "install", "release", "precision", "invalid_precision"} {
		t.Run(fail, func(t *testing.T) {
			m := smallModel()
			ex := &precisionTimingExec{fakeExec: servingExec(m), fail: fail, spans: map[string]CommandTiming{}}
			res, err := Apply(t.Context(), ex, m, nil)
			v := res.Evidence.TimingEvidence()
			if (err == nil) != (fail == "") || v == nil || v.Succeeded != (err == nil) || v.Verified != res.Verified || v.Operation != "apply" || v.Version != 1 || v.Clock != "CLOCK_MONOTONIC" || uint64(v.CandidateGeneration) != m.Generation {
				t.Fatalf("timing=%+v err=%v", v, err)
			}
			d := v.Deadline
			if d != res.Deadline || v.Start != d.ApplyStart || v.Start <= 0 || d.RenderStart < v.Start || d.InstallStart < d.RenderStart || d.InstalledBy < d.InstallStart || v.End < d.InstalledBy {
				t.Fatalf("invalid timeline: %+v", v)
			}
			encloses := func(start, end time.Duration, name string) {
				t.Helper()
				s := ex.spans[name]
				if start <= 0 || start > s.Start || s.Start > s.End || s.End > end || end > v.End {
					t.Fatalf("%s not bracketed: %d <= %+v <= %d", name, start, s, end)
				}
			}
			encloses(d.InstallStart, d.InstalledBy, "install")
			if d.InstallOK != (fail != "install") {
				t.Fatal("install outcome lost")
			}
			if fail == "install" {
				if d.ReleaseStart != 0 || d.ReleaseEnd != 0 || d.GuardReleased || len(v.Verifications) != 0 {
					t.Fatal("invented release/verification after failed install")
				}
			} else {
				encloses(d.ReleaseStart, d.ReleaseEnd, "release")
				if d.ReleaseStart < d.InstalledBy || d.GuardReleased != (fail != "release") {
					t.Fatal("release outcome lost")
				}
			}
			if fail == "release" && len(v.Verifications) != 0 {
				t.Fatal("invented verification after failed release")
			}
			if fail == "" || strings.Contains(fail, "precision") {
				if len(v.Verifications) != 1 {
					t.Fatalf("verifiers=%d", len(v.Verifications))
				}
				r := v.Verifications[0]
				if r.Start < d.ReleaseEnd || r.End > v.End || r.NFTRead.Start < r.Start || r.NFTRead.End > r.PrecisionRead.Start || r.TCRead.Start < r.PrecisionRead.End || r.TCRead.End > r.End || !r.NFTRead.OK || !r.TCRead.OK || r.Verified != (fail == "") || r.SHA256 != res.Evidence.TCReadbackEvidence().Sha256 {
					t.Fatalf("verifier timeline=%+v", r)
				}
				encloses(r.PrecisionRead.Start, r.PrecisionRead.End, "precision")
				if r.PrecisionRead != res.Evidence.DeadlineRead || r.PrecisionRead.ReadOK != (fail != "precision") || r.PrecisionRead.Verified != (fail == "") {
					t.Fatal("precision command success confused with validation")
				}
				v.Verifications[0].Start = 0
				if res.Evidence.TimingEvidence().Verifications[0].Start != r.Start {
					t.Fatal("snapshot aliases caller edits")
				}
			}
			raw, err := json.Marshal(res.Evidence.TimingEvidence())
			if err != nil || strings.Contains(string(raw), "private executor") || strings.Contains(string(raw), "100.64.") || !strings.Contains(string(raw), `"start_ns":"`) {
				t.Fatalf("unsafe or imprecise JSON: %s %v", raw, err)
			}
		})
	}
}

func TestD104_ISOF603_TimingRecoveryAndLiveRead(t *testing.T) {
	ex := newQuarantineExec(t)
	if _, err := Quarantine(t.Context(), ex, ex.m); err != nil {
		t.Fatal(err)
	}
	ex.foreign = false
	res, err := Apply(t.Context(), ex, ex.m, nil)
	v := res.Evidence.TimingEvidence()
	if err != nil || !res.Verified || v == nil || len(v.Verifications) != 2 {
		t.Fatalf("both recovery reads required: %+v %v", v, err)
	}
	if !v.Verifications[0].Verified || !v.Verifications[1].Verified || v.Verifications[0].End > v.Verifications[1].Start {
		t.Fatal("recovery reads out of order")
	}
	live, err := VerifyServing(t.Context(), ex, ex.m, ex.plan.Assignments)
	l := live.TimingEvidence()
	if err != nil || l == nil || l.Operation != "verify" || l.Start < v.End || l.Deadline != (DeadlineEvidence{}) || len(l.Verifications) != 1 || !l.Verified || l.Verifications[0].SHA256 != live.TCReadbackEvidence().Sha256 {
		t.Fatalf("live read confused with Apply: %+v %v", l, err)
	}
	if (ServingEvidence{}).TimingEvidence() != nil {
		t.Fatal("zero evidence invented a read")
	}
}

func TestD104_ISOF603_NoInventedStages(t *testing.T) {
	m := smallModel()
	m.Datapath.MemberIngress = ""
	res, err := Apply(t.Context(), &fakeExec{}, m, nil)
	v := res.Evidence.TimingEvidence()
	if err == nil || v == nil || v.Succeeded || v.Verified || v.Start <= 0 || v.End < v.Start || v.Deadline.RenderStart != 0 || v.Deadline.InstallStart != 0 || len(v.Verifications) != 0 {
		t.Fatalf("pre-render failure: %+v %v", v, err)
	}
	m = smallModel()
	m.Revoke = []netip.Addr{mustAddr("100.64.1.1")}
	ex := servingExec(m)
	res, err = Apply(t.Context(), ex, m, nil)
	v = res.Evidence.TimingEvidence()
	if !IsConntrackDeleteFailure(err) || v == nil || v.Succeeded || !v.Verified || !v.Deadline.GuardReleased || len(v.Verifications) != 1 || !v.Verifications[0].Verified {
		t.Fatalf("verified conntrack failure: %+v %v", v, err)
	}
}
