package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// reviewProbe is a same-uid challenge body. Addresses are parser fixtures:
// nothing in these tests dials them.
func reviewProbe(source string, dests ...string) *healthattest.ChallengeProbe {
	p := &healthattest.ChallengeProbe{PeerID: "peer-probe", SourceV4: source, ResolverDNS: true}
	for _, d := range dests {
		p.Destinations = append(p.Destinations, healthattest.ChallengeProbeDest{AddrV4: d, Proto: "tcp", Port: 443})
	}
	return p
}

func serveBoundGeneration(t *testing.T, r *rig) {
	t.Helper()
	r.srv.StickyLease(false)
	r.eng.prober = &verProbe{version: "0.76.2"}
	enrollBoot(t, r)
	stepOK(t, r)
	r.srv.SetFixture(generationFixture(2, oneBinding(time.Now().UTC())))
	stepOK(t, r)
	if !r.eng.serving || r.eng.appliedGen != 2 || len(r.plane.last.Bindings) == 0 {
		t.Fatalf("setup serving %v applied %d bindings %d reasons %v", r.eng.serving, r.eng.appliedGen, len(r.plane.last.Bindings), r.eng.decision.Reasons)
	}
}

// TestReviewR11_ServingProbeIgnoresUpdateLock is the case the update lock used
// to prevent: generation N is applied and serving, the updater holds the lock
// and has installed the baseline, then a live challenge arrives.
func TestReviewR11_ServingProbeIgnoresUpdateLock(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	lock, err := nodelock.TryAcquire(r.eng.cfg.LockPath)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	dp, err := r.eng.cfg.DatapathValue()
	if err != nil {
		t.Fatal(err)
	}
	if err := r.plane.Baseline(context.Background(), dp); err != nil {
		t.Fatal(err)
	}
	if len(r.plane.last.Bindings) != 0 || r.plane.last.Generation != 0 {
		t.Fatalf("updater baseline did not clear member forwarding: gen %d bindings %d", r.plane.last.Generation, len(r.plane.last.Bindings))
	}
	before := len(r.plane.calls)
	appliesBefore := r.plane.applies
	ch := probeChallenge(t, r, labProbe(), 0)
	stepOK(t, r)
	if other, err := nodelock.TryAcquire(r.eng.cfg.LockPath); !errors.Is(err, nodelock.ErrBusy) {
		if other != nil {
			other.Release()
		}
		t.Fatalf("probe released the updater lock: %v", err)
	}
	var install, cleared *enforce.Model
	for _, c := range r.plane.calls[before:] {
		if c.kind != "baseline_model" {
			continue
		}
		m := c.model
		if m.Probe != nil && install == nil {
			install = &m
			continue
		}
		if m.Probe == nil && install != nil && cleared == nil {
			cleared = &m
		}
	}
	busyBefore := hasCode(r.eng.decision.Errors, api.ErrorCodeUPDATEBUSY)
	appliesAtProbe := r.plane.applies
	if err := r.eng.enforce(context.Background(), Decide(r.eng.input())); err != nil {
		t.Fatal(err)
	}
	if r.plane.applies != appliesAtProbe || !hasCode(r.eng.decision.Errors, api.ErrorCodeUPDATEBUSY) {
		t.Fatalf("enforce itself applied under the lock: applies %d -> %d errors %+v", appliesAtProbe, r.plane.applies, r.eng.decision.Errors)
	}
	restored := install != nil && cleared != nil && install.Generation == 2 && len(install.Bindings) > 0 && cleared.Generation == 2 && len(cleared.Bindings) > 0 && cleared.Probe == nil
	if restored || install == nil || cleared == nil || install.Generation != 0 || cleared.Generation != 0 || len(install.Bindings) != 0 || len(cleared.Bindings) != 0 || r.plane.applies != appliesBefore || r.eng.serving || !busyBefore {
		t.Fatalf("lock-held probe must install and clear only the baseline: install=%+v clear=%+v applies=%d -> %d busy=%v", install, cleared, appliesBefore, r.plane.applies, busyBefore)
	}
	requireAttest(t, r, ch)
}

