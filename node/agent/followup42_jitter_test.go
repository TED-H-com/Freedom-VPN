package agent

// TEST ONLY injected random bytes; deterministic distribution and lease tests.
import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/google/uuid"
)

func f42Entropy(values ...uint16) *bytes.Reader {
	var b bytes.Buffer
	for _, v := range values {
		var pair [2]byte
		binary.BigEndian.PutUint16(pair[:], v)
		b.Write(pair[:])
	}
	return bytes.NewReader(b.Bytes())
}

func TestAT13_F32_01_UniformMillisecondRenewalJitter(t *testing.T) {
	for _, tc := range []struct {
		name string
		lead time.Duration
		draw uint16
		want time.Duration
	}{
		{"default_min", 120 * time.Second, 0, 108 * time.Second},
		{"default_mid", 120 * time.Second, 12000, 120 * time.Second},
		{"default_max", 120 * time.Second, 24000, 132 * time.Second},
		{"capped_min", 30 * time.Second, 0, 27 * time.Second},
		{"capped_mid", 30 * time.Second, 3000, 30 * time.Second},
		{"capped_max", 30 * time.Second, 6000, 33 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.lead + leadJitter(tc.lead, f42Entropy(tc.draw)); got != tc.want {
				t.Fatal(got, tc.want)
			}
		})
	}
	// Every point in the default span maps to its own millisecond; high bytes
	// are significant. The last draw also proves rejection, not modulo bias.
	for n := uint16(0); n <= 24000; n++ {
		if got := leadJitter(120*time.Second, f42Entropy(n)); got != time.Duration(n)*time.Millisecond-12*time.Second {
			t.Fatal(n, got)
		}
	}
	source := f42Entropy(32767, 24000)
	if got := leadJitter(120*time.Second, source); got != 12*time.Second || source.Len() != 0 {
		t.Fatal("biased/non-rejecting mapping", got)
	}
}

func TestAT13_F32_01_RenewalDrawOncePerLease(t *testing.T) {
	start := time.Hour
	window := time.Hour
	e := renewFixture(1, 1, start+window-120*time.Second, start, time.Now().UTC(), window)
	source := f42Entropy(0, 24000)
	e.renewRandom = source
	e.lease.LeaseId = uuid.Must(uuid.NewV7())
	for range 20 {
		if e.needRenew() {
			t.Fatal("minimum lead renewed too early")
		}
	}
	if source.Len() != 2 {
		t.Fatal("redrew same lease", source.Len())
	}
	e.lease.LeaseId = uuid.Must(uuid.NewV7())
	for range 20 {
		if !e.needRenew() {
			t.Fatal("maximum lead did not renew")
		}
	}
	if source.Len() != 0 {
		t.Fatal("next lease did not draw")
	}
}

func TestAT13_F32_01_FailedRandomReadKeepsSafeLead(t *testing.T) {
	for _, partial := range [][]byte{nil, {0}} {
		if got := leadJitter(120*time.Second, bytes.NewReader(partial)); got != 0 {
			t.Fatal("random error changed base", got)
		}
	}
	e := renewFixture(1, 1, 2*time.Hour-120*time.Second, time.Hour, time.Now().UTC(), time.Hour)
	e.renewRandom = bytes.NewReader(nil)
	if !e.needRenew() || e.leaseJitter.value != 0 {
		t.Fatal("failed draw did not use base")
	}
	e.renewRandom = f42Entropy(0)
	if !e.needRenew() {
		t.Fatal("failed draw was retried on same lease")
	}
}

func TestAT13_F32_01_LeaseReplayAndRestartKeepDraw(t *testing.T) {
	r := f42Siblings(t, false)
	r.eng.renewRandom = f42Entropy(0)
	r.clock.Advance(time.Second)
	r.eng.leaseRefreshPending = true
	stepOK(t, r)
	stored, ok, err := r.eng.store.Lease(t.Context())
	if err != nil || !ok {
		t.Fatal(err)
	}
	if stored.RenewalJitter != -12*time.Second {
		t.Fatal("lease did not persist draw", stored.RenewalJitter)
	}
	r.eng.renewRandom = f42Entropy(24000)
	if err := r.eng.acceptLease(t.Context(), stored.Envelope, r.clock.Mono(), r.clock.Wall()); err != nil {
		t.Fatal(err)
	}
	if r.eng.leaseJitter.value != stored.RenewalJitter {
		t.Fatal("replay redrew")
	}
	r20Crash(r.eng)
	next, err := Start(Options{Config: r.eng.cfg, Clock: r.clock, Plane: r.plane, MeterSource: staticSource{boot: r.eng.bootID.String()}, Prober: passProbe{}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	if next.leaseJitter.value != stored.RenewalJitter || next.leaseJitter.id != stored.ID {
		t.Fatal("restart redrew")
	}
	// Explicit config change clamps the saved offset against the new base.
	next.cfg.RenewLeadSeconds = 10
	if got := next.renewalJitter(time.Hour); got != -time.Second {
		t.Fatal("new cap ignored", got)
	}
}
