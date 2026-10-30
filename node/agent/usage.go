package agent

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/controlclient"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
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
	return t.client.Exchange(ctx, controlclient.ClassUsage, http.MethodPost, "/api/v1/agent/usage-batches", hdr, body)
}

func (t usageTransport) GetUsageCheckpoint(ctx context.Context) (int, []byte, error) {
	status, body, _, err := t.client.Exchange(ctx, controlclient.ClassUsage, http.MethodGet, "/api/v1/agent/usage-checkpoint", nil, nil)
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
	e.usageOnce = true
	st, err := usagestore.Open(e.cfg.UsageDBPath(), e.nodeID, e.incID, usagestore.Options{Trust: e.trust})
	if err != nil {
		e.usagePersist = true
		e.log.Error("usage store", "err", err.Error())
		return
	}
	e.usage = st
	e.usageStatus = st
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
	m, err := meter.New(meter.Config{
		NodeID: e.nodeID, IncarnationID: e.incID, Store: st, Source: src, Logger: e.log,
	})
	if err != nil {
		e.meterFailed = true
		e.log.Error("usage meter", "err", err.Error())
	} else {
		e.meter = m
	}
	up, err := usageupload.New(usageupload.Config{
		Store: st, Transport: usageTransport{client: e.client}, Logger: e.log,
	})
	if err != nil {
		e.usagePersist = true
		e.log.Error("usage upload", "err", err.Error())
		return
	}
	e.uploader = up
	e.startLoops()
	e.loops.Add(1)
	go func() {
		defer e.loops.Done()
		_ = up.Run(e.loopCtx)
	}()
	e.restoreUsage(ctx)
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

// readMeter copies the meter's health onto the forwarding gate.
// Store caps win inside meter.Health, so a full outbox is not reported
// as METER_PATH_INVALID.
func (e *Engine) readMeter(ctx context.Context) {
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
	if e.sampleOn {
		e.meter.ResumeSampling()
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

// handoff stops new samples, installs the baseline, then takes the final
// sample (§13.6 step 8). A later successful apply calls resumeSampling.
// A clean final sample checkpoints the counters so the next epoch does
// not recount them (step 11).
func (e *Engine) handoff(ctx context.Context) {
	if e.meter == nil || !e.sampleOn || e.finalDone {
		return
	}
	_ = e.meter.Barrier(ctx)
	_ = e.baseline(ctx)
	err := e.meter.FinalSample(ctx)
	if err != nil && ctx.Err() == nil && !errors.Is(err, meter.ErrNoEpoch) && !errors.Is(err, meter.ErrHold) {
		e.log.Warn("final sample", "err", err.Error())
	}
	e.finalDone = true
	e.serving = false
}

// shutdownUsage is the process stop sequence: the same hand-off as a
// forwarding withdrawal, then the uploader and the store are closed.
func (e *Engine) shutdownUsage() {
	if e.shutUsage {
		return
	}
	e.shutUsage = true
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	e.handoff(ctx)
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
	e.usageStatus = nil
	e.hasUsage = false
}
