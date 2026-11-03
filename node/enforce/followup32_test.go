package enforce

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func TestAT13_F32_PostShapeClockReadFailureRollsBack(t *testing.T) {
	m := smallModel()
	ex := servingExec(m)
	clockErr := errors.New("TEST ONLY clock read failed")
	calls := 0
	res, err := apply(context.Background(), ex, m, nil, func() (time.Duration, error) {
		calls++
		if calls == 1 {
			return time.Second, nil
		}
		return 0, clockErr
	})
	if !errors.Is(err, clockErr) || res.Verified || len(ex.tc) != 1 || len(ex.nft) != 1 || !strings.Contains(ex.nft[0], "fvpn_baseline=1") {
		t.Fatalf("clock failure did not close after shape: verified=%v tc=%d nft=%d err=%v", res.Verified, len(ex.tc), len(ex.nft), err)
	}
}

// M-2's alleged post-verify non-command error is rejected: the only such
// deleteConntrack input fails Render before either mutation. Samples has no
// error-producing branch. Command failures retain their existing G7 tests.
func TestR24_M2_FatalRevokeInputRejectedBeforeMutation(t *testing.T) {
	for _, addr := range []netip.Addr{{}, netip.MustParseAddr("2001:db8::1")} {
		m := smallModel()
		m.Revoke = []netip.Addr{addr}
		ex := servingExec(m)
		res, err := Apply(context.Background(), ex, m, nil)
		if err == nil || res.Verified || len(ex.tc) != 0 || len(ex.nft) != 0 {
			t.Fatalf("invalid revoke mutated or verified: tc=%d nft=%d verified=%v err=%v", len(ex.tc), len(ex.nft), res.Verified, err)
		}
	}
}

// Model a crash after guarded commit and before retirement: nft no longer
// references B but tc still contains B. Restarted Apply must reserve its minor,
// keep it until the next commit, then reclaim it before exact verification.
func TestAT13_F32_OrphanCrashCutRecovery(t *testing.T) {
	for _, mode := range []string{"recover", "retire_failure", "foreign_filter", "missing_class", "extra_class"} {
		t.Run(mode, func(t *testing.T) {
			old := goldenModel()
			oldPlan, err := Render(old, State{}, TCState{})
			if err != nil {
				t.Fatal(err)
			}
			m := smallModel()
			m.Generation = old.Generation + 1
			m.QoS.Assignments = oldPlan.Assignments
			plan, err := Render(m, State{}, TCState{})
			if err != nil {
				t.Fatal(err)
			}
			orphan := oldPlan.Assignments[memberB]
			if strings.Contains(plan.NFT, fmt.Sprintf("meta priority set 0x%x", ClassID(orphan))) {
				t.Fatal("candidate still references retired class")
			}
			base := servingExec(m)
			committed, retired, verified, baseline, rootDeleted := false, false, false, false, false
			var tcScripts []string
			ex := &r24HookExec{fakeExec: base}
			ex.onTC = func(_ context.Context, script string) error {
				tcScripts = append(tcScripts, script)
				if strings.Contains(script, "class del") {
					if !committed {
						t.Fatal("retirement preceded commit")
					}
					if !strings.Contains(script, "class del dev fv-ih classid "+classHandle(orphan)) {
						t.Fatal("orphan was not reclaimed")
					}
					if mode == "retire_failure" {
						return errors.New("TEST ONLY retirement failed")
					}
					retired = true
				} else if strings.Contains(script, " del ") {
					t.Fatal("preparation deleted tc")
				}
				return nil
			}
			ex.onBatch = func(_ context.Context, script string) error {
				if strings.Contains(script, "fvpn_baseline=1") {
					baseline = true
				} else if strings.Contains(script, "add chain inet fvpn "+deadlineGuard) {
					committed = true
				} else if strings.Contains(script, "delete chain inet fvpn "+deadlineGuard) && !retired {
					t.Fatal("released before retirement")
				}
				return nil
			}
			ex.onOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
				key := name + " " + strings.Join(args, " ")
				if key == "nft -j list table inet fvpn" {
					return fixtureServing(m, plan.Assignments), nil
				}
				if key == "nft -j list ruleset" {
					if !baseline && mode != "foreign_filter" {
						if !retired {
							t.Fatal("serving verification ran on transitional orphan state")
						}
						verified = true
					}
					return base.Output(ctx, name, args...)
				}
				if strings.HasPrefix(key, "tc qdisc del") {
					rootDeleted = true
					return nil, nil
				}
				if name == "tc" && len(args) > 1 && args[0] == "-j" {
					if args[1] == "filter" {
						if mode == "foreign_filter" && strings.HasSuffix(key, "parent "+classHandle(orphan)) {
							return []byte(`[{"kind":"matchall","handle":"1","options":{"actions":[{"kind":"mirred"}]}}]`), nil
						}
						return []byte(`[]`), nil
					}
					pair := fixtureQoS(m, plan.Assignments)
					if !retired {
						for iface, v := range pair {
							v[0] = []byte(strings.TrimSuffix(string(v[0]), "]") + fmt.Sprintf(`,{"kind":"fq_codel","handle":"%x:","parent":%q}]`, orphan, classHandle(orphan)))
							v[1] = []byte(strings.TrimSuffix(string(v[1]), "]") + fmt.Sprintf(`,{"kind":"htb","handle":%q,"options":{"rate":"10mbit","ceil":"20mbit"}}]`, classHandle(orphan)))
							pair[iface] = v
						}
					}
					if args[1] == "qdisc" {
						return pair[args[len(args)-1]][0], nil
					}
					if args[1] == "class" {
						if retired && mode == "missing_class" {
							return []byte(`[]`), nil
						}
						if retired && mode == "extra_class" {
							return []byte(strings.TrimSuffix(string(pair[args[len(args)-1]][1]), "]") + `,{"kind":"htb","handle":"1:abc","options":{"rate":"1mbit","ceil":"1mbit"}}]`), nil
						}
						return pair[args[len(args)-1]][1], nil
					}
				}
				return base.Output(ctx, name, args...)
			}
			res, err := Apply(context.Background(), ex, m, nil)
			switch mode {
			case "recover":
				if err != nil || !res.Verified || !retired || !verified || baseline {
					t.Fatalf("orphan recovery: verified=%v retired=%v err=%v", res.Verified, retired, err)
				}
			case "retire_failure":
				if err == nil || res.Verified || res.Deadline.GuardReleased || !baseline || !rootDeleted {
					t.Fatalf("failed retirement did not baseline/remove root: %+v err=%v", res, err)
				}
			case "foreign_filter":
				if ReasonCode(err) != ReasonTCForeignFilter || res.Verified || committed || len(tcScripts) != 0 || rootDeleted {
					t.Fatalf("foreign attachment accepted/deleted: tc=%v err=%v", tcScripts, err)
				}
			case "missing_class", "extra_class":
				if err == nil || res.Verified || !baseline {
					t.Fatalf("missing class accepted: %+v err=%v", res, err)
				}
			}
		})
	}
}

