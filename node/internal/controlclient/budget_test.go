package controlclient

import (
	"context"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func TestBudget_UsageDoesNotStarveLease(t *testing.T) {
	release := make(chan struct{})
	usageEntered := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/agent/usage-batches" {
			close(usageEntered)
			select {
			case <-release:
			case <-r.Context().Done():
			}
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"results":[]}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{}`)
	}))
	t.Cleanup(srv.Close)
	c, err := New(Config{
		BaseURL: srv.URL,
		RootCAs: x509.NewCertPool(),
		Timeout: 2 * time.Second,
		Budgets: Budgets{Lease: 1, Usage: 1, Other: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)

	usageDone := make(chan error, 1)
	go func() {
		resp, err := c.Usage(context.Background(), uuid.UUID{1}, []byte(`{}`))
		if resp != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
		usageDone <- err
	}()
	select {
	case <-usageEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("usage request did not start")
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	body, err := c.RenewLease(ctx, uuid.UUID{2}, api.LeaseRenewalRequest{
		AppliedConfigGeneration: presence.NewNullNullable[decstr.U64](),
		CurrentLeaseId:          presence.NewNullNullable[uuid.UUID](),
		NodeId:                  uuid.UUID{3},
		NodeIncarnationId:       uuid.UUID{4},
		RenewalRequestId:        uuid.UUID{5},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 {
		t.Fatal("empty lease body")
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatalf("lease waited %s behind usage", time.Since(start))
	}
	close(release)
	if err := <-usageDone; err != nil {
		t.Fatal(err)
	}
}
