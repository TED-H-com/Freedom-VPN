package agent

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/controlclient"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// These record names and field names are an operator/harness interface.
const observationEnqueued = "observation enqueued"
const observationSubmission = "observation submission"

func (e *Engine) observationDueReason(state string, binding bool) string {
	switch {
	case binding:
		return "closed_recovery_binding"
	case e.lastObservationState == "":
		return "first_report"
	case e.serving != e.lastServing:
		return "serving_changed"
	case state != e.lastObservationState || e.lastObs.IsZero():
		return "state_changed"
	default:
		return "interval_elapsed"
	}
}

// Only a finite error class leaves the client boundary. In particular, a
// url.Error's string can contain a URL, user info or untrusted response text.
func observationErrorClass(err error) (outcome, class string) {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout", "deadline_exceeded"
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return "timeout", "network_timeout"
	}
	switch {
	case errors.Is(err, context.Canceled):
		return "transport_error", "canceled"
	case errors.Is(err, syscall.ECONNREFUSED):
		return "transport_error", "connection_refused"
	case errors.Is(err, syscall.ECONNRESET):
		return "transport_error", "connection_reset"
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		return "transport_error", "connection_closed"
	}
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return "transport_error", "dns"
	}
	var op *net.OpError
	if errors.As(err, &op) {
		return "transport_error", "network"
	}
	return "transport_error", "other"
}

func (e *Engine) logObservationSubmission(id uuid.UUID, attempt uint64, started time.Time, elapsed time.Duration, accepted api.ObservationAccepted, err error, retryAt time.Time) {
	outcome, class, code, status := "accepted", "", "", 202
	acceptedIDs := []uuid.UUID{}
	retryAfter := time.Duration(0)
	if err == nil {
		if bool(accepted.Accepted) {
			acceptedIDs = append(acceptedIDs, accepted.ReportId)
		} else {
			outcome = "unconfirmed_response"
		}
	} else {
		status = 0
		var se *controlclient.StatusError
		if errors.As(err, &se) {
			outcome, status, retryAfter = "rejected", se.Status, se.RetryAfter
			code = se.Code
			if code != "" && !api.ErrorCode(code).Valid() {
				code = "unknown"
			}
		} else {
			outcome, class = observationErrorClass(err)
		}
	}
	var next any
	if !retryAt.IsZero() {
		next = retryAt.UTC().Format(time.RFC3339Nano)
	}
	e.log.Info(observationSubmission,
		"report_ids", []uuid.UUID{id}, "attempt_numbers", []uint64{attempt},
		"attempt_scope", "process", "process_id", os.Getpid(), "boot_id", e.bootID.String(),
		"started_at", started.UTC().Format(time.RFC3339Nano), "duration_ms", elapsed.Milliseconds(),
		"outcome", outcome, "accepted_report_ids", acceptedIDs, "http_status", status,
		"error_code", code, "transport_error_class", class,
		"next_retry_at", next, "retry_after_ms", retryAfter.Milliseconds())
}
