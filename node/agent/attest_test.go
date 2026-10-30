package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// verProbe is a data-path probe with a chosen result and an observed
// NetBird version. It records whether the request was for a published node.
type verProbe struct {
	check   api.HealthCheckStatus
	health  api.DataHealth
	detail  string
	version string
	last    ProbeRequest
}

func (p *verProbe) Probe(_ context.Context, req ProbeRequest) ProbeResult {
	p.last = req
	check := p.check
	if check == "" {
		check = api.HealthCheckStatusPassed
	}
	health := p.health
	if health == "" {
		health = api.DataHealthHealthy
	}
	detail := p.detail
	if detail == "" {
		detail = "overlay_up"
	}
	return ProbeResult{Ran: true, Check: check, Health: health, Detail: detail, NetbirdVersion: p.version}
}

func TestHealthAttest_AnswersMatchingChallenge(t *testing.T) {
	r := newRig(t)
	p := &verProbe{version: "0.76.2"}
	r.eng.prober = p
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	if err := r.eng.Enroll(context.Background(), "token-test-only"); err != nil {
		t.Fatal(err)
	}
	ch := issueChallenge(t, r, nil)
	if err := r.eng.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(attestPath(r)); !os.IsNotExist(err) {
		t.Fatalf("boot answered before desired state was synced: %v", err)
	}
	if r.eng.serving || r.plane.applies != 0 {
		t.Fatalf("boot opened forwarding: serving %v applies %d", r.eng.serving, r.plane.applies)
	}
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	att := requireAttest(t, r, ch)
	if att.Published != api.PublicationIntentUnpublished {
		t.Fatalf("published %s", att.Published)
	}
	if att.AgentVersion != "0.0.0-test" || att.NetbirdVersion != "0.76.2" {
		t.Fatalf("versions agent %q netbird %q", att.AgentVersion, att.NetbirdVersion)
	}
	if att.NetbirdVersion == att.AgentVersion || att.NetbirdVersion == "updater-1" || att.NetbirdVersion == "enf-1" {
		t.Fatalf("netbird version was copied: %q", att.NetbirdVersion)
	}
	assertSameChecks(t, r.eng.decision, att)
	if att.Checks.InternetOnly.Status != api.HealthCheckStatusPassed || att.Checks.InternetOnly.Reason != "baseline held" {
		t.Fatalf("internet %+v", att.Checks.InternetOnly)
	}
	if att.Checks.DNSPolicy.Status != api.HealthCheckStatusPassed || att.Checks.DNSPolicy.Reason != "policy in force" {
		t.Fatalf("dns %+v", att.Checks.DNSPolicy)
	}
	if att.Checks.TunnelProbe.Status != api.HealthCheckStatusPassed || att.Checks.TunnelProbe.Reason != "overlay_up" {
		t.Fatalf("tunnel %+v", att.Checks.TunnelProbe)
	}
	if r.eng.serving || r.plane.applies != 0 || p.last.Published {
		t.Fatalf("unpublished probe opened forwarding: serving %v applies %d published-probe %v", r.eng.serving, r.plane.applies, p.last.Published)
	}
	if strings.Contains(r.log.String(), ch.Nonce) {
		t.Fatal("log contains the challenge nonce")
	}
	ino := fileInode(t, attestPath(r))
	body := readFile(t, attestPath(r))
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fileInode(t, attestPath(r)) != ino || string(readFile(t, attestPath(r))) != string(body) {
		t.Fatal("unchanged result was rewritten")
	}

	p.check = api.HealthCheckStatusFailed
	p.health = api.DataHealthDegraded
	p.detail = "overlay_down"
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	changed := requireAttest(t, r, ch)
	if fileInode(t, attestPath(r)) == ino {
		t.Fatal("changed result kept the inode")
	}
	if changed.Checks.TunnelProbe.Status != api.HealthCheckStatusFailed || changed.Checks.TunnelProbe.Reason != "overlay_down" {
		t.Fatalf("tunnel %+v", changed.Checks.TunnelProbe)
	}
	if changed.Checks.InternetOnly.Status != api.HealthCheckStatusPassed || changed.Checks.DNSPolicy.Status != api.HealthCheckStatusPassed {
		t.Fatalf("baseline checks moved: %+v", changed.Checks)
	}
	assertSameChecks(t, r.eng.decision, changed)
	if r.eng.serving || r.plane.applies != 0 || changed.Published != api.PublicationIntentUnpublished {
		t.Fatalf("failed probe changed publication: serving %v applies %d published %s", r.eng.serving, r.plane.applies, changed.Published)
	}
	held := readFile(t, attestPath(r))
	for _, tc := range []struct {
		name string
		mut  func(*healthattest.Input)
	}{
		{"release", func(in *healthattest.Input) { in.ReleaseID = uuid.Must(uuid.NewV7()).String() }},
		{"boot", func(in *healthattest.Input) { in.BootID = uuid.Must(uuid.NewV7()).String() }},
		{"incarnation", func(in *healthattest.Input) { in.IncarnationID = uuid.Must(uuid.NewV7()).String() }},
		{"node", func(in *healthattest.Input) { in.NodeID = uuid.Must(uuid.NewV7()).String() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			issueChallenge(t, r, tc.mut)
			if err := r.eng.Step(context.Background()); err != nil {
				t.Fatal(err)
			}
			if string(readFile(t, attestPath(r))) != string(held) {
				t.Fatalf("%s challenge was answered", tc.name)
			}
		})
	}

	if err := os.Remove(attestPath(r)); err != nil {
		t.Fatal(err)
	}
	wall := r.clock.Wall()
	issueChallenge(t, r, func(in *healthattest.Input) {
		in.Now = wall.Add(-time.Minute)
		in.TTL = time.Second
	})
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(attestPath(r)); !os.IsNotExist(err) {
		t.Fatalf("expired challenge was answered: %v", err)
	}
	issueChallenge(t, r, func(in *healthattest.Input) {
		in.Now = wall.Add(time.Hour)
	})
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(attestPath(r)); !os.IsNotExist(err) {
		t.Fatalf("future challenge was answered: %v", err)
	}
	stale := issueChallenge(t, r, nil)
	if err := healthattest.MarkConsumed(r.eng.cfg.JournalDir, stale); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(attestPath(r)); !os.IsNotExist(err) {
		t.Fatalf("consumed challenge was answered: %v", err)
	}

	p.check = ""
	p.health = ""
	p.detail = ""
	p.version = "0.76.2"
	bootCh := issueChallenge(t, r, nil)
	applies := r.plane.applies
	if err := r.eng.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
	bootAtt := requireAttest(t, r, bootCh)
	if bootAtt.Published != api.PublicationIntentUnpublished || bootAtt.Checks.TunnelProbe.Status != api.HealthCheckStatusPassed {
		t.Fatalf("boot answer %+v", bootAtt)
	}
	if r.eng.serving || r.plane.applies != applies {
		t.Fatalf("boot answer forwarded: serving %v applies %d -> %d", r.eng.serving, applies, r.plane.applies)
	}

	r.srv.SetPublication(api.PublicationIntentDisabled)
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	dis := requireAttest(t, r, bootCh)
	if dis.Published != api.PublicationIntentDisabled {
		t.Fatalf("published %s", dis.Published)
	}
	assertSameChecks(t, r.eng.decision, dis)
	if r.eng.serving || r.plane.applies != applies || p.last.Published {
		t.Fatalf("disabled answer opened forwarding: serving %v applies %d probe %v", r.eng.serving, r.plane.applies, p.last.Published)
	}
	if dis.AgentVersion != "0.0.0-test" || dis.NetbirdVersion != "0.76.2" {
		t.Fatalf("versions agent %q netbird %q", dis.AgentVersion, dis.NetbirdVersion)
	}
}

