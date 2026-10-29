package controlclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

const (
	// ClassLease is the budget for serving-lease renewal (§14.6, §16.3).
	ClassLease Class = "lease"
	// ClassUsage is the budget for usage uploads. It cannot borrow lease slots.
	ClassUsage Class = "usage"
	// ClassOther is the budget for desired state, observations, enrollment,
	// certificate renewal and bootstrap.
	ClassOther Class = "other"
)

// Class selects a concurrency budget and an HTTP connection pool.
type Class string

// ErrNotModified is a desired-state 304. The caller must not treat it as a
// lease extension (§14.7).
var ErrNotModified = errors.New("controlclient: not modified")

// StatusError is an HTTP failure. Code is the API error code when the body
// was an error envelope. The body is not retained.
type StatusError struct {
	Status     int
	Code       string
	RetryAfter time.Duration
}

func (e *StatusError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("controlclient: http %d %s", e.Status, e.Code)
	}
	return fmt.Sprintf("controlclient: http %d", e.Status)
}

// Budgets is the per-class concurrency limit. Zero means DefaultBudgets.
type Budgets struct {
	Lease int
	Usage int
	Other int
}

// DefaultBudgets keeps a lease slot free of usage work (§14.6).
func DefaultBudgets() Budgets {
	return Budgets{Lease: 2, Usage: 1, Other: 2}
}

func (b Budgets) limit(c Class) int {
	if b == (Budgets{}) {
		b = DefaultBudgets()
	}
	switch c {
	case ClassLease:
		if b.Lease < 1 {
			return 1
		}
		return b.Lease
	case ClassUsage:
		if b.Usage < 1 {
			return 1
		}
		return b.Usage
	default:
		if b.Other < 1 {
			return 1
		}
		return b.Other
	}
}

// Config is the pinned control endpoint.
type Config struct {
	BaseURL              string
	RootCAs              *x509.CertPool
	ServerName           string
	GetClientCertificate func(*tls.CertificateRequestInfo) (*tls.Certificate, error)
	Timeout              time.Duration
	DialTimeout          time.Duration
	HandshakeTimeout     time.Duration
	Budgets              Budgets
}

// Client calls the agent API. Copies of the client share no connection pool
// across classes.
type Client struct {
	base    string
	http    map[Class]*http.Client
	sem     map[Class]chan struct{}
	closeMu sync.Mutex
	closed  bool
}

// New builds the client. A missing CA pool or base URL fails closed.
func New(cfg Config) (*Client, error) {
	if cfg.RootCAs == nil {
		return nil, errors.New("controlclient: no pinned CA")
	}
	u, err := parseBase(cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	if cfg.HandshakeTimeout <= 0 {
		cfg.HandshakeTimeout = 5 * time.Second
	}
	c := &Client{
		base: u,
		http: map[Class]*http.Client{},
		sem:  map[Class]chan struct{}{},
	}
	for _, class := range []Class{ClassLease, ClassUsage, ClassOther} {
		n := cfg.Budgets.limit(class)
		c.sem[class] = make(chan struct{}, n)
		c.http[class] = &http.Client{
			Timeout:       cfg.Timeout,
			CheckRedirect: refuseRedirect,
			Transport:     newTransport(cfg, n),
		}
	}
	return c, nil
}

func parseBase(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return "", errors.New("controlclient: control URL must be absolute http(s)")
	}
	return strings.TrimRight(u.String(), "/"), nil
}

func refuseRedirect(*http.Request, []*http.Request) error {
	return errors.New("controlclient: redirect refused")
}

func newTransport(cfg Config, conns int) *http.Transport {
	dial := &net.Dialer{Timeout: cfg.DialTimeout}
	return &http.Transport{
		Proxy:                 nil,
		DialContext:           dial.DialContext,
		TLSHandshakeTimeout:   cfg.HandshakeTimeout,
		ResponseHeaderTimeout: cfg.Timeout,
		IdleConnTimeout:       30 * time.Second,
		MaxConnsPerHost:       conns,
		MaxIdleConnsPerHost:   conns,
		ForceAttemptHTTP2:     false,
		TLSNextProto:          map[string]func(string, *tls.Conn) http.RoundTripper{},
		TLSClientConfig: &tls.Config{
			MinVersion:           tls.VersionTLS12,
			RootCAs:              cfg.RootCAs,
			ServerName:           cfg.ServerName,
			GetClientCertificate: cfg.GetClientCertificate,
		},
	}
}

