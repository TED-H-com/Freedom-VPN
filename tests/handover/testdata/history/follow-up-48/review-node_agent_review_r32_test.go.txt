package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/controlclient"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TestR32_Held_Q1_Q2_DedupExceptionAndRebuildCost proves:
//   - Q1: canBindRecoveryReport matches noteRecoveryReport's predicate on every branch;
//   - Q2: when closedRecoveryRenewal is armed & unbound while items are pending in the outbox
//     and state == lastObservationState, observe rebuilds a candidate report via e.report()
//     on due calls, skips queuing when !canBindRecoveryReport(rep), leaves lastObs unchanged,
//     mutates no durable outbox state or observationAttempts, and immediately queues + binds
//     (with observationDueReason and noteRecoveryReport agreeing) if the report becomes bindable.
func TestR32_Held_Q1_Q2_DedupExceptionAndRebuildCost(t *testing.T) {
	r, base := f50Rig(t)
	p := &r31IsolationQuarantinePlane{r27BootMissingInterfacePlane: base, failIsolationQuarantine: true}
	r.eng.plane = p
	r.eng.cfg.ObservationSeconds = 30
	r.srv.SetFixture(generationFixture(14, oneBinding(r.clock.Wall())))
	advanceToRenewal(r)
	stepOK(t, r)
	if r.eng.containment == nil || r.eng.containment.Code != api.ErrorCodeISOLATIONPOLICYINVALID {
		t.Fatalf("expected isolation containment latch, got %+v", r.eng.containment)
	}

	// Enter closed recovery with an observation outage so the initial post-Apply report stays pending.
	p.failIsolationQuarantine = false
	r.srv.SetMemberGrantsReady(false)
	r.srv.SetObservationRetryAfter(64, 60)
	advanceToRenewal(r)
	stepOK(t, r)

	if !r.eng.closedRecoveryHeld || !r.eng.closedRecoveryRenewal.armed || r.eng.closedRecoveryRenewal.reportID != uuid.Nil {
		t.Fatalf("expected held armed closed recovery with unbound reportID, got held=%v renewal=%+v",
			r.eng.closedRecoveryHeld, r.eng.closedRecoveryRenewal)
	}
	firstObsTime := r.eng.lastObs
	if firstObsTime.IsZero() {
		t.Fatal("expected lastObs to be set on initial queued observation")
	}
	pending0 := f52Pending(t, r)
	if len(pending0) != 1 {
		t.Fatalf("initial pending = %d, want 1", len(pending0))
	}

	buf := f52LogCapture(r)
	obsInterval := time.Duration(r.eng.cfg.ObservationSeconds) * time.Second
	for i := 0; i < 5; i++ {
		r.clock.Advance(obsInterval)
		if err := r.eng.observe(t.Context()); err != nil {
			t.Fatalf("observe iter %d: %v", i, err)
		}
		// Because !canBindRecoveryReport(rep) returns early at observe.go:55 before line 78,
		// lastObs is not advanced and no new item is queued.
		if !r.eng.lastObs.Equal(firstObsTime) {
			t.Fatalf("iter %d: lastObs changed from %v to %v on skipped candidate report", i, firstObsTime, r.eng.lastObs)
		}
		if got := len(f52Pending(t, r)); got != 1 {
			t.Fatalf("iter %d: pending = %d, want 1", i, got)
		}
	}
	if got := len(f52Records(t, buf, observationEnqueued)); got != 0 {
		t.Fatalf("enqueued records during 5 skipped intervals = %d, want 0", got)
	}

	// Verify what happens when the report becomes bindable even while state == lastObservationState:
	// clear containment and keep lastObservationState equal to observationState() so observe enters
	// the state == e.lastObservationState dedup branch with pending0 still in the outbox.
	r.eng.containment = nil
	r.eng.evaluate(t.Context())
	r.eng.lastObservationState = r.eng.observationState()

	if err := r.eng.observe(t.Context()); err != nil {
		t.Fatalf("observe on bindable candidate: %v", err)
	}
	pendingAfterBind := f52Pending(t, r)
	if len(pendingAfterBind) != 2 {
		t.Fatalf("pending after bind = %d, want 2", len(pendingAfterBind))
	}
	boundID := r.eng.closedRecoveryRenewal.reportID
	if boundID == uuid.Nil || boundID != pendingAfterBind[1].ReportID {
		t.Fatalf("bound reportID = %s, want second pending report %s", boundID, pendingAfterBind[1].ReportID)
	}
	if !r.eng.lastObs.Equal(r.clock.Wall().UTC()) {
		t.Fatalf("lastObs = %v, want %v after binding report", r.eng.lastObs, r.clock.Wall().UTC())
	}
	enq := f52Records(t, buf, observationEnqueued)
	if len(enq) != 1 || enq[0]["due_reason"] != "closed_recovery_binding" || enq[0]["report_id"] != boundID.String() {
		t.Fatalf("enqueued record on bind = %v, want due_reason=closed_recovery_binding report_id=%s", enq, boundID)
	}
}

