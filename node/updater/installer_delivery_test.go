package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

func TestD089_Installer404IsRetriable(t *testing.T) {
	ctx := context.Background()
	w := newWorld(t)
	quietDesired(w)
	rel := w.makeRelease("installer", 1, []int{1})
	if err := w.eng.ApplyPending(ctx, stageInstaller(t, w, rel)); err != nil {
		t.Fatal(err)
	}
	before, _ := w.eng.Journal.Outbox(ctx)
	job, err := w.eng.Journal.GetJob(ctx, before[0].JobID, 1)
	if err != nil {
		t.Fatal(err)
	}
	w.eng.Jitter = func() float64 { return 1 }
	w.eng.Download.Initial, w.eng.Download.MaxWait = time.Second, 4*time.Second
	status, calls := http.StatusNotFound, 0
	var logs bytes.Buffer
	w.eng.UseLog(slog.New(slog.NewJSONHandler(&logs, nil)))
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		calls++
		var result api.UpdateResult
		if err := json.NewDecoder(req.Body).Decode(&result); err != nil || result.Source == nil || *result.Source != api.UpdateResultSourceInstaller {
			t.Errorf("missing installer source: %+v %v", result, err)
		}
		if req.Header.Get("Idempotency-Key") != before[0].ResultID {
			t.Error("result identity changed")
		}
		rw.WriteHeader(status)
		if status == http.StatusConflict {
			_, _ = rw.Write([]byte(`{"error":{"code":"ACK_CONFLICT"}}`))
		} else {
			_, _ = rw.Write([]byte(`{"error":{"code":"RESOURCE_NOT_FOUND","message":"private server detail"}}`))
		}
	}))
	defer srv.Close()
	w.eng.Sender = HTTPSink{URL: srv.URL, AllowLoopback: true}
	for attempt := 1; attempt <= 4; attempt++ {
		err := w.eng.Flush(ctx)
		var delivery *DeliveryError
		if !errors.As(err, &delivery) || !delivery.Op.Retry || delivery.HTTPStatus != 404 || delivery.APICode != string(apierr.ResourceNotFound) {
			t.Fatalf("404 was not retriable: %v", err)
		}
		out, err := w.eng.Journal.Outbox(ctx)
		if err != nil || len(out) != 1 {
			t.Fatalf("outbox: %+v %v", out, err)
		}
		got := out[0]
		wantDelay := min(time.Second<<uint(attempt-1), 4*time.Second)
		if got.Status != updatejournal.OutboxPending || got.SendAttempts != attempt || calls != attempt || got.LastHTTPStatus != 404 || got.LastAPICode != "RESOURCE_NOT_FOUND" || got.RetryNotBefore.Sub(w.now) != wantDelay || !bytes.Equal(got.Payload, before[0].Payload) || got.PayloadHash != before[0].PayloadHash {
			t.Fatalf("lost evidence or retry schedule: %+v calls=%d", got, calls)
		}
		if err := w.eng.Journal.Close(); err != nil {
			t.Fatal(err)
		}
		w.eng.Journal = openJournal(t, w.layout.Journal)
		if err := w.eng.Flush(ctx); err == nil || calls != attempt {
			t.Fatalf("reopened flush retried before deadline: %v calls=%d", err, calls)
		}
		gotJob, _ := w.eng.Journal.GetJob(ctx, job.JobID, 1)
		if !reflect.DeepEqual(job, gotJob) {
			t.Fatal("delivery rewrote apply evidence")
		}
		w.now = got.RetryNotBefore
	}
	status = http.StatusConflict
	if err := w.eng.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	if err := w.eng.Flush(ctx); err != nil || calls != 5 {
		t.Fatalf("conflict was resent: %v calls=%d", err, calls)
	}
	out, _ := w.eng.Journal.Outbox(ctx)
	if out[0].Status != updatejournal.OutboxConflict || out[0].LastAPICode != "ACK_CONFLICT" || !bytes.Equal(out[0].Payload, before[0].Payload) {
		t.Fatal("conflict lost immutable evidence")
	}
	if !strings.Contains(logs.String(), "central not yet accepting installer evidence") || !strings.Contains(logs.String(), "update result conflict") || strings.Contains(logs.String(), "private server detail") {
		t.Fatalf("delivery diagnostics: %s", &logs)
	}
}

func TestD089_404ExceptionRequiresInstallerAndResourceNotFound(t *testing.T) {
	for _, tc := range []struct{ source, code string }{{"", "RESOURCE_NOT_FOUND"}, {"central", "RESOURCE_NOT_FOUND"}, {"installer", "VALIDATION_FAILED"}} {
		t.Run(tc.source+tc.code, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(404)
				_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": tc.code}})
			}))
			defer srv.Close()
			payload, _ := json.Marshal(map[string]string{"source": tc.source})
			_, _, err := (HTTPSink{URL: srv.URL, AllowLoopback: true}).Submit(context.Background(), payload)
			var delivery *DeliveryError
			if !errors.As(err, &delivery) || delivery.Op.Retry || !delivery.rejected() || installerResultPending(payload, 404, tc.code) {
				t.Fatalf("unexpected retry exception: %v", err)
			}
		})
	}
}
