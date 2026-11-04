package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/controlclient"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usageupload"
)

// usageTransport posts batches on the control client's usage budget. TLS is
// the client's existing config (TLS 1.2 minimum, pinned CA, the node
// certificate). The lease budget is a different pool (§14.6).
type usageTransport struct {
	client *controlclient.Client
}

func (t usageTransport) PostUsageBatch(ctx context.Context, key string, body []byte) (int, []byte, time.Duration, error) {
	hdr := make(http.Header)
	hdr.Set("Content-Type", "application/json")
	hdr.Set("Idempotency-Key", key)
	return t.client.Exchange(controlContext(ctx), controlclient.ClassUsage, http.MethodPost, "/api/v1/agent/usage-batches", hdr, body)
}

func (t usageTransport) GetUsageCheckpoint(ctx context.Context) (int, []byte, error) {
	status, body, _, err := t.client.Exchange(controlContext(ctx), controlclient.ClassUsage, http.MethodGet, "/api/v1/agent/usage-checkpoint", nil, nil)
	return status, body, err
}

// kernelSource reads forward-admission counters from table inet fvpn and
// keeps the last State. A missing table is an empty reading so the meter
// can open before the first apply. Any other nft failure is a meter-path
// failure. The agent and the meter share one pointer so a sample and a
// step see the same external-reset latch.
type kernelSource struct {
	boot     string
	bindings func() []enforce.Binding
	expect   func() bool
	// read overrides the nft listing. Tests set it; production leaves it nil.
	read func(context.Context) (enforce.State, error)

	mu       sync.Mutex
	prev     map[string]enforce.CounterValue
	state    enforce.State
	nudge    bool
	switched bool
	pending  bool
	reported bool
	external bool
}

func (k *kernelSource) Read(ctx context.Context) (meter.Observation, error) {
	st, err := k.load(ctx)
	if err != nil {
		return meter.Observation{}, err
	}
	if st.Counters == nil {
		st.Counters = map[string]enforce.CounterValue{}
	}
	obs := meter.Observation{Counters: map[string]enforce.CounterValue{}, BootID: k.boot}
	if k.bindings != nil {
		obs.Bindings = k.bindings()
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	k.state = st
	real := st.Counters
	switch {
	case k.nudge:
		// A step already restored the ruleset. This read still has to look
		// like a drop so the meter's value comparison opens the epoch.
		obs.Counters = nudgeOne(real)
		k.nudge = false
		k.switched = true
		k.pending = true
		k.external = true
	default:
		expect := k.expect != nil && k.expect()
		_, force, report := classifyExternal(k.prev, st, expect)
		switch {
		case force && !k.switched:
			obs.Counters = nudgeOne(real)
			k.switched = true
			k.pending = true
			k.external = true
		case report && !k.reported:
			obs.Counters = cloneCounters(real)
			k.pending = true
			if brokenRuleset(st) {
				k.external = true
			}
		default:
			obs.Counters = cloneCounters(real)
			// A later healthy read ends the incident. A pending report is
			// left intact so the step can still emit it once.
			if !brokenRuleset(st) && !k.nudge && !k.pending {
				k.switched = false
				k.reported = false
				k.external = false
			}
		}
	}
	if len(real) > 0 || len(k.prev) == 0 {
		k.prev = cloneCounters(real)
	}
	return obs, nil
}

func nftMissing(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "No such file") || strings.Contains(msg, "does not exist")
}

