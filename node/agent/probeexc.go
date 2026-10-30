package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

// Keep this agent-side bound aligned with enforce.ProbeMaxDestinations.
const maxProbeTargets = 4

// probeIssuance survives process restarts. The monotonic deadline belongs to
// one boot and can only shrink. Cleared issuances cannot be opened again.
type probeIssuance struct {
	Nonce    string        `json:"nonce"`
	Hash     string        `json:"hash"`
	IssuedAt time.Time     `json:"issued_at"`
	Deadline time.Duration `json:"deadline_ns,string"`
	Cleared  bool          `json:"cleared"`
}

const probeIssuanceKey = "probe_issuance"

// withProbeException binds the result to the exact challenge read before the
// probe, even when a free update lock means probing the normal node state.
func (e *Engine) withProbeException(ctx context.Context, run func()) {
	ch, ok := e.matchingChallenge()
	e.probedChallenge = nil
	if ok {
		e.probedChallenge = &ch
	}
	if !ok || ch.Probe == nil || (e.healthNonce == ch.Nonce && e.probeFresh(ch)) {
		run()
		return
	}
	clear, installed := e.installProbeException(ctx, ch)
	if installed {
		defer clear()
	}
	run()
}

// updateLockHeld never waits for or keeps the lock. Only contention is
// authority for D-064; a missing or inaccessible lock is not an update.
func (e *Engine) updateLockHeld() bool {
	lock, err := nodelock.TryAcquire(e.cfg.LockPath)
	if err == nil {
		lock.Release()
		return false
	}
	return errors.Is(err, nodelock.ErrBusy)
}

func (e *Engine) installProbeException(ctx context.Context, ch healthattest.Challenge) (func(), bool) {
	e.prepareProbe(ctx)
	live, ok := e.matchingChallenge()
	if !ok || !live.Same(ch) || !e.updateLockHeld() || e.dependencyMissing ||
		(e.probeChallenge != nil && e.probeChallenge.Nonce == ch.Nonce) {
		return nil, false
	}
	now := e.trustedNow()
	_, ok = e.challengeProbe(ch, now)
	if !ok {
		return nil, false
	}
	// Store a nonce-bound digest, never the probe's destination list (SEC02).
	// Same compares timestamp instants; normalize their JSON spelling too.
	ch.IssuedAt = ch.IssuedAt.UTC()
	ch.ExpiresAt = ch.ExpiresAt.UTC()
	raw, err := canonjson.Marshal(ch)
	if err != nil {
		return nil, false
	}
	sum := sha256.Sum256(raw)
	issuance := probeIssuance{Nonce: ch.Nonce, Hash: hex.EncodeToString(sum[:]), IssuedAt: ch.IssuedAt,
		Deadline: e.clock.Mono() + ch.ExpiresAt.Sub(now)}
	previous, exists, err := e.store.Meta(ctx, probeIssuanceKey)
	if err != nil {
		e.storageOK = false
		return nil, false
	}
	if exists {
		var old probeIssuance
		if json.Unmarshal([]byte(previous), &old) != nil {
			e.storageOK = false
			return nil, false
		}
		if old.Nonce == ch.Nonce {
			if old.Hash != issuance.Hash || old.Cleared || old.Deadline <= e.clock.Mono() {
				return nil, false
			}
			if old.Deadline < issuance.Deadline {
				issuance.Deadline = old.Deadline
			}
		} else if !ch.IssuedAt.After(old.IssuedAt) {
			return nil, false
		}
	}
	// Persist before touching the kernel, including partial apply failures.
	if !e.saveProbeIssuance(ctx, issuance) {
		return nil, false
	}
	// Re-check time and authority after the durable write. Neither elapsed
	// time nor a new anchor may increase the original kernel lifetime.
	now = e.trustedNow()
	exc, ok := e.challengeProbe(ch, now)
	live, matches := e.matchingChallenge()
	if !ok || !matches || !live.Same(ch) || !e.updateLockHeld() {
		return nil, false
	}
	remain := issuance.Deadline - e.clock.Mono()
	if remain < time.Second {
		return nil, false
	}
	if capSeconds := uint64(remain / time.Second); capSeconds < exc.TimeoutCeilSeconds {
		exc.TimeoutCeilSeconds = capSeconds
	}
	_, ok = e.baselineProbeModel(now, exc)
	if !ok {
		return nil, false
	}
	e.probeChallenge = &ch
	// D-064: even an agent that was serving enters the baseline. No probe
	// installation or removal may call the serving-policy Apply path.
	if e.serving {
		e.withdraw(ctx)
	}
	e.serving = false
	e.expectServing.Store(false)
	e.policyExpires = 0
	// Handoff can take time. Compute the actual install cap after it, using
	// the same trusted clock and the durable original monotonic deadline.
	now = e.trustedNow()
	exc, ok = e.challengeProbe(ch, now)
	live, matches = e.matchingChallenge()
	remain = issuance.Deadline - e.clock.Mono()
	if !ok || !matches || !live.Same(ch) || !e.updateLockHeld() || remain < time.Second {
		return nil, false
	}
	exc.TimeoutCeilSeconds = min(exc.TimeoutCeilSeconds, uint64(remain/time.Second))
	model, ok := e.baselineProbeModel(now, exc)
	if !ok {
		return nil, false
	}
	clear := func() {
		bare := model
		bare.Probe = nil
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := e.plane.BaselineModel(cleanupCtx, bare); err != nil {
			e.baselineOK = false
			e.lastProbe.Check = api.HealthCheckStatusFailed
			e.lastProbe.Health = api.DataHealthFailed
			e.lastProbe.Detail = "probe_clear_failed"
			e.log.Warn("probe exception", "err", "not removed")
		}
		issuance.Cleared = true
		e.saveProbeIssuance(cleanupCtx, issuance)
	}
	if err := e.plane.BaselineModel(ctx, model); err != nil {
		clear()
		e.log.Warn("probe exception", "err", "not installed")
		return nil, false
	}
	e.baselineOK = true
	return clear, true
}

