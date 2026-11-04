package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/controlclient"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func f52LogCapture(r *rig) *bytes.Buffer {
	var buf bytes.Buffer
	r.eng.log = slog.New(RedactHandler{Next: slog.NewJSONHandler(&buf, nil)})
	return &buf
}
func f52Records(t *testing.T, buf *bytes.Buffer, name string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatal(err)
		}
		if row["msg"] == name {
			out = append(out, row)
		}
	}
	return out
}
func f52LogKeys(t *testing.T, row map[string]any, want []string) {
	t.Helper()
	got := make([]string, 0, len(row))
	for k := range row {
		got = append(got, k)
	}
	want = append(slices.Clone(want), "time", "level", "msg")
	slices.Sort(got)
	slices.Sort(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("log keys=%v want=%v", got, want)
	}
}
func TestF52_ObservationSubmissionRecords(t *testing.T) {
	for _, tc := range []struct {
		name, outcome, class string
		status               int
		code                 string
	}{
		{"accepted", "accepted", "", 202, ""},
		{"5xx", "rejected", "", 503, "DEPENDENCY_UNAVAILABLE"},
		{"429", "rejected", "", 429, "RATE_LIMITED"},
		{"transport", "transport_error", "connection_refused", 0, ""},
		{"timeout", "timeout", "deadline_exceeded", 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t)
			serveBoundGeneration(t, r)
			buf := f52LogCapture(r)
			// TEST ONLY sentinels are sent in the body/config, never selected for logs.
			const bodyMarker = "TEST ONLY report-body-marker"
			r.eng.cfg.AgentRelease = bodyMarker
			r.eng.decision.Errors = append(r.eng.decision.Errors, api.NodeError{Code: api.ErrorCodeHEALTHCHECKFAILED, Message: bodyMarker, Stage: "test", ObservedAt: r.clock.Wall().UTC()})
			r.eng.lastObs = time.Time{}
			r.eng.lastObservationState = ""
			if err := r.eng.observe(t.Context()); err != nil {
				t.Fatal(err)
			}
			items := f52Pending(t, r)
			if len(items) != 1 {
				t.Fatalf("pending=%d", len(items))
			}
			id := items[0].ReportID.String()
			switch tc.name {
			case "5xx":
				r.srv.FailNext(1)
			case "429":
				r.srv.SetObservationRetryAfter(1, 60)
			case "transport":
				if err := r.srv.Close(); err != nil {
					t.Fatal(err)
				}
			case "timeout":
				r.srv.SetDelay(time.Second)
			}
			flush := func() {
				ctx := t.Context()
				if tc.name == "timeout" {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
					defer cancel()
				}
				if err := r.eng.flush(ctx); err != nil {
					t.Fatal(err)
				}
			}
			started := r.clock.Wall().UTC().Format(time.RFC3339Nano)
			flush()
			enqueue := f52Records(t, buf, observationEnqueued)
			attempts := f52Records(t, buf, observationSubmission)
			if len(enqueue) != 1 || len(attempts) != 1 {
				t.Fatalf("record counts enqueue=%d attempt=%d", len(enqueue), len(attempts))
			}
			f52LogKeys(t, enqueue[0], []string{"report_id", "observed_at", "due_reason"})
			if enqueue[0]["report_id"] != id || enqueue[0]["observed_at"] != started || enqueue[0]["due_reason"] != "first_report" {
				t.Fatalf("enqueue=%v", enqueue[0])
			}
			row := attempts[0]
			f52LogKeys(t, row, []string{"report_ids", "attempt_numbers", "attempt_scope", "process_id", "boot_id", "started_at", "duration_ms", "outcome", "accepted_report_ids", "http_status", "error_code", "transport_error_class", "next_retry_at", "retry_after_ms"})
			if !reflect.DeepEqual(row["report_ids"], []any{id}) || !reflect.DeepEqual(row["attempt_numbers"], []any{float64(1)}) || row["attempt_scope"] != "process" || row["process_id"].(float64) <= 0 || row["boot_id"] != r.eng.bootID.String() || row["started_at"] != started || row["duration_ms"].(float64) < 0 || row["outcome"] != tc.outcome || row["http_status"] != float64(tc.status) || row["error_code"] != tc.code || row["transport_error_class"] != tc.class {
				t.Fatalf("attempt=%v", row)
			}
			if tc.name == "accepted" {
				if !reflect.DeepEqual(row["accepted_report_ids"], []any{id}) || row["next_retry_at"] != nil || row["retry_after_ms"] != float64(0) {
					t.Fatalf("accept=%v", row)
				}
			} else {
				if len(row["accepted_report_ids"].([]any)) != 0 {
					t.Fatal("failed report logged accepted")
				}
				want := r.clock.Wall().Add(r.eng.observationRetryAt - r.clock.Mono()).UTC().Format(time.RFC3339Nano)
				if row["next_retry_at"] != want {
					t.Fatalf("retry=%v want=%s", row, want)
				}
				if tc.name == "429" && row["retry_after_ms"] != float64(60000) {
					t.Fatal("lost Retry-After")
				}
				if tc.name == "5xx" && row["retry_after_ms"] != float64(1000) {
					t.Fatal("lost default 503 Retry-After")
				}
				if tc.name == "timeout" && row["duration_ms"].(float64) < 50 {
					t.Fatal("timeout duration missing")
				}
			}
			// Both a deduplicated observation and a retry slot that has not arrived
			// are silent. The second attempt must be numbered 2, never the outbox's
			// two-writes-per-failure uncertainty count (which would yield 3).
			if err := r.eng.observe(t.Context()); err != nil {
				t.Fatal(err)
			}
			flush()
			if tc.name == "accepted" {
				stepOK(t, r) // An actual idle Step emits neither receipt.
			}
			if len(f52Records(t, buf, observationSubmission)) != 1 || len(f52Records(t, buf, observationEnqueued)) != 1 {
				t.Fatal("idle call logged a record")
			}
			if tc.name != "accepted" {
				r.srv.SetDelay(0)
				r.clock.Advance(r.eng.observationRetryAt - r.clock.Mono())
				flush()
				rows := f52Records(t, buf, observationSubmission)
				if len(rows) != 2 || !reflect.DeepEqual(rows[1]["attempt_numbers"], []any{float64(2)}) || !reflect.DeepEqual(rows[1]["report_ids"], []any{id}) {
					t.Fatalf("retry records=%v", rows)
				}
			}
			for _, marker := range []string{bodyMarker, "Authorization", "Enrollment ", "PRIVATE KEY", "token-test-only"} {
				if marker != "" && strings.Contains(buf.String(), marker) {
					t.Fatalf("log carries sensitive field marker")
				}
			}
		})
	}
}

