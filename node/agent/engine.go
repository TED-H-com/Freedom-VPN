package agent

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log/slog"
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
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usageupload"
	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
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

	usage         *usagestore.Store
	meter         *meter.Meter
	kernel        *kernelSource
	expectServing atomic.Bool
	uploader      *usageupload.Uploader
	usageOnce     bool
	sampleOn      bool
	finalDone     bool
	shutUsage     bool
	meterSnap     []enforce.Binding
	usageOutbox   bool
	usageDisk     bool
	usagePersist  bool
	loopCtx       context.Context
	loopCancel    context.CancelFunc
	loops         sync.WaitGroup

	mu      sync.Mutex
	tlsCert *tls.Certificate

	nodeID         uuid.UUID
	incID          uuid.UUID
	enrolled       bool
	revoked        bool
	bootstrapLost  bool
	desired        *api.DesiredState
	desiredETag    string
	desiredOK      bool
	desiredExpired bool
	lease          *api.ServingLease
	leaseMono      time.Duration
	leaseBoot      uuid.UUID
	highestIssued  time.Time
	leaseFresh     bool
	anchor         timeanchor.Anchor
	anchorOK       bool
	baselineOK     bool
	storageOK      bool
	serving        bool
	appliedGen     uint64
	assignments    map[uuid.UUID]uint16
	counters       map[string]enforce.CounterValue
	addresses      []string
	recoverBlocked bool
	recovered      bool
	latchBlocked   bool
	outboxFull     bool
	meterFailed    bool
	decision       Decision
	policyExpires  time.Duration
	lastObs        time.Time
	lastServing    bool
	prober         Prober
	lastProbe      ProbeResult
	// probeAt is the wall time of the last probe. An answer is written only
	// when this is strictly after the challenge issued_at, so a cached probe
	// cannot satisfy a challenge that did not exist yet.
	probeAt time.Time
	// healthNonce and healthSig remember the last answer this process wrote.
	// A restart leaves them empty, so the next answer is a new file. An
	// attestation already on disk when the challenge appeared is not evidence
	// until that rename.
	healthNonce string
	healthSig   string
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
	if !e.dnsPinned {
		e.reloadResolvers()
	}
	return e, nil
}

// Close finishes the usage hand-off, then closes the client and, when
// Start opened it, the agent store.
func (e *Engine) Close() error {
	if e == nil {
		return nil
	}
	e.shutdownUsage()
	if e.client != nil {
		e.client.Close()
		e.client = nil
	}
	if e.ownStore && e.store != nil {
		err := e.store.Close()
		e.store = nil
		return err
	}
	return nil
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
	id, ok, err := e.store.Identity(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	e.nodeID = id.NodeID
	e.incID = id.IncarnationID
	e.enrolled = true
	if err := e.loadCert(); err != nil {
		e.log.Warn("identity certificate is unavailable")
	}
	return nil
}

// Boot installs the fail-closed baseline before any member forwarding (§8.2, §13.5).
// A stored lease does not count as fresh until this process accepts one.
func (e *Engine) Boot(ctx context.Context) error {
	dp, err := e.cfg.DatapathValue()
	if err != nil {
		return err
	}
	if err := e.plane.Baseline(ctx, dp); err != nil {
		e.baselineOK = false
		e.log.Error("baseline", "err", err.Error())
	} else {
		e.baselineOK = true
	}
	if err := e.loadRuntime(ctx); err != nil {
		e.storageOK = false
		return err
	}
	e.leaseFresh = false
	e.serving = false
	e.recoverOnce(ctx)
	e.latch()
	e.ensureUsage(ctx)
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
	interval := time.Duration(e.cfg.HeartbeatSeconds) * time.Second
	poll := time.Duration(e.cfg.PollSeconds) * time.Second
	if poll > 0 && poll < interval {
		interval = poll
	}
	if interval <= 0 {
		interval = time.Second
	}
	// SIGHUP reloads resolver credentials between steps. It does not stop
	// the process. SIGINT and SIGTERM still cancel ctx.
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	defer signal.Stop(hup)
	for {
		if err := e.Step(ctx); err != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		if signaled(hup) {
			e.reloadResolvers()
		}
		timer := time.NewTimer(interval)
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
	defer e.beat()
	if err := ctx.Err(); err != nil {
		return err
	}
	e.latch()
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
	// Refresh the anchor before any 304. A wall jump must be visible to the
	// lease renewal decision even when desired state is unchanged (§16.3).
	_, _ = e.verifyTime()
	_ = e.flush(ctx)
	if err := e.syncDesired(ctx); err != nil && ctx.Err() != nil {
		return err
	}
	if e.revoked {
		e.withdraw(ctx)
		e.answerHealth(ctx)
		return nil
	}
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
	if err := e.enforce(ctx, e.decision); err != nil && ctx.Err() != nil {
		return err
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
	e.addresses, err = e.store.Addresses(ctx)
	return err
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
		HostReserveBPS: e.cfg.HostReserveBPS, Revoked: e.revoked, Enrolled: e.enrolled,
		BaselineOK: e.baselineOK, StorageOK: e.storageOK, BootstrapLost: e.bootstrapLost,
		OutboxFull: e.outboxFull, MeterFailed: e.meterFailed,
		UsageOutbox: e.usageOutbox, UsageDisk: e.usageDisk, UsagePersist: e.usagePersist,
		ProbeRan: e.lastProbe.Ran, ProbeCheck: e.lastProbe.Check, ProbeHealth: e.lastProbe.Health,
		ProbeSource: e.probeSource,
		DNSDocument: e.dnsDocument, DNSFailed: e.dnsFailed, DNSCode: e.dnsCode, DNSDetail: e.dnsDetail,
	}
}

func (e *Engine) refreshProbe(ctx context.Context) {
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
		e.probeAt = e.clock.Wall().UTC()
	})
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
	e.latchBlocked = ForwardingBlocked(e.cfg.JournalDir)
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
	if err := e.plane.Baseline(ctx, dp); err != nil {
		e.baselineOK = false
		return err
	}
	e.baselineOK = true
	e.policyExpires = 0
	return nil
}

