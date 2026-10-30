package updater

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// Health-attestation window (D-053, §13.9). These are suggested defaults
// pending measurement (待實測), not a verified SLO. The challenge outlives
// one wait so an answer written at the end of the window is still inside
// expires_at. A config file cannot change them.
const (
	HealthAttestWait   = 30 * time.Second
	HealthAttestPoll   = 250 * time.Millisecond
	HealthChallengeTTL = 45 * time.Second
)

// AttestWindow is the per-attempt wait, the poll interval and the challenge
// lifetime. Zero fields use the defaults above.
type AttestWindow struct {
	Wait time.Duration
	Poll time.Duration
	TTL  time.Duration
}

func (e *Engine) attestWindow() AttestWindow {
	a := e.Attest
	if a.Wait <= 0 {
		a.Wait = HealthAttestWait
	}
	if a.Poll <= 0 {
		a.Poll = HealthAttestPoll
	}
	if a.TTL <= a.Wait {
		a.TTL = a.Wait + a.Poll
	}
	return a
}

// healthDetail is the journal record for one health evaluation. Statuses are
// what the updater decided. The agent_* fields are the raw attestation, so a
// not_run answer stays visible next to the failed status.
type healthDetail struct {
	Kind                    string           `json:"kind"`
	Phase                   string           `json:"phase"`
	Reason                  string           `json:"reason,omitempty"`
	Statuses                api.UpdateHealth `json:"statuses"`
	InternetOnlyReason      string           `json:"internet_only_reason,omitempty"`
	DNSPolicyReason         string           `json:"dns_policy_reason,omitempty"`
	TunnelProbeReason       string           `json:"tunnel_probe_reason,omitempty"`
	AgentInternet           string           `json:"agent_internet_only,omitempty"`
	AgentDNS                string           `json:"agent_dns_policy,omitempty"`
	AgentTunnel             string           `json:"agent_tunnel_probe,omitempty"`
	AgentVersion            string           `json:"agent_version,omitempty"`
	NetbirdVersion          string           `json:"netbird_version,omitempty"`
	Published               string           `json:"published,omitempty"`
	UpdaterVersionCheck     string           `json:"updater_version_check"`
	EnforcementVersionCheck string           `json:"enforcement_version_check"`
}

// checkRT is the shared state of one NodeChecks value. The engine keeps a
// pointer so Bind is visible to the value methods on the Health interface.
type checkRT struct {
	mu    sync.Mutex
	bound *healthattest.Challenge
	prior healthattest.Observed
	phase string
	cache *checkSnap
}

type checkSnap struct {
	nonce   string
	release string
	health  api.UpdateHealth
	detail  healthDetail
}

// Bind installs the challenge this evaluation must match. prior is the
// attestation path as it looked before that challenge was written.
func (n *NodeChecks) Bind(ch healthattest.Challenge, prior healthattest.Observed, phase string) {
	if n.rt == nil {
		n.rt = &checkRT{}
	}
	n.rt.mu.Lock()
	n.rt.bound = &ch
	n.rt.prior = prior
	n.rt.phase = phase
	n.rt.cache = nil
	n.rt.mu.Unlock()
}

// Detail returns the last evaluation, including reasons.
func (n NodeChecks) Detail() (healthDetail, bool) {
	if n.rt == nil {
		return healthDetail{}, false
	}
	n.rt.mu.Lock()
	defer n.rt.mu.Unlock()
	if n.rt.cache == nil {
		return healthDetail{}, false
	}
	return n.rt.cache.detail, true
}

func (n NodeChecks) attestStatus(ctx context.Context, v ReleaseView, pick func(api.UpdateHealth) api.HealthCheckStatus) api.HealthCheckStatus {
	if n.Journal == "" || n.rt == nil {
		return api.HealthCheckStatusFailed
	}
	return pick(n.evaluate(ctx, v).health)
}

func (n NodeChecks) evaluate(ctx context.Context, v ReleaseView) checkSnap {
	n.rt.mu.Lock()
	bound := n.rt.bound
	prior := n.rt.prior
	phase := n.rt.phase
	if n.rt.cache != nil && bound != nil && n.rt.cache.nonce == bound.Nonce && n.rt.cache.release == v.ReleaseID {
		s := *n.rt.cache
		n.rt.mu.Unlock()
		return s
	}
	n.rt.mu.Unlock()
	var s checkSnap
	if bound == nil {
		s = n.decide(v, healthattest.Challenge{}, nil, healthattest.ReasonMissing, phase)
	} else {
		s = n.poll(ctx, v, *bound, prior, phase)
	}
	n.rt.mu.Lock()
	n.rt.cache = &s
	n.rt.mu.Unlock()
	return s
}

