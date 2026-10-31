package meter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
)

// Fault steps observed by the store hook (§10.5, AT16). A crash here is
// before any database write.
const (
	StepRead = "sample.read"
	StepPlan = "sample.plan"
)

// ReasonMeterPath is the Health reason when the counter source cannot be
// read. Member forwarding stops; the control channel is not this package's
// concern (§10.7).
const ReasonMeterPath = "meter_path_invalid"

var (
	// ErrNoEpoch means FinalSample found no open meter epoch (§13.6 step 8).
	ErrNoEpoch = errors.New("meter: no open meter epoch to finish")
	// ErrBusy means a sample is already running.
	ErrBusy = errors.New("meter: sample already in progress")
	// ErrClosed means Close has been called.
	ErrClosed = errors.New("meter: closed")
	// ErrPaused means Barrier is in force. ResumeSampling clears it.
	ErrPaused = errors.New("meter: sampling is paused")
)

// Source reads kernel counters and the binding snapshot. The agent supplies
// the nftables reader. Tests supply a fake.
type Source interface {
	Read(ctx context.Context) (Observation, error)
}

// Config builds a Meter. Store and Source are required. Interval defaults
// to 10s (§10.5, 建議預設).
type Config struct {
	NodeID        uuid.UUID
	IncarnationID uuid.UUID
	Store         *usagestore.Store
	Source        Source
	Interval      time.Duration
	Now           func() time.Time
	// PreserveClosedEpoch keeps a completed final_sample closed at blocked
	// startup. PrepareSampling must open the next epoch before admission.
	PreserveClosedEpoch bool
	// Mono is the monotonic clock used to space samples (§10.5). Nil means
	// time.Now, whose Sub uses the monotonic reading. Timestamps stored on
	// events still come from Now.
	Mono   func() time.Time
	Logger *slog.Logger
}

// Controller is the surface the updater and the forwarding gate call
// (§13.6 steps 8 and 11, §10.7). The agent process owns the node lock and
// the actual forwarding change. This package only reports the decision.
type Controller interface {
	Barrier(ctx context.Context) error
	FinalSample(ctx context.Context) error
	Health(ctx context.Context) (usagestore.Health, error)
}

// Meter samples forward-admission counters into the usage store.
type Meter struct {
	store    *usagestore.Store
	src      Source
	interval time.Duration
	now      func() time.Time
	mono     func() time.Time
	log      *slog.Logger
	lastMono time.Time

	mu           sync.Mutex
	cond         *sync.Cond
	paused       bool
	sampling     bool
	closed       bool
	inFinal      bool
	sourceFailed bool
	wake         chan struct{}
}

var _ Controller = (*Meter)(nil)

