package agent

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"net/netip"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/TED-H-com/Freedom-VPN/node/internal/controlclient"
	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usageupload"
	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

// certRenewLead is how early a certificate is renewed. Five minutes is a
// TEST ONLY overlap until an operator chooses the production window (§6.4).
const certRenewLead = 5 * time.Minute

// Options builds an Engine. Nil dependencies take the production defaults.
type Options struct {
	Config Config
	Store  *agentstore.Store
	Client *controlclient.Client
	Plane  Plane
	Clock  timeanchor.Clock
	Hooks  Hooks
	Log    *slog.Logger
	// MeterSource reads admission counters. Nil reads table inet fvpn.
	// Tests pass a fake so they do not need nft.
	MeterSource meter.Source
	Trust       *envelope.TrustStore
	// Prober runs the data-path probe. Nil uses NetBird status.
	Prober Prober
	// DNS applies a verified resolver document. Nil fails a referenced
	// document closed and does not open an AdGuard client. *dns.Applier
	// is the production implementation; tests pass a fake.
	DNS DNSApplier
}

// Engine is one agent process (§5, §16.3). It is not safe for concurrent Step.
type Engine struct {
	cfg         Config
	store       *agentstore.Store
	client      *controlclient.Client
	plane       Plane
	clock       timeanchor.Clock
	bootID      uuid.UUID
	trust       *envelope.TrustStore
	hooks       Hooks
	log         *slog.Logger
	meterSource meter.Source
	ownStore    bool

	usage          *usagestore.Store
	meter          *meter.Meter
	kernel         *kernelSource
	expectServing  atomic.Bool
	uploader       *usageupload.Uploader
	usageOnce      bool
	sampleOn       bool
	finalDone      bool
	finalBarrierID string
	usageLock      *nodelock.Lock
	shutUsage      bool
	meterSnap      []enforce.Binding
	usageOutbox    bool
	usageDisk      bool
	usagePersist   bool
	loopCtx        context.Context
	loopCancel     context.CancelFunc
	loops          sync.WaitGroup
	closeOnce      sync.Once
	closeDone      chan struct{}
	closeCtx       context.Context
	closeErr       error

	// usageStatus is the open store, read when an observation is built.
	// Nil before the store opens and after it closes. usageStatusBound
	// overrides the 2s read limit; zero keeps the default. The reason
	// pair logs one line per transition. lastUsage is this process's
	// last successful read and is what a timed-out read reports.
	usageStatus       usageStatusSource
	usageStatusBound  time.Duration
	usageStatusReason string
	usageStatusKnown  bool
	lastUsage         api.UsageOutboxStatus
	hasUsage          bool

	mu      sync.Mutex
	tlsCert *tls.Certificate

	nodeID               uuid.UUID
	incID                uuid.UUID
	enrolled             bool
	revoked              bool
	identityFrozen       bool
	freezeDurable        bool
	freezeUncertain      bool
	bootstrapLost        bool
	desired              *api.DesiredState
	desiredETag          string
	desiredOK            bool
	desiredExpired       bool
	lease                *api.ServingLease
	leaseMono            time.Duration
	leaseBoot            uuid.UUID
	highestIssued        time.Time
	leaseFresh           bool
	leaseRefreshPending  bool
	anchor               timeanchor.Anchor
	anchorOK             bool
	baselineOK           bool
	storageOK            bool
	serving              bool
	appliedModel         *enforce.Model
	appliedAssignments   map[uuid.UUID]uint16
	appliedGen           uint64
	planeLocked          bool
	repairing            bool
	quarantinePending    bool
	containment          *containmentEvidence
	memberPath           *MemberPath
	memberPathPending    bool
	assignments          map[uuid.UUID]uint16
	counters             map[string]enforce.CounterValue
	addresses            []addrID
	pendingRevoke        []addrID
	conntrackAttempts    int
	conntrackBlocked     bool
	conntrackDetail      string
	recoverBlocked       bool
	recovered            bool
	latchBlocked         bool
	outboxFull           bool
	meterFailed          bool
	applyError           *api.NodeError
	failedCandidate      *failedCandidate
	servingRead          *servingRead
	tcReadback           *api.TCReadbackEvidence
	measurementRun       uuid.UUID
	measurementSequence  uint64
	dependencyMissing    bool
	leaseRetryAt         time.Duration
	leaseAttempted       bool
	leaseAttemptAt       time.Duration
	staleHold            staleLeaseHold
	provisionalStreak    int
	provisionalRetryAt   time.Duration
	decision             Decision
	policyExpires        time.Duration
	lastObs              time.Time
	lastServing          bool
	lastObservationState string
	observationRetryAt   time.Duration
	observationFailures  int
	prober               Prober
	lastProbe            ProbeResult
	// probeAt is the trusted time of the last probe. An answer is written only
	// when this is strictly after the challenge issued_at, so a cached probe
	// cannot satisfy a challenge that did not exist yet.
	probeAt time.Time
	// Keep omission evidence after the updater consumes its sidecars. Only a
	// newer probe-bearing or omission-bearing challenge supersedes it.
	probeOmitted        string
	probeOmittedAt      time.Time
	probeEvidenceIssued time.Time
	// healthNonce and healthSig remember the last answer this process wrote.
	// A restart leaves them empty, so the next answer is a new file. An
	// attestation already on disk when the challenge appeared is not evidence
	// until that rename.
	healthNonce string
	healthSig   string
	// probeChallenge records the issuance already installed by this process.
	probeChallenge  *healthattest.Challenge
	probedChallenge *healthattest.Challenge
	// desiredSynced is set when a verified document was loaded, or when this
	// process received a desired-state response (200, 304, or revocation).
	// A transport error with no document does not set it. A health answer
	// also requires desired != nil, so a body that fails verification
	// writes nothing.
	desiredSynced bool
	dnsApply      DNSApplier
	// dnsPinned is set when the caller passed Options.DNS. Reload leaves it.
	dnsPinned bool
	// dnsCredDetail is set when configured credentials cannot be used.
	// It is a fixed sentence and never contains a path, URL, or secret.
	dnsCredDetail string
	dnsDocument   bool
	dnsFailed     bool
	dnsCode       api.ErrorCode
	dnsDetail     string
	dnsEv         *api.DnsPolicyEvidence
	appliedDNS    appliedDNS
	probeSource   string
	// updateSnapReason is the last updater-journal snapshot class. A repeat
	// of the same class is not logged again. Success clears it.
	updateSnapReason string
}

