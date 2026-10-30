package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"log/slog"
	"os"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	"github.com/TED-H-com/Freedom-VPN/node/dns"
	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// exampleResolverBody is a publisher-normalized resolver document. The signed
// digest is the SHA-256 of these exact bytes. The schema example leaves
// bootstrap_dns empty; Compile copies the upstream list, which is what
// dnspublish.Verify accepts before Apply.
var exampleResolverBody = mustExampleResolver()

func mustExampleResolver() string {
	compiled, err := dnspublish.Compile(dnspublish.Input{
		ProfileCode:       "standard",
		BlockResponse:     "nxdomain",
		BlockedTTLSeconds: 60,
		Upstreams:         []string{"192.0.2.53"},
	})
	if err != nil {
		panic(err)
	}
	return string(compiled.Bytes)
}

func exampleRef(body []byte) api.ResolverDocumentRef {
	sum := sha256.Sum256(body)
	return api.ResolverDocumentRef{
		DocumentId:        uuid.MustParse("01900000-0000-7000-8000-000000000056"),
		ProfileRevisionId: uuid.MustParse("01900000-0000-7000-8000-000000000012"),
		Generation:        decstr.U64(9007199254740993),
		ContentSha256:     hex.EncodeToString(sum[:]),
		SizeBytes:         len(body),
	}
}

type scriptDNS struct {
	calls int
	gens  []string
	docs  []dnspublish.Document
	ret   dns.Evidence
	err   error
}

func (s *scriptDNS) Apply(_ context.Context, generation string, doc dnspublish.Document) (dns.Evidence, error) {
	s.calls++
	s.gens = append(s.gens, generation)
	s.docs = append(s.docs, doc)
	if s.err != nil {
		return s.ret, s.err
	}
	ev := s.ret
	ev.Verified = true
	if ev.PolicyGeneration == "" {
		ev.PolicyGeneration = generation
	}
	if ev.DocumentHash == "" {
		ev.DocumentHash = strings.Repeat("ab", 32)
	}
	if ev.AppliedAt.IsZero() {
		ev.AppliedAt = time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	}
	return ev, nil
}