func (e *Engine) enforce(ctx context.Context, d Decision) error {
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
	model := d.Model
	model.Revoke = droppedAddrs(e.addresses, model)
	prev := map[string]enforce.CounterValue{}
	for name, c := range e.counters {
		prev[name] = c
	}
	wasServing := e.expectServing.Load()
	res, err := e.plane.Apply(ctx, model, prev)
	if err != nil {
		e.handoff(ctx)
		e.serving = false
		e.policyExpires = 0
		e.noteApply(err)
		_ = e.baseline(ctx)
		return nil
	}
	e.serving = true
	e.expectServing.Store(true)
	e.baselineOK = true
	e.policyExpires = d.PolicyExpires
	e.appliedGen = model.Generation
	if err := e.store.MarkApplied(ctx, model.Generation); err != nil {
		e.storageOK = false
		e.log.Warn("applied generation", "err", err.Error())
	}
	if res.Plan.Assignments != nil {
		e.assignments = res.Plan.Assignments
		if err := e.store.SaveQoS(ctx, e.assignments); err != nil {
			e.storageOK = false
		}
	}
	e.counters = map[string]enforce.CounterValue{}
	stored := map[string]agentstore.Counter{}
	for name, c := range res.Counters {
		e.counters[name] = c
		stored[name] = agentstore.Counter{Packets: c.Packets, Bytes: c.Bytes}
	}
	if err := e.store.SaveCounters(ctx, stored); err != nil {
		e.storageOK = false
	}
	e.addresses = bindingAddrs(model)
	if err := e.store.SaveAddresses(ctx, e.addresses); err != nil {
		e.storageOK = false
	}
	e.setMeterBindings(model.Bindings)
	if e.kernel != nil {
		if err := e.kernel.capture(ctx); err != nil {
			e.log.Warn("meter state", "err", err.Error())
		}
		e.kernel.noteApply(prev, res.Resets, res.Counters, wasServing)
	} else {
		e.noteResetsWithoutKernel(prev, res.Resets, res.Counters, wasServing)
	}
	e.resumeSampling()
	e.switchEpoch(ctx)
	return nil
}

func (e *Engine) ackApplied(ctx context.Context, gen uint64) {
	if gen == 0 || e.desired == nil || !e.desiredOK {
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
}

func (e *Engine) noteApply(err error) {
	var ce *enforce.CheckError
	code := api.ErrorCodeHEALTHCHECKFAILED
	msg := "apply failed"
	if errors.As(err, &ce) {
		msg = string(ce.Reason)
		if api.ErrorCode(ce.Reason).Valid() {
			code = api.ErrorCode(ce.Reason)
		}
	}
	e.decision.Errors = append(e.decision.Errors, api.NodeError{
		Code: code, Message: clipMsg(msg), ObservedAt: e.clock.Wall().UTC(), Stage: "apply",
	})
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
	err := watchdog.WriteHeartbeat(e.cfg.HeartbeatPath, watchdog.Beat{
		BootID: e.bootID, Mono: mono, Wall: e.clock.Wall().UTC(),
		PolicyExpires: e.policyExpires, BaselineOK: e.baselineOK, Serving: e.serving,
		Generation: e.appliedGen, Reason: reason,
	})
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

func droppedAddrs(prev []string, m enforce.Model) []netip.Addr {
	keep := map[string]bool{}
	for _, b := range m.Bindings {
		for _, p := range b.AllowedV4 {
			keep[p.Addr().String()] = true
		}
	}
	var out []netip.Addr
	for _, s := range prev {
		if keep[s] {
			continue
		}
		a, err := netip.ParseAddr(s)
		if err == nil {
			out = append(out, a)
		}
	}
	return out
}

func bindingAddrs(m enforce.Model) []string {
	var out []string
	seen := map[string]bool{}
	for _, b := range m.Bindings {
		for _, p := range b.AllowedV4 {
			s := p.Addr().String()
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	return out
}
