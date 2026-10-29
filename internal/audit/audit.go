// Package audit records redacted configuration history, never browsing history.
package audit

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

type Store struct{ Pool *pgxpool.Pool }

func (s Store) ListAuditEvents(ctx context.Context, r gen.ListAuditEventsRequestObject) (gen.ListAuditEventsResponseObject, error) {
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
func (s Store) List(ctx context.Context, p api.ListAuditEventsParams) (page api.AuditEventList, err error) {
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
	var cursorTime *time.Time
	var cursorID *uuid.UUID
	if p.Cursor != nil {
		b, e := base64.RawURLEncoding.DecodeString(*p.Cursor)
		if e != nil || len(b) != 57 || b[0] != 1 || string(b[25:]) != string(hash[:]) {
			return page, apierr.New(apierr.MalformedRequest, "Invalid audit cursor.")
		}
		t := time.UnixMicro(int64(binary.BigEndian.Uint64(b[1:9]))).UTC()
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
	for rows.Next() {
		var event api.AuditEvent
		var actor, resource string
		var before, after []byte
		if e = rows.Scan(&event.AuditEventId, &event.OccurredAt, &actor, &event.Actor.Type, &event.Action, &resource, &event.RequestId, &event.OperationId, &event.Result, &before, &after); e != nil {
			return page, e
		}
		if id, e := uuid.Parse(actor); e == nil {
			event.Actor.Id = &id
		}
		parts := strings.SplitN(resource, ":", 2)
		if len(parts) != 2 {
			return page, errors.New("invalid stored audit resource reference")
		}
		id, e := uuid.Parse(parts[1])
		if e != nil {
			return page, e
		}
		event.Resource = api.ResourceRef{Type: api.ResourceType(parts[0]), Id: id}
		var b, a map[string]any
		d := json.NewDecoder(strings.NewReader(string(before)))
		d.UseNumber()
		if e = d.Decode(&b); e != nil {
			return page, e
		}
		d = json.NewDecoder(strings.NewReader(string(after)))
		d.UseNumber()
		if e = d.Decode(&a); e != nil {
			return page, e
		}
		b = redact(b).(map[string]any)
		a = redact(a).(map[string]any)
		event.Changes.Before = &b
		event.Changes.After = &a
		event.OccurredAt = event.OccurredAt.UTC()
		page.Items = append(page.Items, event)
	}
	if e = rows.Err(); e != nil {
		return page, e
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[limit-1]
		b := make([]byte, 57)
		b[0] = 1
		binary.BigEndian.PutUint64(b[1:9], uint64(last.OccurredAt.UnixMicro()))
		copy(b[9:25], last.AuditEventId[:])
		copy(b[25:], hash[:])
		cursor := base64.RawURLEncoding.EncodeToString(b)
		page.NextCursor = &cursor
	}
	return page, nil
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