// New checks the open meter epoch against the current counters (§10.7) and
// returns a meter ready to sample. A source that cannot be read fails the
// constructor: the agent must not forward when it cannot meter.
func New(cfg Config) (*Meter, error) {
	if cfg.Store == nil || cfg.Source == nil || cfg.NodeID == uuid.Nil || cfg.IncarnationID == uuid.Nil {
		return nil, fmt.Errorf("meter: store, source, node and incarnation are required")
	}
	id, inc := cfg.Store.Identity()
	if id != cfg.NodeID || inc != cfg.IncarnationID {
		return nil, fmt.Errorf("meter: store identity does not match the meter")
	}
	interval := cfg.Interval
	if interval <= 0 {
		interval = DefaultSampleInterval
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	mono := cfg.Mono
	if mono == nil {
		mono = time.Now
	}
	log := cfg.Logger
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	m := &Meter{
		store: cfg.Store, src: cfg.Source, interval: interval, now: now, mono: mono, log: log,
		wake: make(chan struct{}, 1),
	}
	m.cond = sync.NewCond(&m.mu)
	if cfg.PreserveClosedEpoch {
		if _, err := m.store.ClosedCheckpoint(context.Background()); err == nil {
			m.paused = true
			return m, nil
		}
	}
	if err := m.reconcile(context.Background()); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Meter) nowUTC() time.Time { return m.now().UTC() }

func (m *Meter) reconcile(ctx context.Context) error {
	obs, err := m.src.Read(ctx)
	if err != nil {
		m.noteSource(err)
		return err
	}
	head, err := m.store.Head(ctx)
	if err != nil {
		m.store.NotePersistFailure()
		return err
	}
	p, err := planContinuity(head, obs, m.nowUTC())
	if err != nil {
		return err
	}
	if p.noop {
		m.clearSource()
		m.markSampled()
		return nil
	}
	if err := m.persist(ctx, p, obs.BootID); err != nil {
		return err
	}
	m.markSampled()
	return nil
}

// Sample takes one interval. A quiet interval updates the checkpoint time
// and does not invent an event (§10.5).
func (m *Meter) Sample(ctx context.Context) error {
	return m.sample(ctx, false)
}

func (m *Meter) sample(ctx context.Context, final bool) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return ErrClosed
	}
	if m.paused && !m.inFinal && !final {
		m.mu.Unlock()
		return ErrPaused
	}
	if m.sampling {
		m.mu.Unlock()
		return ErrBusy
	}
	m.sampling = true
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.sampling = false
		m.cond.Broadcast()
		m.mu.Unlock()
	}()

	if err := m.store.Fault(StepRead); err != nil {
		return err
	}
	obs, err := m.src.Read(ctx)
	if err != nil {
		m.noteSource(err)
		return err
	}
	if err := m.store.Fault(StepPlan); err != nil {
		return err
	}
	head, err := m.store.Head(ctx)
	if err != nil {
		m.store.NotePersistFailure()
		return err
	}
	if final && head.OpenEpoch == uuid.Nil {
		m.clearSource()
		return ErrNoEpoch
	}
	p, err := planSample(head, obs, m.nowUTC(), m.interval, final)
	if err != nil {
		return err
	}
	if p.hold {
		return ErrHold
	}
	if p.noop {
		m.clearSource()
		m.markSampled()
		return nil
	}
	if err := m.persist(ctx, p, obs.BootID); err != nil {
		return err
	}
	m.markSampled()
	if final && p.commit.Open != nil {
		// The switch left a fresh epoch open. Close it without an event
		// so the next process does not treat the handoff as a crash
		// (§13.6 step 11).
		head2, err := m.store.Head(ctx)
		if err != nil {
			m.store.NotePersistFailure()
			return err
		}
		closePlan := plan{commit: usagestore.Commit{
			Epoch:       head2.OpenEpoch,
			Counters:    head2.Counters,
			BootID:      obs.BootID,
			CloseEpoch:  true,
			CloseReason: "final_sample",
			SampledAt:   head2.LastSample,
		}}
		if err := m.persist(ctx, closePlan, obs.BootID); err != nil {
			return err
		}
	}
	return nil
}

func (m *Meter) persist(ctx context.Context, p plan, boot string) error {
	seq, err := m.store.Commit(ctx, p.commit)
	if err != nil {
		if errors.Is(err, usagestore.ErrInjectedCrash) {
			// A crash after commit left the row durable. Do not flag the
			// ledger as unwritable (§10.5, AT16).
			if seq != 0 || p.commit.Event == nil {
				m.store.NotePersistSuccess()
			}
			return err
		}
		m.store.NotePersistFailure()
		return err
	}
	m.store.NotePersistSuccess()
	m.clearSource()
	if p.commit.Open != nil {
		m.log.Info("meter epoch opened",
			"epoch_id", p.commit.Open.ID.String(),
			"reason", p.commit.Open.Reason,
			"boot_id", boot,
			"event_seq", seq)
	} else if p.commit.CloseEpoch {
		m.log.Info("meter epoch closed",
			"epoch_id", p.commit.Epoch.String(),
			"reason", p.commit.CloseReason,
			"event_seq", seq)
	}
	return nil
}

// Barrier pauses the sampler and waits until the in-flight sample finishes
// (§13.6 step 8). ResumeSampling continues a sample loop that has not yet
// taken the final sample.
func (m *Meter) Barrier(ctx context.Context) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return ErrClosed
	}
	m.paused = true
	if !m.sampling {
		m.mu.Unlock()
		return nil
	}
	stop := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			m.mu.Lock()
			m.cond.Broadcast()
			m.mu.Unlock()
		case <-stop:
		}
	}()
	for m.sampling {
		m.cond.Wait()
		if ctx.Err() != nil && m.sampling {
			close(stop)
			m.mu.Unlock()
			return ctx.Err()
		}
	}
	m.mu.Unlock()
	close(stop)
	return nil
}

// FinalSample takes the closing sample and shuts the meter epoch (§13.6
// step 8). The next process opens a new epoch at the checkpointed counters.
func (m *Meter) FinalSample(ctx context.Context) error {
	return m.pausedSample(ctx, true)
}

// CheckpointForRestart persists the last observed delta without closing the
// epoch or clearing the kernel counters. A same-boot successor accounts for
// bytes admitted during the process gap using the existing continuity path.
// Updater withdrawal must still use FinalSample.
func (m *Meter) CheckpointForRestart(ctx context.Context) error {
	return m.pausedSample(ctx, false)
}