// ensureUsage opens the ledger, the meter and the uploader once the node
// is enrolled. Sampling stays off until forwarding is admitted. The uploader
// runs immediately so a full outbox can drain while forwarding is stopped.
func (e *Engine) ensureUsage(ctx context.Context) {
	if e.usageOnce || !e.enrolled || e.client == nil {
		return
	}
	// A previous process must finish its final sample, stop BOTH loops and
	// close SQLite before this process can adopt the ledger (D-118).
	lifetime, err := nodelock.TryAcquire(e.cfg.UsageDBPath() + ".lock")
	if err != nil {
		e.usagePersist = true
		return // retry on a later Step; never admit an unowned ledger
	}
	st, err := e.openUsage(ctx)
	if err != nil {
		lifetime.Release()
		e.usagePersist = true
		e.log.Error("usage store", "err", err.Error())
		return
	}
	// Initialization is retryable. Neither the ownership lock nor a partially
	// constructed meter may survive a failed attempt (r22 M-1).
	var m *meter.Meter
	defer func() {
		if !e.usageOnce {
			if m != nil {
				m.Close()
			}
			_ = st.Close()
			lifetime.Release()
		}
	}()
	id, blocked := meterBarrierIdentity(e.cfg.JournalDir)
	if id == "" && !e.usageAdopting {
		if err := e.invalidateMeterCheckpoint(); err != nil {
			e.usagePersist = true
			return
		}
	}
	src := e.meterSource
	if src == nil {
		ks := &kernelSource{
			boot:     e.bootID.String(),
			bindings: e.meterBindings,
			expect:   func() bool { return e.expectServing.Load() },
		}
		e.kernel = ks
		src = ks
	}
	if e.retainedPending {
		head, err := st.Head(ctx)
		if err != nil {
			e.usagePersist = true
			return
		}
		bindings, err := e.restartHeadBindings(ctx, st, head)
		if err != nil {
			e.usagePersist = true
			return
		}
		e.setMeterBindings(bindings)
	}
	m, err = meter.NewContext(ctx, meter.Config{
		NodeID: e.nodeID, IncarnationID: e.incID, Store: st, Source: src, Logger: e.log,
		PreserveClosedEpoch: blocked || e.retainedPending,
		ExistingEpochOnly:   e.usageAdopting,
	})
	if err != nil {
		e.meterFailed = true
		e.usagePersist = true
		e.log.Error("usage meter", "err", err.Error())
		return
	}
	up, err := usageupload.New(usageupload.Config{
		Store: st, Transport: usageTransport{client: e.client}, Logger: e.log,
	})
	if err != nil {
		e.usagePersist = true
		e.log.Error("usage upload", "err", err.Error())
		return
	}
	e.usageLock = lifetime
	e.usage, e.usageStatus = st, st
	e.meter = m
	e.uploader = up
	e.usagePersist, e.meterFailed = false, false
	e.usageOnce = true
	if !e.usageAdopting {
		e.activateUsage(ctx)
	}
	if blocked && !e.usageAdopting {
		e.handoff(ctx)
	}
}

// activateUsage is called only after initialization and the adoption decision
// succeed. Preparing a candidate cannot start uploads or restore ledger state.
func (e *Engine) activateUsage(ctx context.Context) {
	if e.uploader == nil || e.uploadOn {
		return
	}
	up := e.uploader
	// Uploads follow the process caller, not Boot's cancellation-independent
	// local context or a Step's short HTTP budget.
	if e.runCtx == nil || e.runCtx.Err() == nil {
		e.uploadOn = true
		e.startLoops()
		parent := e.loopCtx
		if e.runCtx != nil {
			parent = e.runCtx
		}
		// Signal cancellation propagates synchronously to the request context.
		// An AfterFunc on runCtx could otherwise race the first upload.
		uploadCtx, cancel := context.WithCancel(parent)
		stop := func() bool { return false }
		if e.runCtx != nil {
			stop = context.AfterFunc(e.loopCtx, cancel)
		}
		e.loops.Add(1)
		go func() {
			defer e.loops.Done()
			defer cancel()
			defer stop()
			_ = up.Run(uploadCtx)
		}()
	}
	e.restoreUsage(ctx)
}

// discardUsagePreparation has no running loops and no ledger writes to undo.
func (e *Engine) discardUsagePreparation() {
	if e.meter != nil {
		e.meter.Close()
	}
	if e.usage != nil {
		_ = e.usage.Close()
	}
	if e.usageLock != nil {
		e.usageLock.Release()
	}
	e.meter, e.uploader, e.usage, e.usageStatus, e.usageLock = nil, nil, nil, nil, nil
	e.usageOnce = false
}

func (e *Engine) openUsage(ctx context.Context) (*usagestore.Store, error) {
	opt := usagestore.Options{Trust: e.trust}
	st, err := usagestore.OpenContext(ctx, e.cfg.UsageDBPath(), e.nodeID, e.incID, opt)
	if !errors.Is(err, usagestore.ErrIdentity) {
		return st, err
	}
	// Same-incarnation update restarts can open while the updater owns the
	// node lock. Only adoption of a different incarnation requires it here.
	lock, err := nodelock.Acquire(localWorkContext(ctx), e.cfg.LockPath, 3, 50*time.Millisecond)
	if err != nil {
		return nil, err
	}
	defer lock.Release()
	opt.RetirePreviousAt = e.clock.Wall()
	return usagestore.OpenContext(ctx, e.cfg.UsageDBPath(), e.nodeID, e.incID, opt)
}

func (e *Engine) startLoops() {
	if e.loopCancel != nil {
		return
	}
	e.loopCtx, e.loopCancel = context.WithCancel(context.Background())
}

func (e *Engine) restoreUsage(ctx context.Context) {
	if e.uploader == nil {
		return
	}
	if err := e.uploader.Restore(ctx); err != nil {
		e.log.Warn("usage checkpoint", "err", err.Error())
	}
}