// Start opens the store, the trust bundle and the control client.
func Start(opt Options) (*Engine, error) {
	if opt.Clock == nil {
		opt.Clock = systemClock{}
	}
	if opt.Plane == nil {
		opt.Plane = SystemPlane{}
	}
	if opt.Log == nil {
		opt.Log = slog.New(RedactHandler{Next: slog.DiscardHandler})
	} else {
		opt.Log = slog.New(RedactHandler{Next: opt.Log.Handler()})
	}
	boot, err := watchdog.ReadBootID(opt.Config.BootIDFile)
	if err != nil {
		return nil, err
	}
	trust := opt.Trust
	if trust == nil {
		raw, err := os.ReadFile(opt.Config.TrustBundle)
		if err != nil {
			return nil, err
		}
		trust, err = envelope.ParseNodeTrustStoreJSON(raw)
		if err != nil {
			return nil, err
		}
	}
	ownStore := false
	store := opt.Store
	if store == nil {
		store, err = agentstore.Open(opt.Config.DBPath())
		if err != nil {
			return nil, err
		}
		ownStore = true
	}
	e := &Engine{
		cfg: opt.Config, store: store, client: opt.Client, plane: opt.Plane,
		clock: opt.Clock, bootID: boot, trust: trust, hooks: opt.Hooks,
		log: opt.Log, meterSource: opt.MeterSource, ownStore: ownStore, storageOK: true,
		prober: opt.Prober, dnsApply: opt.DNS, dnsPinned: opt.DNS != nil,
	}
	if e.client == nil {
		pool := x509.NewCertPool()
		pem, err := os.ReadFile(opt.Config.ServerCAFile)
		if err != nil {
			_ = e.Close()
			return nil, err
		}
		if !pool.AppendCertsFromPEM(pem) {
			_ = e.Close()
			return nil, errors.New("agent: server CA is empty")
		}
		client, err := controlclient.New(controlclient.Config{
			BaseURL:              opt.Config.ControlURL,
			RootCAs:              pool,
			GetClientCertificate: e.clientCert,
		})
		if err != nil {
			_ = e.Close()
			return nil, err
		}
		e.client = client
	}
	if err := e.loadIdentity(context.Background()); err != nil {
		_ = e.Close()
		return nil, err
	}
	executable, err := os.Executable()
	if err == nil {
		err = e.loadActiveRelease(context.Background(), executable)
	}
	if err != nil {
		_ = e.Close()
		return nil, errActiveRelease
	}
	if !e.dnsPinned {
		e.reloadResolvers()
	}
	return e, nil
}

// Close checkpoints a healthy same-boot restart or finishes the withdrawal
// hand-off, then closes the client and, when Start opened it, the agent store.
func (e *Engine) Close() error {
	if e == nil {
		return nil
	}
	// Bound the whole close, including loop joins and synchronous filesystem
	// work. A timed-out worker retains ledger ownership until it actually
	// finishes or the process exits; never close a store under a live sampler.
	e.closeOnce.Do(func() {
		var cancel context.CancelFunc
		e.closeCtx, cancel = newLocalBudget(context.Background(), 3*time.Second)
		e.closeDone = make(chan struct{})
		go func() {
			defer cancel()
			defer close(e.closeDone)
			e.shutdownUsage(e.closeCtx)
			if e.client != nil {
				e.client.Close()
				e.client = nil
			}
			if e.ownStore && e.store != nil {
				e.closeErr = e.store.Close()
				e.store = nil
			}
		}()
	})
	select {
	case <-e.closeDone:
		return e.closeErr
	case <-e.closeCtx.Done():
		select {
		case <-e.closeDone:
			return e.closeErr
		default:
			return e.closeCtx.Err()
		}
	}
}

func (e *Engine) clientCert(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.tlsCert == nil {
		// An empty certificate tells the handshake not to send one. A nil
		// certificate panics inside crypto/tls when the server asks.
		return &tls.Certificate{}, nil
	}
	c := *e.tlsCert
	return &c, nil
}

func (e *Engine) setCert(c tls.Certificate) {
	e.mu.Lock()
	e.tlsCert = &c
	e.mu.Unlock()
}

func (e *Engine) loadCert() error {
	certPEM, err := os.ReadFile(e.cfg.certPath())
	if err != nil {
		return err
	}
	keyPEM, err := os.ReadFile(e.cfg.keyPath())
	if err != nil {
		return err
	}
	pair, err := tlsPair(certPEM, keyPEM)
	if err != nil {
		return err
	}
	e.setCert(pair)
	if e.client != nil {
		e.client.ResetConnections()
	}
	return nil
}

func (e *Engine) loadIdentity(ctx context.Context) error {
	// A retry of the freeze-intent write must already know its incarnation,
	// including when Start is interrupted before Boot/loadRuntime.
	id, ok, err := e.store.Identity(ctx)
	if err != nil {
		return err
	}
	if ok {
		e.nodeID, e.incID, e.enrolled = id.NodeID, id.IncarnationID, true
	}
	if err := e.loadFreeze(ctx); err != nil {
		return err
	}
	if v, ok, err := e.store.Meta(ctx, "revoked"); err != nil {
		return err
	} else if ok && v == "1" {
		e.revoked = true
	}
	if v, ok, err := e.store.Meta(ctx, "bootstrap_lost"); err != nil {
		return err
	} else if ok && v == "1" && !fileExists(e.cfg.setupKeyPath()) {
		e.bootstrapLost = true
	}
	if !ok {
		return nil
	}
	if err := e.loadCert(); err != nil {
		e.log.Warn("identity certificate is unavailable")
	}
	return nil
}