func TestReviewR11_SameUIDChallengeOpensProbeHole(t *testing.T) {
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	enrollBoot(t, r)
	stepOK(t, r)
	if r.eng.desired == nil {
		t.Fatal("no desired document")
	}
	r.eng.desired.Dns.ResolverIpv4 = []string{"10.0.0.53"}
	r.eng.desired.DeviceBindings = []api.DeviceBinding{{
		PeerId: "peer-a1", AllowedIpv4: []string{"100.64.1.1/32"},
	}}
	// RFC1918 source, four destinations the render does not deny, plus DNS to
	// a resolver inside 10.0.0.0/8. TxnID is one this process minted.
	p := reviewProbe("10.9.8.7/32", "1.1.1.1/32", "8.8.8.8/32", "192.88.99.1/32", "192.31.196.1/32")
	if !probePeerOK("peer-a1") {
		t.Fatal("probePeerOK rejected a member peer id shape")
	}
	ch := probeChallenge(t, r, p, 0)
	info, err := os.Lstat(filepath.Join(r.eng.cfg.JournalDir, healthattest.ChallengeFile))
	if err != nil {
		t.Fatal(err)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || uint64(st.Uid) != uint64(os.Geteuid()) || info.Mode().Perm() != 0o600 {
		t.Fatalf("challenge writer uid %d euid %d mode %o", st.Uid, os.Geteuid(), info.Mode().Perm())
	}
	if _, parsed := uuid.Parse(ch.TxnID); parsed != nil {
		t.Fatal(parsed)
	}
	before := len(r.plane.calls)
	clear, installedOK := r.eng.installProbeException(context.Background(), ch)
	if installedOK {
		clear()
		t.Fatal("same-euid writer installed a probe while the update lock was free")
	}
	if len(r.plane.calls) != before {
		t.Fatal("refused probe changed the plane")
	}
	// During an update the same body is constrained by the agent's guards.
	// Private overlay sources are allowed; resolver ordering belongs to enforce.
	holdUpdateLock(t, r)
	clear, installedOK = r.eng.installProbeException(context.Background(), ch)
	if !installedOK {
		t.Fatal("valid lock-held private overlay source was refused")
	}
	defer clear()
	var installed *enforce.Model
	for _, c := range r.plane.calls[before:] {
		if c.kind == "baseline_model" && c.model.Probe != nil {
			m := c.model
			installed = &m
			break
		}
	}
	if installed == nil {
		t.Fatal("proof inconclusive: same-uid challenge was not installed")
	}
	script, err := enforce.RenderBaseline(*installed, enforce.State{})
	if err != nil {
		t.Fatal(err)
	}
	rules := script[strings.Index(script, "add rule"):]
	dns := strings.Index(rules, "ip daddr @resolver_v4 tcp dport 53")
	deny := strings.Index(rules, "ip daddr @denied_v4")
	opened := installed.Probe.SourceV4.String() == "10.9.8.7/32" &&
		len(installed.Probe.Destinations) == 4 &&
		installed.Probe.ResolverDNS &&
		strings.Contains(script, "10.9.8.7 timeout") &&
		strings.Contains(script, "192.88.99.1") &&
		strings.Contains(script, "1.1.1.1") &&
		strings.Contains(script, "8.8.8.8") &&
		strings.Contains(script, "192.31.196.1") &&
		dns >= 0 && deny >= 0 && dns < deny &&
		strings.Contains(script, "10.0.0.53")
	if !opened || installed.Generation != 0 || len(installed.Bindings) != 0 || installed.Probe.TimeoutCeilSeconds > uint64(ch.ExpiresAt.Sub(r.eng.trustedNow())/time.Second) {
		t.Fatalf("lock-held baseline probe changed the requested bounds: %+v", installed.Probe)
	}
	t.Logf("lock-held baseline sends private overlay /32, four exact TCP/443 sockets, resolver_dns=true; DNS rule position=%d deny=%d", dns, deny)
}

func TestReviewR11_ProbePeerCollidesOnlyWithCurrentDocument(t *testing.T) {
	r := newRig(t)
	enrollBoot(t, r)
	stepOK(t, r)
	now := r.eng.trustedNow()
	member := labProbe()
	member.PeerID = "peer-a1"
	member.SourceV4 = "100.64.1.1/32"
	if _, ok := r.eng.challengeProbe(healthattest.Challenge{IssuedAt: now.Add(-time.Second), ExpiresAt: now.Add(30 * time.Second), Probe: member}, now); !ok {
		t.Fatal("member /32 absent from the current document was refused")
	}
	r.eng.desired = &api.DesiredState{DeviceBindings: []api.DeviceBinding{{
		PeerId: "peer-a1", AllowedIpv4: []string{"100.64.1.1/32"},
	}}}
	if _, ok := r.eng.challengeProbe(healthattest.Challenge{IssuedAt: now.Add(-time.Second), ExpiresAt: now.Add(30 * time.Second), Probe: member}, now); ok {
		t.Fatal("current member peer id and allowed address were accepted")
	}
	other := labProbe()
	other.PeerID = "peer-a1"
	if _, ok := r.eng.challengeProbe(healthattest.Challenge{IssuedAt: now.Add(-time.Second), ExpiresAt: now.Add(30 * time.Second), Probe: other}, now); ok {
		t.Fatal("probe peer id equal to a current member was accepted")
	}
	// A member who is not in this document is not a collision.
	r.eng.desired.DeviceBindings = nil
	if !probePeerOK("peer-a1") {
		t.Fatal("probePeerOK knows member ids")
	}
	exc, ok := r.eng.challengeProbe(healthattest.Challenge{IssuedAt: now.Add(-time.Second), ExpiresAt: now.Add(30 * time.Second), Probe: other}, now)
	if !ok || exc.PeerID != "peer-a1" {
		t.Fatal("member peer id absent from the current document was refused", ok)
	}
	own := labProbe()
	own.SourceV4 = "192.0.2.37/32"
	if _, ok := r.eng.challengeProbe(healthattest.Challenge{IssuedAt: now.Add(-time.Second), ExpiresAt: now.Add(30 * time.Second), Probe: own}, now); ok {
		t.Fatal("node SNAT was accepted as the probe source")
	}
	meta := labProbe()
	meta.Destinations[0].AddrV4 = "169.254.169.254/32"
	if _, ok := r.eng.challengeProbe(healthattest.Challenge{IssuedAt: now.Add(-time.Second), ExpiresAt: now.Add(30 * time.Second), Probe: meta}, now); ok {
		t.Fatal("metadata address was accepted")
	}
	wide := labProbe()
	wide.Destinations[0].AddrV4 = "1.2.3.0/24"
	if _, ok := r.eng.challengeProbe(healthattest.Challenge{IssuedAt: now.Add(-time.Second), ExpiresAt: now.Add(30 * time.Second), Probe: wide}, now); ok {
		t.Fatal("destination wider than /32 was accepted")
	}
	r.eng.desired.DeviceBindings = []api.DeviceBinding{{PeerId: "peer-a1", AllowedIpv4: []string{"100.64.1.1/32"}}}
	overlap := labProbe()
	overlap.Destinations[0].AddrV4 = "100.64.1.1/32"
	if _, ok := r.eng.challengeProbe(healthattest.Challenge{IssuedAt: now.Add(-time.Second), ExpiresAt: now.Add(30 * time.Second), Probe: overlap}, now); ok {
		t.Fatal("destination inside a member AllowedV4 was accepted")
	}
	r.eng.cfg.HostV4 = []string{"9.9.9.9/32"}
	local := labProbe()
	local.Destinations[0].AddrV4 = "9.9.9.9/32"
	local.Destinations[0].Port = 9
	_, ok = r.eng.challengeProbe(healthattest.Challenge{IssuedAt: now.Add(-time.Second), ExpiresAt: now.Add(30 * time.Second), Probe: local}, now)
	if ok {
		t.Fatal("node-local destination on a non-admin port passed the agent check")
	}
}

func TestReviewR11_TrustedNowInstallsWhenWallRefusesAnswer(t *testing.T) {
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	enrollBoot(t, r)
	stepOK(t, r)
	holdUpdateLock(t, r)
	wall := r.clock.Wall().UTC()
	r.eng.anchor = timeanchor.Anchor{
		BootID: r.eng.bootID, Server: wall.Add(30 * time.Second), Mono: r.clock.Mono(), Wall: wall, Trusted: true,
	}
	r.eng.anchorOK = true
	skewed := r.eng.trustedNow()
	if !skewed.After(wall.Add(20*time.Second)) || !skewed.Before(wall.Add(40*time.Second)) {
		t.Fatalf("trustedNow %s wall %s", skewed, wall)
	}
	ch := issueChallenge(t, r, func(in *healthattest.Input) {
		in.Now = wall
		in.TTL = 45 * time.Second
		in.Probe = labProbe()
	})
	before := len(r.plane.calls)
	stepOK(t, r)
	installed := false
	for _, c := range r.plane.calls[before:] {
		if c.kind == "baseline_model" && c.model.Probe != nil {
			installed = true
		}
	}
	_, answeredErr := os.Lstat(attestPath(r))
	answered := answeredErr == nil
	if installed && !answered {
		t.Fatalf("trustedNow %s installed the probe for challenge issued %s expires %s, and wall %s wrote no attestation", skewed, ch.IssuedAt, ch.ExpiresAt, wall)
	}
	if !installed && answered {
		t.Fatal("answered a probe challenge without installing it")
	}
	if !installed || !answered {
		t.Fatalf("trusted clock did not install and answer: installed=%v answered=%v", installed, answered)
	}
	att, _, err := healthattest.ReadAttestation(r.eng.cfg.JournalDir)
	if err != nil || !att.GeneratedAt.Equal(skewed) {
		t.Fatalf("answer clock=%s want=%s err=%v", att.GeneratedAt, skewed, err)
	}
}

func TestReviewR11_WallAnswersDoNotSkipProbeInstall(t *testing.T) {
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	enrollBoot(t, r)
	stepOK(t, r)
	holdUpdateLock(t, r)
	wall := r.clock.Wall().UTC()
	// trustedNow sits before issued_at, so the exception is refused, while wall
	// is inside the challenge window.
	r.eng.anchor = timeanchor.Anchor{
		BootID: r.eng.bootID, Server: wall.Add(-30 * time.Second), Mono: r.clock.Mono(), Wall: wall, Trusted: true,
	}
	r.eng.anchorOK = true
	ch := issueChallenge(t, r, func(in *healthattest.Input) {
		in.Now = wall.Add(-10 * time.Second)
		in.TTL = 45 * time.Second
		in.Probe = labProbe()
	})
	before := len(r.plane.calls)
	stepOK(t, r)
	for _, c := range r.plane.calls[before:] {
		if c.model.Probe != nil {
			t.Fatal("installed a probe trustedNow does not consider issued")
		}
	}
	if _, err := os.Lstat(attestPath(r)); err == nil {
		att := requireAttest(t, r, ch)
		t.Fatalf("answered without an install: %+v", att.Checks.TunnelProbe)
	}
}

func TestReviewR11_ChallengeSameFields(t *testing.T) {
	r := newRig(t)
	enrollBoot(t, r)
	ch := issueChallenge(t, r, func(in *healthattest.Input) { in.Probe = labProbe() })
	if !ch.Same(ch) {
		t.Fatal("challenge is not Same as itself")
	}
	mutants := []struct {
		name string
		edit func(*healthattest.Challenge)
	}{
		{"txn", func(c *healthattest.Challenge) { c.TxnID = uuid.Must(uuid.NewV7()).String() }},
		{"nonce", func(c *healthattest.Challenge) { c.Nonce = strings.Repeat("ab", 32) }},
		{"release", func(c *healthattest.Challenge) { c.ReleaseID = uuid.Must(uuid.NewV7()).String() }},
		{"node", func(c *healthattest.Challenge) { c.NodeID = uuid.Must(uuid.NewV7()).String() }},
		{"incarnation", func(c *healthattest.Challenge) { c.IncarnationID = uuid.Must(uuid.NewV7()).String() }},
		{"boot", func(c *healthattest.Challenge) { c.BootID = uuid.Must(uuid.NewV7()).String() }},
		{"issued", func(c *healthattest.Challenge) { c.IssuedAt = c.IssuedAt.Add(time.Second) }},
		{"expires", func(c *healthattest.Challenge) { c.ExpiresAt = c.ExpiresAt.Add(time.Second) }},
		{"component", func(c *healthattest.Challenge) { c.Components.NodeAgent = "other" }},
		{"consumed", func(c *healthattest.Challenge) { c.Consumed = true }},
		{"peer", func(c *healthattest.Challenge) { c.Probe.PeerID = "other-peer" }},
		{"source", func(c *healthattest.Challenge) { c.Probe.SourceV4 = "10.1.2.3/32" }},
		{"dns", func(c *healthattest.Challenge) { c.Probe.ResolverDNS = !c.Probe.ResolverDNS }},
		{"dest", func(c *healthattest.Challenge) { c.Probe.Destinations[0].AddrV4 = "9.9.9.9/32" }},
	}
	for _, tc := range mutants {
		t.Run(tc.name, func(t *testing.T) {
			edited := ch
			edited.Probe = &healthattest.ChallengeProbe{}
			*edited.Probe = *ch.Probe
			edited.Probe.Destinations = append([]healthattest.ChallengeProbeDest(nil), ch.Probe.Destinations...)
			tc.edit(&edited)
			if edited.Same(ch) {
				t.Fatalf("Same ignored %s", tc.name)
			}
		})
	}
}

func TestReviewR11_RewriteDuringProbeDoesNotAttach(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*healthattest.Challenge)
	}{
		{"expiry", func(c *healthattest.Challenge) { c.ExpiresAt = c.ExpiresAt.Add(time.Second) }},
		{"release", func(c *healthattest.Challenge) { c.ReleaseID = uuid.Must(uuid.NewV7()).String() }},
		{"probe", func(c *healthattest.Challenge) {
			c.Probe = labProbe()
			c.Probe.Destinations[0].AddrV4 = "9.9.9.9/32"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t)
			snap := &snapProbe{verProbe: verProbe{version: "0.76.2"}}
			r.eng.prober = snap
			r.srv.SetPublication(api.PublicationIntentUnpublished)
			enrollBoot(t, r)
			stepOK(t, r)
			holdUpdateLock(t, r)
			ch := probeChallenge(t, r, labProbe(), 0)
			snap.during = func() {
				edited := ch
				edited.Probe = &healthattest.ChallengeProbe{}
				*edited.Probe = *ch.Probe
				edited.Probe.Destinations = append([]healthattest.ChallengeProbeDest(nil), ch.Probe.Destinations...)
				tc.edit(&edited)
				if err := healthattest.WriteChallenge(r.eng.cfg.JournalDir, edited); err != nil {
					t.Fatal(err)
				}
			}
			stepOK(t, r)
			if _, err := os.Lstat(attestPath(r)); err == nil {
				t.Fatalf("old probe result attached after %s rewrite", tc.name)
			}
		})
	}
}