func issueChallenge(t *testing.T, r *rig, mut func(*healthattest.Input)) healthattest.Challenge {
	t.Helper()
	in := healthattest.Input{
		TxnID:         uuid.Must(uuid.NewV7()).String(),
		ReleaseID:     r.eng.cfg.ReleaseUUID().String(),
		NodeID:        r.eng.nodeID.String(),
		IncarnationID: r.eng.incID.String(),
		BootID:        r.eng.bootID.String(),
		Now:           r.clock.Wall().Add(-2 * time.Second),
		TTL:           45 * time.Second,
		Components: healthattest.Components{
			NodeAgent:   r.eng.cfg.AgentRelease,
			Netbird:     "0.76.2",
			NodeUpdater: "updater-1",
			Enforcement: "enf-1",
		},
	}
	if mut != nil {
		mut(&in)
	}
	ch, err := healthattest.Issue(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(r.eng.cfg.JournalDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := healthattest.WriteChallenge(r.eng.cfg.JournalDir, ch); err != nil {
		t.Fatal(err)
	}
	return ch
}

func requireAttest(t *testing.T, r *rig, ch healthattest.Challenge) healthattest.Attestation {
	t.Helper()
	path := attestPath(r)
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", info.Mode())
	}
	tmp, err := filepath.Glob(filepath.Join(r.eng.cfg.JournalDir, ".health-attest.json.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tmp) != 0 {
		t.Fatalf("temp files left behind: %v", tmp)
	}
	att, _, err := healthattest.ReadAttestation(r.eng.cfg.JournalDir)
	if err != nil {
		t.Fatal(err)
	}
	if reason := healthattest.Validate(ch, att, r.clock.Wall(), ch.Components); reason != "" {
		t.Fatalf("validate %s", reason)
	}
	chJSON, err := json.Marshal(ch)
	if err != nil {
		t.Fatal(err)
	}
	wrapped, err := ValidateHealthAnswer(chJSON, readFile(t, path), r.clock.Wall(), ch.Components.NodeAgent, ch.Components.Netbird, ch.Components.NodeUpdater, ch.Components.Enforcement)
	if err != nil || wrapped != "" {
		t.Fatalf("wrapper %q %v", wrapped, err)
	}
	if !att.GeneratedAt.After(ch.IssuedAt) || !att.GeneratedAt.Before(ch.ExpiresAt) {
		t.Fatalf("generated %s issued %s expires %s", att.GeneratedAt, ch.IssuedAt, ch.ExpiresAt)
	}
	return att
}

func assertSameChecks(t *testing.T, d Decision, att healthattest.Attestation) {
	t.Helper()
	want := map[api.NodeCheckName]api.HealthCheckStatus{
		api.NodeCheckNameInternetOnly: att.Checks.InternetOnly.Status,
		api.NodeCheckNameDnsPolicy:    att.Checks.DNSPolicy.Status,
		api.NodeCheckNameTunnelProbe:  att.Checks.TunnelProbe.Status,
	}
	for name, status := range want {
		got, ok := decisionStatus(d, name)
		if !ok || got != status {
			t.Fatalf("%s decision %s attestation %s", name, got, status)
		}
	}
	if d.Publication != att.Published {
		t.Fatalf("decision publication %s attestation %s", d.Publication, att.Published)
	}
}

func attestPath(r *rig) string {
	return filepath.Join(r.eng.cfg.JournalDir, healthattest.AttestationFile)
}

func fileInode(t *testing.T, path string) uint64 {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Ino == 0 {
		t.Fatal("inode unavailable")
	}
	return st.Ino
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