// TestR32_Held_Q3_Q4_SubmitObservationBranchesAndDiagnosticMap proves:
//   - Q3: every branch of submitObservation preserves flush's exact control flow and state
//     mutations (accepted, IdentityFrozen when !freezeDurable vs freezeDurable, 403 revoked,
//     409/422 conflict continuing to subsequent pending items in the same flush pass, and
//     retryable failure stopping flush after one failed request);
//   - Q4: observationAttempts is bounded by the pending outbox and pruned at the start of flush
//     across accepted, conflict, and retry cycles.
func TestR32_Held_Q3_Q4_SubmitObservationBranchesAndDiagnosticMap(t *testing.T) {
	t.Run("conflict_422_continues_in_same_flush_and_prunes_map", func(t *testing.T) {
		r := newRig(t)
		serveBoundGeneration(t, r)
		buf := f52LogCapture(r)

		now := r.clock.Wall().UTC()
		// 1st pending item: wrong NodeId -> fakecontrol returns 422 VALIDATION_FAILED.
		rep422a, err := r.eng.report(t.Context(), now)
		if err != nil {
			t.Fatal(err)
		}
		rep422a.NodeId = uuid.Must(uuid.NewV7())
		raw422a, err := json.Marshal(rep422a)
		if err != nil {
			t.Fatal(err)
		}
		if err := r.eng.store.Enqueue(t.Context(), rep422a.ReportId, raw422a, now); err != nil {
			t.Fatal(err)
		}
		// 2nd pending item: wrong NodeIncarnationId -> fakecontrol returns 422 VALIDATION_FAILED.
		rep422b, err := r.eng.report(t.Context(), now.Add(time.Millisecond))
		if err != nil {
			t.Fatal(err)
		}
		rep422b.NodeIncarnationId = uuid.Must(uuid.NewV7())
		raw422b, err := json.Marshal(rep422b)
		if err != nil {
			t.Fatal(err)
		}
		if err := r.eng.store.Enqueue(t.Context(), rep422b.ReportId, raw422b, now.Add(time.Millisecond)); err != nil {
			t.Fatal(err)
		}
		// 3rd pending item: valid observation enqueued through observe().
		r.clock.Advance(time.Second)
		r.eng.lastObs = time.Time{}
		r.eng.lastObservationState = "valid-3rd"
		if err := r.eng.observe(t.Context()); err != nil {
			t.Fatal(err)
		}
		pending := f52Pending(t, r)
		if len(pending) != 3 {
			t.Fatalf("pending before flush = %d, want 3", len(pending))
		}
		id202 := pending[2].ReportID

		// Single flush pass must process all 3 items because 409/422 return stop=false.
		if err := r.eng.flush(t.Context()); err != nil {
			t.Fatal(err)
		}
		if got := len(f52Pending(t, r)); got != 0 {
			t.Fatalf("pending after flush = %d, want 0", got)
		}
		if r.eng.observationFailures != 0 || r.eng.observationRetryAt != 0 {
			t.Fatalf("failures=%d retryAt=%v, want 0/0 after 422+422+202", r.eng.observationFailures, r.eng.observationRetryAt)
		}

		subs := f52Records(t, buf, observationSubmission)
		if len(subs) != 3 {
			t.Fatalf("submission records = %d, want 3", len(subs))
		}
		if subs[0]["http_status"] != float64(422) || subs[0]["outcome"] != "rejected" || subs[0]["error_code"] != string(api.ErrorCodeVALIDATIONFAILED) || subs[0]["next_retry_at"] != nil {
			t.Fatalf("record[0] (422a) = %v", subs[0])
		}
		if subs[1]["http_status"] != float64(422) || subs[1]["outcome"] != "rejected" || subs[1]["error_code"] != string(api.ErrorCodeVALIDATIONFAILED) || subs[1]["next_retry_at"] != nil {
			t.Fatalf("record[1] (422b) = %v", subs[1])
		}
		if subs[2]["http_status"] != float64(202) || subs[2]["outcome"] != "accepted" || subs[2]["accepted_report_ids"].([]any)[0] != id202.String() {
			t.Fatalf("record[2] (202) = %v", subs[2])
		}

		// Next flush with empty outbox prunes observationAttempts to 0 entries.
		if err := r.eng.flush(t.Context()); err != nil {
			t.Fatal(err)
		}
		if len(r.eng.observationAttempts) != 0 {
			t.Fatalf("observationAttempts after pruning flush = %d, want 0", len(r.eng.observationAttempts))
		}
	})

	t.Run("identity_frozen_marks_durable_and_emits_accepted_record", func(t *testing.T) {
		r := newRig(t)
		serveBoundGeneration(t, r)
		buf := f52LogCapture(r)

		r.clock.Advance(time.Second)
		r.eng.lastObs = time.Time{}
		r.eng.lastObservationState = "freeze-0"
		if err := r.eng.observe(t.Context()); err != nil {
			t.Fatal(err)
		}

		frozen := true
		r.srv.SetIdentityFrozen(&frozen)
		postsBefore := r.srv.ObservationPosts()

		if err := r.eng.flush(t.Context()); err != nil {
			t.Fatal(err)
		}
		if !r.eng.identityFrozen || !r.eng.freezeDurable {
			t.Fatalf("expected identityFrozen=true and freezeDurable=true, got frozen=%v durable=%v", r.eng.identityFrozen, r.eng.freezeDurable)
		}
		if got := r.srv.ObservationPosts() - postsBefore; got != 1 {
			t.Fatalf("observation posts = %d, want 1", got)
		}
		if got := len(f52Pending(t, r)); got != 0 {
			t.Fatalf("pending = %d, want 0", got)
		}
		subs := f52Records(t, buf, observationSubmission)
		if len(subs) != 1 || subs[0]["outcome"] != "accepted" || subs[0]["http_status"] != float64(202) || subs[0]["next_retry_at"] != nil {
			t.Fatalf("submission record on IdentityFrozen = %v", subs)
		}
	})

	t.Run("revoked_403_marks_revoked_and_stops_flush", func(t *testing.T) {
		r := newRig(t)
		serveBoundGeneration(t, r)
		buf := f52LogCapture(r)

		for i := 0; i < 2; i++ {
			r.clock.Advance(time.Second)
			r.eng.lastObs = time.Time{}
			r.eng.lastObservationState = fmt.Sprintf("revoked-%d", i)
			if err := r.eng.observe(t.Context()); err != nil {
				t.Fatal(err)
			}
		}

		r.srv.Revoke()
		if err := r.eng.flush(t.Context()); err != nil {
			t.Fatal(err)
		}
		if !r.eng.revoked {
			t.Fatal("expected revoked=true on 403 IDENTITY_REVOKED")
		}
		if got := len(f52Pending(t, r)); got != 2 {
			t.Fatalf("pending = %d, want 2 (stopped on first 403)", got)
		}
		subs := f52Records(t, buf, observationSubmission)
		if len(subs) != 1 || subs[0]["http_status"] != float64(403) || subs[0]["outcome"] != "rejected" || subs[0]["error_code"] != string(api.ErrorCodeIDENTITYREVOKED) || subs[0]["next_retry_at"] != nil {
			t.Fatalf("submission record on 403 = %v", subs)
		}
	})
}