// Boot adopts a verified same-boot ruleset or installs the fail-closed baseline.
// Reboot/clock loss still requires new authorization (§16.3).
func (e *Engine) Boot(ctx context.Context) error {
	if p, ok := e.plane.(interface{ MissingExecutables() []string }); ok {
		missing := p.MissingExecutables()
		e.dependencyMissing = len(missing) != 0
		for _, name := range missing {
			e.log.Error("enforcement dependency unavailable", "executable", name)
		}
	}
	if err := e.loadRuntime(ctx); err != nil {
		e.storageOK = false
		_ = e.baseline(ctx)
		return err
	}
	e.leaseFresh = false
	e.serving = false
	e.recoverOnce(ctx)
	e.latch()
	if e.identityFrozen && !e.freezeDurable {
		e.persistFreeze(ctx)
	}
	if e.adoptStoredServing(ctx) {
		e.beat()
		e.answerHealthOnBoot(ctx)
		return nil
	}
	dp, err := e.cfg.DatapathValue()
	if err != nil {
		return err
	}
	err = e.baselineCall("boot_unverified", func() error { return e.plane.Baseline(ctx, dp) })
	e.baselineOK = err == nil
	e.serving, e.leaseFresh = false, false
	e.policyExpires = 0
	e.ensureUsage(ctx)
	e.decision = Decide(e.input())
	e.beat()
	// A stored document that is still inside its validity window can be
	// answered at start. A missing document, or one that expired while the
	// process was stopped, is not a DNS result yet.
	e.answerHealthOnBoot(ctx)
	return nil
}

// Run boots, then steps until ctx ends.
func (e *Engine) Run(ctx context.Context) error {
	if err := e.Boot(ctx); err != nil {
		return err
	}
	interval := e.stepInterval()
	// SIGHUP reloads resolver credentials between steps. It does not stop
	// the process. SIGINT and SIGTERM still cancel ctx.
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	defer signal.Stop(hup)
	for {
		started := e.clock.Mono()
		if err := e.Step(ctx); err != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		if signaled(hup) {
			e.reloadResolvers()
		}
		timer := time.NewTimer(min(e.nextStepDelay(interval), e.remainingStepDelay(started, interval)))
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-hup:
			timer.Stop()
			e.reloadResolvers()
		case <-timer.C:
		}
	}
}

// nextStepDelay wakes at a grant-less renewal slot even when the preceding
// control work took time or the configured poll is longer than the slot.
// Failed requests and replayed expired leases retry at the control cadence,
// rather than spinning on an already elapsed deadline. Retry-After wins.
func (e *Engine) nextStepDelay(interval time.Duration) time.Duration {
	if e.serving && e.policyExpires > 0 {
		interval = min(interval, max(0, e.policyExpires-e.clock.Mono()))
	}
	if !e.enrolled || e.revoked || !e.leaseGrantless() {
		return interval
	}
	interval = min(interval, provisionalRenewCap)
	at := max(e.provisionalRetryAt, e.leaseRetryAt)
	if e.leaseAttempted && at <= e.leaseAttemptAt {
		at = e.leaseAttemptAt + interval
	}
	return min(interval, max(0, at-e.clock.Mono()))
}

