package controlclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestReviewR11_RenewLeaseReturnsBeforeRetryAfter(t *testing.T) {
	// TODO(review): Do parses Retry-After onto StatusError. postJSON returns
	// that error immediately, and RenewLease does not wait. The agent
	// syncLease logs the error and returns nil, so the next step tries again
	// inside the delay.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)
	c := &Client{
		base: srv.URL,
		http: map[Class]*http.Client{ClassLease: srv.Client()},
		sem:  map[Class]chan struct{}{ClassLease: make(chan struct{}, 1)},
	}
	req := api.LeaseRenewalRequest{
		NodeId: uuid.Must(uuid.NewV7()), NodeIncarnationId: uuid.Must(uuid.NewV7()),
		RenewalRequestId: uuid.Must(uuid.NewV7()),
	}
	req.AppliedConfigGeneration.SetNull()
	req.CurrentLeaseId.SetNull()
	start := time.Now()
	_, err := c.RenewLease(context.Background(), req.RenewalRequestId, req)
	elapsed := time.Since(start)
	var status *StatusError
	if errors.As(err, &status) && status.Status == http.StatusTooManyRequests && status.RetryAfter >= 30*time.Second {
		if elapsed < 30*time.Second {
			t.Fatalf("Retry-After was parsed (%s) but RenewLease returned in %s", status.RetryAfter, elapsed)
		}
		return
	}
	t.Fatalf("RenewLease dropped Retry-After on 429 and returned in %s: %v", elapsed, err)
}