func (e *Engine) meterBindings() []enforce.Binding {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]enforce.Binding, len(e.meterSnap))
	copy(out, e.meterSnap)
	return out
}

func (e *Engine) setMeterBindings(b []enforce.Binding) {
	e.mu.Lock()
	e.meterSnap = append([]enforce.Binding(nil), b...)
	e.mu.Unlock()
}

// settleRetainedUsage closes a pending restart or runtime hand-off only after
// verified Baseline stopped admission. Bindings describe the candidates that
// could have counted, scoped to the ledger interval when intent is available.
// A failed/held sample keeps the delta and its provenance pending; Step must
// retry before fetching desired state or applying a generation. Closing the
// epoch prevents retired bindings contaminating the next baseline (r21 H1/H2/M1).
func (e *Engine) settleRetainedUsage(ctx context.Context) bool {
	if !e.retainedPending {
		return true
	}
	if e.meter == nil {
		e.usagePersist = true
		return false
	}
	if !e.baselineOK {
		if err := e.baseline(ctx); err != nil {
			return false
		}
	}
	err := e.meter.FinalSample(ctx)
	if err != nil && !errors.Is(err, meter.ErrNoEpoch) {
		e.log.Warn("usage settlement", "err", err.Error())
		e.readMeter(ctx)
		e.usagePersist = true
		return false
	}
	e.retainedPending = false
	e.epochClosed = true
	e.finalDone = true
	e.setMeterBindings(nil)
	e.readMeter(ctx)
	return true
}

// readMeter copies the meter's health onto the forwarding gate.
// Store caps win inside meter.Health, so a full outbox is not reported
// as METER_PATH_INVALID.
func (e *Engine) readMeter(ctx context.Context) {
	defer func() {
		if e.retainedPending {
			e.usagePersist = true
		}
	}()
	if e.meter == nil {
		return
	}
	h, err := e.meter.Health(ctx)
	if err != nil {
		e.usagePersist = true
		e.meterFailed = false
		return
	}
	e.meterFailed = false
	e.usageOutbox = false
	e.usageDisk = false
	e.usagePersist = false
	switch h.Reason {
	case "":
		if !h.ForwardingAllowed {
			e.usagePersist = true
		}
	case meter.ReasonMeterPath:
		e.meterFailed = true
	case usagestore.ReasonPersistFail:
		e.usagePersist = true
	case usagestore.ReasonOutboxAge, usagestore.ReasonOutboxBytes, usagestore.ReasonArchiveBytes:
		e.usageOutbox = true
	case usagestore.ReasonDiskSpace:
		e.usageDisk = true
	default:
		if !h.ForwardingAllowed {
			e.usagePersist = true
		}
	}
}

// resumeSampling starts the sampler the first time forwarding is admitted
// and continues it after a hand-off that did not need a new recount.
func (e *Engine) resumeSampling() {
	if e.meter == nil {
		return
	}
	e.finalDone = false
	e.epochClosed = false
	e.finalBarrierID = ""
	e.meter.ResumeSampling()
	if e.sampleOn {
		return
	}
	e.sampleOn = true
	e.startLoops()
	e.loops.Add(1)
	go func() {
		defer e.loops.Done()
		_ = e.meter.Run(e.loopCtx)
	}()
}