func TestReviewR11_IssuanceWindowAndRenderCap(t *testing.T) {
	r := newRig(t)
	enrollBoot(t, r)
	stepOK(t, r)
	now := r.eng.trustedNow()
	issued := now.Add(-time.Nanosecond)
	expires := issued.Add(time.Duration(enforce.ProbeMaxSeconds) * time.Second)
	exc, ok := r.eng.challengeProbe(healthattest.Challenge{IssuedAt: issued, ExpiresAt: expires, Probe: labProbe()}, now)
	if !ok {
		t.Fatal("expires_at = issued_at + 300s was refused")
	}
	if exc.TimeoutCeilSeconds > enforce.ProbeMaxSeconds {
		t.Fatalf("ceil %d exceeds ProbeMaxSeconds", exc.TimeoutCeilSeconds)
	}
	model, built := r.eng.baselineProbeModel(now, exc)
	if !built {
		t.Fatal("300s window did not render")
	}
	script, err := enforce.RenderBaseline(model, enforce.State{})
	if err != nil {
		t.Fatal(err)
	}
	want := "timeout " + itoaU64(exc.TimeoutCeilSeconds) + "s"
	if !strings.Contains(script, want) || strings.Contains(script, "timeout 301s") {
		t.Fatalf("rendered timeout is not the ceil %d:\n%s", exc.TimeoutCeilSeconds, script)
	}
	if _, ok := r.eng.challengeProbe(healthattest.Challenge{IssuedAt: now, ExpiresAt: now.Add(45 * time.Second), Probe: labProbe()}, now); ok {
		t.Fatal("issued_at = now was accepted")
	}
	ch, err := healthattest.Issue(healthattest.Input{
		TxnID: uuid.Must(uuid.NewV7()).String(), ReleaseID: r.eng.cfg.ReleaseUUID().String(),
		NodeID: r.eng.nodeID.String(), IncarnationID: r.eng.incID.String(), BootID: r.eng.bootID.String(),
		Now: now, TTL: 300 * time.Second, Probe: labProbe(),
		Components: healthattest.Components{NodeAgent: "a", Netbird: "b", NodeUpdater: "c", Enforcement: "d"},
	})
	if err != nil || !ch.ExpiresAt.Equal(ch.IssuedAt.Add(300*time.Second)) {
		t.Fatal("issuer refused a 300s probe", err)
	}
	if _, err := healthattest.Issue(healthattest.Input{
		TxnID: uuid.Must(uuid.NewV7()).String(), ReleaseID: r.eng.cfg.ReleaseUUID().String(),
		NodeID: r.eng.nodeID.String(), IncarnationID: r.eng.incID.String(), BootID: r.eng.bootID.String(),
		Now: now, TTL: 300*time.Second + time.Nanosecond, Probe: labProbe(),
		Components: healthattest.Components{NodeAgent: "a", Netbird: "b", NodeUpdater: "c", Enforcement: "d"},
	}); err == nil {
		t.Fatal("issuer accepted a probe TTL past 300s")
	}
}

