package nodes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CreateGroupTx creates an immutable membership group in the command transaction (§15.2, D-057).
func (s *Service) CreateGroupTx(ctx context.Context, tx pgx.Tx, principal string, requestID *uuid.UUID, body api.MemberGroupCreate) (api.MemberGroup, error) {
	if !regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`).MatchString(body.Code) {
		return api.MemberGroup{}, apierr.Validation("Invalid group code.", apierr.FieldError{Field: "/code", Reason: "invalid"})
	}
	if strings.TrimSpace(body.Name) == "" || utf8.RuneCountInString(body.Name) > 100 {
		return api.MemberGroup{}, apierr.Validation("Invalid group name.", apierr.FieldError{Field: "/name", Reason: "invalid"})
	}
	id, err := uuid.NewV7()
	if err != nil {
		return api.MemberGroup{}, err
	}
	out := api.MemberGroup{GroupId: id, Code: body.Code, Name: body.Name}
	err = tx.QueryRow(ctx, `INSERT INTO member_groups(id,code,name) VALUES($1,$2,$3) ON CONFLICT DO NOTHING RETURNING id`, id, body.Code, body.Name).Scan(&out.GroupId)
	if errors.Is(err, pgx.ErrNoRows) {
		return api.MemberGroup{}, apierr.New(apierr.StateConflict, "Group code or name already exists.")
	}
	if err != nil {
		return api.MemberGroup{}, err
	}
	actor, actorType, _ := ResolveAdminActor(ctx, principal)
	event, err := audit.NewEvent(audit.Entry{
		Actor:  audit.Actor{Type: api.AuditEventActorType(actorType), ID: actor},
		Action: "group.create", Resource: api.ResourceRef{Type: api.ResourceTypeMemberGroup, Id: id},
		Result: api.AuditEventResultSucceeded, RequestID: requestID,
		After: map[string]any{"code": body.Code, "name": body.Name}, Fields: []string{"code", "name"},
	})
	if err != nil {
		return api.MemberGroup{}, err
	}
	return out, audit.AppendEvent(ctx, tx, event)
}

// CreatedGroup reads the immutable creation receipt; its initial exit_count is zero.
func (s *Service) CreatedGroup(ctx context.Context, id uuid.UUID) (out api.MemberGroup, err error) {
	err = s.pool.QueryRow(ctx, `SELECT id,code,name FROM member_groups WHERE id=$1`, id).Scan(&out.GroupId, &out.Code, &out.Name)
	return
}

type groupCursor struct {
	After string `json:"after"`
	Code  string `json:"code"`
}

// ListGroups pages immutable groups by their unique code; counts include unavailable exits.
func (s *Service) ListGroups(ctx context.Context, p api.ListAdminGroupsParams) (api.MemberGroupList, error) {
	out := api.MemberGroupList{Items: []api.MemberGroup{}, NextCursor: presence.NewNullNullable[string]()}
	limit, code, after := 50, "", ""
	if p.Limit != nil {
		limit = *p.Limit
	}
	if limit < 1 || limit > 100 {
		return out, apierr.New(apierr.MalformedRequest, "Invalid page limit.")
	}
	if p.Code != nil {
		code = *p.Code
	}
	if p.Cursor != nil {
		raw, err := base64.RawURLEncoding.DecodeString(*p.Cursor)
		var cursor groupCursor
		if err != nil || json.Unmarshal(raw, &cursor) != nil || cursor.After == "" || cursor.Code != code {
			return out, apierr.New(apierr.MalformedRequest, "Invalid group cursor.")
		}
		after = cursor.After
	}
	rows, err := s.pool.Query(ctx, `SELECT g.id,g.code,g.name,(SELECT count(*) FROM exit_groups e WHERE e.group_id=g.id)
		FROM member_groups g WHERE ($1='' OR g.code=$1) AND g.code>$2 ORDER BY g.code LIMIT $3`, code, after, limit+1)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var g api.MemberGroup
		if err = rows.Scan(&g.GroupId, &g.Code, &g.Name, &g.ExitCount); err != nil {
			return out, err
		}
		out.Items = append(out.Items, g)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	if len(out.Items) > limit {
		out.Items = out.Items[:limit]
		raw, _ := json.Marshal(groupCursor{After: out.Items[limit-1].Code, Code: code})
		out.NextCursor = presence.NewNullableWithValue(base64.RawURLEncoding.EncodeToString(raw))
	}
	return out, nil
}
