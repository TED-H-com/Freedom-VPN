package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/fakecontrol"
	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func labProbe() *healthattest.ChallengeProbe {
	return &healthattest.ChallengeProbe{
		PeerID:   "peer-probe",
		SourceV4: "100.64.8.1/32",
		Destinations: []healthattest.ChallengeProbeDest{{
			AddrV4: "1.2.3.4/32", Proto: "tcp", Port: 80,
		}},
	}
}

func probeChallenge(t *testing.T, r *rig, p *healthattest.ChallengeProbe, ttl time.Duration) healthattest.Challenge {
	t.Helper()
	ch := issueChallenge(t, r, func(in *healthattest.Input) {
		if ttl > 0 {
			in.TTL = ttl
		}
	})
	ch.Probe = p
	if err := healthattest.WriteChallenge(r.eng.cfg.JournalDir, ch); err != nil {
		t.Fatal(err)
	}
	return ch
}

type snapProbe struct {
	verProbe
	during func()
}

func (p *snapProbe) Probe(ctx context.Context, req ProbeRequest) ProbeResult {
	if p.during != nil {
		p.during()
	}
	return p.verProbe.Probe(ctx, req)
}

func countKind(calls []planeCall, kind string) int {
	n := 0
	for _, c := range calls {
		if c.kind == kind {
			n++
		}
	}
	return n
}

