package agent

import (
	"context"
	"net/netip"
	"slices"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// servingPresenceExecutor checks completeness on the same live nft snapshot
// the serving verifier consumes. It issues no extra commands or mutations.
// enforce's lifetime check bounds elements present but does not require them.
type servingPresenceExecutor struct {
	enforce.Executor
	model    enforce.Model
	snapshot []byte
}

func (x *servingPresenceExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	raw, err := x.Executor.Output(ctx, name, args...)
	if err == nil && name == "nft" {
		if slices.Equal(args, []string{"-j", "list", "ruleset"}) {
			x.snapshot = raw
		}
		if slices.Equal(args, []string{"list", "set", "inet", "fvpn", "allowed_v4"}) {
			// This read is reached only after the existing nft ruleset checks pass.
			// Preserve their failure precedence (especially generation repair), and
			// let a missing element fail the same measured deadline-verification gate.
			at, clockErr := enforce.MonotonicNow()
			if clockErr != nil {
				return nil, clockErr
			}
			if err := verifyAllowedPresence(x.snapshot, x.model, at); err != nil {
				return nil, err
			}
		}
	}
	return raw, err
}

func verifyAllowedPresence(raw []byte, m enforce.Model, at time.Duration) error {
	rs, err := enforce.ParseRuleset(raw)
	if err != nil {
		return err
	}
	now := m.Now
	if m.NowMono > 0 {
		if at < m.NowMono {
			return &enforce.CheckError{Reason: enforce.ReasonIsolation, Detail: "monotonic presence read precedes model"}
		}
		now = now.Add(at - m.NowMono)
	}
	present := map[netip.Addr]bool{}
	for _, el := range rs.Elements["allowed_v4"] {
		present[el.Addr] = true
	}
	for _, b := range m.Bindings {
		if !b.ValidFrom.IsZero() && now.Before(b.ValidFrom) {
			continue
		}
		for _, l := range m.Leases {
			if l.MemberID != b.MemberID {
				continue
			}
			end, ok := enforce.ForwardingDeadline(l)
			if !ok {
				continue
			} // the full serving verifier rejects invalid leases
			if !b.ValidUntil.IsZero() && b.ValidUntil.Before(end) {
				end = b.ValidUntil
			}
			remaining := end.Sub(now)
			if l.DeadlineMono != nil {
				remaining = min(remaining, *l.DeadlineMono-at)
			}
			if l.TimeoutCeilSeconds > 0 && m.NowMono > 0 && l.TimeoutCeilSeconds <= uint64((1<<63-1)/int64(time.Second)) {
				remaining = min(remaining, time.Duration(l.TimeoutCeilSeconds)*time.Second-(at-m.NowMono))
			}
			// A legitimate D-104 element can disappear this much before D. Never
			// call the existing safety reserve a drift or extend its signed window.
			if remaining <= enforce.DeadlineInstallBound+enforce.DeadlineGranularity {
				continue
			}
			for _, prefix := range b.AllowedV4 {
				if !present[prefix.Addr()] {
					return &enforce.CheckError{Reason: enforce.ReasonIsolation, Detail: "allowed_v4 is missing a live binding"}
				}
			}
		}
	}
	return nil
}
