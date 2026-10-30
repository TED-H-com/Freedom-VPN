package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// IssueHealthChallenge builds a challenge with healthattest.Issue and
// returns its JSON. The netlab module cannot import the internal package.
// The caller writes the bytes as health-challenge.json, mode 0600.
func IssueHealthChallenge(txnID, releaseID, nodeID, incarnationID, bootID string, now time.Time, ttl time.Duration, nodeAgent, netbird, updater, enforcement string) ([]byte, error) {
	ch, err := healthattest.Issue(healthattest.Input{
		TxnID:         txnID,
		ReleaseID:     releaseID,
		NodeID:        nodeID,
		IncarnationID: incarnationID,
		BootID:        bootID,
		Now:           now,
		TTL:           ttl,
		Components: healthattest.Components{
			NodeAgent:   nodeAgent,
			Netbird:     netbird,
			NodeUpdater: updater,
			Enforcement: enforcement,
		},
	})
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(ch)
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}

// ValidateHealthAnswer is healthattest.Validate for a challenge and an
// attestation already encoded as JSON. The netlab module lives outside
// node/ and cannot import node/internal/healthattest. manifest versions
// are the release's node_agent, netbird, node_updater and enforcement
// strings. An empty reason means the binding held. A non-nil error means
// the JSON itself could not be read.
func ValidateHealthAnswer(challenge, attestation []byte, now time.Time, nodeAgent, netbird, updater, enforcement string) (string, error) {
	var ch healthattest.Challenge
	if err := json.Unmarshal(challenge, &ch); err != nil {
		return "", err
	}
	var att healthattest.Attestation
	if err := json.Unmarshal(attestation, &att); err != nil {
		return "", err
	}
	return string(healthattest.Validate(ch, att, now, healthattest.Components{
		NodeAgent:   nodeAgent,
		Netbird:     netbird,
		NodeUpdater: updater,
		Enforcement: enforcement,
	})), nil
}

// evaluate runs the data-path probe and Decide. A live matching challenge
// whose probe is not strictly newer than issued_at is probed once more
// before Decide, so the step does not keep a result from before the challenge.
func (e *Engine) evaluate(ctx context.Context) {
	e.refreshProbe(ctx)
	if ch, ok := e.matchingChallenge(); ok && !e.probeFresh(ch) {
		now := e.trustedNow().UTC()
		if e.anchorOK && now.After(ch.IssuedAt) && now.Before(ch.ExpiresAt) {
			e.refreshProbe(ctx)
		}
	}
	e.applyResolver(ctx)
	e.decision = Decide(e.input())
}

// answerHealthOnBoot answers at start only when a verified desired document
// is already in force. It does not probe when no challenge is present, and
// it does not install member forwarding.
func (e *Engine) answerHealthOnBoot(ctx context.Context) {
	if _, ok := e.matchingChallenge(); !ok {
		return
	}
	if !e.healthAnswerReady() {
		return
	}
	e.refreshDesired()
	e.evaluate(ctx)
	e.answerHealth(ctx)
}

// answerHealth writes health-attest.json for a challenge that names this
// process. The file reports the latest publication intent and the checks
// this process just ran. It does not change publication and it does not
// open member forwarding. A missing or foreign challenge is ignored.
func (e *Engine) answerHealth(ctx context.Context) {
	ch, ok := e.matchingChallenge()
	if !ok {
		return
	}
	now := e.trustedNow().UTC()
	if !e.anchorOK {
		e.decision = Decide(e.input())
		return
	}
	if !now.After(ch.IssuedAt) || !now.Before(ch.ExpiresAt) {
		return
	}
	// Write only after a verified document is in hand. A transport error,
	// a 304 with nothing loaded, or a body that fails verification leaves
	// desired nil and writes nothing. The updater keeps polling until its
	// deadline. That deadline, not a premature failed answer, is fail-closed.
	if !e.healthAnswerReady() {
		return
	}
	e.refreshDesired()
	if !e.probeFresh(ch) || (ch.ProbeOmitted != nil && e.probedChallenge == nil) {
		e.refreshProbe(ctx)
		if !e.probeFresh(ch) {
			return
		}
	}
	// Re-read the authority after probing: replacement or expiry during the
	// probe must not attach that result to a different issuance.
	live, ok := e.matchingChallenge()
	now = e.trustedNow().UTC()
	if !e.anchorOK || !ok || !live.Same(ch) || !now.After(ch.IssuedAt) || !now.Before(ch.ExpiresAt) {
		return
	}
	if (ch.Probe != nil || ch.ProbeOmitted != nil) && (e.probedChallenge == nil || !e.probedChallenge.Same(ch)) {
		return
	}
	// Enforce or exception cleanup may have failed after the earlier Decide.
	// Recompute health without losing operational apply errors in the report.
	d := Decide(e.input())
	if e.cfg.AgentRelease == "" || !d.Publication.Valid() {
		return
	}
	checks, ok := attestChecks(d, e.lastProbe, e.probeOmitted != "")
	if !ok {
		return
	}
	att := healthattest.Attestation{
		TxnID:          ch.TxnID,
		Nonce:          ch.Nonce,
		ReleaseID:      ch.ReleaseID,
		NodeID:         ch.NodeID,
		AgentVersion:   e.cfg.AgentRelease,
		NetbirdVersion: netbirdVersion(e.lastProbe.NetbirdVersion),
		BootID:         ch.BootID,
		IncarnationID:  ch.IncarnationID,
		GeneratedAt:    now,
		Published:      d.Publication,
		Checks:         checks,
	}
	sig := healthSignature(att)
	if e.healthNonce == att.Nonce && e.healthSig == sig && attestFileOK(e.cfg.JournalDir) {
		return
	}
	if err := healthattest.WriteAttestation(e.cfg.JournalDir, att); err != nil {
		e.log.Warn("health attestation", "err", err.Error())
		return
	}
	e.healthNonce = att.Nonce
	e.healthSig = sig
}

