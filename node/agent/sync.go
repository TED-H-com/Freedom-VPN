package agent

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/TED-H-com/Freedom-VPN/node/internal/controlclient"
	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

func (e *Engine) syncDesired(ctx context.Context) error {
	// No verified document means no validator. If-None-Match is omitted,
	// so an expired envelope stored from the previous process cannot 304.
	match := ""
	if e.desired != nil {
		match = e.desiredETag
	}
	body, etag, err := e.client.DesiredState(ctx, match)
	if errors.Is(err, controlclient.ErrNotModified) {
		// A 304 is not a new document and does not move the lease (§14.7).
		e.noteDesiredSync()
		return nil
	}
	if isRevoked(err) {
		e.markRevoked(ctx)
		e.noteDesiredSync()
		return nil
	}
	if err != nil {
		// A transport error is not a completed sync. With no document in
		// hand the updater keeps polling until its deadline.
		if e.desired != nil {
			e.noteDesiredSync()
		}
		e.log.Warn("desired state", "err", safeErr(err))
		return nil
	}
	e.noteDesiredSync()
	now, accept := e.verifyTime()
	if !accept {
		return nil
	}
	ds, err := verifyDesired(body, e.trust, now)
	if err != nil || !desiredInForce(ds, e.nodeID, e.incID, now) {
		e.log.Warn("desired state rejected")
		return nil
	}
	gen := uint64(ds.ConfigGeneration)
	err = e.store.AcceptDesired(ctx, agentstore.Desired{
		Generation: gen, ETag: etag, Envelope: body,
		IssuedAt: ds.IssuedAt.UTC(), ExpiresAt: ds.ExpiresAt.UTC(),
		Publication: string(ds.PublicationIntent),
	})
	if errors.Is(err, agentstore.ErrStaleGeneration) {
		return nil
	}
	if err != nil {
		e.storageOK = false
		return err
	}
	if err := writeAtomic(e.cfg.DesiredStatePath, body, 0o644); err != nil {
		e.log.Warn("desired state file", "err", err.Error())
	}
	e.desired = &ds
	e.desiredETag = etag
	e.desiredOK = true
	e.desiredExpired = false
	return nil
}

func (e *Engine) syncLease(ctx context.Context) error {
	if e.leaseAttempted || e.clock.Mono() < e.leaseRetryAt {
		return nil
	}
	if !e.leaseGrantless() {
		e.provisionalStreak = 0
		e.provisionalRetryAt = 0
	}
	if !e.needRenew() {
		return nil
	}
	mono := e.clock.Mono()
	wall := e.clock.Wall()
	if mono <= 0 {
		return nil
	}
	reqID, err := uuid.NewV7()
	if err != nil {
		return err
	}
	req := api.LeaseRenewalRequest{
		AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
		CurrentLeaseId:          presence.NewNullNullable[uuid.UUID](),
		NodeId:                  e.nodeID,
		NodeIncarnationId:       e.incID,
		RenewalRequestId:        reqID,
	}
	if !e.identityFrozen && !e.freezeUncertain && e.applyError == nil && e.appliedGen > 0 {
		req.AppliedConfigGeneration = presence.NewNullableWithValue(decstr.U64(e.appliedGen))
	}
	if e.lease != nil && e.lease.LeaseId != uuid.Nil {
		req.CurrentLeaseId = presence.NewNullableWithValue(e.lease.LeaseId)
	}
	e.leaseAttempted = true
	e.leaseAttemptAt = mono
	raw, err := e.client.RenewLeaseOnce(ctx, reqID, req)
	if isRevoked(err) {
		e.markRevoked(ctx)
		return nil
	}
	if err != nil {
		var status *controlclient.StatusError
		if errors.As(err, &status) && status.RetryAfter > 0 {
			remaining := status.RetryAfter
			if !status.RetryAt.IsZero() {
				remaining = max(0, time.Until(status.RetryAt))
			}
			e.leaseRetryAt = e.clock.Mono() + remaining
			e.decision = Decide(e.input())
			if !e.decision.Serve {
				e.withdraw(context.WithoutCancel(ctx))
			}
		}
		e.log.Warn("lease", "err", safeErr(err))
		return ctx.Err()
	}
	e.leaseRetryAt = 0
	return e.acceptLease(ctx, raw, mono, wall)
}

