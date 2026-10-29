package usageupload

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

const (
	// DefaultInterval is the §10.5 recommended upload period.
	DefaultInterval = 60 * time.Second
	// DefaultMaxEvents and DefaultMaxBytes start a batch early. The spec
	// does not set these numbers. The contract maximum is 1000 events.
	DefaultMaxEvents = 100
	DefaultMaxBytes  = 512_000

	defaultBackoffBase   = time.Second
	defaultBackoffCap    = 5 * time.Minute
	defaultBackoffFactor = 2
	jitterFrac           = 0.2
)

// Fault steps. A hook error aborts the attempt before the named side effect.
const (
	StepBuild = "upload.build"
	StepPost  = "upload.post"
	StepApply = "upload.apply"
)

// Transport is the central usage API. The agent passes an mTLS client
// through HTTPTransport. Tests pass a fake.
type Transport interface {
	PostUsageBatch(ctx context.Context, idempotencyKey string, body []byte) (status int, resp []byte, retryAfter time.Duration, err error)
	GetUsageCheckpoint(ctx context.Context) (status int, body []byte, err error)
}

// Hook is a test fault injector. Production leaves it nil.
type Hook func(step string) error

// Sleeper waits. Tests record the delay instead of sleeping.
type Sleeper func(ctx context.Context, d time.Duration) error

// Config builds an Uploader.
type Config struct {
	Store     *usagestore.Store
	Transport Transport
	Interval  time.Duration
	MaxEvents int
	MaxBytes  int
	Now       func() time.Time
	Sleep     Sleeper
	// Rand returns a value in [0, 1). Nil disables jitter.
	Rand   func() float64
	Logger *slog.Logger
	Hook   Hook
	// BackoffBase, BackoffCap and BackoffFactor override the engineering
	// defaults. Zero keeps the default.
	BackoffBase   time.Duration
	BackoffCap    time.Duration
	BackoffFactor int
}

// Uploader drains the outbox. Upload continues while member forwarding is
// stopped (§10.8).
type Uploader struct {
	store    *usagestore.Store
	tr       Transport
	interval time.Duration
	maxEv    int
	maxBytes int
	now      func() time.Time
	sleep    Sleeper
	rand     func() float64
	log      *slog.Logger
	hook     Hook

	base   time.Duration
	cap    time.Duration
	factor int

	bo       time.Duration
	lastSend time.Time
}

