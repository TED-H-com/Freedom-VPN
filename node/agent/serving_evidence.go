package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// servingRead is a retained read, not the observation time or a generation ACK.
type servingRead struct {
	Evidence   enforce.ServingEvidence
	Generation uint64
	At         time.Time
	Source     string
	Verified   bool
}

func (e *Engine) retainServingRead(ev enforce.ServingEvidence, gen uint64, at time.Time, source string, verified bool) {
	e.retainTCReadback(ev)
	e.servingRead = &servingRead{ev, gen, at.UTC(), source, verified}
}

// Capture before diagnostic serialization: the enforce snapshot is private and
// authoritative for read time, complete verifier outcome, bounds and digest.
// A call with no read does not erase the last actual read or fabricate success.
func (e *Engine) retainTCReadback(ev enforce.ServingEvidence) {
	if tc := ev.TCReadbackEvidence(); tc != nil {
		e.tcReadback = tc
	}
}

func (e *Engine) attachServingRead(rep *api.ObservationReport) {
	rep.TcEvidence.SetNull()
	if e.tcReadback != nil {
		rep.TcEvidence.Set(*e.tcReadback)
	}
	r := e.servingRead
	if r == nil {
		return
	}
	for i := range rep.Checks {
		c := &rep.Checks[i]
		if c.Name != api.NodeCheckNameQos {
			continue
		}
		// Copying the checks in report prevents mutation of the actual decision.
		// Status/error still come from Decide, including after withdrawal.
		detail := fmt.Sprintf("read_at=%s source=%s candidate=%d verified=%t %s", r.At.Format(time.RFC3339Nano), r.Source, r.Generation, r.Verified, r.Evidence.ReportDetail())
		// ReportDetail already bounds and sanitizes kernel tokens. Generic
		// detail redaction would mistake read timestamps for IPv6 addresses.
		detail = clipMsg(detail)
		c.Detail = &detail
	}
}

type applyReadKey struct{}
type applyReadTiming struct {
	now       func() time.Time
	at        time.Time
	stopped   bool
	baselines []MeasurementBaseline
}
type timedApplyExecutor struct {
	enforce.Executor
	timing *applyReadTiming
}

func (x timedApplyExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	b, err := x.Executor.Output(ctx, name, args...)
	if name == "tc" && len(args) > 0 && args[0] == "-j" && !x.timing.stopped {
		x.timing.at = x.timing.now().UTC()
	}
	return b, err
}

func (x timedApplyExecutor) NftBatch(ctx context.Context, script string) error {
	// Quarantine/rollback may read again. Stop the timestamp before the first
	// rollback mutation; the library has already retained the rejected rows.
	if strings.Contains(script, "fvpn_baseline=1") {
		x.timing.stopped = true
		// Memory only inside Apply; serialize after the enclosing operation
		// returns, outside every install/release guard bracket.
		start, _ := enforce.MonotonicNow()
		err := x.Executor.NftBatch(ctx, script)
		end, _ := enforce.MonotonicNow()
		x.timing.baselines = append(x.timing.baselines, MeasurementBaseline{Start: start, End: end, Succeeded: err == nil})
		return err
	} else if strings.Contains(script, "fvpn_generation=") {
		x.timing.stopped = false
	}
	return x.Executor.NftBatch(ctx, script)
}