func (e *Engine) needRenew() bool {
	if e.leaseRetryAt > 0 || !e.leaseFresh || e.lease == nil || e.leaseMono <= 0 || !e.anchorOK || (!e.identityFrozen && ((e.applyError == nil && e.leaseBehindApplied()) || (e.applyError != nil && e.leaseBehindDesired()))) {
		return true
	}
	// D-067 bounds every grant-less renewal, including an empty lease
	// received before desired bindings arrive. Reporting remains separate:
	// only in-force matching bindings report AWAITING_MEMBER_GRANTS.
	if e.leaseGrantless() {
		return e.provisionalRetryAt == 0 || e.clock.Mono() >= e.provisionalRetryAt
	}
	window := e.lease.ExpiresAt.Sub(e.lease.IssuedAt)
	deadline, ok := timeanchor.MemberDeadline(e.leaseMono, window)
	if !ok {
		return true
	}
	return timeanchor.Remaining(deadline, e.clock.Mono()) <= e.renewLead(window)+leadJitter(e.renewLead(window))
}

// A retained failure must not suppress acquisition of the desired generation's
// authorization. syncLease still omits the applied-generation ACK on that
// request; only a fresh, clean Apply can clear the failure and acknowledge it.
func (e *Engine) leaseBehindDesired() bool {
	return e.lease != nil && e.desired != nil && e.desiredOK && !e.desiredExpired &&
		e.desired.ConfigGeneration != 0 && e.lease.ConfigGeneration != e.desired.ConfigGeneration
}

func (e *Engine) leaseGrantless() bool {
	return e.leaseFresh && e.lease != nil && (e.identityFrozen || len(e.lease.Members) == 0)
}

// provisionalRenewCap is the longest gap between grant-less renewals.
// The first wait is at most one heartbeat. Every wait respects this cap.
const provisionalRenewCap = 60 * time.Second

func (e *Engine) heartbeatInterval() time.Duration {
	sec := e.cfg.HeartbeatSeconds
	if sec <= 0 {
		sec = 10
	}
	return time.Duration(sec) * time.Second
}

// provisionalWait is the delay after the streak-th provisional lease.
// Streak 1 waits one heartbeat, capped even when the heartbeat is longer.
// Each later wait doubles and then stops at provisionalRenewCap.
func (e *Engine) provisionalWait(streak int) time.Duration {
	hb := e.heartbeatInterval()
	if streak < 1 {
		streak = 1
	}
	wait := min(hb, provisionalRenewCap)
	for i := 1; i < streak; i++ {
		if wait >= provisionalRenewCap/2 {
			return provisionalRenewCap
		}
		wait *= 2
	}
	return wait
}

// noteProvisionalSchedule arms the next grant-less renewal, or clears the
// schedule when this lease carries grants. Anchor the wait to the request,
// not its response, and never let backoff postpone the lease's renewal lead.
func (e *Engine) noteProvisionalSchedule(monoSent time.Duration) {
	if !e.leaseGrantless() {
		e.provisionalStreak = 0
		e.provisionalRetryAt = 0
		return
	}
	if monoSent <= 0 {
		return
	}
	e.provisionalStreak++
	window := e.lease.ExpiresAt.Sub(e.lease.IssuedAt)
	leadAt := e.leaseMono + max(0, window-e.renewLead(window))
	e.provisionalRetryAt = min(monoSent+e.provisionalWait(e.provisionalStreak), leadAt)
}

