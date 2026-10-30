package updater

import (
	"errors"
	"net/netip"
	"slices"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
)

// A fixed error keeps peer IDs, addresses, destination lists and malformed
// config values out of diagnostics (SEC02).
var errProbePeer = errors.New("updater: invalid probe_peer")

// probeRefusal carries only a fixed reason class to the logger. The public
// error text is identical for every refusal, including address-source failures.
type probeRefusal string

func (probeRefusal) Error() string { return errProbePeer.Error() }
func (probeRefusal) Unwrap() error { return errProbePeer }

type probeAddresses struct {
	node, control, resolvers []netip.Addr
}

func validateProbePeer(p *healthattest.ChallengeProbe, known probeAddresses) error {
	if p == nil {
		return nil
	}
	if strings.TrimSpace(p.PeerID) == "" || len(p.PeerID) > 128 ||
		strings.ContainsFunc(p.PeerID, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
		return probeRefusal("malformed")
	}
	source, ok := probeHostV4(p.SourceV4)
	if !ok {
		return probeRefusal("malformed")
	}
	if slices.Contains(known.node, source) {
		return probeRefusal("node_source")
	}
	if len(p.Destinations) < 1 || len(p.Destinations) > 4 {
		return probeRefusal("malformed")
	}
	denied := enforce.SpecDeniedV4()
	for _, d := range p.Destinations {
		addr, ok := probeHostV4(d.AddrV4)
		if !ok || d.Port == 0 || (d.Proto != "tcp" && d.Proto != "udp") {
			return probeRefusal("malformed")
		}
		for _, prefix := range denied {
			if prefix.Contains(addr) {
				return probeRefusal("denied_destination")
			}
		}
		switch {
		case slices.Contains(known.node, addr):
			return probeRefusal("node_destination")
		case slices.Contains(known.control, addr):
			return probeRefusal("control_destination")
		case slices.Contains(known.resolvers, addr):
			return probeRefusal("resolver_destination")
		}
	}
	return nil
}

func probeHostV4(raw string) (netip.Addr, bool) {
	p, err := netip.ParsePrefix(raw)
	if err != nil || !p.Addr().Is4() || p.Bits() != 32 {
		return netip.Addr{}, false
	}
	a := p.Addr()
	return a, a.IsGlobalUnicast() && !a.IsLoopback() && !a.IsLinkLocalUnicast() && a.As4()[0] != 0 && a.As4()[0] < 224
}

func cloneProbePeer(p *healthattest.ChallengeProbe) *healthattest.ChallengeProbe {
	if p == nil {
		return nil
	}
	copy := *p
	copy.Destinations = append([]healthattest.ChallengeProbeDest(nil), p.Destinations...)
	return &copy
}
