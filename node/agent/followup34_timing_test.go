package agent

import (
	"context"
	"fmt"
	"net/netip"
	"slices"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
)

// TEST ONLY: isolates the durable work added immediately before Apply's first
// instruction. This is disk write latency, not a kernel/AT timing claim.
func BenchmarkF34_ConntrackReservation(b *testing.B) {
	for _, n := range []int{1, 64} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			store, err := agentstore.Open(b.TempDir() + "/agent.db")
			if err != nil {
				b.Fatal(err)
			}
			defer store.Close()
			e := &Engine{store: store, storageOK: true}
			m := enforce.Model{}
			for i := 0; i < n; i++ {
				addr := netip.AddrFrom4([4]byte{100, 64, 1, byte(i + 1)})
				e.addresses = append(e.addresses, addrID{Addr: addr})
				m.Revoke = append(m.Revoke, addr)
			}
			var samples []int64
			for b.Loop() {
				// Repeated independent TEST ONLY batches, never a production reset.
				e.conntrackAttempts %= maxConntrackDeleteAttempts
				start := time.Now()
				if err := e.reserveConntrack(context.Background(), m); err != nil {
					b.Fatal(err)
				}
				samples = append(samples, time.Since(start).Nanoseconds())
			}
			slices.Sort(samples)
			for _, p := range []int{50, 90, 99, 100} {
				b.ReportMetric(float64(samples[(len(samples)*p+99)/100-1]), fmt.Sprintf("p%d-ns", p))
			}
		})
	}
}