// leaseBehindApplied is a stored lease whose config generation is not the
// generation this process has applied. The deadline path is unchanged.
// appliedGen 0 means nothing has been applied yet, so the generation
// comparison waits.
func (e *Engine) leaseBehindApplied() bool {
	if e.appliedGen == 0 || e.lease == nil {
		return false
	}
	return uint64(e.lease.ConfigGeneration) != e.appliedGen
}

// renewAppliedLease runs after this step applies a generation. One renewal
// is attempted in that same step. A lease that comes back for the applied
// generation is what the rest of the step serves. A failure leaves the
// decision already computed: forwarding stays off, and the reason is not
// lease_expired.
func (e *Engine) renewAppliedLease(ctx context.Context) error {
	if e.identityFrozen || e.applyError != nil || e.dependencyMissing || !e.leaseBehindApplied() {
		return nil
	}
	if err := e.syncLease(ctx); err != nil && ctx.Err() != nil {
		return err
	}
	if e.lease == nil || !e.leaseFresh || uint64(e.lease.ConfigGeneration) != e.appliedGen {
		return nil
	}
	e.evaluate(ctx)
	if err := e.enforce(ctx, e.decision); err != nil && ctx.Err() != nil {
		return err
	}
	return nil
}

func leadJitter(lead time.Duration) time.Duration {
	span := lead / 10
	if span <= 0 {
		return 0
	}
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0
	}
	n := int(b[0])<<8 | int(b[1])
	mod := int(span)*2 + 1
	return time.Duration(n%mod) - span
}

func (e *Engine) acceptLease(ctx context.Context, raw []byte, monoSent time.Duration, wall time.Time) error {
	sigNow := wall
	if e.anchorOK {
		if t, err := e.anchor.TrustedNow(e.bootID, e.clock, timeanchor.DefaultBackwardSkew, timeanchor.DefaultForwardSkew); err == nil {
			sigNow = t
		}
	}
	lease, err := verifyLease(raw, e.trust, sigNow)
	if err != nil {
		e.log.Warn("lease rejected")
		return nil
	}
	if lease.NodeId != e.nodeID || lease.NodeIncarnationId != e.incID || lease.SchemaVersion != api.ServingLeaseSchemaVersionN1 || !lease.ExpiresAt.After(lease.IssuedAt) || lease.MaxOfflineSeconds < 60 || lease.MaxOfflineSeconds > 86400 {
		e.log.Warn("lease rejected")
		return nil
	}
	sameBoot := e.leaseBoot == uuid.Nil || e.leaseBoot == e.bootID
	replay := e.lease != nil && lease.LeaseId == e.lease.LeaseId && !e.highestIssued.IsZero() && !lease.IssuedAt.After(e.highestIssued)
	if replay {
		if sameBoot {
			// Same lease id keeps the original monotonic send time (§16.3).
			e.leaseFresh = true
			e.noteProvisionalSchedule(monoSent)
		}
		return nil
	}
	if !e.highestIssued.IsZero() && !lease.IssuedAt.After(e.highestIssued) {
		return nil
	}
	if monoSent <= 0 {
		return nil
	}
	e.lease = &lease
	e.leaseMono = monoSent
	e.leaseBoot = e.bootID
	e.highestIssued = lease.IssuedAt.UTC()
	e.leaseFresh = true
	e.anchor = timeanchor.Anchor{
		BootID: e.bootID, Server: lease.IssuedAt.UTC(), Mono: monoSent, Wall: wall.UTC(), Trusted: true,
	}
	e.anchorOK = true
	if err := e.store.SaveLease(ctx, agentstore.Lease{
		ID: lease.LeaseId, IssuedAt: lease.IssuedAt.UTC(), ExpiresAt: lease.ExpiresAt.UTC(),
		HighestIssuedAt: e.highestIssued, Envelope: raw, MonoSent: monoSent, BootID: e.bootID,
	}); err != nil {
		e.storageOK = false
		return err
	}
	if err := e.store.SaveAnchor(ctx, agentstore.Anchor{
		BootID: e.bootID, Server: e.anchor.Server, Mono: e.anchor.Mono, Wall: e.anchor.Wall, Trusted: true,
	}); err != nil {
		e.storageOK = false
		return err
	}
	e.noteProvisionalSchedule(monoSent)
	return nil
}