type timeoutNetError struct{ msg string }

func (e timeoutNetError) Error() string   { return e.msg }
func (e timeoutNetError) Timeout() bool   { return true }
func (e timeoutNetError) Temporary() bool { return true }

// TestR32_Held_Q5_Q6_ObservationRecordFieldsAndErrorClasses proves:
//   - Q5: unconfirmed_response (202 with accepted=false), mismatched accepted.ReportId,
//     unknown api.ErrorCode sanitization, and all 9 finite observationErrorClass tokens;
//   - Invariant 5: no sentinel URL, hostname, IP address, or error string ever appears in logs.
func TestR32_Held_Q5_Q6_ObservationRecordFieldsAndErrorClasses(t *testing.T) {
	t.Run("unconfirmed_response_and_mismatched_report_id", func(t *testing.T) {
		r, p := f50Rig(t)
		r.srv.SetObservationRetryAfter(4, 60)
		f50Recovery(t, r, p, 14)
		r.eng.evaluate(t.Context())
		if err := r.eng.observe(t.Context()); err != nil {
			t.Fatal(err)
		}
		boundID := r.eng.closedRecoveryRenewal.reportID
		if boundID == uuid.Nil || r.eng.closedRecoveryRenewal.accepted {
			t.Fatalf("expected bound, unaccepted recovery report, got %+v", r.eng.closedRecoveryRenewal)
		}
		rep, err := r.eng.report(t.Context(), r.clock.Wall().UTC())
		if err != nil {
			t.Fatal(err)
		}
		rep.ReportId = boundID

		buf := f52LogCapture(r)

		// Case 1: 202 with accepted=false -> outcome="unconfirmed_response", accepted_report_ids=[],
		// and acceptRecoveryReport refuses to mark closedRecoveryRenewal.accepted.
		unconfirmed := api.ObservationAccepted{
			Accepted:   false,
			ReceivedAt: r.clock.Wall().UTC(),
			ReportId:   boundID,
		}
		r.eng.acceptRecoveryReport(rep, unconfirmed)
		if r.eng.closedRecoveryRenewal.accepted {
			t.Fatal("closedRecoveryRenewal.accepted must remain false when accepted=false")
		}
		r.eng.logObservationSubmission(boundID, 1, r.clock.Wall().UTC(), 2*time.Millisecond, unconfirmed, nil, time.Time{})

		// Case 2: 202 with accepted=true but a different ReportId -> outcome="accepted",
		// accepted_report_ids=[otherID], while acceptRecoveryReport still refuses to mark recovery accepted.
		otherID := uuid.Must(uuid.NewV7())
		mismatched := api.ObservationAccepted{
			Accepted:   true,
			ReceivedAt: r.clock.Wall().UTC(),
			ReportId:   otherID,
		}
		r.eng.acceptRecoveryReport(rep, mismatched)
		if r.eng.closedRecoveryRenewal.accepted {
			t.Fatal("closedRecoveryRenewal.accepted must remain false when accepted.ReportId != rep.ReportId")
		}
		r.eng.logObservationSubmission(boundID, 1, r.clock.Wall().UTC(), 2*time.Millisecond, mismatched, nil, time.Time{})

		subs := f52Records(t, buf, observationSubmission)
		if len(subs) != 2 {
			t.Fatalf("subs = %d, want 2", len(subs))
		}
		if subs[0]["http_status"] != float64(202) || subs[0]["outcome"] != "unconfirmed_response" {
			t.Fatalf("unconfirmed record = %v", subs[0])
		}
		if gotIDs, ok := subs[0]["accepted_report_ids"].([]any); !ok || len(gotIDs) != 0 {
			t.Fatalf("accepted_report_ids = %v, want empty slice", subs[0]["accepted_report_ids"])
		}
		if subs[1]["http_status"] != float64(202) || subs[1]["outcome"] != "accepted" {
			t.Fatalf("mismatched-ID record = %v", subs[1])
		}
		if gotIDs, ok := subs[1]["accepted_report_ids"].([]any); !ok || len(gotIDs) != 1 || gotIDs[0] != otherID.String() {
			t.Fatalf("accepted_report_ids = %v, want [%s]", subs[1]["accepted_report_ids"], otherID)
		}
	})

	t.Run("all_error_classes_and_unknown_error_code", func(t *testing.T) {
		const secretSentinel = "TEST-ONLY-SECRET-HOST.example.invalid/198.51.100.77"
		cases := []struct {
			name        string
			err         error
			wantStatus  float64
			wantOutcome string
			wantClass   any
			wantCode    any
		}{
			{
				name:        "deadline_exceeded",
				err:         &url.Error{Op: "Post", URL: "https://" + secretSentinel, Err: context.DeadlineExceeded},
				wantStatus:  0,
				wantOutcome: "timeout",
				wantClass:   "deadline_exceeded",
				wantCode:    "",
			},
			{
				name:        "network_timeout",
				err:         &url.Error{Op: "Post", URL: "https://" + secretSentinel, Err: timeoutNetError{msg: secretSentinel}},
				wantStatus:  0,
				wantOutcome: "timeout",
				wantClass:   "network_timeout",
				wantCode:    "",
			},
			{
				name:        "canceled",
				err:         &url.Error{Op: "Post", URL: "https://" + secretSentinel, Err: context.Canceled},
				wantStatus:  0,
				wantOutcome: "transport_error",
				wantClass:   "canceled",
				wantCode:    "",
			},
			{
				name:        "connection_refused",
				err:         &net.OpError{Op: "dial", Net: "tcp", Err: os.NewSyscallError("connect", syscall.ECONNREFUSED)},
				wantStatus:  0,
				wantOutcome: "transport_error",
				wantClass:   "connection_refused",
				wantCode:    "",
			},
			{
				name:        "connection_reset",
				err:         &net.OpError{Op: "read", Net: "tcp", Err: os.NewSyscallError("read", syscall.ECONNRESET)},
				wantStatus:  0,
				wantOutcome: "transport_error",
				wantClass:   "connection_reset",
				wantCode:    "",
			},
			{
				name:        "connection_closed_unexpected_eof",
				err:         &url.Error{Op: "Post", URL: "https://" + secretSentinel, Err: io.ErrUnexpectedEOF},
				wantStatus:  0,
				wantOutcome: "transport_error",
				wantClass:   "connection_closed",
				wantCode:    "",
			},
			{
				name:        "dns_wrapped_in_op_error",
				err:         &net.OpError{Op: "dial", Net: "tcp", Err: &net.DNSError{Err: "no such host", Name: secretSentinel}},
				wantStatus:  0,
				wantOutcome: "transport_error",
				wantClass:   "dns",
				wantCode:    "",
			},
			{
				name:        "network_op_error",
				err:         &net.OpError{Op: "dial", Net: "tcp", Err: errors.New(secretSentinel)},
				wantStatus:  0,
				wantOutcome: "transport_error",
				wantClass:   "network",
				wantCode:    "",
			},
			{
				name:        "other_transport_error",
				err:         fmt.Errorf("controlclient: /api/v1/agent/observations status 200 (%s)", secretSentinel),
				wantStatus:  0,
				wantOutcome: "transport_error",
				wantClass:   "other",
				wantCode:    "",
			},
			{
				name:        "unknown_error_code_sanitized",
				err:         &controlclient.StatusError{Status: 502, Code: "INTERNAL_UPSTREAM_" + secretSentinel},
				wantStatus:  502,
				wantOutcome: "rejected",
				wantClass:   "",
				wantCode:    "unknown",
			},
		}

		r := newRig(t)
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				buf := f52LogCapture(r)
				id := uuid.Must(uuid.NewV7())
				r.eng.logObservationSubmission(id, 1, r.clock.Wall().UTC(), 3*time.Millisecond, api.ObservationAccepted{}, tc.err, time.Time{})
				if strings.Contains(buf.String(), secretSentinel) {
					t.Fatalf("log leaked sentinel %q: %s", secretSentinel, buf.String())
				}
				subs := f52Records(t, buf, observationSubmission)
				if len(subs) != 1 {
					t.Fatalf("subs = %d, want 1", len(subs))
				}
				row := subs[0]
				if row["http_status"] != tc.wantStatus || row["outcome"] != tc.wantOutcome || row["transport_error_class"] != tc.wantClass || row["error_code"] != tc.wantCode {
					t.Fatalf("row = %v, want status=%v outcome=%v class=%v code=%v", row, tc.wantStatus, tc.wantOutcome, tc.wantClass, tc.wantCode)
				}
			})
		}
	})
}

