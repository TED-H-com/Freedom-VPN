//go:build integration

package updater

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

// Source-less results still require a central-issued job (D-089 revision 13).
func TestD080_InstallerResultNeedsCentralJobRegistration(t *testing.T) {
	l := newCentralLab(t)
	id := l.node("amd64", "disabled", "registered", true)
	w := newWorld(t)
	quietDesired(w)
	w.eng.Node.ID = id.NodeID
	w.eng.Node.Incarnation = id.IncarnationID
	rel := w.makeRelease("installer", 1, []int{1})
	pending := stageInstaller(t, w, rel)
	if err := w.eng.ApplyPending(context.Background(), pending); err != nil {
		t.Fatal(err)
	}
	_, result := lastPayload(t, w)
	result.Source = nil
	raw := resultPayload(t, result)
	ack, err := l.svc.SubmitResult(l.ctx, id, raw)
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != apierr.ResourceNotFound || ae.HTTPStatus() != 404 || ack.Accepted {
		t.Fatalf("unexpected central response: %+v %v", ack, err)
	}
	out, err := w.eng.Journal.Outbox(context.Background())
	if err != nil || len(out) != 1 || out[0].ResultID != result.ResultId.String() || out[0].Status != updatejournal.OutboxPending {
		t.Fatalf("lost unaccepted result: %+v %v", out, err)
	}
}

// Exercise the current real central service through delivery. The old service
// returns 404; a service with fleet-update follow-up 10 accepts installer
// evidence. Both outcomes must preserve the local apply and immutable result.
func TestD089_InstallerResultCentralDelivery(t *testing.T) {
	l := newCentralLab(t)
	id := l.node("amd64", "disabled", "registered", true)
	w := newWorld(t)
	quietDesired(w)
	w.eng.Node.ID, w.eng.Node.Incarnation = id.NodeID, id.IncarnationID
	rel := w.makeRelease("installer", 1, []int{1})
	if err := w.eng.ApplyPending(l.ctx, stageInstaller(t, w, rel)); err != nil {
		t.Fatal(err)
	}
	raw, result := lastPayload(t, w)
	assertInstallerWireSource(t, raw)
	ack, centralErr := l.svc.SubmitResult(l.ctx, id, raw)
	if centralErr == nil {
		if !ack.Accepted {
			t.Fatal("central did not accept installer evidence")
		}
		t.Log("central accepts source: installer; pre-rollout 404 retry is covered by TestD089_Installer404IsRetriable")
		return
	}
	var ae *apierr.Error
	if !errors.As(centralErr, &ae) || ae.Code != apierr.ResourceNotFound || ae.HTTPStatus() != 404 {
		t.Fatalf("unexpected installer response: %v", centralErr)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(ae.HTTPStatus())
		_, _ = rw.Write([]byte(`{"error":{"code":"RESOURCE_NOT_FOUND"}}`))
	}))
	defer srv.Close()
	w.eng.Sender = HTTPSink{URL: srv.URL, AllowLoopback: true}
	w.eng.Jitter = func() float64 { return 1 }
	if err := w.eng.Flush(l.ctx); err == nil || !retryable(err) {
		t.Fatalf("central 404 lost retry: %v", err)
	}
	out, err := w.eng.Journal.Outbox(l.ctx)
	if err != nil || len(out) != 1 || out[0].ResultID != result.ResultId.String() || string(out[0].Payload) != string(raw) || out[0].Status != updatejournal.OutboxPending || !out[0].RetryNotBefore.After(w.now) {
		t.Fatalf("central 404 lost evidence: %+v %v", out, err)
	}
	t.Log("current central returns 404 RESOURCE_NOT_FOUND for source: installer; result remains pending with durable retry deadline")
}