func (e *Engine) syncCert(ctx context.Context) error {
	id, ok, err := e.store.Identity(ctx)
	if err != nil || !ok {
		return err
	}
	now, good := e.verifyTime()
	if !good {
		now = e.clock.Wall()
	}
	if id.NotAfter.Sub(now) > certRenewLead {
		return nil
	}
	key, err := loadOrCreateKey(e.cfg.keyPath())
	if err != nil {
		return err
	}
	csr, reqID, err := e.persistedRenewal(ctx, key)
	if err != nil {
		return err
	}
	resp, err := e.client.RenewCertificate(ctx, reqID, api.CertificateRenewRequest{
		CsrPem: csr, NodeId: e.nodeID, NodeIncarnationId: e.incID, RenewalRequestId: reqID,
	})
	if isRevoked(err) {
		e.markRevoked(ctx)
		return nil
	}
	if err != nil {
		e.log.Warn("certificate renewal", "err", safeErr(err))
		return nil
	}
	cert, err := certFromPEM([]byte(resp.CertificatePem))
	if err != nil || !keyMatches(cert, key) || resp.CertificateSerial == "" {
		e.log.Warn("certificate renewal rejected")
		return nil
	}
	if err := writeAtomic(e.cfg.certPath(), []byte(resp.CertificatePem), 0o644); err != nil {
		return err
	}
	if resp.CaChainPem != "" {
		if err := writeAtomic(e.cfg.caPath(), []byte(resp.CaChainPem), 0o644); err != nil {
			return err
		}
	}
	notAfter := resp.NotAfter.UTC()
	if notAfter.IsZero() {
		notAfter = cert.NotAfter.UTC()
	}
	if err := e.store.UpdateCertificate(ctx, resp.CertificateSerial, notAfter); err != nil {
		e.storageOK = false
		return err
	}
	if err := writeNodeFile(e.cfg.nodePath(), nodeFile{
		NodeID: e.nodeID.String(), NodeIncarnationID: e.incID.String(),
		Serial: resp.CertificateSerial, NotAfter: notAfter,
	}); err != nil {
		return err
	}
	_ = os.Remove(e.cfg.renewCSRPath())
	_ = e.store.SetMeta(ctx, "renewal_request_id", "")
	return e.loadCert()
}

// persistedRenewal reuses the CSR and renewal id that were stored before the
// request, so a crash does not mint a second renewal (§6.4).
func (e *Engine) persistedRenewal(ctx context.Context, key *ecdsa.PrivateKey) (string, uuid.UUID, error) {
	if b, err := os.ReadFile(e.cfg.renewCSRPath()); err == nil && len(bytes.TrimSpace(b)) > 0 {
		if v, ok, err := e.store.Meta(ctx, "renewal_request_id"); err != nil {
			return "", uuid.Nil, err
		} else if ok && v != "" {
			id, err := uuid.Parse(v)
			if err == nil {
				return string(b), id, nil
			}
		}
	}
	csr, err := newCSR(key)
	if err != nil {
		return "", uuid.Nil, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return "", uuid.Nil, err
	}
	if err := writeAtomic(e.cfg.renewCSRPath(), []byte(csr), 0o600); err != nil {
		return "", uuid.Nil, err
	}
	if err := e.store.SetMeta(ctx, "renewal_request_id", id.String()); err != nil {
		return "", uuid.Nil, err
	}
	return csr, id, nil
}