func signaled(ch <-chan os.Signal) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// Step is one control pass. A 304 does not move the lease (§14.7). Member
// forwarding stays off unless every gate in Decide passes.
func (e *Engine) Step(ctx context.Context) error {
	caller := ctx
	// Leave time for live verification and heartbeat even during a control
	// outage. A 30s HTTP timeout previously exceeded the watchdog's stale
	// threshold before the next heartbeat (ISO-F6-03).
	ioCtx, cancelIO := context.WithTimeout(ctx, e.controlPassTimeout())
	defer cancelIO()
	ctx = context.WithValue(ctx, controlBudgetKey{}, ioCtx)
	// One local allowance starts at entry, including the initial/final reads,
	// quarantine, baseline, persistence and observation. Nested cleanup may
	// shorten it but cannot discard it or start another quarantine allowance.
	ctx, cancelLocal := newLocalBudget(ctx, e.controlPassTimeout()+5*time.Second)
	defer cancelLocal()
	defer e.finishServingStep(ctx)
	previousDesired, previousLease := e.desired, e.lease
	e.staleHold.counted = false
	e.leaseAttempted = false
	if err := caller.Err(); err != nil {
		return err
	}
	e.latch()
	if e.identityFrozen && !e.freezeDurable {
		e.persistFreeze(ctx)
	}
	if e.revoked {
		e.withdraw(ctx)
		e.evaluate(ctx)
		e.publishMeterEpoch()
		_ = e.observe(ctx)
		_ = e.flush(ctx)
		e.answerHealth(ctx)
		return nil
	}
	if !e.enrolled {
		return nil
	}
	// Check the retained model before Apply can overwrite evidence of drift.
	// A valid same-generation mismatch is repaired immediately after retaining
	// the rejected read. Other failures still withdraw before a later Apply.
	if !e.verifyCurrentServing(ctx) {
		_ = e.observe(ctx)
		_ = e.flush(ctx)
		return nil
	}
	// The independent live read can finish after SIGTERM. Stop here before
	// canceled local store/meter reads manufacture a storage failure and
	// prevent Close from checkpointing the still-valid same-boot rules.
	if err := caller.Err(); err != nil {
		return err
	}
	// Refresh the anchor before any 304. A wall jump must be visible to the
	// lease renewal decision even when desired state is unchanged (§16.3).
	_, _ = e.verifyTime()
	// A due grant-less renewal must not wait behind observation retries or
	// desired-state HTTP work. Generation changes are still handled below.
	if e.leaseGrantless() {
		if err := e.syncLease(ctx); err != nil && ctx.Err() != nil {
			return err
		}
		if e.revoked {
			e.answerHealth(ctx)
			return nil
		}
	}
	_ = e.flush(ctx)
	if err := e.syncDesired(ctx); err != nil && ctx.Err() != nil {
		return err
	}
	if e.revoked {
		e.withdraw(ctx)
		e.answerHealth(ctx)
		return nil
	}
	// Classify the document before the lease decision. After a restart the
	// stored desired state is loaded with desiredOK still false, and the
	// provisional schedule is armed in the same step that accepts the lease.
	e.refreshDesired()
	if err := e.syncLease(ctx); err != nil && ctx.Err() != nil {
		return err
	}
	if e.revoked {
		e.withdraw(ctx)
		e.answerHealth(ctx)
		return nil
	}
	_ = e.syncCert(ctx)
	_ = e.syncBootstrap(ctx)
	e.refreshDesired()
	e.ensureUsage(ctx)
	if e.uploader != nil && e.uploader.NeedsRestore() {
		e.restoreUsage(ctx)
	}
	e.readMeter(ctx)
	if pending, err := e.store.Pending(ctx); err == nil {
		e.outboxFull = len(pending) >= 32
	}
	e.evaluate(ctx)
	// A timed-out control pass did not authorize a new policy. Keep the
	// just-verified generation instead of re-installing the same expiring
	// allowset on each outage retry. No guard/timeout is reset by this path.
	retain := ioCtx.Err() != nil && e.serving && e.decision.Serve && e.applyError == nil &&
		previousDesired == e.desired && previousLease == e.lease && e.appliedModel != nil &&
		e.appliedGen == e.decision.Model.Generation && e.policyExpires == e.decision.PolicyExpires &&
		len(e.pendingRevoke) == 0 && !e.memberPathPending && !e.quarantinePending
	if retain {
		e.checkMemberPath(ctx, *e.appliedModel)
	} else {
		if err := e.enforce(ctx, e.decision); err != nil && ctx.Err() != nil {
			return err
		}
	}
	// The generation this step just applied is visible only after enforce.
	// The matching lease is requested before heartbeat and observation.
	if err := e.renewAppliedLease(ctx); err != nil && ctx.Err() != nil {
		return err
	}
	e.publishMeterEpoch()
	_ = e.observe(ctx)
	_ = e.flush(ctx)
	e.answerHealth(ctx)
	return nil
}

func (e *Engine) loadRuntime(ctx context.Context) error {
	if err := e.loadFreeze(ctx); err != nil {
		return err
	}
	if raw, ok, err := e.store.Meta(ctx, "retained_applied"); err != nil {
		return err
	} else if ok {
		var retained retainedApplied
		if err := json.Unmarshal([]byte(raw), &retained); err != nil {
			return err
		}
		e.appliedModel, e.appliedAssignments = &retained.Model, retained.Assignments
	}

	if raw, ok, err := e.store.Meta(ctx, "member_path"); err != nil {
		return err
	} else if ok {
		var path MemberPath
		if err := json.Unmarshal([]byte(raw), &path); err != nil {
			return err
		}
		e.memberPath = &path
	}

	if a, ok, err := e.store.Anchor(ctx); err != nil {
		return err
	} else if ok && a.Trusted && a.BootID == e.bootID {
		e.anchor = timeanchor.Anchor{BootID: a.BootID, Server: a.Server, Mono: a.Mono, Wall: a.Wall, Trusted: true}
	}
	if d, ok, err := e.store.Desired(ctx); err != nil {
		return err
	} else if ok {
		// The applied generation stays even when the envelope is rejected,
		// so ack and anti-rollback are not lost. The ETag is kept only
		// after the signature and the validity window both succeed. An
		// envelope that expired while the process was stopped must not
		// leave a stale validator for the next sync.
		if d.HasApplied {
			e.appliedGen = d.Applied
		}
		now, accept := e.verifyTime()
		if !accept {
			now = e.clock.Wall()
		}
		if ds, err := verifyDesired(d.Envelope, e.trust, now); err == nil && desiredInForce(ds, e.nodeID, e.incID, now) {
			e.desired = &ds
			e.desiredETag = d.ETag
			e.desiredSynced = true
			if err := writeAtomic(e.cfg.DesiredStatePath, d.Envelope, 0o644); err != nil {
				e.log.Warn("desired state file", "err", err.Error())
			}
		}
	}
	if err := e.migrateMemberPath(ctx); err != nil {
		return err
	}
	if l, ok, err := e.store.Lease(ctx); err != nil {
		return err
	} else if ok {
		if lease, err := verifyLease(l.Envelope, e.trust, e.clock.Wall()); err == nil {
			e.lease = &lease
		}
		e.leaseMono = l.MonoSent
		e.leaseBoot = l.BootID
		e.highestIssued = l.HighestIssuedAt
	}
	var err error
	if e.assignments, err = e.store.QoS(ctx); err != nil {
		return err
	}
	raw, err := e.store.Counters(ctx)
	if err != nil {
		return err
	}
	e.counters = map[string]enforce.CounterValue{}
	for name, c := range raw {
		e.counters[name] = enforce.CounterValue{Packets: c.Packets, Bytes: c.Bytes}
	}
	rows, err := e.store.Addresses(ctx)
	if err != nil {
		return err
	}
	e.addresses = nil
	for _, row := range rows {
		addr, err := netip.ParseAddr(row.Addr)
		if err != nil || !addr.Is4() {
			return errors.New("agent: invalid applied address")
		}
		device, err := uuid.Parse(row.DeviceID)
		if err != nil {
			return errors.New("agent: invalid applied address")
		}
		member, err := uuid.Parse(row.MemberID)
		if err != nil {
			return errors.New("agent: invalid applied address")
		}
		e.addresses = append(e.addresses, addrID{Addr: addr, DeviceID: device, MemberID: member})
	}
	if err := e.loadRevocations(ctx); err != nil {
		return err
	}
	return e.loadConntrackState(ctx)
}

