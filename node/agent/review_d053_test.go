package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestAG_D053_Review_ExpiredStoredDesiredETagOn304WritesFailedDNSAttestation(t *testing.T) {
	// TODO(review): In loadRuntime (engine.go:413), `e.desiredETag = d.ETag` is
	// assigned BEFORE `verifyDesired(d.Envelope, e.trust, now)` checks whether
	// the persisted desired-state envelope has expired.
	//
	// If the stored desired-state envelope in agent.db expired while the old
	// service was stopped for the update, a newly started candidate agent loads
	// `e.desired == nil` but keeps `e.desiredETag = d.ETag`. When Step runs
	// `syncDesired` and the server returns 304 Not Modified for that ETag,
	// `syncDesired` sets `e.desiredSynced = true` while `e.desired` remains nil,
	// and `answerHealth` immediately writes `dns_policy: "failed"` ("policy not
	// in force") to health-attest.json. Because NodeChecks.poll exits on the
	// first bound attestation (`got != nil`), this immediately fails the
	// updater's health check attempt and persists across retries.
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	if err := r.eng.Enroll(context.Background(), "token-test-only"); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.eng.desired == nil || r.eng.desiredETag == "" {
		t.Fatalf("expected stored desired state and ETag after first step, got desired=%v etag=%q", r.eng.desired, r.eng.desiredETag)
	}

	// Advance clock past the stored desired-state envelope's ExpiresAt while the
	// service is stopped for an update, then restart (fresh Engine on the same
	// StateDir/IdentityDir/JournalDir).
	r.clock.Advance(2 * time.Hour)
	_ = r.eng.Close()

	cfg := r.eng.cfg
	cand, err := Start(Options{
		Config:      cfg,
		Clock:       r.clock,
		Plane:       r.plane,
		Log:         r.eng.log,
		MeterSource: staticSource{boot: r.eng.bootID.String()},
		Prober:      &verProbe{version: "0.76.2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cand.Close()
	r.eng = cand

	ch := issueChallenge(t, r, nil)
	if err := cand.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
	// On Step, cand sends If-None-Match with the expired document's ETag;
	// fakecontrol returns 304 Not Modified because its ETag has not changed.
	if err := cand.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	att, _, err := healthattest.ReadAttestation(cand.cfg.JournalDir)
	if err == nil && att.Nonce == ch.Nonce && att.Checks.DNSPolicy.Status == api.HealthCheckStatusFailed {
		t.Fatalf("candidate retained expired desiredETag (%q) with nil desired on 304 and attested dns_policy=%s (%q)",
			cand.desiredETag, att.Checks.DNSPolicy.Status, att.Checks.DNSPolicy.Reason)
	}
}

func TestAG_D053_Review_TransientFirstSyncErrorPrematurelyAttestsFailedDNS(t *testing.T) {
	// TODO(review): In syncDesired (sync.go:26), `defer func() { e.desiredSynced = true }()`
	// sets `e.desiredSynced = true` even when `e.client.DesiredState` fails with
	// a transient transport error and `e.desired == nil`.
	//
	// Despite the comment at engine.go:313-315 ("A missing document before the
	// first sync is not a DNS result yet: a binding failure would end the
	// updater's wait before Step corrects it"), a single transient HTTP failure
	// on the first Step marks `e.desiredSynced = true` with `e.desired == nil`,
	// causing `answerHealth` to write `dns_policy: "failed"` into
	// health-attest.json and immediately terminate the updater's 30s wait loop
	// before the next Step can succeed.
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	if err := r.eng.Enroll(context.Background(), "token-test-only"); err != nil {
		t.Fatal(err)
	}
	ch := issueChallenge(t, r, nil)
	if err := r.eng.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}

	// Simulate a transient control-plane failure on the first Step by closing
	// the test server before Step runs.
	_ = r.srv.Close()
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}

	if att, _, err := healthattest.ReadAttestation(r.eng.cfg.JournalDir); err == nil {
		t.Fatalf("first Step with transient DesiredState transport error and nil desired prematurely wrote health-attest.json (nonce=%s dns_policy=%s reason=%q), aborting updater wait window",
			att.Nonce, att.Checks.DNSPolicy.Status, att.Checks.DNSPolicy.Reason)
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected error reading attestation after transient sync failure: %v (challenge=%s)", err, ch.Nonce)
	}
}

func TestAG_D053_Review_Held_NetbirdVersionAndProbeReasonSanitization(t *testing.T) {
	// 1. netbirdVersion rejects URLs, spaces, colons, slashes, control chars, >64B, and empty strings.
	badVersions := []string{
		"",
		"   ",
		"https://evil.example.com/0.76.2",
		"http://100.64.0.1",
		"0.76.2/../../etc/passwd",
		"0.76.2:8080",
		"0.76.2 beta",
		"0.76\n.2",
		"0.76.2\x00",
		"0.76.2\x7f",
		strings.Repeat("1", 65),
	}
	for _, raw := range badVersions {
		if got := netbirdVersion(raw); got != "" {
			t.Fatalf("netbirdVersion(%q) = %q, want empty string", raw, got)
		}
	}
	if got := netbirdVersion(" 0.76.2-rc1+build.4 "); got != "0.76.2-rc1+build.4" {
		t.Fatalf("netbirdVersion valid token = %q", got)
	}

	// 2. tunnelReason / detailToken strips any non-token detail (URLs, DNS names, IPs, uppercase, >40B).
	leaks := []string{
		"https://member.example.com/secret",
		"dns=corp.internal",
		"100.64.0.42",
		"Overlay_Up",
		"overlay up with spaces",
		strings.Repeat("a", 41),
	}
	for _, d := range leaks {
		if got := tunnelReason(api.HealthCheckStatusFailed, d); got != "probe failed" {
			t.Fatalf("tunnelReason(failed, %q) leaked %q, want \"probe failed\"", d, got)
		}
		if got := tunnelReason(api.HealthCheckStatusPassed, d); got != "overlay up" {
			t.Fatalf("tunnelReason(passed, %q) leaked %q, want \"overlay up\"", d, got)
		}
	}

	// 3. Oversized NetBird status file (> 1 MiB) fails closed.
	statusPath := filepath.Join(t.TempDir(), "nb-status.json")
	huge := `{"daemonVersion":"0.76.2","management":{"connected":true},"padding":"` + strings.Repeat("P", probeFileMax) + `"}`
	if err := os.WriteFile(statusPath, []byte(huge), 0o600); err != nil {
		t.Fatal(err)
	}
	pr := netbirdProber{statusFile: statusPath}
	res := pr.Probe(context.Background(), ProbeRequest{})
	if res.Check != api.HealthCheckStatusFailed || res.NetbirdVersion != "" {
		t.Fatalf("oversized status file probe result: %+v", res)
	}
}

func TestAG_D053_Review_Held_OldAgentDoesNotOverwriteCandidateAttestation(t *testing.T) {
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	if err := r.eng.Enroll(context.Background(), "token-test-only"); err != nil {
		t.Fatal(err)
	}

	// Issue a challenge for a new candidate release (different from r.eng.cfg.ReleaseID)
	// and write the candidate's valid attestation.
	candidateRelease := uuid.Must(uuid.NewV7()).String()
	ch := issueChallenge(t, r, func(in *healthattest.Input) {
		in.ReleaseID = candidateRelease
	})
	candAtt := healthattest.Attestation{
		TxnID:          ch.TxnID,
		Nonce:          ch.Nonce,
		ReleaseID:      candidateRelease,
		NodeID:         ch.NodeID,
		AgentVersion:   ch.Components.NodeAgent,
		NetbirdVersion: ch.Components.Netbird,
		BootID:         ch.BootID,
		IncarnationID:  ch.IncarnationID,
		GeneratedAt:    ch.IssuedAt.Add(time.Millisecond),
		Published:      api.PublicationIntentUnpublished,
		Checks: healthattest.Checks{
			InternetOnly: healthattest.Check{Status: api.HealthCheckStatusPassed, Reason: "baseline held"},
			DNSPolicy:    healthattest.Check{Status: api.HealthCheckStatusPassed, Reason: "policy in force"},
			TunnelProbe:  healthattest.Check{Status: api.HealthCheckStatusPassed, Reason: "overlay_up"},
		},
	}
	if err := healthattest.WriteAttestation(r.eng.cfg.JournalDir, candAtt); err != nil {
		t.Fatal(err)
	}
	before := readFile(t, attestPath(r))

	// Old agent (still running with old ReleaseID) boots and steps; it must not
	// overwrite or touch the candidate's health-attest.json.
	if err := r.eng.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	after := readFile(t, attestPath(r))
	if string(after) != string(before) {
		t.Fatalf("old agent overwrote candidate attestation:\nbefore: %s\nafter:  %s", before, after)
	}
}