func (e *Engine) syncBootstrap(ctx context.Context) error {
	if fileExists(e.cfg.setupKeyPath()) {
		e.bootstrapLost = false
		return nil
	}
	if e.bootstrapLost {
		return nil
	}
	reqID, err := uuid.NewV7()
	if err != nil {
		return err
	}
	resp, err := e.client.Bootstrap(ctx, reqID)
	if err != nil {
		var se *controlclient.StatusError
		if errors.As(err, &se) {
			switch se.Code {
			case string(api.ErrorCodeBOOTSTRAPNOTREADY):
				return nil
			case string(api.ErrorCodeBOOTSTRAPCONSUMED), string(api.ErrorCodeBOOTSTRAPEXPIRED):
				e.bootstrapLost = true
				if serr := e.store.SetMeta(ctx, "bootstrap_lost", "1"); serr != nil {
					e.storageOK = false
				}
				return nil
			}
		}
		e.log.Warn("bootstrap", "err", safeErr(err))
		return nil
	}
	if resp.SetupKey == "" {
		e.log.Warn("bootstrap empty")
		return nil
	}
	if err := writeAtomic(e.cfg.setupKeyPath(), []byte(resp.SetupKey), 0o600); err != nil {
		return err
	}
	meta, err := json.Marshal(map[string]string{
		"external_key_id": resp.ExternalKeyId,
		"expires_at":      resp.ExpiresAt.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return err
	}
	if err := writeAtomic(e.cfg.setupMetaPath(), append(meta, '\n'), 0o600); err != nil {
		return err
	}
	if e.hooks.AfterBootstrapPersist != nil {
		if err := e.hooks.AfterBootstrapPersist(); err != nil {
			return err
		}
	}
	e.bootstrapLost = false
	e.log.Info("bootstrap stored")
	return nil
}

// noteDesiredSync records a desired-state response, or that a verified
// document is already held. A transport error with desired == nil does
// not call it.
func (e *Engine) noteDesiredSync() {
	e.desiredSynced = true
}

// desiredInForce is this node's document inside [IssuedAt, ExpiresAt).
// The signature check is verifyDesired. The caller still has to do it.
func desiredInForce(ds api.DesiredState, node, inc uuid.UUID, now time.Time) bool {
	if ds.NodeId != node || ds.NodeIncarnationId != inc {
		return false
	}
	if ds.SchemaVersion != api.DesiredStateSchemaVersionN1 || !ds.ExpiresAt.After(ds.IssuedAt) || uint64(ds.ConfigGeneration) == 0 {
		return false
	}
	return !now.Before(ds.IssuedAt) && now.Before(ds.ExpiresAt)
}

func verifyDesired(raw []byte, ts *envelope.TrustStore, now time.Time) (api.DesiredState, error) {
	signed, _, err := envelope.Verify(raw, ts, envelope.PurposeDesiredState, now)
	if err != nil {
		return api.DesiredState{}, err
	}
	var ds api.DesiredState
	dec := json.NewDecoder(bytes.NewReader(signed))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&ds); err != nil {
		return api.DesiredState{}, err
	}
	return ds, nil
}

func verifyLease(raw []byte, ts *envelope.TrustStore, now time.Time) (api.ServingLease, error) {
	signed, _, err := envelope.Verify(raw, ts, envelope.PurposeLease, now)
	if err != nil {
		return api.ServingLease{}, err
	}
	var lease api.ServingLease
	dec := json.NewDecoder(bytes.NewReader(signed))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&lease); err != nil {
		return api.ServingLease{}, err
	}
	return lease, nil
}

func isRevoked(err error) bool {
	var se *controlclient.StatusError
	return errors.As(err, &se) && se.Code == string(api.ErrorCodeIDENTITYREVOKED)
}

func safeErr(err error) string {
	if err == nil {
		return ""
	}
	var se *controlclient.StatusError
	if errors.As(err, &se) {
		return se.Error()
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "control request ended"
	}
	return "control request failed"
}