func (e *Engine) trustedNow() time.Time {
	_, _ = e.verifyTime()
	now := e.clock.Wall()
	if e.anchorOK {
		if t, err := e.anchor.TrustedNow(e.bootID, e.clock, timeanchor.DefaultBackwardSkew, timeanchor.DefaultForwardSkew); err == nil {
			now = t
		}
	}
	return now
}

func (e *Engine) input() Input {
	e.guardNamedDNS()
	dp, _ := e.cfg.DatapathValue()
	exempt, _ := parsePrefixes(e.cfg.TestExemptV4)
	extra, _ := parsePrefixes(e.cfg.ExtraDeniedV4)
	host, _ := parsePrefixes(e.cfg.HostV4)
	now := e.trustedNow()
	return Input{
		Now: now, MonoNow: e.clock.Mono(), Datapath: dp,
		Desired: e.desired, DesiredOK: e.desiredOK, DesiredExpired: e.desiredExpired,
		Lease: e.lease, LeaseFresh: e.leaseFresh && e.lease != nil, LeaseMonoSent: e.leaseMono,
		AnchorOK: e.anchorOK, LatchBlocked: e.latchBlocked, RecoverBlocked: e.recoverBlocked,
		Backend: e.cfg.NetbirdBackend, ReleaseID: e.cfg.ReleaseUUID(), Assignments: e.assignments,
		TestMode: e.cfg.TestMode, ExemptV4: exempt, ExtraDenied: extra, HostV4: host,
		IdentityFrozen: e.identityFrozen,
		HostReserveBPS: e.cfg.HostReserveBPS, Revoked: e.revoked, Enrolled: e.enrolled,
		BaselineOK: e.baselineOK, StorageOK: e.storageOK && !e.freezeUncertain, BootstrapLost: e.bootstrapLost,
		ApplyError: e.applyError, DependencyMissing: e.dependencyMissing, LeaseRetryPending: e.leaseRetryAt > 0,
		RevocationPending: len(e.pendingRevoke) > 0, ConntrackBlocked: e.conntrackBlocked,
		ConntrackDetail: e.conntrackDetail,
		OutboxFull:      e.outboxFull, MeterFailed: e.meterFailed,
		UsageOutbox: e.usageOutbox, UsageDisk: e.usageDisk, UsagePersist: e.usagePersist,
		ProbeRan: e.lastProbe.Ran && e.probeOmitted == "", ProbeCheck: e.lastProbe.Check, ProbeHealth: e.lastProbe.Health,
		ProbeSource: e.probeSource, ProbeOmitted: e.probeOmitted, ProbeOmittedAt: e.probeOmittedAt,
		DNSDocument: e.dnsDocument, DNSFailed: e.dnsFailed, DNSCode: e.dnsCode, DNSDetail: e.dnsDetail,
	}
}

func (e *Engine) refreshProbe(ctx context.Context) {
	e.prepareProbe(ctx)
	e.withProbeException(ctx, func() {
		p := e.prober
		if p == nil {
			nb := newNetbirdProber(e.cfg)
			p = nb
		}
		published := e.desired != nil && e.desired.PublicationIntent == api.PublicationIntentPublished
		targets, timeout, source, bad := egressTargets(e.desired, e.desiredOK, e.cfg.EgressProbeTarget)
		e.probeSource = probeSourceDetail(source, bad)
		e.lastProbe = p.Probe(ctx, ProbeRequest{
			Published: published,
			Targets:   targets,
			Timeout:   timeout,
			Rejected:  bad,
			Ingress:   e.cfg.Datapath.MemberIngress,
			Uplink:    e.cfg.Datapath.Uplink,
		})
		e.probeAt = e.trustedNow().UTC()
		if !e.anchorOK {
			e.probeAt = time.Time{}
		}
	})
	e.noteProbeOmission()
}

// noteProbeOmission retains only the shared reader's fixed reason, never the
// peer ID. Re-read the challenge after the probe to bind this evidence to the
// same live issuance. A plain overlay check cannot supersede a refused probe.
func (e *Engine) noteProbeOmission() {
	ch, ok := e.matchingChallenge()
	now := e.trustedNow().UTC()
	if !ok || !e.anchorOK || e.probedChallenge == nil || !ch.Same(*e.probedChallenge) ||
		!now.After(ch.IssuedAt) || !now.Before(ch.ExpiresAt) || !ch.IssuedAt.After(e.probeEvidenceIssued) {
		return
	}
	if ch.Probe == nil && ch.ProbeOmitted == nil {
		return
	}
	e.probeEvidenceIssued = ch.IssuedAt
	e.probeOmitted = ""
	e.probeOmittedAt = time.Time{}
	if ch.ProbeOmitted != nil {
		e.probeOmitted = ch.ProbeOmitted.Reason
		e.probeOmittedAt = now
	}
	// The node may stay non-serving across this change; do not wait for the
	// serving-state trigger to publish the new check evidence.
	e.lastObs = time.Time{}
}

// verifyTime reports the time used to check a signed document.
// The second result is false during a clock anomaly: a new desired state
// must not be accepted (§16.3). Before the first anchor, wall time is used.
func (e *Engine) verifyTime() (time.Time, bool) {
	if e.anchor.Trusted {
		t, err := e.anchor.TrustedNow(e.bootID, e.clock, timeanchor.DefaultBackwardSkew, timeanchor.DefaultForwardSkew)
		if err != nil {
			e.anchorOK = false
			return e.clock.Wall(), false
		}
		e.anchorOK = true
		return t, true
	}
	e.anchorOK = false
	return e.clock.Wall(), true
}

