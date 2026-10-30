package config

import (
	"errors"
	"net/netip"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// D-058: the optional signed probe names exact TCP destinations. It cannot
// introduce DNS lookups, address ranges or scoped interface exceptions (§13.9).
func validateNodeEgressProbe(p *api.EgressProbe) error {
	if p == nil {
		return nil
	}
	if len(p.Targets) < 1 || len(p.Targets) > 4 || (p.TimeoutMs != nil && (*p.TimeoutMs < 100 || *p.TimeoutMs > 10000)) {
		return errors.New("invalid node egress_probe targets or timeout_ms")
	}
	seen := map[netip.AddrPort]bool{}
	for _, target := range p.Targets {
		addr, err := netip.ParseAddr(target.Address)
		if err != nil || addr.Zone() != "" || target.Protocol != api.EgressProbeTargetProtocolTcp || target.Port < 1 || target.Port > 65535 {
			return errors.New("invalid node egress_probe target")
		}
		endpoint := netip.AddrPortFrom(addr.Unmap(), uint16(target.Port))
		if seen[endpoint] {
			return errors.New("duplicate node egress_probe target")
		}
		seen[endpoint] = true
	}
	return nil
}
