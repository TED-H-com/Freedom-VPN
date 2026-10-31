//go:build e2e

package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/members"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GrantTransition exports read-back only, including incomplete transitions.
// Accepted is deliberately separate from Projected; HTTP 202 is not completion.
type GrantTransition struct {
	RevokeReplayVerified bool        `json:"revoke_replay_verified"`
	CreateReplayVerified bool        `json:"create_replay_verified"`
	MemberID             uuid.UUID   `json:"member_id"`
	PlanRevisionID       uuid.UUID   `json:"plan_revision_id"`
	Revoked              []uuid.UUID `json:"revoked_grants"`
	OperationID          uuid.UUID   `json:"operation_id"`
	Accepted             bool        `json:"accepted"`
	Projected            bool        `json:"projected"`
	BeforeRevision       string      `json:"before_revision"`
	AfterRevision        string      `json:"after_revision"`
	DeviceLimit          int         `json:"device_limit"`
	Authorized           int         `json:"authorized"`
	Pending              int         `json:"pending"`
	ValidUntil           time.Time   `json:"valid_until"`
	ObservedAt           time.Time   `json:"observed_at"`
}

// ReplaceFiniteGrant replaces a grant-only member's finite plan through the
// production domain revoke command and admin grant API (D-050). No projector
// rows are written by this hook. The current authenticated admin supplies the
// actor; no arbitrary administrator or NetBird principal is substituted.
// This also supports restoration to a higher plan. A paid subscription is
// refused, since its independent entitlement could mask the requested limit.
func (s *Stack) ReplaceFiniteGrant(ctx context.Context, admin BrowserSession, member, plan uuid.UUID, until time.Time) (out GrantTransition, err error) {
	out = GrantTransition{MemberID: member, PlanRevisionID: plan, ValidUntil: until.UTC(), Revoked: []uuid.UUID{}}
	if member == uuid.Nil || plan == uuid.Nil || admin.Client == nil || admin.Client.Jar == nil || !until.After(time.Now()) {
		return out, fmt.Errorf("finite grant: member, plan, admin session and future deadline required")
	}
	raw, err := os.ReadFile(filepath.Join(s.lab.state, "config", "members.json"))
	var cfg config.Members
	if err != nil || json.Unmarshal(raw, &cfg) != nil {
		return out, fmt.Errorf("finite grant: prepared members config unavailable")
	}
	ec, err := cfg.Entitlements()
	if err != nil {
		return out, fmt.Errorf("finite grant: invalid prepared entitlement config")
	}
	if time.Until(until) > ec.MaxGrantDuration {
		return out, fmt.Errorf("finite grant: deadline exceeds configured maximum")
	}
	store, err := entitlement.NewStore(ec)
	if err != nil {
		return out, err
	}
	conn, err := s.lab.db(ctx)
	if err != nil {
		return out, err
	}
	poolCfg, err := pgxpool.ParseConfig(conn.Config().ConnString())
	_ = conn.Close(ctx)
	if err != nil {
		return out, fmt.Errorf("finite grant: pool config")
	}
	poolCfg.MaxConns = 2
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return out, fmt.Errorf("finite grant: pool")
	}
	defer pool.Close()
	check, checkErr := s.Call(ctx, admin, "GET", "/api/v1/admin/members/"+member.String(), nil, nil)
	if checkErr != nil || check.Status != 200 {
		return out, fmt.Errorf("finite grant: active administrator session required")
	}
	var actor uuid.UUID
	origin, _ := url.Parse(s.lab.public)
	for _, cookie := range admin.Client.Jar.Cookies(origin) {
		if cookie.Name != auth.AdminSessionCookie {
			continue
		}
		hash := sha256.Sum256([]byte(cookie.Value))
		err = pool.QueryRow(ctx, `SELECT admin_id FROM admin_sessions WHERE token_hash=$1 AND absolute_expires_at>clock_timestamp()`, hash[:]).Scan(&actor)
		break
	}
	if err != nil || actor == uuid.Nil {
		return out, fmt.Errorf("finite grant: authenticated admin actor unavailable")
	}
	var subscriptions int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM subscriptions WHERE member_id=$1 AND is_primary`, member).Scan(&subscriptions); err != nil || subscriptions != 0 {
		return out, fmt.Errorf("finite grant: requires a grant-only fixture member")
	}
	if err = pool.QueryRow(ctx, `SELECT device_limit FROM plan_revisions WHERE id=$1`, plan).Scan(&out.DeviceLimit); err != nil {
		return out, fmt.Errorf("finite grant: plan revision unavailable")
	}
	if err = pool.QueryRow(ctx, `SELECT revision::text FROM entitlements WHERE member_id=$1`, member).Scan(&out.BeforeRevision); err != nil {
		return out, fmt.Errorf("finite grant: existing entitlement required")
	}
	rows, err := pool.Query(ctx, `SELECT id FROM entitlement_grants WHERE member_id=$1 AND revoked_at IS NULL AND valid_until>clock_timestamp() ORDER BY id`, member)
	if err != nil {
		return out, fmt.Errorf("finite grant: grant inventory")
	}
	var grants []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return out, fmt.Errorf("finite grant: grant inventory")
		}
		grants = append(grants, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil || len(grants) == 0 {
		return out, fmt.Errorf("finite grant: live finite grant required")
	}
	domain := members.Service{Pool: pool, Entitlements: store}
	for _, id := range grants {
		cmd := entitlement.Command{Actor: entitlement.Actor{ID: actor, Role: "admin", RequestID: uuid.New()}, Key: uuid.New()}
		if err = domain.RevokeGrant(ctx, cmd, member, id, "TEST ONLY finite plan transition"); err != nil {
			return out, fmt.Errorf("finite grant: production revoke failed")
		}
		if err = domain.RevokeGrant(ctx, cmd, member, id, "TEST ONLY finite plan transition"); err != nil {
			return out, fmt.Errorf("finite grant: revoke replay failed")
		}
		out.Revoked = append(out.Revoked, id)
	}
	out.RevokeReplayVerified = true
	key := uuid.NewString()
	body := api.GrantCreateRequest{PlanRevisionId: &plan, Source: api.GrantSourceAdminGrant, Reason: "TEST ONLY finite plan transition", ValidUntil: out.ValidUntil}
	resp, err := s.Call(ctx, admin, "POST", "/api/v1/admin/members/"+member.String()+"/grants", body, map[string]string{"Idempotency-Key": key})
	if err != nil || resp.Status != 202 {
		return out, fmt.Errorf("finite grant: create HTTP %d", resp.Status)
	}
	out.Accepted = true
	var accepted struct {
		ID uuid.UUID `json:"operation_id"`
	}
	if json.Unmarshal(resp.Body, &accepted) != nil || accepted.ID == uuid.Nil {
		return out, fmt.Errorf("finite grant: operation absent")
	}
	out.OperationID = accepted.ID
	replay, e := s.Call(ctx, admin, "POST", "/api/v1/admin/members/"+member.String()+"/grants", body, map[string]string{"Idempotency-Key": key})
	var replayOp struct {
		ID uuid.UUID `json:"operation_id"`
	}
	out.CreateReplayVerified = e == nil && replay.Status == 202 && json.Unmarshal(replay.Body, &replayOp) == nil && replayOp.ID == out.OperationID
	if !out.CreateReplayVerified {
		return out, fmt.Errorf("finite grant: create replay did not return the same operation")
	}
	wait, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	out.Projected = waitUntil(wait, time.Second, func() bool {
		resp, e := s.Call(wait, admin, "GET", "/api/v1/admin/operations/"+out.OperationID.String(), nil, nil)
		var op struct {
			Status string `json:"status"`
		}
		if e != nil || resp.Status != 200 || json.Unmarshal(resp.Body, &op) != nil || op.Status != "succeeded" {
			return false
		}
		var gotPlan uuid.UUID
		var limit int
		e = pool.QueryRow(wait, `SELECT e.revision::text,e.plan_revision_id,e.device_limit,
		 (SELECT count(*) FROM member_devices WHERE member_id=e.member_id AND status='authorized'),
		 (SELECT count(*) FROM member_devices WHERE member_id=e.member_id AND status='pending')
		 FROM entitlements e WHERE e.member_id=$1`, member).Scan(&out.AfterRevision, &gotPlan, &limit, &out.Authorized, &out.Pending)
		out.ObservedAt = time.Now().UTC()
		return e == nil && gotPlan == plan && limit == out.DeviceLimit && out.AfterRevision != out.BeforeRevision && out.Authorized <= limit
	})
	if !out.Projected {
		return out, fmt.Errorf("finite grant: worker operation/projection deadline")
	}
	return out, nil
}