func (e *Engine) refreshDesired() {
	now, ok := e.verifyTime()
	if !ok {
		now = e.clock.Wall()
	}
	if e.desired == nil {
		e.desiredOK = false
		e.desiredExpired = false
		return
	}
	e.desiredExpired = now.Before(e.desired.IssuedAt) || !now.Before(e.desired.ExpiresAt)
	e.desiredOK = !e.desiredExpired
}

func (e *Engine) latch() {
	id, blocked := meterBarrierIdentity(e.cfg.JournalDir)
	e.latchBlocked = blocked
	if id == "" {
		if err := e.invalidateMeterCheckpoint(); err != nil {
			e.usagePersist = true
		}
	}
}

func (e *Engine) recoverOnce(ctx context.Context) {
	if e.recovered {
		return
	}
	e.recovered = true
	if e.cfg.UpdaterRecover == "" {
		return
	}
	if e.cfg.UpdaterConfig == "" {
		e.recoverBlocked = true
		return
	}
	cmd := exec.CommandContext(ctx, e.cfg.UpdaterRecover, "recover", "-config", e.cfg.UpdaterConfig)
	err := cmd.Run()
	if err == nil {
		e.recoverBlocked = false
		return
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 2 {
		e.recoverBlocked = true
		return
	}
	e.recoverBlocked = true
	e.log.Error("updater recover", "err", "forwarding stays blocked")
}

func (e *Engine) baseline(ctx context.Context) error {
	// A planned baseline is not an external meter reset. Clear this before
	// FinalSample reads the counters.
	e.expectServing.Store(false)
	dp, err := e.cfg.DatapathValue()
	if err != nil {
		return err
	}
	if err := e.baselineCall(e.baselineReason(), func() error { return e.plane.Baseline(ctx, dp) }); err != nil {
		e.baselineOK = false
		return err
	}
	e.baselineOK = true
	e.policyExpires = 0
	return nil
}

func (e *Engine) enforce(ctx context.Context, d Decision) error {
	baselineOK, meterFailed, applyError := e.baselineOK, e.meterFailed, e.applyError
	storageOK, revocationPending := e.storageOK, len(e.pendingRevoke) > 0
	blocked := e.conntrackBlocked
	defer func() {
		pendingChanged := revocationPending != (len(e.pendingRevoke) > 0)
		blockedChanged := blocked != e.conntrackBlocked
		if pendingChanged || blockedChanged {
			// Publish the new pending or failed-closed condition on this tick.
			e.lastObs = time.Time{}
		}
		if baselineOK != e.baselineOK || meterFailed != e.meterFailed || applyError != e.applyError || storageOK != e.storageOK || pendingChanged || blockedChanged {
			errs := e.decision.Errors
			e.decision = Decide(e.input())
			for _, err := range errs {
				if err.Code == api.ErrorCodeUPDATEBUSY || err.Code == api.ErrorCodeMETEREPOCHCHANGE {
					e.decision.Errors = append(e.decision.Errors, err)
				}
			}
		}
	}()
	if e.conntrackBlocked && !e.conntrackMissing() && e.flushPendingConntrack(ctx) {
		e.clearConntrackBlock(ctx)
		d = Decide(e.input())
		e.decision = d
	}
	if e.conntrackMissing() {
		e.markConntrackBlocked(ctx, conntrackMissingDetail)
		d.Serve = false
		e.decision.Serve = false
	} else if e.conntrackBlocked {
		e.serving = false
		e.policyExpires = 0
		_ = e.baseline(ctx)
		return nil
	}
	// A previous failure remains reportable while we retry. Only the candidate
	// decision omits it; successful read-back below clears the evidence.
	closedRecovery := false
	if e.applyError != nil {
		in := e.input()
		in.ApplyError = nil
		d = Decide(in)
		if !d.Serve && (leaseGenerationStale(in) || grantsProvisional(in)) {
			// Verify a deny-all candidate before ACK/renewal. There are no
			// member grants in this model, so it cannot forward using an old
			// generation's authorization. All other safety gates still apply.
			d, closedRecovery = recoveryDecision(in)
		}
	}
	if !d.Serve {
		e.handoff(ctx)
		e.serving = false
		e.policyExpires = 0
		if err := e.baseline(ctx); err != nil {
			return err
		}
		// Unpublished (and any other non-serving) desired generation is
		// acknowledged only after the baseline is in place. The serving
		// model was cleared, so this does not forward member traffic.
		e.ackApplied(ctx, d.Applied)
		return nil
	}
	lock, err := nodelock.Acquire(ctx, e.cfg.LockPath, 3, 50*time.Millisecond)
	if errors.Is(err, nodelock.ErrBusy) {
		e.decision.Errors = append(e.decision.Errors, api.NodeError{
			Code: api.ErrorCodeUPDATEBUSY, Message: "UPDATE_BUSY",
			ObservedAt: e.clock.Wall().UTC(), Stage: "apply",
		})
		if !e.serving {
			e.decision.Reasons = append(e.decision.Reasons, api.ForwardingBlockReasonUpdateInProgress)
			e.decision.Serve = false
		}
		return nil
	}
	if err != nil {
		return err
	}
	defer lock.Release()
	// The updater may have published its block since this step's decision.
	// Re-read under the node lock before invalidating evidence or admitting.
	e.latch()
	if e.latchBlocked {
		e.decision = Decide(e.input())
		e.withdraw(ctx)
		return nil
	}
	if !closedRecovery {
		if err := e.invalidateMeterCheckpoint(); err != nil {
			e.usagePersist = true
			e.withdraw(ctx)
			return err
		}
	}
	e.planeLocked = true
	defer func() { e.planeLocked = false }()
	model := d.Model
	queued := planRevokes(e.pendingRevoke, e.addresses, model.Bindings)
	model.Revoke = queued
	prev := map[string]enforce.CounterValue{}
	for name, c := range e.counters {
		prev[name] = c
	}
	// Capture the initial topology before Apply can itself quarantine a
	// foreign attachment and discard routes/IPv6 addresses. A failed capture
	// leaves forwarding off until the existing topology is configured.
	if e.memberPath == nil && !e.checkMemberPath(ctx, model) {
		return nil
	}
	// Keep a candidate separate from the durable expectation. Capture before
	// Apply: quarantine may itself remove routes and IPv6 addresses. A retry
	// of the same generation can only restore the existing topology (D-100).
	pathGen := uint64(e.memberPath.Generation)
	if pathGen == 0 {
		pathGen = e.appliedGen // upgrade from the pre-D-100 durable format
	}
	var candidate *MemberPath
	if model.Generation > pathGen && !closedRecovery {
		path, ok := e.readMemberPath(ctx, model, nil)
		if ok {
			candidate = &path
		} else if !e.quarantinePending {
			return nil
		}
		// If quarantine lowered the link, let Apply raise it for repair.
		// Without a valid pre-Apply candidate the old expectation still applies.
	}
	wasServing := e.expectServing.Load()
	if !closedRecovery && e.meter != nil && e.finalDone {
		// A closed epoch must be replaced before Apply can admit the first
		// packet. Keep the sampler paused across this real durable read.
		e.finalDone = false
		if err := e.meter.PrepareSampling(ctx); err != nil {
			e.usagePersist = true
			e.withdraw(ctx)
			return err
		}
	}
	timing := &applyReadTiming{now: e.clock.Wall}
	host := e.measureHostBeforeApply()
	applyCtx := context.WithValue(ctx, applyReadKey{}, timing)
	if e.measurementEnabled() {
		// TEST ONLY command observations use the existing local timing gate.
		// They do not enter the wire evidence or its D-106 digest.
		applyCtx = enforce.WithCommandLifecycleForTest(applyCtx)
	}
	res, err := e.plane.Apply(applyCtx, model, prev)
	host.finish(err)
	e.measureTiming(res.Evidence, "apply", host)
	for _, b := range timing.baselines {
		b.Cause = "apply_rollback"
		if res.Quarantine != nil {
			b.Cause = "apply_quarantine"
		}
		e.log.Info("baseline write", "writer", "agent_apply", "cause", b.Cause,
			"start_mono_ns", int64(b.Start), "end_mono_ns", int64(b.End), "succeeded", b.Succeeded)
		e.writeMeasurement(Measurement{Source: "baseline", Baseline: &b})
	}
	e.retainTCReadback(res.Evidence)
	if !timing.at.IsZero() {
		e.retainServingRead(res.Evidence, model.Generation, timing.at, "apply", res.Verified)
	}
	// A verified conntrack failure keeps serving. Only Failed stays pending.
	// An empty Failed list is a successful delete and resets the bound.
	var ctErr *enforce.ConntrackDeleteError
	verifiedDelete := err != nil && res.Verified && errors.As(err, &ctErr)
	pending := verifiedDelete && len(ctErr.Failed) > 0
	if res.Quarantine != nil {
		e.recordApplyQuarantine(res, err)
	}
	if res.Verified || res.Evidence.TCSummary.Phase == "verify" {
		// Apply reached its installed-rules verification. Even a rollback or
		// later topology rejection must attribute bytes admitted before
		// withdrawal, including on restart with no in-memory snapshot.
		e.setMeterBindings(model.Bindings)
	}
	if err != nil && !verifiedDelete {
		e.failedCandidate = &failedCandidate{Generation: model.Generation}
		e.rejectServing(ctx, model, res.Evidence, err)
		e.failedCandidate.Error = *e.applyError
		return nil
	}
	e.failedCandidate = nil
	// Retain the model actually installed, including its class assignments,
	// before any subsequent read can reject it.
	e.appliedModel = &model
	e.appliedAssignments = maps.Clone(res.Plan.Assignments)
	// Apply alone releases quarantine. A missing member route/address below
	// must not lower the clean link again and prevent its owner's repair.
	e.quarantinePending = false
	// Apply's evidence can already be stale by its return (for example a
	// filter inserted during conntrack deletion). Read before acknowledging.
	if !e.verifyServingModel(ctx, model, res.Plan.Assignments) {
		return nil
	}
	retained, _ := json.Marshal(retainedApplied{Model: model, Assignments: e.appliedAssignments})
	if err := e.store.SetMeta(ctx, "retained_applied", string(retained)); err != nil {
		e.storageOK = false
		e.rejectServing(ctx, model, res.Evidence, err)
		return nil
	}
	if candidate != nil && err == nil && res.Verified {
		// Recheck the pre-Apply candidate after full serving verification. A
		// failed Apply, including a pending conntrack delete, cannot refresh it.
		path, ok := e.readMemberPath(ctx, model, candidate)
		if !ok || !e.saveMemberPath(ctx, path, model.Generation) {
			return nil
		}
		e.memberPathPending = false
	} else if !e.checkMemberPath(ctx, model) {
		return nil
	}
	e.serving = !closedRecovery
	e.applyError = nil
	e.expectServing.Store(!closedRecovery)
	e.baselineOK = true
	e.policyExpires = d.PolicyExpires
	if closedRecovery {
		e.policyExpires = 0
	}
	e.ackApplied(ctx, model.Generation)
	if pending {
		e.conntrackAttempts++
		if e.conntrackAttempts > maxConntrackDeleteAttempts {
			e.conntrackAttempts = maxConntrackDeleteAttempts
		}
		e.saveRevocations(ctx, pendingAfterFailure(e.pendingRevoke, e.addresses, model.Bindings, ctErr.Failed))
		if e.conntrackAttempts >= maxConntrackDeleteAttempts {
			e.markConntrackBlocked(ctx, conntrackBoundDetail)
			e.handoff(ctx)
			e.serving = false
			e.policyExpires = 0
			_ = e.baseline(ctx)
			return nil
		}
		e.persistConntrack(ctx)
	} else {
		e.saveRevocations(ctx, nil)
		if e.conntrackAttempts != 0 || e.conntrackBlocked {
			e.clearConntrackBlock(ctx)
		}
	}
	if res.Plan.Assignments != nil {
		e.assignments = res.Plan.Assignments
		if err := e.store.SaveQoS(ctx, e.assignments); err != nil {
			e.storageOK = false
		}
	}
	// A verified conntrack failure returns Plan/Resets without a counter
	// snapshot. Keep the previous snapshot until the sampler or a full apply
	// reads the counters; absence of a snapshot is not a counter reset.
	if !pending || res.Counters != nil {
		e.counters = cloneCounters(res.Counters)
		stored := map[string]agentstore.Counter{}
		for name, c := range res.Counters {
			stored[name] = agentstore.Counter{Packets: c.Packets, Bytes: c.Bytes}
		}
		if err := e.store.SaveCounters(ctx, stored); err != nil {
			e.storageOK = false
		}
	}
	e.addresses = bindingIDs(model)
	stored := make([]agentstore.AppliedAddress, 0, len(e.addresses))
	for _, id := range e.addresses {
		stored = append(stored, agentstore.AppliedAddress{
			Addr: id.Addr.String(), DeviceID: id.DeviceID.String(), MemberID: id.MemberID.String(),
		})
	}
	if err := e.store.SaveAddresses(ctx, stored); err != nil {
		e.storageOK = false
	}
	post := res.Counters
	if pending && post == nil {
		post = prev
	}
	if e.kernel != nil {
		if err := e.kernel.capture(ctx); err != nil {
			e.log.Warn("meter state", "err", err.Error())
		}
		e.kernel.noteApply(prev, res.Resets, post, wasServing)
	} else {
		e.noteResetsWithoutKernel(prev, res.Resets, post, wasServing)
	}
	if !closedRecovery {
		e.resumeSampling()
		e.switchEpoch(ctx)
	}
	e.beat() // publish renewed expiry before releasing the shared Apply lock
	return nil
}

func (e *Engine) ackApplied(ctx context.Context, gen uint64) {
	if e.identityFrozen || e.freezeUncertain || gen == 0 || e.desired == nil || !e.desiredOK || e.applyError != nil || e.dependencyMissing {
		return
	}
	if e.appliedGen == gen {
		return
	}
	if err := e.store.MarkApplied(ctx, gen); err != nil {
		e.storageOK = false
		e.log.Warn("applied generation", "err", err.Error())
		return
	}
	e.appliedGen = gen
	// Start elapsed hold time at the ACK, before a matching-lease request can
	// spend the rest of the control allowance. Observation is not its origin.
	e.updateStaleHold()
}

func (e *Engine) noteApply(err error) {
	var ce *enforce.CheckError
	code := api.ErrorCodeHEALTHCHECKFAILED
	msg := "apply failed"
	if errors.Is(err, context.DeadlineExceeded) {
		msg = "serving verification timed out"
	} else if errors.Is(err, context.Canceled) {
		msg = "serving verification canceled"
	}
	if errors.As(err, &ce) {
		msg = string(ce.Reason)
		if api.ErrorCode(ce.Reason).Valid() {
			code = api.ErrorCode(ce.Reason)
		}
	}
	e.applyError = &api.NodeError{
		Code: code, Message: clipMsg(msg), ObservedAt: e.clock.Wall().UTC(), Stage: "apply",
	}
	e.decision.Errors = append(e.decision.Errors, *e.applyError)
	e.log.Warn("apply", "err", msg)
}

func (e *Engine) markRevoked(ctx context.Context) {
	e.revoked = true
	e.policyExpires = 0
	if err := e.store.SetMeta(ctx, "revoked", "1"); err != nil {
		e.storageOK = false
	}
	e.withdraw(ctx)
	e.log.Warn("identity revoked")
}

// withdraw is the forwarding stop: the usage hand-off (barrier, baseline,
// final sample) and a baseline again for the case where sampling never
// started. A later successful apply resumes sampling.
func (e *Engine) withdraw(ctx context.Context) {
	e.handoff(ctx)
	_ = e.baseline(ctx)
	e.serving = false
	e.policyExpires = 0
}

func (e *Engine) beat() {
	mono := e.clock.Mono()
	if mono <= 0 || e.bootID == uuid.Nil {
		return
	}
	reason := "-"
	if e.serving {
		reason = "serving"
	} else if len(e.decision.Reasons) > 0 {
		reason = string(e.decision.Reasons[0])
	}
	if e.memberPathPending && !e.quarantinePending {
		reason = "member_path_unconfigured"
	}
	if e.identityFrozen {
		reason = "identity_frozen"
	}
	err := watchdog.WriteHeartbeat(e.cfg.HeartbeatPath, watchdog.Beat{
		BootID: e.bootID, Mono: mono, Wall: e.clock.Wall().UTC(),
		PolicyExpires: e.policyExpires, BaselineOK: e.baselineOK, Serving: e.serving,
		Generation: e.appliedGen, Reason: reason, Evidence: e.containmentJSON(),
	})
	e.writeMeasurement(Measurement{Source: "heartbeat", Heartbeat: &MeasurementHeartbeat{
		Mono: mono, PolicyExpires: e.policyExpires, Generation: decstr.U64(e.appliedGen),
		BaselineOK: e.baselineOK, Serving: e.serving, Reason: reason, Written: err == nil,
	}})
	if err != nil {
		e.log.Warn("heartbeat", "err", err.Error())
	}
}

func (e *Engine) renewLead(window time.Duration) time.Duration {
	lead := time.Duration(e.cfg.RenewLeadSeconds) * time.Second
	if window > 0 && lead > window/2 {
		lead = window / 2
	}
	if lead < time.Second {
		lead = time.Second
	}
	return lead
}
