// Package audit records redacted configuration history, never browsing history.
package audit

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"reflect"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func sensitive(key string) bool {
	k := strings.ToLower(strings.NewReplacer("-", "", "_", "", " ", "").Replace(key))
	for _, term := range []string{"token", "secret", "password", "authorization", "cookie", "privatekey", "setupkey", "signature", "certificate", "csr", "queryname", "url", "destination", "packet", "paymentpayload"} {
		if strings.Contains(k, term) {
			return true
		}
	}
	return false
}
func redact(v any) any {
	switch x := v.(type) {
	case map[string]any:
		m := map[string]any{}
		for k, v := range x {
			if sensitive(k) {
				m[k] = "[REDACTED]"
			} else {
				m[k] = redact(v)
			}
		}
		return m
	case []any:
		a := make([]any, len(x))
		for i, v := range x {
			a[i] = redact(v)
		}
		return a
	case string:
		if strings.Contains(x, "-----BEGIN ") || strings.Contains(x, "Bearer ") || strings.Contains(x, "Enrollment ") || strings.Contains(x, "://") {
			return "[REDACTED]"
		}
	}
	return v
}

// Diff only records explicitly allowlisted top-level fields. Sensitive names are
// still redacted recursively. Callers must not allowlist unbounded free-text or
// provider payloads; allowlists are reviewed with the owning domain.
func Diff(before, after map[string]any, allow ...string) (json.RawMessage, json.RawMessage, error) {
	b, a := map[string]any{}, map[string]any{}
	for _, k := range allow {
		if sensitive(k) {
			continue
		}
		if !reflect.DeepEqual(before[k], after[k]) {
			var err error
			b[k], err = normalizeRedacted(before[k])
			if err != nil {
				return nil, nil, err
			}
			a[k], err = normalizeRedacted(after[k])
			if err != nil {
				return nil, nil, err
			}
		}
	}
	bb, e := json.Marshal(b)
	if e != nil {
		return nil, nil, e
	}
	aa, e := json.Marshal(a)
	return bb, aa, e
}
func Append(ctx context.Context, tx pgx.Tx, event db.AuditEvent, before, after map[string]any, allow ...string) error {
	var e error
	event.Before, event.After, e = Diff(before, after, allow...)
	if e != nil {
		return e
	}
	event.SchemaVersion = 1
	return db.AppendAudit(ctx, tx, event)
}

// Store must not be copied after use. Anomalies counts malformed row encounters,
// not distinct rows; a fresh page may encounter the same malformed row again.
type Store struct {
	Pool      *pgxpool.Pool
	Logger    *slog.Logger
	anomalies atomic.Uint64
}

func (s *Store) AnomalyCount() uint64 { return s.anomalies.Load() }

func (s *Store) recordAnomaly(ctx context.Context, id uuid.UUID, reason string) {
	count := s.anomalies.Add(1)
	logger := s.Logger
	if logger == nil {
		logger = slog.Default()
	}
	// Never log the malformed values, SQL, diffs or underlying parse errors.
	logger.WarnContext(ctx, "audit row omitted", "audit_event_id", id, "reason", reason, "audit_anomalies_total", count)
}