func (e *Engine) saveProbeIssuance(ctx context.Context, issuance probeIssuance) bool {
	raw, err := json.Marshal(issuance)
	if err == nil {
		err = e.store.SetMeta(ctx, probeIssuanceKey, string(raw))
	}
	if err != nil {
		e.storageOK = false
		return false
	}
	return true
}

// prepareProbe refreshes the serving gates before a probe can touch the
// plane. Losing a gate first withdraws members onto the baseline.
func (e *Engine) prepareProbe(ctx context.Context) {
	e.latch()
	e.decision = Decide(e.input())
	if e.serving && !e.decision.Serve {
		e.withdraw(ctx)
		e.decision = Decide(e.input())
	}
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
	if !e.anchorOK || req == nil || !probePeerOK(req.PeerID) || len(req.Destinations) > maxProbeTargets {
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
		if e.memberCollision(req.PeerID, p.Addr()) || e.nodeLocal(p.Addr()) {
			return nil, false
		}
		for _, denied := range enforce.SpecDeniedV4() {
			if denied.Contains(p.Addr()) {
				return nil, false
			}
		}
		dests = append(dests, enforce.ProbeDestination{AddrV4: p, Proto: d.Proto, Port: d.Port})
	}
	if len(dests) == 0 && !req.ResolverDNS {
		return nil, false
	}
	if ch.IssuedAt.IsZero() || ch.ExpiresAt.IsZero() || now.IsZero() || !now.After(ch.IssuedAt) {
		return nil, false
	}
	exp := ch.ExpiresAt
	capAt := ch.IssuedAt.Add(time.Duration(enforce.ProbeMaxSeconds) * time.Second)
	if exp.After(capAt) {
		return nil, false
	}
	if !exp.After(now) || exp.Sub(now) < time.Second {
		return nil, false
	}
	return &enforce.ProbeException{
		PeerID:             req.PeerID,
		SourceV4:           src,
		Destinations:       dests,
		ResolverDNS:        req.ResolverDNS,
		ExpiresAt:          exp.UTC(),
		TimeoutCeilSeconds: uint64(exp.Sub(now) / time.Second),
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