func (n NodeChecks) poll(ctx context.Context, v ReleaseView, bound healthattest.Challenge, prior healthattest.Observed, phase string) checkSnap {
	deadline := n.now().Add(n.wait())
	// The injected clock controls the deadline. Also bound a stalled clock
	// so clock adjustments cannot keep an updater waiting indefinitely.
	ctx, cancel := context.WithTimeout(ctx, n.wait())
	defer cancel()
	var last healthattest.Reason = healthattest.ReasonMissing
	var got *healthattest.Attestation
	for {
		reason, at := n.readOnce(bound, prior)
		if at != nil {
			last, got = reason, at
			if reason == "" && at.Checks.InternetOnly.Status == api.HealthCheckStatusPassed &&
				at.Checks.DNSPolicy.Status == api.HealthCheckStatusPassed && at.Checks.TunnelProbe.Status == api.HealthCheckStatusPassed {
				break
			}
		} else if got == nil {
			last = reason
		}
		if !n.now().Before(deadline) || ctx.Err() != nil {
			break
		}
		if err := n.sleep(ctx, n.pollInterval()); err != nil {
			break
		}
	}
	return n.decide(v, bound, got, last, phase)
}

func (n NodeChecks) readOnce(bound healthattest.Challenge, prior healthattest.Observed) (healthattest.Reason, *healthattest.Attestation) {
	fileCh, _, err := healthattest.ReadChallenge(n.Journal)
	if err != nil {
		if r := healthattest.ReasonOf(err); r != "" {
			return r, nil
		}
		return healthattest.ReasonChallengeReplaced, nil
	}
	if !fileCh.Same(bound) {
		return healthattest.ReasonChallengeReplaced, nil
	}
	if n.Node.ID.String() != "" && fileCh.NodeID != n.Node.ID.String() {
		return healthattest.ReasonNodeMismatch, nil
	}
	at, cur, err := healthattest.ReadObservedAttestation(n.Journal)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return healthattest.ReasonMissing, nil
		}
		if r := healthattest.ReasonOf(err); r != "" {
			return r, nil
		}
		return healthattest.ReasonUnreadable, nil
	}
	if !cur.Rewritten(prior) {
		if prior.Present {
			return healthattest.ReasonPreseeded, nil
		}
		return healthattest.ReasonMissing, nil
	}
	reason := healthattest.Validate(bound, at, n.now(), bound.Components)
	if reason != "" && reason != healthattest.ReasonNotRun {
		return reason, nil
	}
	return reason, &at
}

func (n NodeChecks) decide(v ReleaseView, bound healthattest.Challenge, at *healthattest.Attestation, reason healthattest.Reason, phase string) checkSnap {
	h := api.UpdateHealth{
		Processes:        checkProcesses(v),
		IdentityAndState: checkPersistent(v),
		InternetOnly:     api.HealthCheckStatusFailed,
		DnsPolicy:        api.HealthCheckStatusFailed,
		TunnelProbe:      api.HealthCheckStatusFailed,
	}
	d := healthDetail{
		Kind:                    "health_attest",
		Phase:                   phase,
		Reason:                  string(reason),
		UpdaterVersionCheck:     "tree_hash_only",
		EnforcementVersionCheck: "tree_hash_only",
	}
	versionsOK := false
	if at != nil && (reason == "" || reason == healthattest.ReasonNotRun) {
		versionsOK = true
		d.AgentVersion = at.AgentVersion
		d.NetbirdVersion = at.NetbirdVersion
		d.Published = string(at.Published)
		d.InternetOnlyReason = at.Checks.InternetOnly.Reason
		d.DNSPolicyReason = at.Checks.DNSPolicy.Reason
		d.TunnelProbeReason = at.Checks.TunnelProbe.Reason
		d.AgentInternet = string(at.Checks.InternetOnly.Status)
		d.AgentDNS = string(at.Checks.DNSPolicy.Status)
		d.AgentTunnel = string(at.Checks.TunnelProbe.Status)
		h.InternetOnly = closeNotRun(at.Checks.InternetOnly.Status)
		h.DnsPolicy = closeNotRun(at.Checks.DNSPolicy.Status)
		h.TunnelProbe = closeNotRun(at.Checks.TunnelProbe.Status)
	}
	if h.Processes != api.HealthCheckStatusPassed || !versionsOK {
		h.Processes = api.HealthCheckStatusFailed
	}
	d.Statuses = h
	return checkSnap{nonce: bound.Nonce, release: v.ReleaseID, health: h, detail: d}
}

func closeNotRun(st api.HealthCheckStatus) api.HealthCheckStatus {
	if st == api.HealthCheckStatusPassed {
		return st
	}
	return api.HealthCheckStatusFailed
}

func (n NodeChecks) now() time.Time {
	if n.Now != nil {
		return n.Now()
	}
	return time.Now()
}

func (n NodeChecks) wait() time.Duration {
	if n.Wait > 0 {
		return n.Wait
	}
	return HealthAttestWait
}

func (n NodeChecks) pollInterval() time.Duration {
	if n.Poll > 0 {
		return n.Poll
	}
	return HealthAttestPoll
}

