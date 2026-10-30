package healthattest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func validPair(t *testing.T) (Challenge, Attestation, time.Time, Components) {
	t.Helper()
	now := time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC)
	versions := Components{NodeAgent: "1.2.0", Netbird: "0.76.2", NodeUpdater: "1.2.0", Enforcement: "1.2.0"}
	ch, err := Issue(Input{
		TxnID: "018f5b2a-6c3e-7a10-8d11-111111111111", ReleaseID: "018f5b2a-6c3e-7a10-8d11-222222222222",
		NodeID: "018f5b2a-6c3e-7a10-8d11-333333333333", IncarnationID: "018f5b2a-6c3e-7a10-8d11-444444444444",
		BootID: "018f5b2a-6c3e-7a10-8d11-555555555555", Now: now, TTL: 45 * time.Second, Components: versions,
	})
	if err != nil {
		t.Fatal(err)
	}
	at := Attestation{
		TxnID: ch.TxnID, Nonce: ch.Nonce, ReleaseID: ch.ReleaseID, NodeID: ch.NodeID,
		AgentVersion: versions.NodeAgent, NetbirdVersion: versions.Netbird,
		BootID: ch.BootID, IncarnationID: ch.IncarnationID,
		GeneratedAt: ch.IssuedAt.Add(time.Millisecond), Published: api.PublicationIntentUnpublished,
		Checks: Checks{
			InternetOnly: Check{Status: api.HealthCheckStatusPassed, Reason: "baseline held"},
			DNSPolicy:    Check{Status: api.HealthCheckStatusPassed, Reason: "policy loaded"},
			TunnelProbe:  Check{Status: api.HealthCheckStatusPassed, Reason: "probe reached the lab endpoint"},
		},
	}
	return ch, at, now.Add(time.Second), versions
}

func TestValidateAcceptsBoundAttestation(t *testing.T) {
	ch, at, now, versions := validPair(t)
	if reason := Validate(ch, at, now, versions); reason != "" {
		t.Fatal(reason)
	}
	// Publication intent is evidence, not a pass bit. Every legal intent binds.
	for _, intent := range []api.PublicationIntent{api.PublicationIntentPublished, api.PublicationIntentDisabled, api.PublicationIntentUnpublished} {
		at.Published = intent
		if reason := Validate(ch, at, now, versions); reason != "" {
			t.Fatal(intent, reason)
		}
	}
	at.NodeID = strings.ToUpper(at.NodeID)
	if reason := Validate(ch, at, now, versions); reason != "" {
		t.Fatal("canonical uuid case", reason)
	}
}