func TestProbeException_BaselineThenClear(t *testing.T) {
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	enrollBoot(t, r)
	stepOK(t, r)
	if r.eng.serving {
		t.Fatal("unpublished node served")
	}
	if countKind(r.plane.calls, "baseline_model") != 0 {
		t.Fatal("baseline model ran before a probe challenge")
	}
	before := len(r.plane.calls)
	ch := probeChallenge(t, r, labProbe(), 0)
	stepOK(t, r)
	att := requireAttest(t, r, ch)
	if att.Checks.TunnelProbe.Status != api.HealthCheckStatusPassed {
		t.Fatalf("tunnel %+v", att.Checks.TunnelProbe)
	}
	var probed, cleared *enforce.Model
	for _, c := range r.plane.calls[before:] {
		if c.kind != "baseline_model" {
			continue
		}
		m := c.model
		if m.Probe != nil && probed == nil {
			probed = &m
			continue
		}
		if m.Probe == nil && probed != nil && cleared == nil {
			cleared = &m
		}
	}
	if probed == nil || cleared == nil {
		t.Fatalf("baseline calls missing probe then clear: %+v", r.plane.calls[before:])
	}
	if probed.Probe.PeerID != "peer-probe" || probed.Probe.SourceV4.String() != "100.64.8.1/32" {
		t.Fatalf("probe %+v", probed.Probe)
	}
	if probed.Probe.ExpiresAt.IsZero() || len(probed.Probe.Destinations) != 1 || probed.Probe.Destinations[0].AddrV4.Bits() != 32 || probed.Probe.Destinations[0].Port != 80 {
		t.Fatalf("probe %+v", probed.Probe)
	}
	now := probed.Now
	if probed.Probe.ExpiresAt.After(ch.ExpiresAt) || probed.Probe.ExpiresAt.After(now.Add(time.Duration(enforce.ProbeMaxSeconds)*time.Second)) || probed.Probe.ExpiresAt.Sub(now) < time.Second {
		t.Fatalf("expiry %s now %s deadline %s", probed.Probe.ExpiresAt, now, ch.ExpiresAt)
	}
	if cleared.Datapath != probed.Datapath || !cleared.Now.Equal(probed.Now) {
		t.Fatal("clear used a different baseline model")
	}
	script, err := enforce.RenderBaseline(*probed, enforce.State{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(script, "100.64.8.1 timeout") || !strings.Contains(script, "1.2.3.4 . tcp . 80") || !strings.Contains(script, `counter name "probe_up"`) {
		t.Fatalf("rendered exception missing:\n%s", script)
	}
	clearScript, err := enforce.RenderBaseline(*cleared, enforce.State{})
	if err != nil {
		t.Fatal(err)
	}
	rules := clearScript[strings.Index(clearScript, "add rule"):]
	if strings.Contains(rules, "probe_up") || strings.Contains(rules, " accept") {
		t.Fatalf("cleared baseline still accepts:\n%s", rules)
	}
	if countKind(r.plane.calls[before:], "baseline") < 1 {
		t.Fatal("steady path did not keep ApplyBaseline")
	}
	raw, err := r.eng.store.Counters(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for name := range raw {
		if name == "probe_up" || name == "probe_down" {
			t.Fatalf("stored diagnostic counter %s", name)
		}
	}
}

func TestProbeException_ServingApplyThenClear(t *testing.T) {
	r := newRig(t)
	snap := &snapProbe{verProbe: verProbe{version: "0.76.2"}}
	r.eng.prober = snap
	enrollBoot(t, r)
	stepOK(t, r)
	if !r.eng.serving {
		t.Fatalf("not serving: %v", r.eng.decision.Reasons)
	}
	snap.during = func() {
		if r.plane.last.Probe == nil || r.plane.last.Probe.PeerID != "peer-probe" {
			t.Fatal("probe ran without the serving exception")
		}
		for name := range r.eng.counters {
			if name == "probe_up" || name == "probe_down" {
				t.Fatalf("counter %s was copied during the probe", name)
			}
		}
	}
	before := len(r.plane.calls)
	ch := probeChallenge(t, r, labProbe(), 0)
	stepOK(t, r)
	requireAttest(t, r, ch)
	sawProbe, sawClear := false, false
	for _, c := range r.plane.calls[before:] {
		if c.kind != "apply" {
			t.Fatalf("serving check used %s", c.kind)
		}
		if c.model.Probe != nil {
			sawProbe = true
			if c.model.Probe.SourceV4.Bits() != 32 || c.model.Probe.ExpiresAt.IsZero() || c.model.Probe.ExpiresAt.After(ch.ExpiresAt) {
				t.Fatalf("serving probe %+v", c.model.Probe)
			}
		} else if sawProbe {
			sawClear = true
		}
	}
	if !sawProbe || !sawClear || r.plane.last.Probe != nil {
		t.Fatalf("steady model kept a probe (saw %v %v last %v)", sawProbe, sawClear, r.plane.last.Probe != nil)
	}
	samples, err := enforce.Samples(r.plane.last, map[string]enforce.CounterValue{
		"probe_up":   {Packets: 3, Bytes: 30},
		"probe_down": {Packets: 2, Bytes: 20},
	})
	if err != nil || len(samples) != 0 {
		t.Fatalf("probe counters entered usage samples: %+v %v", samples, err)
	}
}

func TestProbeException_AbsentChallengeDoesNotInstall(t *testing.T) {
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	enrollBoot(t, r)
	stepOK(t, r)
	ch := issueChallenge(t, r, nil)
	n := countKind(r.plane.calls, "baseline_model")
	applies := r.plane.applies
	stepOK(t, r)
	requireAttest(t, r, ch)
	if countKind(r.plane.calls, "baseline_model") != n || r.plane.applies != applies {
		t.Fatalf("challenge without a probe installed one: models %d applies %d -> %d", countKind(r.plane.calls, "baseline_model"), applies, r.plane.applies)
	}
}

func TestProbeException_RefusedRequestIsNotInstalled(t *testing.T) {
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	enrollBoot(t, r)
	stepOK(t, r)
	bad := labProbe()
	bad.SourceV4 = "100.64.8.0/24"
	probeChallenge(t, r, bad, 0)
	stepOK(t, r)
	for _, c := range r.plane.calls {
		if c.model.Probe != nil {
			t.Fatalf("wide source was installed: %+v", c.model.Probe)
		}
	}
}

func TestProbeException_CapsExpiryAndRefusesUnsafe(t *testing.T) {
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	enrollBoot(t, r)
	stepOK(t, r)
	now := r.eng.trustedNow()
	long := healthattest.Challenge{ExpiresAt: now.Add(10 * time.Minute), Probe: labProbe()}
	p, ok := r.eng.challengeProbe(long, now)
	if !ok {
		t.Fatal("long challenge was refused")
	}
	cap := now.Add(time.Duration(enforce.ProbeMaxSeconds) * time.Second)
	if p.ExpiresAt.After(cap) || p.ExpiresAt.Before(now.Add(299*time.Second)) || p.ExpiresAt.After(long.ExpiresAt) {
		t.Fatalf("expiry %s cap %s deadline %s", p.ExpiresAt, cap, long.ExpiresAt)
	}
	r.eng.desired.DeviceBindings = []api.DeviceBinding{{
		PeerId:      "member-peer",
		AllowedIpv4: []string{"100.64.1.9/32"},
	}}
	refused := []healthattest.ChallengeProbe{
		{PeerID: "", SourceV4: "100.64.8.1/32", Destinations: labProbe().Destinations},
		{PeerID: "peer\nprobe", SourceV4: "100.64.8.1/32", Destinations: labProbe().Destinations},
		{PeerID: "peer-probe", SourceV4: "100.64.8.0/24", Destinations: labProbe().Destinations},
		{PeerID: "peer-probe", SourceV4: "100.64.8.1/32"},
		{PeerID: "member-peer", SourceV4: "100.64.8.1/32", Destinations: labProbe().Destinations},
		{PeerID: "peer-probe", SourceV4: "100.64.1.9/32", Destinations: labProbe().Destinations},
		{PeerID: "peer-probe", SourceV4: "100.64.8.1/32", Destinations: []healthattest.ChallengeProbeDest{{AddrV4: "192.0.2.37/32", Proto: "tcp", Port: 22}}},
		{PeerID: "peer-probe", SourceV4: "100.64.8.1/32", Destinations: []healthattest.ChallengeProbeDest{{AddrV4: "1.2.3.0/24", Proto: "tcp", Port: 80}}},
		{PeerID: "peer-probe", SourceV4: "100.64.8.1/32", Destinations: []healthattest.ChallengeProbeDest{{AddrV4: "1.2.3.4/32", Proto: "icmp", Port: 80}}},
		{PeerID: "peer-probe", SourceV4: "100.64.8.1/32", Destinations: []healthattest.ChallengeProbeDest{{AddrV4: "1.2.3.4/32", Proto: "tcp", Port: 0}}},
	}
	for _, bad := range refused {
		ch := healthattest.Challenge{ExpiresAt: now.Add(30 * time.Second), Probe: &bad}
		if _, ok := r.eng.challengeProbe(ch, now); ok {
			t.Fatalf("installed %+v", bad)
		}
	}
	short := healthattest.Challenge{ExpiresAt: now.Add(200 * time.Millisecond), Probe: labProbe()}
	if _, ok := r.eng.challengeProbe(short, now); ok {
		t.Fatal("subsecond expiry was installed")
	}
	past := healthattest.Challenge{ExpiresAt: now.Add(-time.Second), Probe: labProbe()}
	if _, ok := r.eng.challengeProbe(past, now); ok {
		t.Fatal("past expiry was installed")
	}
	dnsOnly := labProbe()
	dnsOnly.Destinations = nil
	dnsOnly.ResolverDNS = true
	req := healthattest.Challenge{ExpiresAt: now.Add(30 * time.Second), Probe: dnsOnly}
	exc, ok := r.eng.challengeProbe(req, now)
	if !ok {
		t.Fatal("resolver DNS request was refused")
	}
	if _, ok := r.eng.baselineProbeModel(now, exc); ok {
		t.Fatal("resolver DNS rendered with no resolvers")
	}
	r.eng.desired.Dns.ResolverIpv4 = []string{"1.1.1.1"}
	model, ok := r.eng.baselineProbeModel(now, exc)
	if !ok || len(model.Resolvers) != 1 || model.Resolvers[0].String() != "1.1.1.1" || model.Probe == nil || !model.Probe.ResolverDNS || model.Probe.ExpiresAt.IsZero() {
		t.Fatalf("resolver model %+v ok %v", model.Probe, ok)
	}
	script, err := enforce.RenderBaseline(model, enforce.State{})
	if err != nil || !strings.Contains(script, "tcp dport 53") || !strings.Contains(script, "100.64.8.1 timeout") {
		t.Fatalf("resolver rules: %v\n%s", err, script)
	}
}

func TestProbeException_MemberPeerFromDesiredIsNotInstalled(t *testing.T) {
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetFixture(fakecontrol.Fixture{
		ReleaseID:   uuid.MustParse("019a0000-0000-7000-8000-0000000000c1"),
		Publication: api.PublicationIntentUnpublished,
		Generation:  1,
		TTL:         time.Hour,
		Capacity:    &api.NodeCapacity{UpBps: 200_000_000, DownBps: 200_000_000},
		Bindings: []api.DeviceBinding{{
			AllowedIpv4:     []string{"100.64.1.9/32"},
			AllowedIpv6:     []string{},
			BindingRevision: decstr.U64(1),
			DeviceId:        uuid.Must(uuid.NewV7()),
			MemberId:        uuid.Must(uuid.NewV7()),
			PeerId:          "member-peer",
			ValidFrom:       time.Now().Add(-time.Hour).UTC(),
			ValidUntil:      time.Now().Add(time.Hour).UTC(),
		}},
	})
	r.srv.BumpGeneration()
	enrollBoot(t, r)
	stepOK(t, r)
	if r.eng.desired == nil || len(r.eng.desired.DeviceBindings) != 1 {
		t.Fatal("member binding was not loaded")
	}
	member := labProbe()
	member.PeerID = "member-peer"
	probeChallenge(t, r, member, 0)
	stepOK(t, r)
	for _, c := range r.plane.calls {
		if c.model.Probe != nil {
			t.Fatalf("member peer was installed: %+v", c.model.Probe)
		}
	}
	addr := labProbe()
	addr.SourceV4 = "100.64.1.9/32"
	probeChallenge(t, r, addr, 0)
	stepOK(t, r)
	for _, c := range r.plane.calls {
		if c.model.Probe != nil {
			t.Fatalf("member address was installed: %+v", c.model.Probe)
		}
	}
}