// ResetConnections drops idle connections so the next request presents the
// current client certificate. A connection opened before enrollment has no
// certificate and must not be reused for mTLS calls.
func (c *Client) ResetConnections() {
	if c == nil {
		return
	}
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	for _, h := range c.http {
		h.CloseIdleConnections()
	}
}

// Close idle connections. In-flight requests are left to finish.
func (c *Client) Close() {
	if c == nil {
		return
	}
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	c.closed = true
	for _, h := range c.http {
		h.CloseIdleConnections()
	}
}

type releaseBody struct {
	io.ReadCloser
	once sync.Once
	fn   func()
}

func (b *releaseBody) Close() error {
	err := b.ReadCloser.Close()
	b.once.Do(b.fn)
	return err
}

func (c *Client) acquire(ctx context.Context, class Class) error {
	sem := c.sem[class]
	if sem == nil {
		return errors.New("controlclient: unknown class")
	}
	select {
	case sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("controlclient: %s budget: %w", class, ctx.Err())
	}
}

func (c *Client) release(class Class) {
	sem := c.sem[class]
	if sem == nil {
		return
	}
	select {
	case <-sem:
	default:
	}
}

// exchangeLimit matches the usage uploader's response cap.
const exchangeLimit = 8 << 20

// Exchange performs one request on the class budget and returns every HTTP
// status with its body. It uses the client and TLS config New already built
// (TLS 1.2 minimum, pinned CA, no redirects, HTTP/2 off). It does not
// construct a transport. 409, 429 and 503 are not errors; the caller reads
// the status. A transport failure is an error and status is 0.
func (c *Client) Exchange(ctx context.Context, class Class, method, path string, header http.Header, body []byte) (status int, resp []byte, retryAfter time.Duration, err error) {
	if c == nil {
		return 0, nil, 0, errors.New("controlclient: closed")
	}
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, r)
	if err != nil {
		return 0, nil, 0, err
	}
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if err := c.acquire(ctx, class); err != nil {
		return 0, nil, 0, err
	}
	defer c.release(class)
	h := c.http[class]
	if h == nil {
		return 0, nil, 0, errors.New("controlclient: unknown class")
	}
	res, err := h.Do(req)
	if err != nil {
		return 0, nil, 0, err
	}
	defer res.Body.Close()
	buf, err := io.ReadAll(io.LimitReader(res.Body, exchangeLimit+1))
	if err != nil {
		return res.StatusCode, nil, 0, err
	}
	if len(buf) > exchangeLimit {
		return res.StatusCode, nil, 0, errors.New("controlclient: response exceeds 8 MiB")
	}
	return res.StatusCode, buf, retryAfterOf(res.Header.Get("Retry-After"), time.Now()), nil
}

func retryAfterOf(v string, now time.Time) time.Duration {
	if v == "" {
		return 0
	}
	if n, err := strconv.Atoi(v); err == nil {
		if n < 0 {
			return 0
		}
		return time.Duration(n) * time.Second
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

// Do performs one request on the class budget. On success the caller must
// close the body; that releases the budget. A non-2xx response is returned
// as *StatusError and the body is already closed.
func (c *Client) Do(ctx context.Context, class Class, req *http.Request) (*http.Response, error) {
	if err := c.acquire(ctx, class); err != nil {
		return nil, err
	}
	h := c.http[class]
	if h == nil {
		c.release(class)
		return nil, errors.New("controlclient: unknown class")
	}
	resp, err := h.Do(req.WithContext(ctx))
	if err != nil {
		c.release(class)
		return nil, err
	}
	resp.Body = &releaseBody{ReadCloser: resp.Body, fn: func() { c.release(class) }}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		se := statusFrom(resp)
		_ = resp.Body.Close()
		return nil, se
	}
	return resp, nil
}

func statusFrom(resp *http.Response) *StatusError {
	se := &StatusError{Status: resp.StatusCode}
	if v := resp.Header.Get("Retry-After"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			se.RetryAfter = time.Duration(n) * time.Second
		}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return se
	}
	se.Code = errorCode(body)
	return se
}

func readLimited(r io.Reader) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, canonjson.MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > canonjson.MaxBytes {
		return nil, errors.New("controlclient: response exceeds 1 MiB")
	}
	return b, nil
}
