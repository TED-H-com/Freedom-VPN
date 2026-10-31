package nodestest

import (
	"sync/atomic"

	"github.com/TED-H-com/Freedom-VPN/internal/leases"
	"github.com/jackc/pgx/v5"
)

// MemberGrantEligibility is a TEST ONLY, process-local D-087 switch. Each instance
// controls all nodes using its factory; use a dedicated fixture runtime. The zero
// value withholds grants. NewMemberGrantEligibility starts with production reads.
type MemberGrantEligibility struct{ ready atomic.Bool }

func NewMemberGrantEligibility() *MemberGrantEligibility {
	g := &MemberGrantEligibility{}
	g.SetMemberGrantsReady(true)
	return g
}

// SetMemberGrantsReady changes only future renewals. It neither revokes nor
// rewrites issued leases, and a replay still returns its original signed lease.
func (g *MemberGrantEligibility) SetMemberGrantsReady(ready bool) { g.ready.Store(ready) }

// ReaderFactory is for httpwire.WithTestOnlyGrantEligibilityReaderFactory.
// Sample once so both the initial read and the pre-commit recheck agree. Returning
// the concrete production reader preserves the lease service's FOR SHARE path.
func (g *MemberGrantEligibility) ReaderFactory(_ pgx.Tx) leases.EntitlementReader {
	if g.ready.Load() {
		return leases.DBEntitlementReader{}
	}
	return leases.NewFakeEntitlementReader()
}
