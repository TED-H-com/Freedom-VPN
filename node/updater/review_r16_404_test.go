package updater

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

// TestReviewR16_Installer404BackoffIsCapped keeps an unknown release pending,
// caps the retry, survives a journal reopen, and does not record a health pass.
func TestReviewR16_Installer404BackoffIsCapped(t *testing.T) {
	e := &Engine{Download: DownloadConfig{Initial: time.Second, MaxWait: 4 * time.Second}, Jitter: func() float64 { return 1 }}
	if e.backoff(1) != time.Second || e.backoff(2) != 2*time.Second || e.backoff(3) != 4*time.Second || e.backoff(8) != 4*time.Second {
		t.Fatalf("cap: %s %s %s %s", e.backoff(1), e.backoff(2), e.backoff(3), e.backoff(8))
	}
	e.Jitter = func() float64 { return 0 }
	if e.backoff(4) != 0 {
		t.Fatalf("jitter 0 = %s", e.backoff(4))
	}
	e.Jitter = func() float64 { return 0.5 }
	if e.backoff(3) != 2*time.Second {
		t.Fatalf("jitter 0.5 = %s", e.backoff(3))
	}

	ctx := context.Background()
	w := newWorld(t)
	quietDesired(w)
	rel := w.makeRelease("installer", 1, []int{1})
	if err := w.eng.ApplyPending(ctx, stageInstaller(t, w, rel)); err != nil {
		t.Fatal(err)
	}
	before, err := w.eng.Journal.Outbox(ctx)
	if err != nil || len(before) != 1 {
		t.Fatal(before, err)
	}
	w.eng.Jitter = func() float64 { return 1 }
	w.eng.Download.Initial, w.eng.Download.MaxWait = time.Second, 4*time.Second
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		calls++
		rw.WriteHeader(http.StatusNotFound)
		_, _ = rw.Write([]byte(`{"error":{"code":"RESOURCE_NOT_FOUND","message":"private server detail"}}`))
	}))
	defer srv.Close()
	w.eng.Sender = HTTPSink{URL: srv.URL, AllowLoopback: true}
	beforePtr, err := w.eng.Journal.Pointers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = w.eng.Flush(ctx)
	var delivery *DeliveryError
	if !errors.As(err, &delivery) || !delivery.Op.Retry || delivery.HTTPStatus != 404 || delivery.APICode != string(apierr.ResourceNotFound) {
		t.Fatalf("404 was not retriable: %v", err)
	}
	out, err := w.eng.Journal.Outbox(ctx)
	if err != nil || len(out) != 1 || out[0].Status != updatejournal.OutboxPending || out[0].SendAttempts != 1 || out[0].RetryNotBefore.Sub(w.now) != time.Second {
		t.Fatalf("pending schedule: %+v %v", out, err)
	}
	ptr, err := w.eng.Journal.Pointers(ctx)
	if err != nil || ptr != beforePtr {
		t.Fatalf("404 moved release pointers: before %+v after %+v (%v)", beforePtr, ptr, err)
	}
	if err := w.eng.Journal.Close(); err != nil {
		t.Fatal(err)
	}
	w.eng.Journal = openJournal(t, w.layout.Journal)
	if err := w.eng.Flush(ctx); err == nil || calls != 1 {
		t.Fatalf("reopened flush retried before the deadline: %v calls=%d", err, calls)
	}
	out, err = w.eng.Journal.Outbox(ctx)
	if err != nil || len(out) != 1 || out[0].Status != updatejournal.OutboxPending || string(out[0].Payload) != string(before[0].Payload) {
		t.Fatal("reopen lost the pending installer result", err)
	}
	ptr, err = w.eng.Journal.Pointers(ctx)
	if err != nil || ptr != beforePtr {
		t.Fatalf("reopen moved release pointers: before %+v after %+v (%v)", beforePtr, ptr, err)
	}
}