func enrollBoot(t *testing.T, r *rig) {
	t.Helper()
	if err := r.eng.Enroll(context.Background(), "token-test-only"); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func stepOK(t *testing.T, r *rig) {
	t.Helper()
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func mustDNS(t *testing.T, checks []api.NodeCheck) api.NodeCheck {
	t.Helper()
	c, ok := checkBy(checks, api.NodeCheckNameDnsPolicy)
	if !ok {
		t.Fatal("missing dns_policy")
	}
	return c
}

func codeOf(t *testing.T, c api.NodeCheck) api.ErrorCode {
	t.Helper()
	code, err := c.ErrorCode.Get()
	if err != nil {
		t.Fatal(err)
	}
	return code
}

func TestResolver_AbsentDoesNotFetch(t *testing.T) {
	r := newRig(t)
	enrollBoot(t, r)
	stepOK(t, r)
	if n := r.srv.ResolverFetches(); n != 0 {
		t.Fatalf("fetches %d", n)
	}
	c := mustDNS(t, r.eng.decision.Checks)
	if c.Status != api.HealthCheckStatusPassed || c.Detail != nil {
		t.Fatalf("dns %+v", c)
	}
	rep, ok := r.srv.LastObservation()
	if !ok {
		t.Fatal("no observation")
	}
	if _, err := rep.DnsPolicyEvidence.Get(); err == nil {
		t.Fatal("evidence reported without a resolver document")
	}
	if r.eng.desired.PublicationIntent != api.PublicationIntentPublished || !r.eng.serving {
		t.Fatalf("publication %s serving %v", r.eng.desired.PublicationIntent, r.eng.serving)
	}
}

func TestResolver_DigestMismatch(t *testing.T) {
	r := newRig(t)
	dnsApply := &scriptDNS{}
	r.eng.dnsApply = dnsApply
	good := []byte(exampleResolverBody)
	bad := append([]byte(nil), good...)
	bad[len(bad)-8] ^= 0x01
	ref := exampleRef(good)
	r.srv.PutResolverBody(ref.DocumentId, bad)
	r.srv.SetResolverRef(&ref)
	enrollBoot(t, r)
	stepOK(t, r)
	if dnsApply.calls != 0 {
		t.Fatalf("apply calls %d", dnsApply.calls)
	}
	c := mustDNS(t, r.eng.decision.Checks)
	if c.Status != api.HealthCheckStatusPassed {
		if c.Detail == nil || *c.Detail != "Resolver document digest differs from the signed reference." {
			t.Fatalf("detail %v", c.Detail)
		}
		if codeOf(t, c) != api.ErrorCodeHASHMISMATCH {
			t.Fatalf("code %s", codeOf(t, c))
		}
	} else {
		t.Fatal("dns_policy passed")
	}
	if r.eng.serving || r.eng.decision.Safety != api.SafetyStatusFailed {
		t.Fatalf("serving %v safety %s", r.eng.serving, r.eng.decision.Safety)
	}
	if r.eng.desired.PublicationIntent != api.PublicationIntentPublished {
		t.Fatalf("publication %s", r.eng.desired.PublicationIntent)
	}
	ev := mustEvidence(t, r)
	if ev.Outcome != api.DnsPolicyEvidenceOutcomeFailed || ev.DocumentId != ref.DocumentId {
		t.Fatalf("%+v", ev)
	}
	if ev.ContentSha256 == ref.ContentSha256 {
		t.Fatal("evidence claimed the signed digest was applied")
	}
	sum := sha256.Sum256(bad)
	if ev.ContentSha256 != hex.EncodeToString(sum[:]) || ev.Generation != ref.Generation {
		t.Fatalf("hash %s gen %s", ev.ContentSha256, ev.Generation)
	}
	if _, err := ev.AppliedAt.Get(); !errorsIsNull(err) {
		t.Fatalf("applied_at %v", err)
	}
	got, err := ev.ErrorCode.Get()
	if err != nil || got != api.ErrorCodeHASHMISMATCH {
		t.Fatalf("error %v %v", got, err)
	}
}

func TestResolver_SizeOverflow(t *testing.T) {
	r := newRig(t)
	dnsApply := &scriptDNS{}
	r.eng.dnsApply = dnsApply
	good := []byte(exampleResolverBody)
	ref := exampleRef(good)
	over := append(append([]byte(nil), good...), ' ')
	r.srv.PutResolverBody(ref.DocumentId, over)
	r.srv.SetResolverRef(&ref)
	enrollBoot(t, r)
	stepOK(t, r)
	if dnsApply.calls != 0 {
		t.Fatalf("apply calls %d", dnsApply.calls)
	}
	c := mustDNS(t, r.eng.decision.Checks)
	if c.Status != api.HealthCheckStatusFailed || c.Detail == nil || *c.Detail != "Resolver document exceeds the signed size." {
		t.Fatalf("%+v", c)
	}
	if codeOf(t, c) != api.ErrorCodePAYLOADTOOLARGE {
		t.Fatalf("code %s", codeOf(t, c))
	}
	ev := mustEvidence(t, r)
	if ev.Outcome != api.DnsPolicyEvidenceOutcomeFailed {
		t.Fatalf("outcome %s", ev.Outcome)
	}
}

func TestResolver_NotFound(t *testing.T) {
	r := newRig(t)
	dnsApply := &scriptDNS{}
	r.eng.dnsApply = dnsApply
	ref := exampleRef([]byte(exampleResolverBody))
	r.srv.SetResolverRef(&ref)
	enrollBoot(t, r)
	stepOK(t, r)
	if dnsApply.calls != 0 {
		t.Fatalf("apply calls %d", dnsApply.calls)
	}
	if r.srv.ResolverFetches() != 1 {
		t.Fatalf("fetches %d", r.srv.ResolverFetches())
	}
	c := mustDNS(t, r.eng.decision.Checks)
	if c.Status != api.HealthCheckStatusFailed || c.Detail == nil || *c.Detail != "Resolver document was not found." {
		t.Fatalf("%+v", c)
	}
	if codeOf(t, c) != api.ErrorCodeNOTFOUND {
		t.Fatalf("code %s", codeOf(t, c))
	}
	ev := mustEvidence(t, r)
	if ev.Outcome != api.DnsPolicyEvidenceOutcomeFailed || ev.ContentSha256 != ref.ContentSha256 || ev.DocumentId != ref.DocumentId {
		t.Fatalf("%+v", ev)
	}
	if r.eng.desired.PublicationIntent != api.PublicationIntentPublished || r.eng.serving {
		t.Fatalf("publication %s serving %v", r.eng.desired.PublicationIntent, r.eng.serving)
	}
}

func TestResolver_StaleDocumentID(t *testing.T) {
	r := newRig(t)
	good := []byte(exampleResolverBody)
	ref := exampleRef(good)
	dnsApply := successDNS(ref)
	r.eng.dnsApply = dnsApply
	r.srv.PutResolverBody(ref.DocumentId, good)
	r.srv.SetResolverRef(&ref)
	enrollBoot(t, r)
	stepOK(t, r)
	if dnsApply.calls != 1 || r.srv.ResolverFetches() != 1 {
		t.Fatalf("calls %d fetches %d", dnsApply.calls, r.srv.ResolverFetches())
	}
	ev := mustEvidence(t, r)
	if ev.Outcome != api.DnsPolicyEvidenceOutcomeApplied || ev.Generation.String() != ref.Generation.String() || ev.ContentSha256 != ref.ContentSha256 {
		t.Fatalf("%+v", ev)
	}
	if len(ev.Feeds) != 1 || ev.Feeds[0].FeedCode != "ads" || ev.Feeds[0].RuleCount != 2 {
		t.Fatalf("feeds %+v", ev.Feeds)
	}
	if _, err := ev.ErrorCode.Get(); !errorsIsNull(err) {
		t.Fatal(err)
	}
	if _, err := ev.AppliedAt.Get(); err != nil {
		t.Fatal(err)
	}
	next := ref
	next.DocumentId = uuid.MustParse("01900000-0000-7000-8000-000000000057")
	r.srv.SetResolverRef(&next)
	stepOK(t, r)
	if r.srv.ResolverFetches() != 2 || dnsApply.calls != 1 {
		t.Fatalf("fetches %d calls %d", r.srv.ResolverFetches(), dnsApply.calls)
	}
	c := mustDNS(t, r.eng.decision.Checks)
	if c.Status != api.HealthCheckStatusFailed || codeOf(t, c) != api.ErrorCodeNOTFOUND {
		t.Fatalf("%+v", c)
	}
	ev = mustEvidence(t, r)
	if ev.DocumentId != next.DocumentId || ev.Outcome != api.DnsPolicyEvidenceOutcomeFailed {
		t.Fatalf("%+v", ev)
	}
}

func TestResolver_RestartUsesCache(t *testing.T) {
	r := newRig(t)
	good := []byte(exampleResolverBody)
	ref := exampleRef(good)
	dnsApply := successDNS(ref)
	r.eng.dnsApply = dnsApply
	r.srv.PutResolverBody(ref.DocumentId, good)
	r.srv.SetResolverRef(&ref)
	enrollBoot(t, r)
	stepOK(t, r)
	if dnsApply.calls != 1 || r.srv.ResolverFetches() != 1 {
		t.Fatalf("calls %d fetches %d", dnsApply.calls, r.srv.ResolverFetches())
	}
	cfg := r.eng.cfg
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	boot, err := os.ReadFile(cfg.BootIDFile)
	if err != nil {
		t.Fatal(err)
	}
	store, err := agentstore.Open(cfg.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	eng, err := Start(Options{
		Config: cfg, Store: store, Clock: r.clock, Plane: &memPlane{},
		Log: slog.New(slog.DiscardHandler), MeterSource: staticSource{boot: strings.TrimSpace(string(boot))},
		Prober: passProbe{}, DNS: dnsApply,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = eng.Close() })
	if err := eng.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.srv.ResolverFetches() != 1 {
		t.Fatalf("restart refetched: %d", r.srv.ResolverFetches())
	}
	if dnsApply.calls != 2 {
		t.Fatalf("apply calls %d", dnsApply.calls)
	}
	if len(dnsApply.docs) != 2 || len(dnsApply.docs[1].Upstreams) != 1 || dnsApply.docs[1].Upstreams[0] != "192.0.2.53" {
		t.Fatalf("cached document %+v", dnsApply.docs)
	}
	rep, ok := r.srv.LastObservation()
	if !ok {
		t.Fatal("no observation")
	}
	ev, err := rep.DnsPolicyEvidence.Get()
	if err != nil || ev.Outcome != api.DnsPolicyEvidenceOutcomeApplied || ev.DocumentId != ref.DocumentId {
		t.Fatalf("%+v %v", ev, err)
	}
}

func successDNS(ref api.ResolverDocumentRef) *scriptDNS {
	return &scriptDNS{ret: dns.Evidence{
		Verified:         true,
		PolicyGeneration: ref.Generation.String(),
		DocumentHash:     ref.ContentSha256,
		AppliedAt:        time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC),
		Feeds:            []dns.FeedEvidence{{FeedCode: "ads", ContentSHA256: strings.Repeat("ab", 32), RuleCount: 2}},
	}}
}

func mustEvidence(t *testing.T, r *rig) api.DnsPolicyEvidence {
	t.Helper()
	rep, ok := r.srv.LastObservation()
	if !ok {
		t.Fatal("no observation")
	}
	ev, err := rep.DnsPolicyEvidence.Get()
	if err != nil {
		t.Fatal(err)
	}
	return ev
}

func errorsIsNull(err error) bool {
	return err != nil && err == presence.ErrNull
}

type recProbe struct {
	reqs []ProbeRequest
}

func (p *recProbe) Probe(_ context.Context, req ProbeRequest) ProbeResult {
	p.reqs = append(p.reqs, req)
	return ProbeResult{Ran: true, Check: api.HealthCheckStatusPassed, Health: api.DataHealthHealthy, Detail: "overlay_up"}
}

func TestEgressProbeSourceSwitch(t *testing.T) {
	r := newRig(t)
	rec := &recProbe{}
	r.eng.prober = rec
	r.eng.cfg.EgressProbeTarget = "192.0.2.10:443"
	enrollBoot(t, r)
	stepOK(t, r)
	if len(rec.reqs) == 0 || len(rec.reqs[len(rec.reqs)-1].Targets) != 1 || rec.reqs[len(rec.reqs)-1].Targets[0] != "192.0.2.10:443" {
		t.Fatalf("local %+v", rec.reqs)
	}
	localDetail := tunnelDetail(t, r.eng.decision.Checks)
	if localDetail != "Egress probe used the local target." {
		t.Fatalf("detail %s", localDetail)
	}
	timeout := 1500
	r.srv.SetEgressProbe(&api.EgressProbe{
		Targets: []api.EgressProbeTarget{
			{Address: "192.0.2.80", Port: 443, Protocol: api.EgressProbeTargetProtocolTcp},
			{Address: "2001:db8::80", Port: 443, Protocol: api.EgressProbeTargetProtocolTcp},
		},
		TimeoutMs: &timeout,
	})
	stepOK(t, r)
	got := rec.reqs[len(rec.reqs)-1]
	want := []string{"192.0.2.80:443", "[2001:db8::80]:443"}
	if len(got.Targets) != len(want) || got.Targets[0] != want[0] || got.Targets[1] != want[1] || got.Timeout != 1500*time.Millisecond || got.Rejected {
		t.Fatalf("signed %+v", got)
	}
	for _, addr := range got.Targets {
		if addr == "192.0.2.10:443" {
			t.Fatal("signed probe still dialed the local target")
		}
	}
	if tunnelDetail(t, r.eng.decision.Checks) != "Egress probe used the signed targets." {
		t.Fatalf("detail %s", tunnelDetail(t, r.eng.decision.Checks))
	}
	if r.eng.desired.PublicationIntent != api.PublicationIntentPublished {
		t.Fatalf("publication %s", r.eng.desired.PublicationIntent)
	}
	r.srv.SetEgressProbe(nil)
	stepOK(t, r)
	got = rec.reqs[len(rec.reqs)-1]
	if len(got.Targets) != 1 || got.Targets[0] != "192.0.2.10:443" {
		t.Fatalf("cleared %+v", got)
	}
	if tunnelDetail(t, r.eng.decision.Checks) != "Egress probe used the local target." {
		t.Fatalf("detail %s", tunnelDetail(t, r.eng.decision.Checks))
	}
	omitted := &api.EgressProbe{Targets: []api.EgressProbeTarget{
		{Address: "192.0.2.80", Port: 443, Protocol: api.EgressProbeTargetProtocolTcp},
	}}
	r.srv.SetEgressProbe(omitted)
	stepOK(t, r)
	got = rec.reqs[len(rec.reqs)-1]
	if got.Timeout != 2*time.Second || len(got.Targets) != 1 || got.Targets[0] != "192.0.2.80:443" {
		t.Fatalf("default timeout %+v", got)
	}
}

func tunnelDetail(t *testing.T, checks []api.NodeCheck) string {
	t.Helper()
	c, ok := checkBy(checks, api.NodeCheckNameTunnelProbe)
	if !ok || c.Detail == nil {
		t.Fatalf("tunnel %+v", c)
	}
	return *c.Detail
}

func TestCheckDetailSanitized(t *testing.T) {
	clean := sanitizeDetail("Resolver document digest differs from the signed reference.")
	if clean == nil || *clean != "Resolver document digest differs from the signed reference." {
		t.Fatalf("clean detail %v", clean)
	}
	in := servingInput()
	in.DNSDocument = true
	in.DNSFailed = true
	in.DNSCode = api.ErrorCodeHASHMISMATCH
	in.DNSDetail = "Resolver document digest differs from the signed reference. token=sekret https://user:pw@198.51.100.9/raw subject CN=hidden 2001:db8::10 " + strings.Repeat("k", 600)
	d := Decide(in)
	c := mustDNS(t, d.Checks)
	if c.Status != api.HealthCheckStatusFailed || codeOf(t, c) != api.ErrorCodeHASHMISMATCH || c.Detail == nil {
		t.Fatalf("%+v", c)
	}
	detail := *c.Detail
	if len(detail) > 500 {
		t.Fatalf("len %d", len(detail))
	}
	for _, secret := range []string{"sekret", "user:pw", "198.51.100.9", "CN=hidden", "2001:db8::10", "https://"} {
		if strings.Contains(detail, secret) {
			t.Fatalf("detail kept %q: %s", secret, detail)
		}
	}
	if d.Safety != api.SafetyStatusFailed || d.Serve {
		t.Fatalf("safety %s serve %v", d.Safety, d.Serve)
	}
	if d.Publication != api.PublicationIntentPublished {
		t.Fatalf("publication %s", d.Publication)
	}
}
