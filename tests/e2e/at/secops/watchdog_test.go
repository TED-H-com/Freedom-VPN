//go:build e2e

package secops

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
	"github.com/google/uuid"
)

type observedEnforcer struct {
	calls   int
	present bool
	err     error
}

func (e *observedEnforcer) Baseline(context.Context, enforce.Datapath) error { e.calls++; return nil }
func (e *observedEnforcer) BaselinePresent(context.Context) (bool, error)    { return e.present, e.err }

// No kernel commands or host clock changes: this is the independent watchdog
// decision boundary. Actual TCP/UDP stopping remains an explicit lab prerequisite.
func TestAT29_IndependentWatchdogFailureMatrix(t *testing.T) {
	boot := uuid.New()
	mono := 10 * time.Minute
	base := watchdog.Beat{BootID: boot, Mono: mono - time.Second, Wall: time.Now().UTC(), PolicyExpires: mono + 20*time.Second, BaselineOK: true, Serving: true, Generation: 9}
	for _, tc := range []struct {
		name                                            string
		edit                                            func(*watchdog.Beat)
		missing, clockError, absent, inspectError, stop bool
	}{
		{name: "valid_policy_control_unavailable"},
		{name: "agent_crashed", edit: func(b *watchdog.Beat) { b.Mono = mono - 31*time.Second }, stop: true},
		{name: "exact_policy_deadline", edit: func(b *watchdog.Beat) { b.PolicyExpires = mono }, stop: true},
		{name: "boot_changed", edit: func(b *watchdog.Beat) { b.BootID = uuid.New() }, stop: true},
		{name: "elapsed_clock_regressed", edit: func(b *watchdog.Beat) { b.Mono = mono + time.Second }, stop: true},
		{name: "wall_clock_back_one_hour", edit: func(b *watchdog.Beat) { b.Wall = b.Wall.Add(-time.Hour); b.PolicyExpires = mono }, stop: true},
		{name: "missing_deadline", edit: func(b *watchdog.Beat) { b.PolicyExpires = 0 }, stop: true},
		{name: "failed_isolation", edit: func(b *watchdog.Beat) { b.BaselineOK = false }, stop: true},
		{name: "missing_heartbeat", missing: true, stop: true},
		{name: "missing_time_anchor", clockError: true, stop: true},
		{name: "missing_kernel_table", absent: true, stop: true},
		{name: "unreadable_kernel_table", inspectError: true, stop: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			beat := base
			if tc.edit != nil {
				tc.edit(&beat)
			}
			path := filepath.Join(t.TempDir(), "heartbeat.json")
			if !tc.missing {
				if err := watchdog.WriteHeartbeat(path, beat); err != nil {
					t.Fatal(err)
				}
			}
			enforcer := &observedEnforcer{present: !tc.absent}
			if tc.inspectError {
				enforcer.err = errors.New("TEST ONLY inspection failure")
			}
			for range 2 { // Reconstruct the process; it must not refresh any deadline.
				w := watchdog.Watch{BootID: boot, Enforce: enforcer,
					Config: watchdog.Config{HeartbeatPath: path, WatchdogStaleSeconds: 30, Datapath: watchdog.DatapathConfig{MemberIngress: "fvmb0", Uplink: "fvup0", SNAT: "192.0.2.10"}},
					NowMono: func() (time.Duration, error) {
						if tc.clockError {
							return 0, errors.New("TEST ONLY missing clock")
						}
						return mono, nil
					},
				}
				if err := w.Once(t.Context()); err != nil {
					t.Fatal(err)
				}
			}
			want := 0
			if tc.stop {
				want = 2
			}
			if enforcer.calls != want {
				t.Fatalf("baseline calls=%d want=%d", enforcer.calls, want)
			}
			if !tc.missing {
				persisted, err := watchdog.ReadHeartbeat(path)
				if err != nil || persisted.PolicyExpires != beat.PolicyExpires {
					t.Fatal("watchdog restart rewrote the deadline")
				}
			}
		})
	}
}

func TestAT29_EarliestFiniteServiceDeadline(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct {
		name                         string
		service, node, offline, want int
	}{
		{"paid_service_ends_first", 10, 30, 60, 10},
		{"finite_grace_node_ends_first", 60, 30, 90, 30},
		{"finite_grant_offline_cap_first", 90, 120, 40, 40},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lease := enforce.MemberLease{MemberID: uuid.New(), EffectiveServiceEnd: now.Add(time.Duration(tc.service) * time.Second), NodeLeaseEnd: now.Add(time.Duration(tc.node) * time.Second), SignedServerTime: now, MaxOfflineSeconds: uint64(tc.offline)}
			got, ok := enforce.ForwardingDeadline(lease)
			if !ok || !got.Equal(now.Add(time.Duration(tc.want)*time.Second)) {
				t.Fatal("earliest deadline not enforced")
			}
			lease.EffectiveServiceEnd = time.Time{}
			if _, ok := enforce.ForwardingDeadline(lease); ok {
				t.Fatal("missing entitlement deadline accepted")
			}
		})
	}
}
