package agent

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"syscall"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// Measurement is an opt-in TEST ONLY local record, never an HTTP observation.
// Timing is the detached enforcement snapshot, including failed/partial calls.
// Anchor brackets wall sampling with CLOCK_MONOTONIC in this boot/time namespace.
// Consumers must reject missing/unordered samples and sequence gaps. Nothing is
// written or called back inside an enforcement operation or guard bracket.
type Measurement struct {
	RunID     uuid.UUID               `json:"run_id"`
	BootID    uuid.UUID               `json:"boot_id"`
	Sequence  decstr.U64              `json:"sequence"`
	Source    string                  `json:"source"`
	Anchor    MeasurementAnchor       `json:"anchor"`
	Timing    *enforce.TimingEvidence `json:"timing,omitempty"`
	TC        string                  `json:"tc_sha256,omitempty"`
	Lease     *MeasurementLease       `json:"lease,omitempty"`
	Renewal   *MeasurementRenewal     `json:"renewal,omitempty"`
	Baseline  *MeasurementBaseline    `json:"baseline,omitempty"`
	Heartbeat *MeasurementHeartbeat   `json:"heartbeat,omitempty"`
	Host      *MeasurementHost        `json:"host,omitempty"`
}

// Baseline brackets the actual writer call, including failed calls. Cause is
// fixed vocabulary, never an executor error, selector, or signed document.
type MeasurementBaseline struct {
	Cause     string        `json:"cause"`
	Start     time.Duration `json:"start_ns,string"`
	End       time.Duration `json:"end_ns,string"`
	Succeeded bool          `json:"succeeded"`
}

type MeasurementHeartbeat struct {
	Mono          time.Duration `json:"mono_ns,string"`
	PolicyExpires time.Duration `json:"policy_expires_mono_ns,string"`
	Generation    decstr.U64    `json:"generation"`
	BaselineOK    bool          `json:"baseline_ok"`
	Serving       bool          `json:"serving"`
	Reason        string        `json:"reason"`
	Written       bool          `json:"written"`
}

func (e *Engine) baselineCall(cause string, call func() error) error {
	start, _ := timeanchor.Mono()
	log := e.log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	log.Info("baseline write", "writer", "agent", "cause", cause, "phase", "begin", "mono_ns", int64(start), "generation", e.appliedGen)
	err := call()
	end, _ := timeanchor.Mono()
	log.Info("baseline write", "writer", "agent", "cause", cause, "phase", "end", "mono_ns", int64(end), "succeeded", err == nil)
	e.writeMeasurement(Measurement{Source: "baseline", Baseline: &MeasurementBaseline{cause, start, end, err == nil}})
	return err
}

func (e *Engine) baselineReason() string {
	switch {
	case e.identityFrozen:
		return "identity_frozen"
	case e.revoked:
		return "identity_revoked"
	case e.latchBlocked:
		return "updater_barrier"
	case e.recoverBlocked:
		return "updater_recovery"
	case e.applyError != nil:
		return "verification_" + string(e.applyError.Code)
	case e.conntrackBlocked:
		return "conntrack_blocked"
	case e.shutUsage:
		return "shutdown"
	case e.policyExpires > 0 && e.clock.Mono() >= e.policyExpires:
		return "policy_deadline"
	case len(e.decision.Reasons) > 0:
		return string(e.decision.Reasons[0])
	default:
		return "non_serving"
	}
}

type MeasurementAnchor struct {
	MonoBefore time.Duration `json:"mono_before_ns,string"`
	Wall       time.Time     `json:"wall"`
	MonoAfter  time.Duration `json:"mono_after_ns,string"`
}

type MeasurementLease struct {
	ID           uuid.UUID     `json:"lease_id"`
	Generation   decstr.U64    `json:"generation"`
	IssuedAt     time.Time     `json:"issued_at"`
	ExpiresAt    time.Time     `json:"expires_at"`
	RequestMono  time.Duration `json:"request_mono_ns,string"`
	NodeDeadline time.Duration `json:"node_deadline_ns,string"`
	Members      int           `json:"members"`
}

type MeasurementRenewal struct {
	RequestID uuid.UUID     `json:"request_id"`
	Start     time.Duration `json:"start_ns,string"`
	End       time.Duration `json:"end_ns,string"`
	Wall      time.Time     `json:"request_wall"`
	// NewLease distinguishes a newly accepted lease from replay, rejection
	// and failed requests. Lease is always the currently retained lease.
	NewLease bool `json:"new_lease"`
}

func (e *Engine) measurementEnabled() bool {
	return e.cfg.TestMode && e.cfg.TestTimingPath != ""
}

func (e *Engine) measureTiming(ev enforce.ServingEvidence, source string, host ...*MeasurementHost) {
	if !e.measurementEnabled() {
		return
	}
	timing := ev.TimingEvidence()
	if timing == nil { // a fake Plane may not run enforcement at all
		return
	}
	m := Measurement{Source: source, Timing: timing}
	if len(host) > 0 {
		m.Host = host[0]
	}
	if tc := ev.TCReadbackEvidence(); tc != nil {
		m.TC = tc.Sha256
	}
	e.writeMeasurement(m)
}

func (e *Engine) measureLease(id uuid.UUID, start time.Duration, wall time.Time) {
	e.writeMeasurement(Measurement{Source: "renewal", Renewal: &MeasurementRenewal{
		RequestID: id, Start: start, End: e.clock.Mono(), Wall: wall.UTC(),
		NewLease: e.leaseFresh && e.leaseMono == start,
	}})
}

func (e *Engine) writeMeasurement(m Measurement) {
	if !e.measurementEnabled() {
		return
	}
	if e.measurementRun == uuid.Nil {
		var err error
		e.measurementRun, err = uuid.NewRandom()
		if err != nil {
			e.log.Warn("measurement unavailable")
			return
		}
	}
	e.measurementSequence++
	m.RunID, m.BootID, m.Sequence = e.measurementRun, e.bootID, decstr.U64(e.measurementSequence)
	m.Anchor.MonoBefore, _ = timeanchor.Mono()
	m.Anchor.Wall = time.Now().UTC()
	m.Anchor.MonoAfter, _ = timeanchor.Mono()
	if l := e.lease; l != nil {
		d, _ := timeanchor.MemberDeadline(e.leaseMono, l.ExpiresAt.Sub(l.IssuedAt))
		m.Lease = &MeasurementLease{l.LeaseId, l.ConfigGeneration, l.IssuedAt.UTC(), l.ExpiresAt.UTC(), e.leaseMono, d, len(l.Members)}
	}
	if err := appendMeasurement(e.cfg.TestTimingPath, m); err != nil {
		// No path or underlying error text. A gap invalidates the diagnostic;
		// measurement I/O cannot rewrite a serving decision or D-106 evidence.
		e.log.Warn("measurement unavailable")
	}
}

func appendMeasurement(path string, m Measurement) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() {
		return errors.New("measurement requires regular file")
	}
	return json.NewEncoder(f).Encode(m)
}
