package agent

import (
	"context"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// usageStatusBoundDefault is the observation's wait for one outbox read.
// The report is sent when the bound expires; the store call is not joined.
const usageStatusBoundDefault = 2 * time.Second

// usageStatusSource is the observation's view of the local usage stream.
// *usagestore.Store satisfies it. A nil source means the store is not open.
type usageStatusSource interface {
	OutboxStatus(context.Context) (api.UsageOutboxStatus, error)
}

var _ usageStatusSource = (*usagestore.Store)(nil)

type usageRead struct {
	st  api.UsageOutboxStatus
	err error
}

// zeroUsageStatus is the contract shape sent before the store is open and
// when a read fails. Nullable members are JSON null. Counts are "0" or 0.
func zeroUsageStatus() api.UsageOutboxStatus {
	return api.UsageOutboxStatus{
		BlockedSeq:      presence.NewNullNullable[decstr.U64](),
		LedgerEpoch:     presence.NewNullNullable[uuid.UUID](),
		MeterEpochId:    presence.NewNullNullable[uuid.UUID](),
		OldestUnackedAt: presence.NewNullNullable[time.Time](),
	}
}

// attachUsage fills rep.Usage from the open store. The read never changes
// safety, provisioning, forwarding, or errors: a miss keeps the zero shape,
// and a timeout reuses the last success from this process.
func (e *Engine) attachUsage(ctx context.Context, rep *api.ObservationReport) {
	st, reason := e.readUsageStatus(ctx)
	rep.Usage = st
	e.noteUsageStatus(reason)
}

func (e *Engine) usageReadBound() time.Duration {
	if e.usageStatusBound > 0 {
		return e.usageStatusBound
	}
	return usageStatusBoundDefault
}

// readUsageStatus bounds the store read. The goroutine only sends on ch.
// It does not touch the engine, so a slow store cannot change a report
// that has already been sent.
func (e *Engine) readUsageStatus(ctx context.Context) (api.UsageOutboxStatus, string) {
	src := e.usageStatus
	if src == nil {
		return zeroUsageStatus(), "not_open"
	}
	readCtx, cancel := context.WithTimeout(ctx, e.usageReadBound())
	defer cancel()
	ch := make(chan usageRead, 1)
	go func() {
		st, err := src.OutboxStatus(readCtx)
		ch <- usageRead{st: st, err: err}
	}()
	select {
	case res := <-ch:
		return e.finishUsageRead(res)
	case <-readCtx.Done():
		select {
		case res := <-ch:
			return e.finishUsageRead(res)
		default:
			if errors.Is(readCtx.Err(), context.DeadlineExceeded) {
				return e.timeoutUsage()
			}
			return zeroUsageStatus(), "canceled"
		}
	}
}

func (e *Engine) finishUsageRead(res usageRead) (api.UsageOutboxStatus, string) {
	if res.err == nil {
		e.lastUsage = res.st
		e.hasUsage = true
		return res.st, ""
	}
	if errors.Is(res.err, context.DeadlineExceeded) {
		return e.timeoutUsage()
	}
	if errors.Is(res.err, context.Canceled) {
		return zeroUsageStatus(), "canceled"
	}
	return zeroUsageStatus(), "read_failed"
}

func (e *Engine) timeoutUsage() (api.UsageOutboxStatus, string) {
	if e.hasUsage {
		return e.lastUsage, "deadline"
	}
	return zeroUsageStatus(), "deadline"
}

// noteUsageStatus logs one line when the class changes. The reason is a
// short token. Store errors can carry a path and stay out of the log.
func (e *Engine) noteUsageStatus(reason string) {
	if e.usageStatusKnown && reason == e.usageStatusReason {
		return
	}
	e.usageStatusKnown = true
	e.usageStatusReason = reason
	if e.log == nil {
		return
	}
	if reason == "" {
		e.log.Info("usage status available")
		return
	}
	e.log.Warn("usage status unavailable: " + reason)
}
