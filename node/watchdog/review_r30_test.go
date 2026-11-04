package watchdog

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestR30_D156_WatchdogAdoptionPendingInvariants verifies D-156's watchdog
// baselineDecision invariants: a valid same-boot boot_adopting receipt
// preserves live rules while within WatchdogStaleSeconds and PolicyExpires with
// BaselinePresent == true, and fails closed on every malformed, cross-boot,
// stale, expired, or missing-baseline variation.
func TestR30_D156_WatchdogAdoptionPendingInvariants(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "heartbeat")
	boot := uuid.Must(uuid.NewV7())
	now := 20 * time.Minute

	valid := Beat{
		BootID:        boot,
		Mono:          now - 5*time.Second,
		Wall:          time.Unix(1_700_000_000, 0).UTC(),
		PolicyExpires: now + 10*time.Minute,
		BaselineOK:    false,
		Serving:       false,
		Generation:    7,
		Reason:        AdoptionPending,
	}
	if err := WriteHeartbeat(path, valid); err != nil {
		t.Fatal(err)
	}
	enf := &fakeEnforcer{present: 1}
	w := &Watch{
		Config: Config{
			HeartbeatPath:        path,
			WatchdogStaleSeconds: 30,
			Datapath:             DatapathConfig{MemberIngress: "fv-fh", Uplink: "fv-ih", SNAT: "192.0.2.37"},
		},
		Enforce: enf,
		BootID:  boot,
		NowMono: func() (time.Duration, error) { return now, nil },
	}
	if err := w.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if enf.calls != 0 {
		t.Fatalf("valid boot_adopting triggered %d baseline writes, want 0", enf.calls)
	}

	cases := []struct {
		name    string
		mutate  func(*Beat, *Watch, *fakeEnforcer)
		wantErr bool
	}{
		{
			name: "zero_generation_rejected",
			mutate: func(b *Beat, _ *Watch, _ *fakeEnforcer) {
				b.Generation = 0
			},
		},
		{
			name: "zero_policy_expires_rejected",
			mutate: func(b *Beat, _ *Watch, _ *fakeEnforcer) {
				b.PolicyExpires = 0
			},
		},
		{
			name: "expired_policy_rejected",
			mutate: func(b *Beat, _ *Watch, _ *fakeEnforcer) {
				b.PolicyExpires = now
			},
		},
		{
			name: "stale_mono_rejected",
			mutate: func(b *Beat, _ *Watch, _ *fakeEnforcer) {
				b.Mono = now - 31*time.Second
			},
		},
		{
			name: "serving_true_with_baseline_false_rejected",
			mutate: func(b *Beat, _ *Watch, _ *fakeEnforcer) {
				b.Serving = true
			},
		},
		{
			name: "baseline_ok_true_with_adoption_pending_rejected",
			mutate: func(b *Beat, _ *Watch, _ *fakeEnforcer) {
				b.BaselineOK = true
			},
		},
		{
			name: "nil_watchdog_boot_id_rejected",
			mutate: func(_ *Beat, w *Watch, _ *fakeEnforcer) {
				w.BootID = uuid.Nil
			},
		},
		{
			name: "different_boot_id_rejected",
			mutate: func(b *Beat, _ *Watch, _ *fakeEnforcer) {
				b.BootID = uuid.Must(uuid.NewV7())
			},
		},
		{
			name: "missing_kernel_baseline_rejected",
			mutate: func(_ *Beat, _ *Watch, e *fakeEnforcer) {
				e.present = 0
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := valid
			enf := &fakeEnforcer{present: 1}
			w := &Watch{
				Config: Config{
					HeartbeatPath:        path,
					WatchdogStaleSeconds: 30,
					Datapath:             DatapathConfig{MemberIngress: "fv-fh", Uplink: "fv-ih", SNAT: "192.0.2.37"},
				},
				Enforce: enf,
				BootID:  boot,
				NowMono: func() (time.Duration, error) { return now, nil },
			}
			tc.mutate(&b, w, enf)
			if err := WriteHeartbeat(path, b); err != nil {
				t.Fatal(err)
			}
			if err := w.Once(context.Background()); err != nil {
				t.Fatal(err)
			}
			if enf.calls != 1 {
				t.Fatalf("%s: baseline calls = %d, want 1", tc.name, enf.calls)
			}
		})
	}
}