// matchingChallenge is the live challenge for this node, incarnation, boot
// and release. Expiry is left to the caller. A consumed, malformed, or
// foreign file does not match.
func (e *Engine) matchingChallenge() (healthattest.Challenge, bool) {
	if e.cfg.JournalDir == "" || !e.enrolled || e.nodeID == uuid.Nil {
		return healthattest.Challenge{}, false
	}
	ch, _, err := healthattest.ReadChallenge(e.cfg.JournalDir)
	if err != nil || ch.Consumed {
		return healthattest.Challenge{}, false
	}
	if !sameUUID(ch.NodeID, e.nodeID) || !sameUUID(ch.IncarnationID, e.incID) || !sameUUID(ch.BootID, e.bootID) || !sameUUID(ch.ReleaseID, e.cfg.ReleaseUUID()) {
		return healthattest.Challenge{}, false
	}
	return ch, true
}

// healthAnswerReady is a verified desired document from a completed sync.
// Loading an in-force stored envelope counts: that sync finished in an
// earlier process. A transport error does not.
func (e *Engine) healthAnswerReady() bool {
	return e.desired != nil && e.desiredSynced
}

func (e *Engine) probeFresh(ch healthattest.Challenge) bool {
	// A validated omission is conclusive not_run evidence even when no
	// tunnel probe ran. answerHealth separately checks the exact issuance;
	// a rewritten challenge must not trigger another probe in this pass.
	return (e.lastProbe.Ran || ch.ProbeOmitted != nil) && e.probeAt.After(ch.IssuedAt)
}

func sameUUID(got string, want uuid.UUID) bool {
	if want == uuid.Nil || got == "" {
		return false
	}
	id, err := uuid.Parse(got)
	if err != nil || id == uuid.Nil {
		return false
	}
	return id == want
}

func attestChecks(d Decision, probe ProbeResult, omitted bool) (healthattest.Checks, bool) {
	internet, okI := decisionStatus(d, api.NodeCheckNameInternetOnly)
	dns, okD := decisionStatus(d, api.NodeCheckNameDnsPolicy)
	tunnel, okT := decisionStatus(d, api.NodeCheckNameTunnelProbe)
	if !okI || !okD || !okT || !internet.Valid() || !dns.Valid() || !tunnel.Valid() {
		return healthattest.Checks{}, false
	}
	reason := tunnelReason(tunnel, probe.Detail)
	if omitted {
		// The shared attestation vocabulary already has this fixed token.
		// NodeCheck.detail and its issuer refusal class never become Reason.
		reason = "probe refused"
	}
	return healthattest.Checks{
		InternetOnly: healthattest.Check{Status: internet, Reason: statusReason(internet, "baseline held", "baseline missing")},
		DNSPolicy:    healthattest.Check{Status: dns, Reason: statusReason(dns, "policy in force", "policy not in force")},
		TunnelProbe:  healthattest.Check{Status: tunnel, Reason: reason},
	}, true
}

func decisionStatus(d Decision, name api.NodeCheckName) (api.HealthCheckStatus, bool) {
	for _, c := range d.Checks {
		if c.Name == name {
			return c.Status, true
		}
	}
	return "", false
}

func statusReason(st api.HealthCheckStatus, pass, fail string) string {
	switch st {
	case api.HealthCheckStatusPassed:
		return pass
	case api.HealthCheckStatusFailed:
		return fail
	case api.HealthCheckStatusNotRun:
		return "not run"
	default:
		return ""
	}
}

func tunnelReason(st api.HealthCheckStatus, detail string) string {
	if st == api.HealthCheckStatusNotRun {
		return "probe not run"
	}
	if detailToken(detail) {
		return detail
	}
	if st == api.HealthCheckStatusPassed {
		return "overlay up"
	}
	return "probe failed"
}

func detailToken(s string) bool {
	if s == "" || len(s) > 40 {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

func healthSignature(a healthattest.Attestation) string {
	return strings.Join([]string{
		a.TxnID, a.Nonce, a.ReleaseID, a.NodeID, a.IncarnationID, a.BootID,
		a.AgentVersion, a.NetbirdVersion, string(a.Published),
		string(a.Checks.InternetOnly.Status), a.Checks.InternetOnly.Reason,
		string(a.Checks.DNSPolicy.Status), a.Checks.DNSPolicy.Reason,
		string(a.Checks.TunnelProbe.Status), a.Checks.TunnelProbe.Reason,
	}, "\n")
}

func attestFileOK(dir string) bool {
	info, err := os.Lstat(filepath.Join(dir, healthattest.AttestationFile))
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return false
	}
	return info.Mode().Perm() == 0o600
}