func TestF52_ObservationEnqueueReasons(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	buf := f52LogCapture(r)
	r.clock.Advance(time.Duration(r.eng.cfg.ObservationSeconds) * time.Second)
	if err := r.eng.observe(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.flush(t.Context()); err != nil {
		t.Fatal(err)
	}
	r.eng.serving = false
	if err := r.eng.observe(t.Context()); err != nil {
		t.Fatal(err)
	}
	r.eng.decision.Reasons = append(r.eng.decision.Reasons, api.ForwardingBlockReasonOutboxFull)
	if err := r.eng.observe(t.Context()); err != nil {
		t.Fatal(err)
	}
	rows := f52Records(t, buf, observationEnqueued)
	if len(rows) != 3 {
		t.Fatalf("enqueue records=%v", rows)
	}
	for i, want := range []string{"interval_elapsed", "serving_changed", "state_changed"} {
		if rows[i]["due_reason"] != want {
			t.Fatalf("reason=%v want=%s", rows[i], want)
		}
	}
	r2, p := f50Rig(t)
	buf2 := f52LogCapture(r2)
	f50Recovery(t, r2, p, 14)
	rows = f52Records(t, buf2, observationEnqueued)
	if len(rows) != 2 || rows[1]["due_reason"] != "closed_recovery_binding" || rows[1]["report_id"] != r2.eng.closedRecoveryRenewal.reportID.String() {
		t.Fatalf("closed binding=%v", rows)
	}
}

func TestF52_ObservationRecordsDoNotExposeErrorText(t *testing.T) {
	r := newRig(t)
	buf := f52LogCapture(r)
	const marker = "TEST ONLY untrusted-error-marker"
	id := uuid.New()
	r.eng.logObservationSubmission(id, 1, r.clock.Wall(), time.Millisecond, api.ObservationAccepted{},
		&controlclient.StatusError{Status: 503, Code: marker}, time.Time{})
	r.eng.logObservationSubmission(id, 2, r.clock.Wall(), time.Millisecond, api.ObservationAccepted{},
		&url.Error{Op: marker, URL: "https://example.invalid/" + marker, Err: context.DeadlineExceeded}, time.Time{})
	rows := f52Records(t, buf, observationSubmission)
	if len(rows) != 2 || rows[0]["error_code"] != "unknown" || rows[1]["outcome"] != "timeout" || strings.Contains(buf.String(), marker) || strings.Contains(buf.String(), "example.invalid") {
		t.Fatalf("untrusted metadata escaped finite fields: %v", rows)
	}
}
