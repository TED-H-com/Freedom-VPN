package agent

import (
	"context"
	"net/netip"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
)

// withProbeException installs the §13.9 exception for the probe that answers
// a challenge, then removes it with the same plane call and a nil probe.
// A challenge that does not name a probe, or that names one this process
// will not render, runs the probe with no exception. The Apply result is
// discarded: probe_up and probe_down are diagnostic counters and are not
// copied into the agent counter map or member usage. A crash skips the
// clear and leaves the kernel timeout.
func (e *Engine) withProbeException(ctx context.Context, run func()) {
	ch, ok := e.matchingChallenge()
	if !ok || ch.Probe == nil {
		run()
		return
	}
	clear, installed := e.installProbeException(ctx, ch)
	if installed {
		defer clear()
	}
	run()
}

func (e *Engine) installProbeException(ctx context.Context, ch healthattest.Challenge) (func(), bool) {
	now := e.trustedNow()
	exc, ok := e.challengeProbe(ch, now)
	if !ok {
		e.log.Warn("probe exception", "err", "not installed")
		return nil, false
	}
	serving := e.serving && e.decision.Serve && e.decision.Model.Generation != 0
	var model enforce.Model
	if serving {
		model = e.decision.Model
		model.Now = now
		model.Probe = exc
		if _, err := enforce.Render(model, enforce.State{}, enforce.TCState{}); err != nil {
			e.log.Warn("probe exception", "err", "not installed")
			return nil, false
		}
	} else {
		var built bool
		model, built = e.baselineProbeModel(now, exc)
		if !built {
			e.log.Warn("probe exception", "err", "not installed")
			return nil, false
		}
	}
	lock, err := nodelock.Acquire(ctx, e.cfg.LockPath, 3, 50*time.Millisecond)
	if err != nil {
		e.log.Warn("probe exception", "err", "not installed")
		return nil, false
	}
	if err := e.applyProbeModel(ctx, serving, model); err != nil {
		lock.Release()
		e.log.Warn("probe exception", "err", "not installed")
		return nil, false
	}
	return func() {
		defer lock.Release()
		bare := model
		bare.Probe = nil
		if err := e.applyProbeModel(ctx, serving, bare); err != nil {
			e.log.Warn("probe exception", "err", "not removed")
		}
	}, true
}

// applyProbeModel calls Apply while serving and ApplyBaselineModel otherwise.
// It does not store counters. ApplyBaseline is left to the watchdog and to
// the steady fail-closed path.
func (e *Engine) applyProbeModel(ctx context.Context, serving bool, m enforce.Model) error {
	if serving {
		prev := map[string]enforce.CounterValue{}
		for name, c := range e.counters {
			if name == "probe_up" || name == "probe_down" {
				continue
			}
			prev[name] = c
		}
		_, err := e.plane.Apply(ctx, m, prev)
		return err
	}
	return e.plane.BaselineModel(ctx, m)
}

func (e *Engine) baselineProbeModel(now time.Time, p *enforce.ProbeException) (enforce.Model, bool) {
	dp, err := e.cfg.DatapathValue()
	if err != nil {
		return enforce.Model{}, false
	}
	host, _ := parsePrefixes(e.cfg.HostV4)
	extra, _ := parsePrefixes(e.cfg.ExtraDeniedV4)
	exempt, _ := parsePrefixes(e.cfg.TestExemptV4)
	m := enforce.Model{
		IPv6Mode:      enforce.IPv6Block,
		Now:           now,
		Datapath:      dp,
		Resolvers:     e.resolverAddrs(),
		ExtraDeniedV4: extra,
		HostV4:        host,
		Probe:         p,
	}
	if e.cfg.TestMode && len(exempt) > 0 {
		m.DeniedV4 = enforce.Without(enforce.SpecDeniedV4(), exempt...)
	}
	if _, err := enforce.RenderBaseline(m, enforce.State{}); err != nil {
		return enforce.Model{}, false
	}
	return m, true
}

