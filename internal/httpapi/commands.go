package httpapi

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReplayHandler re-reads only persisted references, with caller authorization.
// Return the generated response type for operationID. Enrollment replays MUST
// return secret_available=false; neither plaintext responses nor secrets persist.
type ReplayHandler interface {
	Replay(context.Context, string, db.IdempotencyRecord) (any, error)
}
type Commands struct {
	Pool      *pgxpool.Pool
	Retention time.Duration
}
type commandState struct {
	tx     pgx.Tx
	result *db.IdempotencyResult
}
type commandKey struct{}

func Transaction(ctx context.Context) (pgx.Tx, error) {
	s, ok := ctx.Value(commandKey{}).(*commandState)
	if !ok {
		return nil, errors.New("command transaction unavailable")
	}
	return s.tx, nil
}

// SetResult must accompany desired state, operation, audit and outbox writes in
// Transaction(ctx). No external network calls are allowed inside this boundary.
func SetResult(ctx context.Context, r db.IdempotencyResult) error {
	s, ok := ctx.Value(commandKey{}).(*commandState)
	if !ok {
		return errors.New("command transaction unavailable")
	}
	if s.result != nil {
		return errors.New("command result already recorded")
	}
	s.result = &r
	return nil
}
func (c *Composite) owner(name string) any {
	m, ok := reflect.TypeFor[gen.StrictServerInterface]().MethodByName(name)
	if !ok {
		return nil
	}
	for _, s := range c.sets {
		v := reflect.ValueOf(s)
		if !v.IsValid() {
			continue
		}
		fn := v.MethodByName(name)
		if fn.IsValid() && fn.Type() == m.Type {
			return s
		}
	}
	return nil
}
func mutation(r *http.Request) bool {
	return r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS"
}
func commandNeeded(r *http.Request) bool {
	p, _ := auth.FromContext(r.Context())
	return mutation(r) && (p.Role == auth.Admin || p.Role == auth.Member || (p.Role == auth.Agent || r.URL.Path == "/api/v1/agent/enroll") && r.Header.Get("Idempotency-Key") != "")
}
func requireCommandBoundary(next gen.StrictHandlerFunc, _ string) gen.StrictHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
		if commandNeeded(r) {
			return nil, apierr.New(apierr.DependencyUnavailable, "Command persistence is unavailable.")
		}
		return next(ctx, w, r, request)
	}
}
func (c *Commands) middleware(handlers *Composite) gen.StrictMiddlewareFunc {
	return func(next gen.StrictHandlerFunc, operationID string) gen.StrictHandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (response any, err error) {
			owner := handlers.owner(operationID)
			if owner == nil {
				return nil, errNotImplemented
			}
			if !commandNeeded(r) {
				return next(ctx, w, r, request)
			}
			if c.Pool == nil {
				return nil, apierr.New(apierr.DependencyUnavailable, "Command persistence is unavailable.")
			}
			replay, ok := owner.(ReplayHandler)
			if !ok {
				return nil, errors.New("command handler needs Replay")
			}
			key, e := uuid.Parse(r.Header.Get("Idempotency-Key"))
			if e != nil {
				return nil, apierr.New(apierr.MalformedRequest, "UUID Idempotency-Key is required.")
			}
			m, ok := ctx.Value(metaKey{}).(*requestMeta)
			if !ok {
				return nil, errors.New("validated request metadata missing")
			}
			raw := m.raw
			if len(raw) == 0 {
				raw = []byte(`{}`)
			}
			hash, e := canonjson.Hash(raw)
			if e != nil {
				return nil, e
			}
			kh := sha256.Sum256([]byte(key.String()))
			p, _ := auth.FromContext(ctx)
			principalScope := p.Scope()
			if p.Role == "" {
				credential := sha256.Sum256([]byte(r.Header.Get("Authorization")))
				principalScope = "enrollment:" + fmt.Sprintf("%x", credential)
			}
			var record db.IdempotencyRecord
			err = db.InTx(ctx, c.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
				var e error
				record, e = db.BeginIdempotent(ctx, tx, db.IdempotencyScope{Principal: principalScope, Method: r.Method, Route: m.scope, KeyHash: kh[:], RequestHash: hash[:]}, c.Retention)
				if e != nil {
					return e
				}
				if !record.New {
					return nil
				}
				state := &commandState{tx: tx}
				commandCtx := context.WithValue(ctx, commandKey{}, state)
				response, e = next(commandCtx, w, r, request)
				if e != nil {
					return e
				}
				if state.result == nil || response == nil {
					return errors.New("command did not record a durable result")
				}
				return db.CompleteIdempotent(ctx, tx, record.ID, *state.result)
			})
			if errors.Is(err, db.ErrIdempotencyConflict) {
				return nil, apierr.New(apierr.IdempotencyConflict, "Key was already used with a different payload.")
			}
			if err != nil {
				return nil, err
			}
			if !record.New {
				response, err := replay.Replay(ctx, operationID, record)
				if err != nil {
					return nil, err
				}
				return sanitizeReplay(operationID, response)
			}
			return response, nil
		}
	}
}

// ETag is the strong representation of a mutable resource's version (§14.5).
func ETag(version decstr.U64) string { return `"` + version.String() + `"` }

// CheckIfMatch must run under the same row lock or conditional UPDATE as the
// mutation; checking a previously read version outside the transaction is unsafe.
func CheckIfMatch(value string, current decstr.U64) error {
	if value == "" {
		return apierr.New(apierr.PreconditionRequired, "If-Match is required.")
	}
	if value != ETag(current) {
		return apierr.New(apierr.ResourceVersionConflict, "Resource has changed; reload before saving.").WithDetail("current_resource_version", current)
	}
	return nil
}
func NotModified(w http.ResponseWriter, r *http.Request, etag string) bool {
	w.Header().Set("ETag", etag)
	for _, v := range strings.Split(r.Header.Get("If-None-Match"), ",") {
		v = strings.TrimSpace(v)
		if v == "*" || strings.TrimPrefix(v, "W/") == etag {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}
	return false
}

// Cleanup deletes expired references, never resources, operations or audit rows.
func (c *Commands) Cleanup(ctx context.Context) error {
	_, e := c.Pool.Exec(ctx, `DELETE FROM idempotency_requests WHERE expiry<=clock_timestamp()`)
	return e
}

func sanitizeReplay(operationID string, response any) (any, error) {
	if operationID != "CreateEnrollment" {
		return response, nil
	}
	switch v := response.(type) {
	case gen.CreateEnrollment201JSONResponse:
		v.Body.SecretAvailable = false
		v.Body.Token = nil
		return v, nil
	case *gen.CreateEnrollment201JSONResponse:
		if v != nil {
			copy := *v
			copy.Body.SecretAvailable = false
			copy.Body.Token = nil
			return copy, nil
		}
	}
	return nil, errors.New("invalid enrollment replay response")
}
