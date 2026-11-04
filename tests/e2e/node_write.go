package e2e

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const (
	nodeWriteAttempts = 3
	nodeWriteTimeout  = 10 * time.Second
)

// These receipts deliberately exclude bodies, credentials and transport error
// text. A command accepted here still needs its caller's observed-state gates.
type nodeWriteRead struct {
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	IfMatch    string    `json:"if_match"`
	BodySHA256 string    `json:"body_sha256"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Status     int       `json:"status"`
	ErrorCode  string    `json:"error_code"`
	ETag       string    `json:"etag"`
	Transport  bool      `json:"transport_error"`
	Intended   bool      `json:"already_intended"`
}

type nodeWriteAttempt struct {
	Method         string         `json:"method"`
	Path           string         `json:"path"`
	IfMatch        string         `json:"if_match"`
	IdempotencyKey string         `json:"idempotency_key"`
	BodySHA256     string         `json:"body_sha256"`
	StartedAt      time.Time      `json:"started_at"`
	FinishedAt     time.Time      `json:"finished_at"`
	Status         int            `json:"status"`
	ErrorCode      string         `json:"error_code"`
	CurrentVersion string         `json:"current_resource_version,omitempty"`
	Transport      bool           `json:"transport_error"`
	Reread         *nodeWriteRead `json:"reread,omitempty"`
}

type nodeWriteTrace struct {
	StartedAt    time.Time          `json:"started_at"`
	FinishedAt   time.Time          `json:"finished_at"`
	MaxAttempts  int                `json:"max_attempts"`
	LimitSeconds int                `json:"limit_seconds"`
	Status       Status             `json:"status"`
	Outcome      string             `json:"outcome"`
	Attempts     []nodeWriteAttempt `json:"attempts"`
}

type nodeWriteResult struct {
	Reply           reply
	Err             error
	AlreadyIntended bool
	Receipt         nodeWriteTrace
}

// Accepted reports command acceptance, not operation or observed-state completion.
func (r nodeWriteResult) Accepted(statuses ...int) bool { return r.accepted(statuses...) }

func (r nodeWriteResult) accepted(statuses ...int) bool {
	if r.Err != nil {
		return false
	}
	if r.AlreadyIntended {
		return true
	}
	for _, status := range statuses {
		if r.Reply.Status == status {
			return true
		}
	}
	return false
}

// writeNode follows §14.5 only for an exact RESOURCE_VERSION_CONFLICT. The
// caller supplies the initial GET's ETag and the desired-field predicate. A
// failed write's transaction (including its key reservation) is rolled back by
// httpapi.Commands; the body hash excludes If-Match. Keep one key and body for
// this logical command. No other response or transport failure is retried.
func (s *httpSession) writeNode(ctx context.Context, method, writeURL, readURL string, body any, etag string, intended func([]byte) bool, success []int, note func(string, string)) (out nodeWriteResult) {
	trace := nodeWriteTrace{StartedAt: time.Now().UTC(), MaxAttempts: nodeWriteAttempts, LimitSeconds: int(nodeWriteTimeout / time.Second), Status: StatusBlocked}
	defer func() {
		trace.FinishedAt = time.Now().UTC()
		out.Receipt = trace
		raw, _ := json.Marshal(trace)
		note("node_write", string(raw))
	}()
	bounded, cancel := context.WithTimeout(ctx, nodeWriteTimeout)
	defer cancel()
	// net/http can silently replay keyed requests on a reused connection, and
	// Client normally follows redirects. Neither is an allowed extra attempt.
	// Use private HTTP/1 connections for this command only, preserving the lab
	// dial/TLS settings, cookies and CSRF. Other callers retain their client.
	base := s.http.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	transport, ok := base.(*http.Transport)
	if !ok {
		out.Err, trace.Outcome = errors.New("node write requires an HTTP transport"), "unsupported_transport"
		return
	}
	private := transport.Clone()
	private.DisableKeepAlives = true
	private.Protocols = new(http.Protocols)
	private.Protocols.SetHTTP1(true)
	defer private.CloseIdleConnections()
	client := *s.http
	client.Transport = private
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	once := &httpSession{http: &client, csrf: s.csrf}
	key := uuid.NewString()
	for attempt := 0; attempt < nodeWriteAttempts; attempt++ {
		if bounded.Err() != nil {
			out.Err, trace.Outcome = bounded.Err(), "time_limit"
			return
		}
		row := nodeWriteAttempt{Method: method, Path: evidencePath(writeURL), IfMatch: etag, IdempotencyKey: key, StartedAt: time.Now().UTC()}
		out.Reply, out.Err = once.callObserved(bounded, method, writeURL, body, map[string]string{"If-Match": etag, "Idempotency-Key": key}, func(payload []byte) {
			digest := sha256.Sum256(payload)
			row.BodySHA256 = hex.EncodeToString(digest[:])
		})
		row.FinishedAt, row.Status, row.ErrorCode = time.Now().UTC(), out.Reply.Status, out.Reply.Code
		row.Transport = out.Err != nil
		var env struct {
			Error struct {
				Details struct {
					Version string `json:"current_resource_version"`
				} `json:"details"`
			} `json:"error"`
		}
		if json.Unmarshal(out.Reply.Body, &env) == nil && decimalVersion(env.Error.Details.Version) {
			row.CurrentVersion = env.Error.Details.Version
		}
		trace.Attempts = append(trace.Attempts, row)
		if out.Err != nil {
			trace.Outcome = "transport_error"
			if bounded.Err() != nil {
				trace.Outcome = "time_limit"
			}
			return
		}
		if out.Reply.Status != 412 || out.Reply.Code != "RESOURCE_VERSION_CONFLICT" {
			trace.Outcome = "non_conflict_response"
			if out.accepted(success...) {
				trace.Status, trace.Outcome = StatusPass, "accepted"
			}
			return
		}
		if bounded.Err() != nil {
			out.Err, trace.Outcome = bounded.Err(), "time_limit"
			return
		}
		emptyDigest := sha256.Sum256(nil)
		read := &nodeWriteRead{Method: "GET", Path: evidencePath(readURL), BodySHA256: hex.EncodeToString(emptyDigest[:]), StartedAt: time.Now().UTC()}
		got, err := once.call(bounded, "GET", readURL, nil, nil)
		read.FinishedAt, read.Status, read.ErrorCode, read.ETag, read.Transport = time.Now().UTC(), got.Status, got.Code, got.Header.Get("ETag"), err != nil
		trace.Attempts[len(trace.Attempts)-1].Reread = read
		if err != nil || got.Status != 200 {
			out.Err, trace.Outcome = err, "reread_failed"
			if bounded.Err() != nil {
				trace.Outcome = "time_limit"
			}
			return
		}
		read.Intended = intended(got.Body)
		if read.Intended {
			out.AlreadyIntended = true
			trace.Status, trace.Outcome = StatusPass, "already_intended"
			return
		}
		etag = read.ETag
		if len(etag) < 3 || etag[0] != '"' || etag[len(etag)-1] != '"' || !decimalVersion(etag[1:len(etag)-1]) {
			trace.Outcome = "invalid_reread_etag"
			return
		}
	}
	trace.Outcome = "attempt_limit"
	return
}

func evidencePath(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "invalid_path"
	}
	return u.Path
}

func decimalVersion(s string) bool {
	v, err := strconv.ParseUint(s, 10, 64)
	return err == nil && strconv.FormatUint(v, 10) == s
}

// nodeFieldsMatch compares only the requested desired fields. UseNumber keeps
// rates/versions exact; missing/null fields never count as a matching object.
func nodeFieldsMatch(want any) func([]byte) bool {
	raw, err := json.Marshal(want)
	var expected map[string]any
	if err == nil {
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		err = d.Decode(&expected)
	}
	return func(body []byte) bool {
		if err != nil || len(expected) == 0 {
			return false
		}
		var actual map[string]any
		d := json.NewDecoder(bytes.NewReader(body))
		d.UseNumber()
		if d.Decode(&actual) != nil {
			return false
		}
		for key, value := range expected {
			if !reflect.DeepEqual(actual[key], value) {
				return false
			}
		}
		return true
	}
}

func revokedNode(body []byte) bool {
	return nodeFieldsMatch(map[string]string{"lifecycle_state": "revoked"})(body)
}

func publishedNode(group uuid.UUID) func([]byte) bool {
	// The publisher requests exactly one group, so array order is immaterial.
	return nodeFieldsMatch(map[string]any{"publication_intent": "published", "group_ids": []string{group.String()}})
}
