package usageupload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const maxResponse = 8 << 20

// HTTPTransport calls the central usage endpoints. Client carries mTLS when
// the agent has a certificate. BaseURL is the origin, without a trailing slash.
type HTTPTransport struct {
	BaseURL string
	Client  *http.Client
	Now     func() time.Time
}

func (h *HTTPTransport) client() *http.Client {
	if h.Client != nil {
		return h.Client
	}
	return http.DefaultClient
}

func (h *HTTPTransport) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

// PostUsageBatch posts one batch. The Idempotency-Key is the caller's UUID.
func (h *HTTPTransport) PostUsageBatch(ctx context.Context, idempotencyKey string, body []byte) (int, []byte, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.BaseURL+"/api/v1/agent/usage-batches", bytes.NewReader(body))
	if err != nil {
		return 0, nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)
	return h.do(req)
}

// GetUsageCheckpoint fetches the signed recovery checkpoint (§17.3).
func (h *HTTPTransport) GetUsageCheckpoint(ctx context.Context) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.BaseURL+"/api/v1/agent/usage-checkpoint", nil)
	if err != nil {
		return 0, nil, err
	}
	status, body, _, err := h.do(req)
	return status, body, err
}

func (h *HTTPTransport) do(req *http.Request) (int, []byte, time.Duration, error) {
	resp, err := h.client().Do(req)
	if err != nil {
		return 0, nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil {
		return resp.StatusCode, nil, 0, err
	}
	if len(body) > maxResponse {
		return resp.StatusCode, nil, 0, fmt.Errorf("usageupload: response exceeds %d bytes", maxResponse)
	}
	return resp.StatusCode, body, parseRetryAfter(resp.Header.Get("Retry-After"), h.now()), nil
}

func parseRetryAfter(v string, now time.Time) time.Duration {
	if v == "" {
		return 0
	}
	if sec, err := strconv.Atoi(v); err == nil {
		if sec < 0 {
			return 0
		}
		return time.Duration(sec) * time.Second
	}
	when, err := http.ParseTime(v)
	if err != nil {
		return 0
	}
	d := when.Sub(now)
	if d < 0 {
		return 0
	}
	return d
}
