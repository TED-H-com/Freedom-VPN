package fup

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// WatermarkRecord tracks the latest FUP computation timestamp and ledger watermark
// (`AggregateAsOf` and `WatermarkDetail`) for a `(member_id, period_id)` key (§11.1, §15.3).
type WatermarkRecord struct {
	MemberID        uuid.UUID
	PeriodID        string
	LastComputedAt  time.Time
	LedgerWatermark time.Time
	WatermarkDetail *LedgerWatermark
	EvaluationCount int
}

type pendingJob struct {
	MemberID         uuid.UUID
	PeriodID         string
	CurrentPeriodID  string
	AllowRelaxation  bool
	RelaxationReason string
}

// Worker coalesces ledger-change notifications and periodic schedule sweeps into
// deduplicated `Engine.Compute` runs per `(member_id, period_id)` and records
// `last_computed_at` and the ledger watermark (`aggregate_as_of`).
type Worker struct {
	engine *Engine
	wakeCh chan struct{}

	mu         sync.Mutex
	pending    map[string]pendingJob
	watermarks map[string]WatermarkRecord
}

// NewWorker creates a coalesced FUP evaluation worker backed by `engine`.
func NewWorker(engine *Engine) *Worker {
	return &Worker{
		engine:     engine,
		wakeCh:     make(chan struct{}, 1),
		pending:    make(map[string]pendingJob),
		watermarks: make(map[string]WatermarkRecord),
	}
}

// NotifyLedgerChange enqueues a coalesced FUP evaluation for `(memberID, periodID)`.
// Multiple notifications for the same `(memberID, periodID)` before the next flush
// are coalesced into a single `Engine.Compute` call.
func (w *Worker) NotifyLedgerChange(memberID uuid.UUID, periodID string) {
	w.enqueue(pendingJob{
		MemberID: memberID,
		PeriodID: periodID,
	})
}

// NotifyLedgerCorrection enqueues a coalesced FUP evaluation with `AllowRelaxation: true`
// for an audited negative ledger adjustment (§11.1).
func (w *Worker) NotifyLedgerCorrection(memberID uuid.UUID, periodID string, reason string) {
	w.enqueue(pendingJob{
		MemberID:         memberID,
		PeriodID:         periodID,
		AllowRelaxation:  true,
		RelaxationReason: reason,
	})
}

// ScheduleMemberSweep enqueues a periodic sweep for the given members in `periodID`
// (for example, to evaluate override expiries or policy `effective_at` cutovers).
func (w *Worker) ScheduleMemberSweep(memberIDs []uuid.UUID, periodID string) {
	for _, id := range memberIDs {
		w.enqueue(pendingJob{
			MemberID: id,
			PeriodID: periodID,
		})
	}
}

func (w *Worker) enqueue(job pendingJob) {
	key := stateKey(job.MemberID, job.PeriodID)
	w.mu.Lock()
	if existing, ok := w.pending[key]; ok {
		if job.AllowRelaxation {
			existing.AllowRelaxation = true
			if job.RelaxationReason != "" {
				existing.RelaxationReason = job.RelaxationReason
			}
		}
		if job.CurrentPeriodID != "" {
			existing.CurrentPeriodID = job.CurrentPeriodID
		}
		w.pending[key] = existing
	} else {
		w.pending[key] = job
	}
	w.mu.Unlock()

	select {
	case w.wakeCh <- struct{}{}:
	default:
	}
}

// PendingCount returns the number of coalesced `(member_id, period_id)` jobs waiting to run.
func (w *Worker) PendingCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.pending)
}

// Flush drains all coalesced pending jobs and executes `Engine.Compute` once per unique
// `(member_id, period_id)`, recording `last_computed_at` and the ledger watermark.
func (w *Worker) Flush(ctx context.Context, now time.Time) ([]*EvaluationResult, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	w.mu.Lock()
	if len(w.pending) == 0 {
		w.mu.Unlock()
		return nil, nil
	}
	batch := make([]pendingJob, 0, len(w.pending))
	for _, job := range w.pending {
		batch = append(batch, job)
	}
	w.pending = make(map[string]pendingJob)
	w.mu.Unlock()

	sort.Slice(batch, func(i, j int) bool {
		ki := stateKey(batch[i].MemberID, batch[i].PeriodID)
		kj := stateKey(batch[j].MemberID, batch[j].PeriodID)
		return ki < kj
	})

	results := make([]*EvaluationResult, 0, len(batch))
	currentPeriodID := CurrentUTCMonthPeriodID(now)

	for _, job := range batch {
		curPeriod := job.CurrentPeriodID
		if curPeriod == "" {
			curPeriod = currentPeriodID
		}
		periodID := job.PeriodID
		if periodID == "" {
			periodID = curPeriod
		}
		res, err := w.engine.Compute(ctx, ComputeInput{
			MemberID:         job.MemberID,
			PeriodID:         periodID,
			CurrentPeriodID:  curPeriod,
			AllowRelaxation:  job.AllowRelaxation,
			RelaxationReason: job.RelaxationReason,
			Actor:            fupSystemActorUUID,
			Now:              now,
		})
		if err != nil {
			return results, err
		}
		results = append(results, res)

		key := stateKey(res.MemberID, res.PeriodID)
		w.mu.Lock()
		prev := w.watermarks[key]
		w.watermarks[key] = WatermarkRecord{
			MemberID:        res.MemberID,
			PeriodID:        res.PeriodID,
			LastComputedAt:  res.ComputedAt,
			LedgerWatermark: res.AggregateAsOf,
			WatermarkDetail: res.Watermark,
			EvaluationCount: prev.EvaluationCount + 1,
		}
		w.mu.Unlock()
	}
	return results, nil
}

// Run executes the coalesced worker loop, flushing pending jobs on wakeup signals
// or every `interval` until `ctx` is canceled.
func (w *Worker) Run(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.wakeCh:
			if _, err := w.Flush(ctx, time.Now().UTC()); err != nil && ctx.Err() != nil {
				return ctx.Err()
			}
		case t := <-ticker.C:
			if _, err := w.Flush(ctx, t.UTC()); err != nil && ctx.Err() != nil {
				return ctx.Err()
			}
		}
	}
}

// WatermarkFor returns the recorded `WatermarkRecord` for `(memberID, periodID)`.
func (w *Worker) WatermarkFor(memberID uuid.UUID, periodID string) (WatermarkRecord, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	rec, ok := w.watermarks[stateKey(memberID, periodID)]
	return rec, ok
}