func (s *Store) ListAuditEvents(ctx context.Context, r gen.ListAuditEventsRequestObject) (gen.ListAuditEventsResponseObject, error) {
	p, _ := auth.FromContext(ctx)
	if p.Role != auth.Admin || !p.MFAVerified {
		return nil, apierr.New(apierr.Forbidden, "Administrator MFA required.")
	}
	v, e := s.List(ctx, r.Params)
	if e != nil {
		return nil, e
	}
	return gen.ListAuditEvents200JSONResponse{Body: v}, nil
}
func (s *Store) List(ctx context.Context, p api.ListAuditEventsParams) (page api.AuditEventList, err error) {
	page.Items = []api.AuditEvent{}
	limit := 50
	if p.Limit != nil {
		limit = *p.Limit
	}
	if limit < 1 || limit > 100 {
		return page, apierr.New(apierr.ValidationFailed, "Limit must be between 1 and 100.")
	}
	if p.OccurredFrom != nil && p.OccurredTo != nil && !p.OccurredFrom.Before(*p.OccurredTo) {
		return page, apierr.New(apierr.ValidationFailed, "Time range must be increasing.")
	}
	filters := p
	filters.Cursor = nil
	filters.Limit = nil
	raw, e := canonjson.Marshal(filters)
	if e != nil {
		return page, e
	}
	hash := sha256.Sum256(raw)
	var cursorTime *pgtype.Timestamptz
	types := pgtype.NewMap()
	var cursorID *uuid.UUID
	if p.Cursor != nil {
		b, e := base64.RawURLEncoding.DecodeString(*p.Cursor)
		if e != nil || len(b) != 57 || b[0] != 2 || string(b[25:]) != string(hash[:]) {
			return page, apierr.New(apierr.MalformedRequest, "Invalid audit cursor.")
		}
		var t pgtype.Timestamptz
		if e = types.Scan(pgtype.TimestamptzOID, pgtype.BinaryFormatCode, b[1:9], &t); e != nil {
			return page, apierr.New(apierr.MalformedRequest, "Invalid audit cursor.")
		}
		id, e := uuid.FromBytes(b[9:25])
		if e != nil {
			return page, e
		}
		cursorTime = &t
		cursorID = &id
	}
	var resourceType, resourceID, actor any
	if p.ResourceType != nil {
		resourceType = string(*p.ResourceType)
	}
	if p.ResourceId != nil {
		resourceID = p.ResourceId.String()
	}
	if p.ActorId != nil {
		actor = p.ActorId.String()
	}
	rows, e := s.Pool.Query(ctx, `SELECT id,occurred_at,actor,actor_type,action,resource,request_id,operation_id,result,before_diff,after_diff FROM audit_events WHERE ($1::text IS NULL OR split_part(resource,':',1)=$1) AND ($2::text IS NULL OR split_part(resource,':',2)=$2) AND ($3::text IS NULL OR actor=$3) AND ($4::uuid IS NULL OR request_id=$4) AND ($5::timestamptz IS NULL OR occurred_at >= $5) AND ($6::timestamptz IS NULL OR occurred_at < $6) AND ($7::timestamptz IS NULL OR (occurred_at,id)<($7,$8::uuid)) ORDER BY occurred_at DESC,id DESC LIMIT $9`, resourceType, resourceID, actor, p.RequestId, p.OccurredFrom, p.OccurredTo, cursorTime, cursorID, limit+1)
	if e != nil {
		return page, e
	}
	defer rows.Close()
	var last api.AuditEvent
	var lastTime pgtype.Timestamptz
	scanned := 0
	more := false
	for rows.Next() {
		if scanned == limit {
			more = true
			break
		}
		var event api.AuditEvent
		var actor, resource string
		var before, after []byte
		var occurred pgtype.Timestamptz
		if e = rows.Scan(&event.AuditEventId, &occurred, &actor, &event.Actor.Type, &event.Action, &resource, &event.RequestId, &event.OperationId, &event.Result, &before, &after); e != nil {
			return page, e
		}
		scanned++
		last = event
		lastTime = occurred
		// Keep PostgreSQL's full timestamp range in the cursor, while omitting
		// dates which cannot be represented by the contract's RFC 3339 JSON.
		occurredUTC := occurred.Time.UTC()
		if occurred.InfinityModifier != pgtype.Finite || occurredUTC.Year() < 1 || occurredUTC.Year() > 9999 {
			s.recordAnomaly(ctx, event.AuditEventId, "occurred_at")
			continue
		}
		event.OccurredAt = occurredUTC
		if reason := decodeEvent(&event, actor, resource, before, after); reason != "" {
			s.recordAnomaly(ctx, event.AuditEventId, reason)
			continue
		}
		page.Items = append(page.Items, event)
	}
	if e = rows.Err(); e != nil {
		return page, e
	}
	// Advance by scanned rows, including omissions. Even an empty page must
	// progress past malformed rows without losing the following valid event.
	if more {
		b := make([]byte, 57)
		b[0] = 2
		if _, e = types.Encode(pgtype.TimestamptzOID, pgtype.BinaryFormatCode, lastTime, b[:1]); e != nil {
			return page, e
		}
		copy(b[9:25], last.AuditEventId[:])
		copy(b[25:], hash[:])
		cursor := base64.RawURLEncoding.EncodeToString(b)
		page.NextCursor = &cursor
	}
	return page, nil
}

// decodeEvent validates contract representations without synthesizing identities.
// The producer must repair malformed values in its own write path.
func decodeEvent(event *api.AuditEvent, actor, resource string, before, after []byte) string {
	if !event.Actor.Type.Valid() {
		return "actor_type"
	}
	if id, err := uuid.Parse(actor); err == nil {
		event.Actor.Id = &id
	} else if event.Actor.Type != api.AuditEventActorTypeSystem {
		return "actor_id"
	}
	if !event.Result.Valid() {
		return "result"
	}
	if len(event.Action) > 128 || !regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`).MatchString(event.Action) {
		return "action"
	}
	parts := strings.SplitN(resource, ":", 2)
	if len(parts) != 2 || !api.ResourceType(parts[0]).Valid() {
		return "resource_type"
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return "resource_id"
	}
	event.Resource = api.ResourceRef{Type: api.ResourceType(parts[0]), Id: id}
	for i, raw := range [][]byte{before, after} {
		var value map[string]any
		decoder := json.NewDecoder(strings.NewReader(string(raw)))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return "changes"
		}
		if value != nil {
			value = redact(value).(map[string]any)
			if i == 0 {
				event.Changes.Before = &value
			} else {
				event.Changes.After = &value
			}
		}
	}
	event.OccurredAt = event.OccurredAt.UTC()
	return ""
}

func normalizeRedacted(v any) (any, error) {
	raw, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	var normalized any
	if e = decoder.Decode(&normalized); e != nil {
		return nil, e
	}
	return redact(normalized), nil
}