func (n NodeChecks) sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	if n.Sleep != nil {
		return n.Sleep(ctx, d)
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (e *Engine) nodeChecks() *NodeChecks {
	nc, ok := e.Health.(*NodeChecks)
	if !ok {
		return nil
	}
	if nc.rt == nil {
		nc.rt = &checkRT{}
	}
	if nc.Journal == "" {
		nc.Journal = e.Layout.Journal
	}
	nc.Node = e.Node
	nc.Now = e.now
	nc.Sleep = e.sleepCtx
	w := e.attestWindow()
	nc.Wait = w.Wait
	nc.Poll = w.Poll
	return nc
}

// issueChallenge writes one live challenge for this check. A file that was
// already at the attestation path is not an answer (D-053).
func (e *Engine) issueChallenge(ctx context.Context, txn updatejournal.Txn, releaseID, phase string) error {
	m := e.manifestOf(ctx, releaseID)
	if m == nil {
		e.noteHealth(ctx, txn.TxnID, e.blankHealth(ctx, releaseID), healthattest.ReasonManifestUnavailable, phase)
		return errChallenge
	}
	prior, oerr := healthattest.ObserveAttestation(e.Layout.Journal)
	if oerr != nil {
		if r := healthattest.ReasonOf(oerr); r != "" {
			_ = e.Journal.RecordEvent(ctx, txn.TxnID, healthEvent(phase, r))
		}
		prior = healthattest.Observed{}
	}
	ch, err := healthattest.Issue(healthattest.Input{
		TxnID: txn.TxnID, ReleaseID: releaseID,
		NodeID: e.Node.ID.String(), IncarnationID: e.Node.Incarnation.String(), BootID: e.Node.BootID.String(),
		Now: e.now(), TTL: e.attestWindow().TTL,
		Components: healthattest.Components{
			NodeAgent: m.Components.NodeAgent, Netbird: m.Components.Netbird,
			NodeUpdater: m.Components.NodeUpdater, Enforcement: m.Components.Enforcement,
		},
	})
	if err != nil {
		e.noteHealth(ctx, txn.TxnID, e.blankHealth(ctx, releaseID), healthattest.ReasonMalformed, phase)
		return err
	}
	if err := healthattest.WriteChallenge(e.Layout.Journal, ch); err != nil {
		e.noteHealth(ctx, txn.TxnID, e.blankHealth(ctx, releaseID), healthattest.ReasonUnreadable, phase)
		return err
	}
	if prior.Present {
		_ = e.Journal.RecordEvent(ctx, txn.TxnID, healthEvent(phase, healthattest.ReasonPreseeded))
	}
	if nc := e.nodeChecks(); nc != nil {
		nc.Bind(ch, prior, phase)
	}
	return nil
}

func (e *Engine) consumeChallenge() {
	if err := healthattest.Consume(e.Layout.Journal); err != nil {
		if ch, _, rerr := healthattest.ReadChallenge(e.Layout.Journal); rerr == nil {
			_ = healthattest.MarkConsumed(e.Layout.Journal, ch)
		}
	}
}

func (e *Engine) recordHealth(ctx context.Context, txnID string, h api.UpdateHealth) {
	_ = e.Journal.RecordHealth(ctx, txnID, h)
	nc, ok := e.Health.(*NodeChecks)
	if !ok {
		return
	}
	d, ok := nc.Detail()
	if !ok {
		return
	}
	d.Statuses = h
	b, err := json.Marshal(d)
	if err != nil {
		return
	}
	_ = e.Journal.RecordEvent(ctx, txnID, string(b))
}

func (e *Engine) noteHealth(ctx context.Context, txnID string, h api.UpdateHealth, reason healthattest.Reason, phase string) {
	d := healthDetail{
		Kind: "health_attest", Phase: phase, Reason: string(reason), Statuses: h,
		UpdaterVersionCheck: "tree_hash_only", EnforcementVersionCheck: "tree_hash_only",
	}
	if b, err := json.Marshal(d); err == nil {
		_ = e.Journal.RecordEvent(ctx, txnID, string(b))
	}
}

func (e *Engine) blankHealth(ctx context.Context, releaseID string) api.UpdateHealth {
	view := e.viewFor(ctx, releaseID, e.manifestOf(ctx, releaseID))
	return api.UpdateHealth{
		Processes:        api.HealthCheckStatusFailed,
		IdentityAndState: checkPersistent(view),
		InternetOnly:     api.HealthCheckStatusFailed,
		DnsPolicy:        api.HealthCheckStatusFailed,
		TunnelProbe:      api.HealthCheckStatusFailed,
	}
}

func healthEvent(phase string, reason healthattest.Reason) string {
	b, err := json.Marshal(healthDetail{
		Kind: "health_attest", Phase: phase, Reason: string(reason),
		UpdaterVersionCheck: "tree_hash_only", EnforcementVersionCheck: "tree_hash_only",
		Statuses: api.UpdateHealth{
			Processes: api.HealthCheckStatusFailed, IdentityAndState: api.HealthCheckStatusFailed,
			InternetOnly: api.HealthCheckStatusFailed, DnsPolicy: api.HealthCheckStatusFailed,
			TunnelProbe: api.HealthCheckStatusFailed,
		},
	})
	if err != nil {
		return `{"kind":"health_attest","reason":"` + string(reason) + `"}`
	}
	return string(b)
}

// errChallenge means this attempt could not publish a challenge.
var errChallenge = errors.New("updater: health challenge")