// New validates the config. It does not dial the network.
func New(cfg Config) (*Uploader, error) {
	if cfg.Store == nil || cfg.Transport == nil {
		return nil, fmt.Errorf("usageupload: store and transport are required")
	}
	u := &Uploader{
		store: cfg.Store, tr: cfg.Transport,
		interval: cfg.Interval, maxEv: cfg.MaxEvents, maxBytes: cfg.MaxBytes,
		now: cfg.Now, sleep: cfg.Sleep, rand: cfg.Rand, log: cfg.Logger, hook: cfg.Hook,
		base: cfg.BackoffBase, cap: cfg.BackoffCap, factor: cfg.BackoffFactor,
	}
	if u.interval <= 0 {
		u.interval = DefaultInterval
	}
	if u.maxEv <= 0 {
		u.maxEv = DefaultMaxEvents
	}
	if u.maxBytes <= 0 {
		u.maxBytes = DefaultMaxBytes
	}
	if u.now == nil {
		u.now = time.Now
	}
	if u.sleep == nil {
		u.sleep = func(ctx context.Context, d time.Duration) error {
			if d <= 0 {
				return nil
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
	}
	if u.log == nil {
		u.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if u.base <= 0 {
		u.base = defaultBackoffBase
	}
	if u.cap <= 0 {
		u.cap = defaultBackoffCap
	}
	if u.factor < 2 {
		u.factor = defaultBackoffFactor
	}
	return u, nil
}

// Once sends at most one batch if the outbox or a replay-unavailable range
// is pending. It applies the ACK only after a 200 whose ledger epoch matches
// the verified checkpoint.
func (u *Uploader) Once(ctx context.Context) error {
	body, key, events, err := u.prepare(ctx)
	if err != nil || body == nil {
		return err
	}
	if err := u.hookStep(StepPost); err != nil {
		return err
	}
	status, resp, retryAfter, err := u.tr.PostUsageBatch(ctx, key, body)
	if err != nil {
		u.log.Warn("usage batch post failed", "err", err.Error())
		return err
	}
	return u.handle(ctx, status, resp, retryAfter, events)
}

func (u *Uploader) prepare(ctx context.Context) (body []byte, key string, events []usagestore.StoredEvent, err error) {
	head, err := u.store.Head(ctx)
	if err != nil {
		return nil, "", nil, err
	}
	events, err = u.store.Unacked(ctx, head.AckedThrough, u.maxEv, u.maxBytes)
	if err != nil {
		return nil, "", nil, err
	}
	ranges, err := u.store.ReplayUnavailable(ctx)
	if err != nil {
		return nil, "", nil, err
	}
	if len(events) == 0 && len(ranges) == 0 {
		return nil, "", nil, nil
	}
	epoch := json.RawMessage("null")
	if head.LedgerEpoch != uuid.Nil {
		raw, err := json.Marshal(head.LedgerEpoch)
		if err != nil {
			return nil, "", nil, err
		}
		epoch = raw
	}
	node, inc := u.store.Identity()
	wire := wireBatch{
		Events:            make([]json.RawMessage, len(events)),
		LedgerEpoch:       epoch,
		NodeID:            node,
		NodeIncarnationID: inc,
	}
	for i, ev := range events {
		wire.Events[i] = json.RawMessage(ev.Payload)
	}
	if len(ranges) > 0 {
		wire.Replay = ranges
	}
	built, err := json.Marshal(wire)
	if err != nil {
		return nil, "", nil, err
	}
	if err := u.hookStep(StepBuild); err != nil {
		return nil, "", nil, err
	}
	sum := sha256.Sum256(built)
	sha := hex.EncodeToString(sum[:])
	savedKey, savedBody, ok, err := u.store.LookupBatch(ctx, sha)
	if err != nil {
		return nil, "", nil, err
	}
	if ok {
		return savedBody, savedKey, events, nil
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, "", nil, fmt.Errorf("usageupload: idempotency key: %w", err)
	}
	var first, last uint64
	if len(events) > 0 {
		first = events[0].Seq
		last = events[len(events)-1].Seq
	}
	if err := u.store.RememberBatch(ctx, sha, id.String(), built, first, last); err != nil {
		return nil, "", nil, err
	}
	return built, id.String(), events, nil
}

type wireBatch struct {
	Events            []json.RawMessage            `json:"events"`
	LedgerEpoch       json.RawMessage              `json:"ledger_epoch"`
	NodeID            uuid.UUID                    `json:"node_id"`
	NodeIncarnationID uuid.UUID                    `json:"node_incarnation_id"`
	Replay            []api.ReplayUnavailableRange `json:"replay_unavailable,omitempty"`
}

func (u *Uploader) handle(ctx context.Context, status int, resp []byte, retryAfter time.Duration, events []usagestore.StoredEvent) error {
	switch status {
	case 200:
		return u.on200(ctx, resp)
	case 409:
		return u.on409(ctx, resp, events)
	case 429:
		d := retryAfter
		if d < 0 {
			d = 0
		}
		u.log.Warn("usage batch throttled", "retry_after", d.String())
		if d > 0 {
			if err := u.sleep(ctx, d); err != nil {
				return err
			}
		}
		return fmt.Errorf("usageupload: throttled")
	case 503:
		u.log.Warn("usage batch unavailable", "status", status)
		return fmt.Errorf("usageupload: ledger unavailable")
	case 413:
		return fmt.Errorf("usageupload: batch too large")
	case 422, 403:
		code := errorCode(resp)
		u.log.Warn("usage batch rejected", "status", status, "code", code)
		return fmt.Errorf("usageupload: status %d %s", status, code)
	default:
		if status >= 500 {
			return fmt.Errorf("usageupload: status %d", status)
		}
		return fmt.Errorf("usageupload: status %d %s", status, errorCode(resp))
	}
}

func (u *Uploader) on200(ctx context.Context, resp []byte) error {
	var res api.UsageBatchResult
	if err := json.Unmarshal(resp, &res); err != nil {
		return fmt.Errorf("usageupload: batch result: %w", err)
	}
	ack := usagestore.BatchACK{
		LedgerEpoch: res.LedgerEpoch,
		Highest:     uint64(res.HighestContiguousSeq),
		Results:     make([]usagestore.EventResult, 0, len(res.Results)),
	}
	if res.BlockedSeq.IsSpecified() && !res.BlockedSeq.IsNull() {
		v, err := res.BlockedSeq.Get()
		if err != nil {
			return fmt.Errorf("usageupload: blocked_seq: %w", err)
		}
		n := uint64(v)
		ack.Blocked = &n
	}
	for _, r := range res.Results {
		ack.Results = append(ack.Results, usagestore.EventResult{
			Seq: uint64(r.EventSeq), Status: string(r.Status),
		})
	}
	if err := u.hookStep(StepApply); err != nil {
		return err
	}
	err := u.store.ApplyACK(ctx, ack)
	if errors.Is(err, usagestore.ErrLedgerEpoch) {
		u.log.Warn("usage batch ledger epoch mismatch")
		return u.fetchCheckpoint(ctx)
	}
	if err != nil {
		return err
	}
	u.log.Info("usage batch acknowledged", "highest_contiguous_seq", ack.Highest)
	return nil
}

func (u *Uploader) on409(ctx context.Context, resp []byte, events []usagestore.StoredEvent) error {
	var env api.ErrorEnvelope
	if err := json.Unmarshal(resp, &env); err != nil {
		return fmt.Errorf("usageupload: error envelope: %w", err)
	}
	switch env.Error.Code {
	case api.ErrorCodeLEDGEREPOCHMISMATCH:
		u.log.Warn("usage ledger epoch mismatch")
		return u.fetchCheckpoint(ctx)
	case api.ErrorCodeUSAGEEVENTCONFLICT:
		if len(events) == 0 {
			return fmt.Errorf("usageupload: conflict with an empty batch")
		}
		return u.block(ctx, events[0].Seq)
	default:
		return fmt.Errorf("usageupload: status 409 %s", env.Error.Code)
	}
}

// block records a hash conflict on seq and does not drop the payload (§10.6).
func (u *Uploader) block(ctx context.Context, seq uint64) error {
	head, err := u.store.Head(ctx)
	if err != nil {
		return err
	}
	ack := usagestore.BatchACK{
		LedgerEpoch: head.LedgerEpoch,
		Highest:     head.AckedThrough,
		Blocked:     &seq,
		Results:     []usagestore.EventResult{{Seq: seq, Status: usagestore.StatusConflict}},
	}
	if err := u.hookStep(StepApply); err != nil {
		return err
	}
	if err := u.store.ApplyACK(ctx, ack); err != nil {
		return err
	}
	u.log.Warn("usage event conflict", "blocked_seq", seq)
	return fmt.Errorf("usageupload: event conflict at seq %s", decstr.U64(seq).String())
}

func (u *Uploader) fetchCheckpoint(ctx context.Context) error {
	status, body, err := u.tr.GetUsageCheckpoint(ctx)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("usageupload: checkpoint status %d", status)
	}
	if err := u.store.ApplyCheckpoint(ctx, body, u.now()); err != nil {
		return err
	}
	u.log.Info("usage checkpoint applied")
	return nil
}

func (u *Uploader) hookStep(step string) error {
	if u.hook == nil {
		return nil
	}
	if err := u.hook(step); err != nil {
		return fmt.Errorf("usageupload: injected stop at %s: %w", step, err)
	}
	return nil
}

func errorCode(resp []byte) string {
	var env api.ErrorEnvelope
	if err := json.Unmarshal(resp, &env); err != nil {
		return ""
	}
	return string(env.Error.Code)
}

// Run uploads until ctx is cancelled. A full outbox sends before the 60s
// period. Failures wait with bounded exponential backoff and ±20% jitter.
func (u *Uploader) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		due, err := u.due(ctx)
		if err != nil {
			return err
		}
		if due {
			err = u.Once(ctx)
			if err != nil && ctx.Err() == nil {
				d := u.nextBackoff()
				u.log.Warn("usage upload retry", "backoff", d.String(), "err", err.Error())
				if sleepErr := u.sleep(ctx, d); sleepErr != nil {
					return sleepErr
				}
				continue
			}
			if err != nil {
				return err
			}
			u.bo = 0
			u.lastSend = u.now()
			if _, perr := u.store.PurgeArchive(ctx, u.now()); perr != nil {
				u.log.Warn("usage archive purge failed", "err", perr.Error())
			}
		}
		wait := u.interval
		hot, err := u.overThreshold(ctx)
		if err != nil {
			return err
		}
		if hot {
			wait = 0
		} else if !u.lastSend.IsZero() {
			remain := u.interval - u.now().Sub(u.lastSend)
			if remain < wait {
				wait = remain
			}
		}
		if wait < 0 {
			wait = 0
		}
		if err := u.sleep(ctx, wait); err != nil {
			return err
		}
	}
}