// Unlike r24's fake, this executor honors cancellation on every output. The
// canceled verifier's tc read therefore selects quarantine; cleanup must still
// lower the member link and install/read back the baseline using a bounded ctx.
func TestAT13_F32_CanceledVerificationQuarantinesIndependently(t *testing.T) {
	base := newQuarantineExec(t)
	base.foreign = false
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ex := &f32CanceledExec{quarantineExec: base, cancel: cancel, t: t}
	res, err := Apply(ctx, ex, base.m, nil)
	if err == nil || res.Verified || res.Quarantine == nil || !res.Quarantine.Quarantined || base.up || !base.baseline {
		t.Fatalf("canceled verification not contained: %+v err=%v", res, err)
	}
}

type f32CanceledExec struct {
	*quarantineExec
	cancel   context.CancelFunc
	t        *testing.T
	canceled bool
}

func (f *f32CanceledExec) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "nft" && strings.Join(args, " ") == "-j list ruleset" && !f.canceled {
		f.canceled = true
		f.cancel()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.canceled {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > cleanupTimeout {
			f.t.Fatal("cleanup is not bounded")
		}
	}
	return f.quarantineExec.Output(ctx, name, args...)
}
func (f *f32CanceledExec) NftBatch(ctx context.Context, script string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return f.quarantineExec.NftBatch(ctx, script)
}

func TestAT13_F32_BaselineIndependentOfSnapshot(t *testing.T) {
	m := Model{Datapath: smallModel().Datapath}
	cold, err := RenderBaseline(m, State{})
	if err != nil {
		t.Fatal(err)
	}
	stale := State{Table: true, Chains: map[string]bool{}, Sets: map[string]bool{}, Counters: map[string]CounterValue{}}
	for _, name := range []string{"forward", "input", "output", "postrouting", deadlineGuard} {
		stale.Chains[name] = true
	}
	for _, name := range []string{"allowed_v4", "denied_v4", "host_v4", "resolver_v4", setProbeV4, setProbeDstV4} {
		stale.Sets[name] = true
	}
	for _, name := range diagnosticCounters() {
		stale.Counters[name] = CounterValue{Bytes: 123, Packets: 3}
	}
	warm, err := RenderBaseline(m, stale)
	if err != nil || warm != cold {
		t.Fatalf("snapshot changes baseline safety commands: %v\n%s", err, warm)
	}
	if strings.Contains(cold, "flush table") || strings.Contains(cold, "delete counter") {
		t.Fatal("baseline resets counters")
	}
}

func TestAT13_F32_EachMutationFailureUsesBoundedCleanup(t *testing.T) {
	for _, cut := range []string{"shape", "install", "release"} {
		t.Run(cut, func(t *testing.T) {
			type markerKey struct{}
			parent := context.WithValue(context.Background(), markerKey{}, "timing retained")
			ctx, cancel := context.WithCancel(parent)
			defer cancel()
			m := smallModel()
			base := servingExec(m)
			rolledBack := false
			fail := func() error { cancel(); return ctx.Err() }
			ex := &r24HookExec{fakeExec: base}
			ex.onTC = func(context.Context, string) error {
				if cut == "shape" {
					return fail()
				}
				return nil
			}
			ex.onBatch = func(c context.Context, s string) error {
				if strings.Contains(s, "fvpn_baseline=1") {
					deadline, ok := c.Deadline()
					if c.Err() != nil || !ok || time.Until(deadline) > cleanupTimeout || c.Value(markerKey{}) != "timing retained" {
						t.Fatal("invalid cleanup context")
					}
					rolledBack = true
					return nil
				}
				if cut == "install" || cut == "release" && s == "delete chain inet fvpn "+deadlineGuard+"\n" {
					return fail()
				}
				return nil
			}
			res, err := Apply(ctx, ex, m, nil)
			if err == nil || res.Verified || !rolledBack {
				t.Fatalf("cut %s lost rollback: verified=%v rollback=%v err=%v", cut, res.Verified, rolledBack, err)
			}
		})
	}
}