// TestR32_Held_Q7_RecoveryInteractionLateBindAndSingleEarlyRenewal proves:
//   - Q7: an armed recovery initially blocked from binding (while an earlier report sits
//     pending in the outbox) is not stranded — once the blocking condition clears, the next
//     observation binds, its acceptance triggers the early renewal, and closedRecoveryRenewal
//     is disarmed on dispatch so a second early renewal cannot fire in the same D-067 slot.
func TestR32_Held_Q7_RecoveryInteractionLateBindAndSingleEarlyRenewal(t *testing.T) {
	r, base := f50Rig(t)
	p := &r31IsolationQuarantinePlane{r27BootMissingInterfacePlane: base, failIsolationQuarantine: true}
	r.eng.plane = p
	r.srv.SetFixture(generationFixture(14, oneBinding(r.clock.Wall())))
	advanceToRenewal(r)
	stepOK(t, r)

	// Enter closed recovery with an observation outage while isolation containment is latched.
	r.eng.cfg.HeartbeatSeconds = 30
	p.failIsolationQuarantine = false
	r.srv.SetMemberGrantsReady(false)
	r.srv.SetObservationRetryAfter(1, 5)
	advanceToRenewal(r)
	stepOK(t, r)
	if !r.eng.closedRecoveryHeld || !r.eng.closedRecoveryRenewal.armed || r.eng.closedRecoveryRenewal.reportID != uuid.Nil {
		t.Fatalf("expected held armed unbound recovery, got held=%v renewal=%+v", r.eng.closedRecoveryHeld, r.eng.closedRecoveryRenewal)
	}
	if got := len(f52Pending(t, r)); got != 1 {
		t.Fatalf("pending = %d, want 1 failed report in outbox", got)
	}

	// Clear containment after the 5s Retry-After expires (still 24s before D-067's 30s slot);
	// observe+flush queues the clean report, drains both reports, and marks closedRecoveryRenewal.accepted=true.
	r.eng.containment = nil
	r.clock.Advance(6 * time.Second)
	r.eng.evaluate(t.Context())
	if err := r.eng.observe(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.flush(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !r.eng.closedRecoveryRenewal.accepted {
		t.Fatalf("expected recovery report to be bound and accepted after containment cleared: %+v", r.eng.closedRecoveryRenewal)
	}

	slot := r.eng.provisionalRetryAt
	renewalsBefore := r.srv.LeaseRenewals()
	// Run Step well before D-067's 30s provisional backoff slot: early renewal must fire once.
	stepOK(t, r)
	if r.clock.Mono() >= slot {
		t.Fatalf("clock %v reached provisional slot %v", r.clock.Mono(), slot)
	}
	if got := r.srv.LeaseRenewals(); got != renewalsBefore+1 {
		t.Fatalf("renewals after early renewal Step = %d, want %d", got, renewalsBefore+1)
	}
	if r.eng.closedRecoveryRenewal.armed || r.eng.closedRecoveryRenewal.accepted || r.eng.closedRecoveryRenewal.reportID != uuid.Nil {
		t.Fatalf("expected closedRecoveryRenewal to be cleared on dispatch, got %+v", r.eng.closedRecoveryRenewal)
	}

	// Another Step 1s later in the same D-067 slot (even with grants still empty) must NOT trigger a second early renewal.
	r.clock.Advance(time.Second)
	stepOK(t, r)
	if got := r.srv.LeaseRenewals(); got != renewalsBefore+1 {
		t.Fatalf("renewals on next Step in same D-067 slot = %d, want %d (at most once per recovery)", got, renewalsBefore+1)
	}
}