func (u *Uploader) due(ctx context.Context) (bool, error) {
	if u.lastSend.IsZero() || u.now().Sub(u.lastSend) >= u.interval {
		head, err := u.store.Head(ctx)
		if err != nil {
			return false, err
		}
		events, err := u.store.Unacked(ctx, head.AckedThrough, 1, u.maxBytes)
		if err != nil {
			return false, err
		}
		if len(events) > 0 {
			return true, nil
		}
		ranges, err := u.store.ReplayUnavailable(ctx)
		if err != nil {
			return false, err
		}
		return len(ranges) > 0, nil
	}
	return u.overThreshold(ctx)
}

func (u *Uploader) overThreshold(ctx context.Context) (bool, error) {
	st, err := u.store.OutboxStatus(ctx)
	if err != nil {
		return false, err
	}
	if st.UnackedEventCount >= u.maxEv {
		return true, nil
	}
	if uint64(st.UnackedPayloadBytes) >= uint64(u.maxBytes) {
		return true, nil
	}
	return false, nil
}

func (u *Uploader) nextBackoff() time.Duration {
	if u.bo == 0 {
		u.bo = u.base
	} else {
		next := u.bo * time.Duration(u.factor)
		if next < u.bo || next > u.cap {
			next = u.cap
		}
		u.bo = next
	}
	d := u.bo
	if u.rand != nil {
		j := (u.rand()*2 - 1) * jitterFrac
		d = time.Duration(float64(u.bo) * (1 + j))
	}
	if d < 0 {
		d = u.base
	}
	if d > u.cap {
		d = u.cap
	}
	return d
}