func (e *Engine) resolverAddrs() []netip.Addr {
	if e.desired == nil {
		return nil
	}
	out := make([]netip.Addr, 0, len(e.desired.Dns.ResolverIpv4))
	for _, s := range e.desired.Dns.ResolverIpv4 {
		a, err := netip.ParseAddr(s)
		if err != nil || !a.Is4() {
			continue
		}
		out = append(out, a)
	}
	return out
}

// challengeProbe builds the exception the challenge asked for. It returns
// false instead of widening a prefix, substituting a peer, or omitting expiry.
func (e *Engine) challengeProbe(ch healthattest.Challenge, now time.Time) (*enforce.ProbeException, bool) {
	req := ch.Probe
	if req == nil || !probePeerOK(req.PeerID) {
		return nil, false
	}
	src, err := netip.ParsePrefix(strings.TrimSpace(req.SourceV4))
	if err != nil || !src.IsValid() || !src.Addr().Is4() || src.Bits() != 32 {
		return nil, false
	}
	src = src.Masked()
	if unusableV4(src.Addr()) || e.nodeLocal(src.Addr()) || e.memberCollision(req.PeerID, src.Addr()) {
		return nil, false
	}
	dests := make([]enforce.ProbeDestination, 0, len(req.Destinations))
	for _, d := range req.Destinations {
		p, err := netip.ParsePrefix(strings.TrimSpace(d.AddrV4))
		if err != nil || !p.IsValid() || !p.Addr().Is4() || p.Bits() != 32 {
			return nil, false
		}
		p = p.Masked()
		if (d.Proto != "tcp" && d.Proto != "udp") || d.Port == 0 || unusableV4(p.Addr()) {
			return nil, false
		}
		if e.nodeLocal(p.Addr()) && probeAdminPort(d.Port) {
			return nil, false
		}
		dests = append(dests, enforce.ProbeDestination{AddrV4: p, Proto: d.Proto, Port: d.Port})
	}
	if len(dests) == 0 && !req.ResolverDNS {
		return nil, false
	}
	if ch.ExpiresAt.IsZero() || now.IsZero() {
		return nil, false
	}
	exp := ch.ExpiresAt
	capAt := now.Add(time.Duration(enforce.ProbeMaxSeconds) * time.Second)
	if exp.After(capAt) {
		exp = capAt
	}
	if !exp.After(now) || exp.Sub(now) < time.Second {
		return nil, false
	}
	return &enforce.ProbeException{
		PeerID:       req.PeerID,
		SourceV4:     src,
		Destinations: dests,
		ResolverDNS:  req.ResolverDNS,
		ExpiresAt:    exp.UTC(),
	}, true
}

func probePeerOK(id string) bool {
	return id != "" && len(id) <= 128 && !strings.ContainsAny(id, "\r\n")
}

func unusableV4(a netip.Addr) bool {
	return !a.IsValid() || !a.Is4() || a.IsUnspecified() || a.IsMulticast() || a.IsLoopback() || a.IsLinkLocalUnicast()
}

func (e *Engine) nodeLocal(addr netip.Addr) bool {
	dp, err := e.cfg.DatapathValue()
	if err == nil && dp.SNAT.IsValid() && addr == dp.SNAT {
		return true
	}
	host, err := parsePrefixes(e.cfg.HostV4)
	if err != nil {
		return false
	}
	for _, p := range host {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

func (e *Engine) memberCollision(peer string, src netip.Addr) bool {
	if e.desired != nil {
		for _, b := range e.desired.DeviceBindings {
			if b.PeerId != "" && b.PeerId == peer {
				return true
			}
			for _, s := range b.AllowedIpv4 {
				p, err := hostPrefix(s)
				if err == nil && p.Contains(src) {
					return true
				}
			}
		}
	}
	for _, b := range e.decision.Model.Bindings {
		if b.PeerID != "" && b.PeerID == peer {
			return true
		}
		for _, p := range b.AllowedV4 {
			if p.IsValid() && p.Contains(src) {
				return true
			}
		}
	}
	return false
}

func probeAdminPort(port uint16) bool {
	switch port {
	case 22, 80, 443, 3000, 8080, 9091:
		return true
	default:
		return false
	}
}
