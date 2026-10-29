// Package netbird is the sole boundary to NetBird CE (§7.2). Mutations require
// read-back evidence; callers persist intent/outbox before invoking this package.
package netbird

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type ErrorKind string

const (
	NotFound     ErrorKind = "not_found"
	Conflict     ErrorKind = "conflict"
	RateLimited  ErrorKind = "rate_limited"
	Unauthorized ErrorKind = "unauthorized"
	Upstream     ErrorKind = "upstream"
	Unavailable  ErrorKind = "unavailable"
	Invalid      ErrorKind = "invalid"
	Unconfirmed  ErrorKind = "unconfirmed"
)

// Error deliberately excludes URLs, payloads, upstream messages and credentials.
type Error struct {
	Kind       ErrorKind
	Status     int
	RetryAfter time.Duration
}

func (e *Error) Error() string           { return fmt.Sprintf("netbird: %s (HTTP %d)", e.Kind, e.Status) }
func IsKind(err error, k ErrorKind) bool { var e *Error; return errors.As(err, &e) && e.Kind == k }
func failure(k ErrorKind) error          { return &Error{Kind: k} }

// SecretSource must return a token without logging it. Errors are sanitized.
type SecretSource interface {
	Token(context.Context) (string, error)
}
type TokenFile string

func (p TokenFile) Token(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	f, err := os.Open(string(p))
	if err != nil {
		return "", failure(Unauthorized)
	}
	defer f.Close()
	s, err := f.Stat()
	if err != nil || !s.Mode().IsRegular() || s.Mode().Perm()&0077 != 0 {
		return "", failure(Unauthorized)
	}
	b, err := io.ReadAll(io.LimitReader(f, 8193))
	if err != nil || len(b) > 8192 {
		return "", failure(Unauthorized)
	}
	token := strings.TrimSpace(string(b))
	if token == "" || strings.ContainsAny(token, "\r\n") {
		return "", failure(Unauthorized)
	}
	return token, nil
}

type Config struct {
	BaseURL, DeploymentID string
	Token                 SecretSource
	Bindings              BindingStore
	HTTPClient            *http.Client
	Timeout               time.Duration
	// GET retries only, capped at five. Writes are never automatically retried.
	ReadRetries int
	// Explicit opt-in for the isolated HTTP dev network; production requires TLS.
	AllowInsecureHTTP bool
}
type Client struct {
	base, deployment string
	token            SecretSource
	store            BindingStore
	http             *http.Client
	timeout          time.Duration
	retries          int
}

func New(c Config) (*Client, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && !(c.AllowInsecureHTTP && u.Scheme == "http")) || strings.Trim(u.Path, "/") != "api" {
		return nil, failure(Invalid)
	}
	if c.DeploymentID == "" || c.Token == nil || c.Bindings == nil || c.ReadRetries < 0 || c.ReadRetries > 5 {
		return nil, failure(Invalid)
	}
	if c.Timeout == 0 {
		c.Timeout = 10 * time.Second
	}
	if c.Timeout < 0 {
		return nil, failure(Invalid)
	}
	h := http.Client{}
	if c.HTTPClient != nil {
		h = *c.HTTPClient
	}
	h.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{strings.TrimRight(c.BaseURL, "/"), c.DeploymentID, c.Token, c.Bindings, &h, c.Timeout, c.ReadRetries}, nil
}

func (c *Client) call(ctx context.Context, method, path string, in, out any) error {
	var body []byte
	if in != nil {
		var err error
		body, err = json.Marshal(in)
		if err != nil {
			return failure(Invalid)
		}
	}
	attempts := 1
	if method == http.MethodGet {
		attempts += c.retries
	}
	for i := 0; i < attempts; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		token, err := c.token.Token(ctx)
		if err != nil {
			return failure(Unauthorized)
		}
		reqctx, cancel := context.WithTimeout(ctx, c.timeout)
		req, err := http.NewRequestWithContext(reqctx, method, c.base+path, bytes.NewReader(body))
		if err != nil {
			cancel()
			return failure(Invalid)
		}
		req.Header.Set("Authorization", "Token "+token)
		req.Header.Set("Accept", "application/json")
		if in != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := c.http.Do(req)
		var result error
		if err != nil {
			result = failure(Unavailable)
		} else {
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				if out != nil {
					data, e := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024+1))
					if e != nil || len(data) > 4*1024*1024 || json.Unmarshal(data, out) != nil {
						result = failure(Unconfirmed)
					}
				}
			} else {
				k := Invalid
				switch {
				case resp.StatusCode == 404:
					k = NotFound
				case resp.StatusCode == 409 || resp.StatusCode == 412:
					k = Conflict
				case resp.StatusCode == 401 || resp.StatusCode == 403:
					k = Unauthorized
				case resp.StatusCode == 429:
					k = RateLimited
				case resp.StatusCode >= 500:
					k = Upstream
				}
				result = &Error{Kind: k, Status: resp.StatusCode, RetryAfter: retryAfter(resp.Header.Get("Retry-After"))}
			}
			resp.Body.Close()
		}
		cancel()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if result == nil {
			return nil
		}
		if i+1 == attempts || !(IsKind(result, RateLimited) || IsKind(result, Upstream) || IsKind(result, Unavailable)) {
			return result
		}
		delay := time.Duration(50*(1<<i))*time.Millisecond + time.Duration(rand.IntN(50))*time.Millisecond
		var e *Error
		if errors.As(result, &e) && e.RetryAfter > delay {
			delay = e.RetryAfter
		}
		// Do not retry early when the server requests a delay beyond our budget.
		if delay > c.timeout {
			return result
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return failure(Unavailable)
}
func retryAfter(s string) time.Duration {
	if n, e := strconv.ParseInt(s, 10, 32); e == nil && n > 0 {
		return time.Duration(n) * time.Second
	}
	if t, e := http.ParseTime(s); e == nil {
		return max(0, time.Until(t))
	}
	return 0
}
func (c *Client) lock(ctx context.Context) (func(), error) { return c.store.Lock(ctx, c.deployment) }
func endpoint(kind, id string) string {
	p := "/" + kind
	if id != "" {
		p += "/" + url.PathEscape(id)
	}
	return p
}
