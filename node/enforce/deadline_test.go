package enforce

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestD104_AT13_MillisecondDeadline(t *testing.T) {
	for _, remaining := range []time.Duration{-time.Second, 0, time.Nanosecond, time.Millisecond - 1, time.Millisecond, time.Millisecond + 1, 999 * time.Millisecond, time.Second - 1, time.Second, time.Second + time.Millisecond, 2900*time.Millisecond + 999999} {
		t.Run(remaining.String(), func(t *testing.T) {
			m := smallModel()
			m.NowMono = 50 * time.Second
			d := m.NowMono + remaining
			m.Leases[0].DeadlineMono = &d
			plan, err := Render(m, State{}, TCState{})
			if err != nil {
				t.Fatal(err)
			}
			got := plan.ElementTimeouts[m.Bindings[0].DeviceID.String()]
			want := max(remaining, 0).Truncate(DeadlineGranularity)
			if got != want {
				t.Fatalf("timeout=%s want=%s", got, want)
			}
			if want == 0 && strings.Contains(plan.NFT, "100.64.1.1 timeout") {
				t.Fatal("expired element installed")
			}
			if want > 0 && !strings.Contains(plan.NFT, fmt.Sprintf("timeout %dms", want/time.Millisecond)) {
				t.Fatal("render lost millisecond precision")
			}
			m.timeoutReserve = DeadlineInstallBound
			got, _ = memberTimeout(m.Bindings[0], m.Leases[0], m)
			want = max(remaining-DeadlineInstallBound, 0).Truncate(DeadlineGranularity)
			if got != want {
				t.Fatalf("reserved timeout=%s want=%s", got, want)
			}
		})
	}
}

func TestD104_AllowedPrecisionRead(t *testing.T) {
	m := smallModel()
	m.NowMono = 100 * time.Second
	d := m.NowMono + 2900*time.Millisecond
	m.Leases[0].DeadlineMono = &d
	for _, tc := range []struct {
		timeout, expires string
		elapsed          time.Duration
		pass             bool
	}{
		{"2s900ms", "2s900ms", 0, true},
		{"2s901ms", "2s900ms", 0, false},
		{"2s900ms", "2s901ms", 0, false},
		{"2s900ms", "1s999ms", 901 * time.Millisecond, true},
		{"2s900ms", "2s", 901 * time.Millisecond, false},
		{"2s900ms", "1ms", 2899 * time.Millisecond, true},
		{"2s900ms", "0ms", 2900 * time.Millisecond, true},
		{"2s900ms", "1ms", 2900 * time.Millisecond, false},
		{"2s900ms", "1ms", 3 * time.Second, false},
		{"2s900ms", "999us", 0, false},
	} {
		raw := []byte("table inet fvpn { set allowed_v4 { type ipv4_addr flags timeout elements = { 100.64.1.1 timeout " + tc.timeout + " expires " + tc.expires + " } } }")
		_, err := verifyAllowedLifetimes(raw, m, m.NowMono+tc.elapsed)
		if (err == nil) != tc.pass {
			t.Errorf("%+v: %v", tc, err)
		}
	}
	for _, body := range []string{"100.64.1.1", "100.64.1.1 timeout 0ms", "100.64.1.1/24 timeout 1s", "100.64.1.2 timeout 1s", "100.64.1.1 timeout 18446744073709551615d", "100.64.1.1 timeout 1s, 100.64.1.1 timeout 1s"} {
		_, err := verifyAllowedLifetimes([]byte("table inet fvpn { set allowed_v4 { type ipv4_addr flags timeout elements = { "+body+" } } }"), m, m.NowMono)
		if err == nil {
			t.Errorf("accepted %q", body)
		}
	}
}

type delayedDeadlineExec struct {
	*fakeExec
	delay        time.Duration
	guardRelease bool
}

func (f *delayedDeadlineExec) NftBatch(ctx context.Context, script string) error {
	if strings.Contains(script, "add chain inet fvpn "+deadlineGuard) {
		// Deliberately ignore context cancellation: the safety property must
		// survive a command that commits too late, even without a cleanup.
		time.Sleep(f.delay)
	}
	if script == "delete chain inet fvpn "+deadlineGuard+"\n" {
		f.guardRelease = true
	}
	return f.fakeExec.NftBatch(ctx, script)
}

func TestD104_AT13_InstallOverrunNeverOpensGuard(t *testing.T) {
	m := smallModel()
	f := &delayedDeadlineExec{fakeExec: servingExec(m), delay: DeadlineInstallBound + 10*time.Millisecond}
	res, err := Apply(t.Context(), f, m, nil)
	if err == nil || res.Verified || f.guardRelease || res.Deadline.GuardReleased {
		t.Fatalf("late install opened: %+v %v", res.Deadline, err)
	}
	if res.Deadline.InstalledBy-res.Deadline.RenderStart <= DeadlineInstallBound {
		t.Fatal("fixture did not exceed bound")
	}
	if !strings.Contains(f.nft[0], "iifname \"fv-fh\" drop") || !strings.Contains(f.nft[0], "oifname \"fv-fh\" drop") {
		t.Fatal("candidate was not atomically guarded in both directions")
	}
}
