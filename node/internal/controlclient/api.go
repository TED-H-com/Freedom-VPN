package controlclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

const (
	pathEnroll    = "/api/v1/agent/enroll"
	pathDesired   = "/api/v1/agent/desired-state"
	pathObserve   = "/api/v1/agent/observations"
	pathLease     = "/api/v1/agent/lease-renewals"
	pathRenew     = "/api/v1/agent/certificates/renew"
	pathBootstrap = "/api/v1/agent/bootstrap"
	pathUsage     = "/api/v1/agent/usage-batches"
	// pathResolver is the prefix for GET /agent/resolver-documents/{document_id}.
	// The body is the stored RFC 8785 bytes. Callers verify length and SHA-256
	// before they parse or apply (D-056).
	pathResolver = "/api/v1/agent/resolver-documents/"
)

// Enroll posts the enrollment request. token is the enrollment secret and is
// sent only in the Authorization header. status is 201 on first success and
// 200 on a same-key retry.
func (c *Client) Enroll(ctx context.Context, token string, idem uuid.UUID, req api.EnrollRequest) (api.EnrollResponse, int, error) {
	var out api.EnrollResponse
	if token == "" {
		return out, 0, errors.New("controlclient: empty enrollment token")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return out, 0, err
	}
	httpReq, err := c.jsonReq(ctx, http.MethodPost, pathEnroll, body, idem)
	if err != nil {
		return out, 0, err
	}
	httpReq.Header.Set("Authorization", "Enrollment "+token)
	resp, err := c.Do(ctx, ClassOther, httpReq)
	if err != nil {
		return out, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return out, resp.StatusCode, fmt.Errorf("controlclient: enroll status %d", resp.StatusCode)
	}
	raw, err := readLimited(resp.Body)
	if err != nil {
		return out, resp.StatusCode, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, resp.StatusCode, fmt.Errorf("controlclient: enroll response: %w", err)
	}
	return out, resp.StatusCode, nil
}

// DesiredState fetches the signed envelope. etag is sent as If-None-Match.
// A 304 returns ErrNotModified and an empty body (§14.7).
func (c *Client) DesiredState(ctx context.Context, etag string) (body []byte, etagOut string, err error) {
	httpReq, err := c.jsonReq(ctx, http.MethodGet, pathDesired, nil, uuid.Nil)
	if err != nil {
		return nil, "", err
	}
	if etag != "" {
		httpReq.Header.Set("If-None-Match", etag)
	}
	if err := c.acquire(ctx, ClassOther); err != nil {
		return nil, "", err
	}
	resp, err := c.http[ClassOther].Do(httpReq)
	if err != nil {
		c.release(ClassOther)
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	defer c.release(ClassOther)
	if resp.StatusCode == http.StatusNotModified {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
		return nil, etag, ErrNotModified
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", statusFrom(resp)
	}
	raw, err := readLimited(resp.Body)
	if err != nil {
		return nil, "", err
	}
	return raw, resp.Header.Get("ETag"), nil
}

// Observe posts one report. Only 202 is acceptance (§14.4).
func (c *Client) Observe(ctx context.Context, idem uuid.UUID, report api.ObservationReport) (api.ObservationAccepted, error) {
	var out api.ObservationAccepted
	body, err := json.Marshal(report)
	if err != nil {
		return out, err
	}
	raw, err := c.postJSON(ctx, ClassOther, pathObserve, body, idem, http.StatusAccepted)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("controlclient: observation response: %w", err)
	}
	return out, nil
}

// RenewLease posts a renewal and returns the signed lease envelope bytes.
func (c *Client) RenewLease(ctx context.Context, idem uuid.UUID, req api.LeaseRenewalRequest) ([]byte, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	return c.postJSON(ctx, ClassLease, pathLease, body, idem, http.StatusOK)
}

// RenewCertificate posts a CSR and returns the issued certificate.
func (c *Client) RenewCertificate(ctx context.Context, idem uuid.UUID, req api.CertificateRenewRequest) (api.CertificateRenewResponse, error) {
	var out api.CertificateRenewResponse
	body, err := json.Marshal(req)
	if err != nil {
		return out, err
	}
	raw, err := c.postJSON(ctx, ClassOther, pathRenew, body, idem, http.StatusCreated)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("controlclient: certificate response: %w", err)
	}
	return out, nil
}

// Bootstrap consumes the one-time NetBird credential (D-015). The caller
// must persist SetupKey before acknowledging it to anyone else, and must
// not log it.
func (c *Client) Bootstrap(ctx context.Context, idem uuid.UUID) (api.AgentBootstrap, error) {
	var out api.AgentBootstrap
	raw, err := c.postJSON(ctx, ClassOther, pathBootstrap, nil, idem, http.StatusOK)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("controlclient: bootstrap response: %w", err)
	}
	return out, nil
}

// ResolverDocument fetches the canonical resolver document for id. The
// returned bytes are the response body, not a re-encoded JSON value. A 404
// is a StatusError whose Code is NOT_FOUND. The read is capped at 1 MiB.
func (c *Client) ResolverDocument(ctx context.Context, id uuid.UUID) ([]byte, error) {
	if id == uuid.Nil {
		return nil, errors.New("controlclient: empty document id")
	}
	httpReq, err := c.jsonReq(ctx, http.MethodGet, pathResolver+id.String(), nil, uuid.Nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(ctx, ClassOther, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return readLimited(resp.Body)
}

// Usage posts a prepared usage batch on the usage budget and returns the
// response. The caller closes the body. The usage uploader normally calls
// Exchange on ClassUsage instead, so a 409 or 429 body is kept and the lease
// pool stays separate (§14.6).
func (c *Client) Usage(ctx context.Context, idem uuid.UUID, body []byte) (*http.Response, error) {
	httpReq, err := c.jsonReq(ctx, http.MethodPost, pathUsage, body, idem)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, ClassUsage, httpReq)
}

func (c *Client) postJSON(ctx context.Context, class Class, path string, body []byte, idem uuid.UUID, want int) ([]byte, error) {
	httpReq, err := c.jsonReq(ctx, http.MethodPost, path, body, idem)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(ctx, class, httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != want {
		return nil, fmt.Errorf("controlclient: %s status %d", path, resp.StatusCode)
	}
	return readLimited(resp.Body)
}

func (c *Client) jsonReq(ctx context.Context, method, path string, body []byte, idem uuid.UUID) (*http.Request, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, r)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if idem != uuid.Nil {
		req.Header.Set("Idempotency-Key", idem.String())
	}
	return req, nil
}

func errorCode(body []byte) string {
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return ""
	}
	return env.Error.Code
}