func (m *Meter) pausedSample(ctx context.Context, final bool) error {
	if err := m.Barrier(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return ErrClosed
	}
	m.inFinal = true
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.inFinal = false
		m.mu.Unlock()
	}()
	return m.sample(ctx, final)
}

// PrepareSampling durably opens the next epoch from a real counter read while
// keeping the sampler paused. The caller must keep forwarding blocked until
// this succeeds, then install its admission rules and call ResumeSampling.
// Otherwise the first admitted bytes could become the new epoch's baseline.
func (m *Meter) PrepareSampling(ctx context.Context) error {
	if err := m.Barrier(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return ErrClosed
	}
	if m.sampling {
		m.mu.Unlock()
		return ErrBusy
	}
	m.sampling = true
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.sampling = false
		m.cond.Broadcast()
		m.mu.Unlock()
	}()
	return m.reconcile(ctx)
}

// ResumeSampling clears a barrier after any required PrepareSampling.
func (m *Meter) ResumeSampling() {
	m.mu.Lock()
	m.paused = false
	m.mu.Unlock()
	m.poke()
}

// Health is the forwarding gate (§10.7, §10.8). Store caps win when both
// a cap and a source failure are set. A source failure alone stops
// forwarding with ReasonMeterPath.
func (m *Meter) Health(ctx context.Context) (usagestore.Health, error) {
	h, err := m.store.EvaluateGate(ctx)
	if err != nil {
		return usagestore.Health{}, err
	}
	m.mu.Lock()
	failed := m.sourceFailed
	m.mu.Unlock()
	if failed && h.Reason == "" {
		h.ForwardingAllowed = false
		h.Reason = ReasonMeterPath
		h.Detail = "the meter could not read counters; member forwarding is stopped (§10.7)"
	}
	return h, nil
}

// Run samples until ctx is cancelled. The wait is the earlier of the sample
// interval and the next UTC month boundary (§10.5, §10.9).
func (m *Meter) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		m.mu.Lock()
		paused := m.paused || m.closed
		m.mu.Unlock()
		if !paused {
			err := m.Sample(ctx)
			if err != nil && ctx.Err() == nil && !errors.Is(err, ErrHold) && !errors.Is(err, ErrPaused) && !errors.Is(err, ErrBusy) {
				m.log.Warn("usage sample failed", "err", err.Error())
			}
		}
		delay := m.nextWait(ctx)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-m.wake:
			timer.Stop()
		case <-timer.C:
		}
	}
}

// Close stops the sampler. It does not close the store.
func (m *Meter) Close() {
	m.mu.Lock()
	m.closed = true
	m.paused = true
	m.cond.Broadcast()
	m.mu.Unlock()
	m.poke()
}

// nextWait is the earlier of the remaining monotonic sample interval and
// the next UTC month boundary (§10.5, §10.9). A wall-clock jump does not
// shorten or stretch the interval. Once the wall clock is already at or
// past the boundary, the sampler wakes immediately so the interval can be
// stamped there.
func (m *Meter) nextWait(ctx context.Context) time.Duration {
	head, err := m.store.Head(ctx)
	if err != nil {
		return m.interval
	}
	nowWall := m.nowUTC()
	m.mu.Lock()
	last := m.lastMono
	m.mu.Unlock()
	var monoWait time.Duration
	if !last.IsZero() {
		elapsed := m.mono().Sub(last)
		if elapsed < 0 {
			elapsed = 0
		}
		monoWait = m.interval - elapsed
		if monoWait < 0 {
			monoWait = 0
		}
	} else {
		monoWait = NextDelay(head.LastSample, nowWall, m.interval)
	}
	if head.LastSample.IsZero() {
		return monoWait
	}
	boundary := nextBoundaryAfter(head.LastSample)
	if !nowWall.Before(boundary) {
		return 0
	}
	until := boundary.Sub(nowWall)
	if until < monoWait {
		return until
	}
	return monoWait
}

func (m *Meter) markSampled() {
	m.mu.Lock()
	m.lastMono = m.mono()
	m.mu.Unlock()
}

func (m *Meter) poke() {
	select {
	case m.wake <- struct{}{}:
	default:
	}
}

func (m *Meter) noteSource(err error) {
	m.mu.Lock()
	m.sourceFailed = true
	m.mu.Unlock()
	m.log.Warn("usage counter read failed", "err", err.Error())
}

func (m *Meter) clearSource() {
	m.mu.Lock()
	m.sourceFailed = false
	m.mu.Unlock()
}