func TestValidateRejections(t *testing.T) {
	ch, at, now, versions := validPair(t)
	other := "018f5b2a-6c3e-7a10-8d11-666666666666"
	cases := []struct {
		name   string
		mutate func(*Challenge, *Attestation, *time.Time, *Components)
		want   Reason
	}{
		{"txn", func(_ *Challenge, a *Attestation, _ *time.Time, _ *Components) { a.TxnID = other }, ReasonTxnMismatch},
		{"nonce", func(_ *Challenge, a *Attestation, _ *time.Time, _ *Components) {
			a.Nonce = strings.Repeat("ab", 32)
		}, ReasonNonceMismatch},
		{"release", func(_ *Challenge, a *Attestation, _ *time.Time, _ *Components) { a.ReleaseID = other }, ReasonReleaseMismatch},
		{"node", func(_ *Challenge, a *Attestation, _ *time.Time, _ *Components) { a.NodeID = other }, ReasonNodeMismatch},
		{"incarnation", func(_ *Challenge, a *Attestation, _ *time.Time, _ *Components) { a.IncarnationID = other }, ReasonIncarnationMismatch},
		{"boot", func(_ *Challenge, a *Attestation, _ *time.Time, _ *Components) { a.BootID = other }, ReasonBootMismatch},
		{"generated equal issued", func(c *Challenge, a *Attestation, _ *time.Time, _ *Components) {
			a.GeneratedAt = c.IssuedAt
		}, ReasonGeneratedBeforeIssue},
		{"generated before issued", func(c *Challenge, a *Attestation, _ *time.Time, _ *Components) {
			a.GeneratedAt = c.IssuedAt.Add(-time.Millisecond)
		}, ReasonGeneratedBeforeIssue},
		{"generated at expiry", func(c *Challenge, a *Attestation, _ *time.Time, _ *Components) {
			a.GeneratedAt = c.ExpiresAt
		}, ReasonExpired},
		{"now at expiry", func(c *Challenge, a *Attestation, n *time.Time, _ *Components) {
			*n = c.ExpiresAt
		}, ReasonExpired},
		{"now after expiry", func(c *Challenge, _ *Attestation, n *time.Time, _ *Components) {
			*n = c.ExpiresAt.Add(time.Second)
		}, ReasonExpired},
		{"agent version", func(_ *Challenge, a *Attestation, _ *time.Time, _ *Components) { a.AgentVersion = "9.9.9" }, ReasonAgentVersion},
		{"netbird version", func(_ *Challenge, a *Attestation, _ *time.Time, _ *Components) { a.NetbirdVersion = "0.0.1" }, ReasonNetbirdVersion},
		{"empty manifest agent", func(_ *Challenge, _ *Attestation, _ *time.Time, v *Components) { v.NodeAgent = "" }, ReasonAgentVersion},
		{"unknown status", func(_ *Challenge, a *Attestation, _ *time.Time, _ *Components) {
			a.Checks.DNSPolicy.Status = "skipped"
		}, ReasonUnknownStatus},
		{"empty status", func(_ *Challenge, a *Attestation, _ *time.Time, _ *Components) {
			a.Checks.TunnelProbe.Status = ""
		}, ReasonUnknownStatus},
		{"not_run", func(_ *Challenge, a *Attestation, _ *time.Time, _ *Components) {
			a.Checks.TunnelProbe.Status = api.HealthCheckStatusNotRun
		}, ReasonNotRun},
		{"bad publication", func(_ *Challenge, a *Attestation, _ *time.Time, _ *Components) {
			a.Published = "yes"
		}, ReasonBadPublication},
		{"long reason", func(_ *Challenge, a *Attestation, _ *time.Time, _ *Components) {
			a.Checks.InternetOnly.Reason = strings.Repeat("x", MaxReasonBytes+1)
		}, ReasonBadReason},
		{"control reason", func(_ *Challenge, a *Attestation, _ *time.Time, _ *Components) {
			a.Checks.InternetOnly.Reason = "line\nbreak"
		}, ReasonBadReason},
		{"consumed", func(c *Challenge, _ *Attestation, _ *time.Time, _ *Components) { c.Consumed = true }, ReasonConsumed},
		{"bad nonce on challenge", func(c *Challenge, a *Attestation, _ *time.Time, _ *Components) {
			c.Nonce = "ABCD"
			a.Nonce = c.Nonce
		}, ReasonMalformed},
		{"missing generated", func(_ *Challenge, a *Attestation, _ *time.Time, _ *Components) {
			a.GeneratedAt = time.Time{}
		}, ReasonMalformed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, a, n, v := ch, at, now, versions
			tc.mutate(&c, &a, &n, &v)
			if got := Validate(c, a, n, v); got != tc.want {
				t.Fatalf("reason %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRoundTripModeAndPreseededFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	ch, at, now, versions := validPair(t)
	if err := WriteChallenge(dir, ch); err != nil {
		t.Fatal(err)
	}
	got, info, err := ReadChallenge(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Same(ch) || info.Mode().Perm() != 0o600 {
		t.Fatalf("challenge %+v mode %o", got, info.Mode().Perm())
	}
	// A file that is already present is not an answer, even when a later
	// rewrite lands on the same file timestamp as the challenge.
	early := filepath.Join(dir, AttestationFile)
	if err := os.WriteFile(early, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(early, past, past); err != nil {
		t.Fatal(err)
	}
	earlyInfo, err := os.Lstat(early)
	if err != nil {
		t.Fatal(err)
	}
	if !AttestationPrecedes(info.ModTime(), earlyInfo.ModTime()) {
		t.Fatal("pre-seeded file was not older than the challenge")
	}
	prior, err := ObserveAttestation(dir)
	if err != nil || !prior.Present {
		t.Fatal(err, prior.Present)
	}
	if err := WriteAttestation(dir, at); err != nil {
		t.Fatal(err)
	}
	gotAt, _, err := ReadAttestation(dir)
	if err != nil {
		t.Fatal(err)
	}
	cur, err := ObserveAttestation(dir)
	if err != nil || !cur.Rewritten(prior) {
		t.Fatalf("rewrite was not visible: %v %+v", err, cur)
	}
	again, err := ObserveAttestation(dir)
	if err != nil || again.Rewritten(cur) {
		t.Fatal("unchanged file looked rewritten", err)
	}
	if reason := Validate(got, gotAt, now, versions); reason != "" {
		t.Fatal(reason)
	}
	if err := os.Chmod(filepath.Join(dir, AttestationFile), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadAttestation(dir); ReasonOf(err) != ReasonBadMode {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, AttestationFile)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, ChallengeFile), filepath.Join(dir, AttestationFile)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadAttestation(dir); ReasonOf(err) != ReasonSymlink {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, AttestationFile)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, AttestationFile), []byte("{\"txn_id\":\"x\",\"extra\":true}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadAttestation(dir); err == nil {
		t.Fatal("unknown field accepted")
	}
	ch.Consumed = true
	if err := MarkConsumed(dir, ch); err != nil {
		t.Fatal(err)
	}
	marked, _, err := ReadChallenge(dir)
	if err != nil || !marked.Consumed {
		t.Fatal(err, marked.Consumed)
	}
	if err := Consume(dir); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadChallenge(dir); !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := Consume(dir); err != nil {
		t.Fatal("missing files")
	}
}