func itoaU64(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func TestReviewR11_RestartRemainCapAndAnomalyExtension(t *testing.T) {
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	enrollBoot(t, r)
	stepOK(t, r)
	wall := r.clock.Wall()
	r.eng.anchor = timeanchor.Anchor{BootID: r.eng.bootID, Server: wall, Mono: r.clock.Mono(), Wall: wall, Trusted: true}
	r.eng.anchorOK = true
	now := r.eng.trustedNow()
	issued := now.Add(-100 * time.Second)
	expires := issued.Add(300 * time.Second)
	ch := healthattest.Challenge{IssuedAt: issued, ExpiresAt: expires, Probe: labProbe()}
	first, ok := r.eng.challengeProbe(ch, now)
	if !ok || first.TimeoutCeilSeconds != 200 {
		t.Fatalf("first ceil %+v ok %v", first, ok)
	}
	r.clock.Advance(80 * time.Second)
	later := r.eng.trustedNow()
	second, ok := r.eng.challengeProbe(ch, later)
	if !ok || second.TimeoutCeilSeconds != 120 {
		t.Fatalf("honest restart ceil %+v later %s now %s", second, later, now)
	}
	honest, built := r.eng.baselineProbeModel(later, second)
	if !built {
		t.Fatal("honest restart did not render")
	}
	live := enforce.State{Sets: map[string]bool{"probe_v4": true, "probe_dst_v4": true}, ProbeRemain: 120}
	script, err := enforce.RenderBaseline(honest, live)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(script, "timeout 120s") || strings.Contains(script, "timeout 200s") {
		t.Fatalf("ProbeRemain did not cap the second render:\n%s", script)
	}
	// An unreadable remaining lifetime still receives only the trusted window.
	unreadable := enforce.State{Sets: map[string]bool{"probe_v4": true, "probe_dst_v4": true}, ProbeRemain: 0}
	script, err = enforce.RenderBaseline(honest, unreadable)
	if err != nil || !strings.Contains(script, "timeout 120s") || strings.Contains(script, "timeout 200s") {
		t.Fatalf("restart with ProbeRemain 0 extended expiry: %v\n%s", err, script)
	}
	// Kernel time has reached `later`. The wall clock steps back to `now`,
	// which is more than the 2s anomaly limit, so trustedNow falls back to it.
	r.clock.JumpWall(-80 * time.Second)
	fallen := r.eng.trustedNow()
	if !fallen.Before(later.Add(-time.Minute)) {
		t.Fatalf("anomaly did not fall back to the stepped-back wall: trusted %s later %s", fallen, later)
	}
	r.eng.probeChallenge = nil
	if inflated, ok := r.eng.challengeProbe(ch, fallen); ok {
		t.Fatalf("untrusted backward clock accepted an inflated window: %+v", inflated)
	}
}

func TestReviewR11_ClearFailureNextStep(t *testing.T) {
	r := newRig(t)
	fp := &failAfterProbePlane{failClear: true}
	r.eng.plane = fp
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	enrollBoot(t, r)
	stepOK(t, r)
	holdUpdateLock(t, r)
	ch := probeChallenge(t, r, labProbe(), 0)
	callsBefore := len(fp.calls)
	stepOK(t, r)
	att := requireAttest(t, r, ch)
	if r.eng.lastProbe.Detail != "probe_clear_failed" || att.Checks.TunnelProbe.Status != api.HealthCheckStatusFailed || att.Checks.TunnelProbe.Reason != "probe_clear_failed" {
		t.Fatalf("failed step was not honest: probe %+v tunnel %+v baselineOK %v", r.eng.lastProbe, att.Checks.TunnelProbe, r.eng.baselineOK)
	}
	if !r.eng.baselineOK {
		t.Fatal("same-step baseline did not restore baselineOK")
	}
	sawBaseline := false
	for _, c := range fp.calls[callsBefore:] {
		if c.kind == "baseline" {
			sawBaseline = true
		}
		if c.kind == "apply" {
			t.Fatal("unpublished cleanup failure applied the serving model")
		}
	}
	if !sawBaseline {
		t.Fatal("next enforce path did not baseline after probe_clear_failed")
	}
	state := enforce.State{Sets: map[string]bool{"probe_v4": true, "probe_dst_v4": true}}
	dp, err := r.eng.cfg.DatapathValue()
	if err != nil {
		t.Fatal(err)
	}
	script, err := enforce.RenderBaseline(enforce.Model{Datapath: dp}, state)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(script, "delete set inet fvpn probe_v4") || !strings.Contains(script, "delete set inet fvpn probe_dst_v4") {
		t.Fatalf("production baseline left the probe sets:\n%s", script)
	}
	stepOK(t, r)
	if countKind(fp.calls, "baseline_model") != 1 {
		t.Fatalf("same challenge installed again: %+v", fp.calls)
	}
	next := requireAttest(t, r, ch)
	if next.Checks.TunnelProbe.Status != api.HealthCheckStatusPassed || next.Checks.TunnelProbe.Reason != "overlay_up" {
		t.Fatalf("next attestation did not move on to the new probe: %+v", next.Checks.TunnelProbe)
	}
	if r.eng.lastProbe.Detail == "probe_clear_failed" {
		t.Fatal("next step kept the cleanup failure as the current probe result")
	}
}

func TestReviewR11_StaleGenerationLeasePressure(t *testing.T) {
	r := newRig(t)
	enrollBoot(t, r)
	stepOK(t, r)
	if !r.eng.serving || uint64(r.eng.lease.ConfigGeneration) != 1 {
		t.Fatalf("setup serving %v lease %s", r.eng.serving, r.eng.lease.ConfigGeneration)
	}
	// Sticky lease keeps returning generation 1 after the desired generation moves.
	r.srv.SetFixture(generationFixture(2, oneBinding(time.Now().UTC())))
	stepOK(t, r)
	if r.eng.serving || r.eng.appliedGen != 2 || uint64(r.eng.lease.ConfigGeneration) == 2 {
		t.Fatalf("setup did not stick on the old lease: serving %v applied %d lease %s", r.eng.serving, r.eng.appliedGen, r.eng.lease.ConfigGeneration)
	}
	before := r.srv.LeaseRenewals()
	const steps = 6
	start := time.Now()
	for range steps {
		stepOK(t, r)
	}
	elapsed := time.Since(start)
	got := r.srv.LeaseRenewals() - before
	if got < steps {
		t.Fatalf("renewals %d over %d steps", got, steps)
	}
	if elapsed >= time.Duration(steps)*time.Second {
		t.Fatalf("renewal loop waited %s, which is enough to be a Retry-After backoff", elapsed)
	}
	if r.eng.serving || hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonLeaseExpired) || !hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonPolicyMissing) {
		t.Fatalf("serving %v reasons %v", r.eng.serving, r.eng.decision.Reasons)
	}
	beat := mustBeat(t, r)
	if beat.Serving || beat.Reason != string(api.ForwardingBlockReasonPolicyMissing) || strings.Contains(beat.Reason, "lease_expired") {
		t.Fatalf("heartbeat %+v", beat)
	}
	rep, ok := r.srv.LastObservation()
	if !ok {
		t.Fatal("no observation")
	}
	for _, reason := range rep.ForwardingBlockReasons {
		if reason == api.ForwardingBlockReasonLeaseExpired {
			t.Fatal("observation reported lease_expired for a lease of another generation")
		}
	}
	for _, check := range rep.Checks {
		if check.Name != api.NodeCheckNameLease {
			continue
		}
		if check.Detail != nil && strings.Contains(*check.Detail, "lease_expired") {
			t.Fatalf("lease check detail %q", *check.Detail)
		}
		if code, err := check.ErrorCode.Get(); err == nil && code == api.ErrorCodeLEASEEXPIRED {
			t.Fatalf("lease check %+v", check)
		}
	}
	t.Logf("renewals %d in %s (%d steps); serving stayed off with policy_missing", got, elapsed, steps)
}