// handoff installs the baseline, stops new samples, then takes the final
// sample (§13.6 step 8). A later successful apply calls resumeSampling.
// A clean final sample checkpoints the counters so the next epoch does
// not recount them (step 11).
func (e *Engine) handoff(ctx context.Context) {
	if e.usageAdopting {
		return // adoption's rejected read withdraws first; Boot settles later
	}
	// A successful final sample can heal meter health, but this pass still
	// withdrew because of the observed failure. Preserve that cause for its
	// receipt; the next Step's health refresh may clear it before a new Apply.
	persist, path, disk, outbox := e.usagePersist, e.meterFailed, e.usageDisk, e.usageOutbox
	defer func() {
		e.usagePersist = e.usagePersist || persist
		e.meterFailed = e.meterFailed || path
		e.usageDisk = e.usageDisk || disk
		e.usageOutbox = e.usageOutbox || outbox
	}()
	id, blocked := meterBarrierIdentity(e.cfg.JournalDir)
	e.latchBlocked = blocked
	// The pending latch precedes every fallible withdrawal operation, even
	// checkpoint removal. A later successful baseline retry must not allow
	// Apply to overtake the outgoing interval (r23 H-3/M-1).
	complete := e.finalDone && e.finalBarrierID == id && !e.retainedPending
	if !complete && e.meter != nil {
		e.finalDone = false
		e.retainedPending = true
		e.serving = false
		e.policyExpires = 0
		e.decision.Serve = false
		if err := e.baseline(ctx); err != nil {
			return
		}
	}
	if id == "" {
		if err := e.invalidateMeterCheckpoint(); err != nil {
			e.usagePersist = true
			e.serving = false
			e.policyExpires = 0
			// Even a completed/no-meter hand-off must withdraw on a new
			// persistence failure. Do not let ledger health clear this error.
			if complete || e.meter == nil {
				_ = e.baseline(ctx)
			}
			return
		}
	}
	if e.meter == nil || complete {
		return
	}
	// FinalSample pauses internally. If it fails, the same settlement gate
	// used at restart prevents refresh or Apply until the outgoing bytes commit.
	if !e.settleRetainedUsage(ctx) {
		return
	}
	// Accounting is settled; the updater barrier is not acknowledged until
	// its identity has been rechecked and its checkpoint has been published.
	e.finalDone = false
	cp, err := e.usage.ClosedCheckpoint(ctx)
	if err != nil {
		e.usagePersist = true
		return
	}
	current, stillBlocked := meterBarrierIdentity(e.cfg.JournalDir)
	if current != id || stillBlocked != blocked {
		// Withdrawal/replacement while closing is not acknowledgement of
		// either identity. The next pass performs a fresh barrier.
		if err := e.invalidateMeterCheckpoint(); err != nil {
			e.usagePersist = true
		}
		return
	}
	if id != "" {
		cp.BarrierID = id
		raw, err := e.meterCheckpointBytes(cp)
		if err == nil {
			err = writeAtomic(e.cfg.MeterCheckpoint, raw, 0o600)
		}
		if err != nil {
			e.usagePersist = true
			return
		}
	}
	e.finalDone = true
	e.finalBarrierID = id
}

// The published snapshot also durably pins the ACK for this identity. Uploads
// may advance it while blocked; restart must republish the original bytes.
// Reuse requires the SAME identity and completed close, never just finalDone.
func (e *Engine) meterCheckpointBytes(cp usagestore.MeterCheckpoint) ([]byte, error) {
	if raw, err := readBarrierFile(e.cfg.MeterCheckpoint); err == nil {
		var old usagestore.MeterCheckpoint
		d := json.NewDecoder(bytes.NewReader(raw))
		d.DisallowUnknownFields()
		if d.Decode(&old) == nil && d.Decode(new(any)) == io.EOF &&
			old.BarrierID == cp.BarrierID && old.ClosedMeterEpochID == cp.ClosedMeterEpochID &&
			old.LastDurableEventSeq == cp.LastDurableEventSeq && old.NewMeterEpochID == "" &&
			old.CentralAckedThrough <= cp.LastDurableEventSeq {
			canonical, err := json.Marshal(old)
			if err == nil && bytes.Equal(raw, append(canonical, '\n')) {
				return raw, nil
			}
		}
	}
	raw, err := json.Marshal(cp)
	return append(raw, '\n'), err
}

// Remove old barrier evidence before any new forwarding can start. FileMeter
// must never accept a closed epoch while the agent is sampling a later one.
func (e *Engine) invalidateMeterCheckpoint() error {
	err := os.Remove(e.cfg.MeterCheckpoint)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(e.cfg.MeterCheckpoint))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

// shutdownUsage checkpoints a healthy restart without closing its epoch;
// blocked/stale nodes still take the withdrawal hand-off. Both loops and the
// store close before another process can own the ledger.
func (e *Engine) shutdownUsage(ctx context.Context) {
	if e.shutUsage {
		return
	}
	e.shutUsage = true
	if e.sampleOn && !e.preserveServingOnStop(ctx) {
		e.handoff(ctx)
		if !e.serving {
			// Record the withdrawal. A retained serving heartbeat would
			// describe rules this hand-off just replaced, inviting a
			// same-boot successor to verify a ruleset that is gone (r20 M-3).
			// A failed baseline reports uncertainty rather than retaining a
			// successful-serving claim for an unverified kernel.
			e.decision = Decide(e.input())
			e.beat()
		}
	}
	if e.loopCancel != nil {
		e.loopCancel()
	}
	e.loops.Wait()
	if e.meter != nil {
		e.meter.Close()
		e.meter = nil
	}
	if e.usage != nil {
		_ = e.usage.Close()
		e.usage = nil
	}
	if e.usageLock != nil {
		e.usageLock.Release()
		e.usageLock = nil
	}
	e.usageStatus = nil
	e.hasUsage = false
}
