package agent

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TEST ONLY: pure renderer deadlines, not a kernel timing measurement.
func TestAT13_F41_DeviceDeadlinesInEitherOrder(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	for _, reverse := range []bool{false, true} {
		for _, limit := range []time.Duration{2 * time.Minute, 20 * time.Minute, 2 * time.Hour} {
			in := r.eng.input()
			ds, lease := *in.Desired, *in.Lease
			in.Desired, in.Lease = &ds, &lease
			start := lease.IssuedAt
			long := oneBinding(start)
			long.ValidUntil = start.Add(time.Hour)
			short := long
			short.DeviceId = uuid.Must(uuid.NewV7())
			short.AllowedIpv4 = []string{"100.64.0.12/32"}
			short.ValidUntil = start.Add(5 * time.Minute)
			ds.DeviceBindings = []api.DeviceBinding{long, short}
			if reverse {
				ds.DeviceBindings = []api.DeviceBinding{short, long}
			}
			ds.ExpiresAt = start.Add(3 * time.Hour)
			lease.ExpiresAt = start.Add(3 * time.Hour)
			lease.MaxOfflineSeconds = 10800
			lease.Members = []api.MemberLease{{MemberId: long.MemberId, LeaseEnd: start.Add(limit), EffectiveServiceEnd: start.Add(3 * time.Hour)}}
			in.MonoNow = in.LeaseMonoSent
			m, expires, err := buildModel(in)
			if err != nil {
				t.Fatal(err)
			}
			plan, err := enforce.Render(m, enforce.State{}, enforce.TCState{})
			if err != nil {
				t.Fatal(err)
			}
			for _, b := range ds.DeviceBindings {
				want := min(limit, b.ValidUntil.Sub(start))
				if got := plan.ElementTimeouts[b.DeviceId.String()]; got != want {
					t.Fatalf("reverse=%v lease=%s device=%s timeout=%s want=%s", reverse, limit, b.DeviceId, got, want)
				}
			}
			if len(m.Leases) != 1 || *m.Leases[0].DeadlineMono != in.LeaseMonoSent+limit || !m.Leases[0].EffectiveServiceEnd.Equal(start.Add(limit)) || !m.Leases[0].NodeLeaseEnd.Equal(start.Add(limit)) {
				t.Fatalf("member limits depend on device order: %+v", m.Leases)
			}
			if want := in.LeaseMonoSent + min(limit, 5*time.Minute); expires != want {
				t.Fatalf("watchdog=%s want=%s", expires, want)
			}
		}
	}
}
