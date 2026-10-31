//go:build e2e

package members

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	memberdomain "github.com/TED-H-com/Freedom-VPN/internal/members"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// Keep wire bodies: pointer/raw fields distinguish false/null from omission.
type fenceDevice struct {
	ID          string          `json:"device_id"`
	Member      string          `json:"member_id"`
	Status      string          `json:"status"`
	Unconfirmed *bool           `json:"authorization_unconfirmed"`
	Since       json.RawMessage `json:"authorization_unconfirmed_since"`
}
type fencePage struct {
	Items  []fenceDevice `json:"items"`
	Cursor *string       `json:"next_cursor"`
}

func (l *live) fenceCount(id uuid.UUID) int {
	return l.count(`SELECT count(*) FROM device_authorization_fences WHERE device_id=$1`, id)
}
func (l *live) fenceInsert(r *record, id uuid.UUID) bool {
	before := l.fenceCount(id)
	_, err := l.pool.Exec(l.ctx, `INSERT INTO device_authorization_fences(device_id,fenced_at,observed_group_ids) VALUES($1,clock_timestamp(),'{}')`, id)
	after := l.fenceCount(id)
	r.check("D-077 restore fence input", "0 -> 1 fence; no device/binding/observation edits", fmt.Sprintf("device_id=%s; before=%d; after=%d; inserted=%t", id, before, after, err == nil), "lab Postgres INSERT device_authorization_fences per migration 00188 / restore-drill §6.3", err == nil && before == 0 && after == 1)
	return err == nil && before == 0 && after == 1
}
func (l *live) fenceCall(r *record, name string, b *browser, method, path string, headers map[string]string, id uuid.UUID, status int, code string) response {
	before := l.fenceCount(id)
	got := l.api(b, method, path, nil, headers)
	var op api.Operation
	_ = json.Unmarshal(got.body, &op)
	body := json.RawMessage(got.body)
	if !json.Valid(body) {
		body = json.RawMessage(`null`)
	}
	evidence, _ := json.Marshal(map[string]any{"method": method, "path": path, "status": got.status, "error_code": got.code, "operation_id": op.OperationId, "fence_before": before, "fence_after": l.fenceCount(id), "body": body})
	r.check("D-077 "+name, fmt.Sprintf("HTTP %d %s", status, code), string(evidence), method+" "+path+"; cookies/CSRF/idempotency key retained only in memory", got.status == status && (code == "" || got.code == code))
	return got
}
func fenceMatches(raw []byte, id uuid.UUID, status string, flag bool, siblings bool) bool {
	var page fencePage
	if json.Unmarshal(raw, &page) != nil {
		return false
	}
	found, others := false, 0
	for _, d := range page.Items {
		if d.Unconfirmed == nil {
			return false
		}
		if d.ID == id.String() {
			found = *d.Unconfirmed == flag && d.Status == status
		} else {
			others++
			if *d.Unconfirmed {
				return false
			}
		}
	}
	return found && (!siblings || others > 0)
}
func (l *live) fenceList(r *record, b *browser, id uuid.UUID, status string, flag, siblings bool) {
	got := l.fenceCall(r, "member list", b, "GET", "/api/v1/member/devices", nil, id, 200, "")
	r.check("D-077 member flags always emitted", fmt.Sprintf("target status=%s, authorization_unconfirmed=%t; siblings explicitly false", status, flag), string(got.body), "GET /member/devices; test field presence as well as value", got.status == 200 && fenceMatches(got.body, id, status, flag, siblings))
}
func (l *live) fenceAdminPages(r *record, b *browser, id uuid.UUID) {
	// Compare complete stable keysets twice; no decoding or constructing a cursor.
	var previous []string
	var firstCursor string
	expected := l.count(`SELECT count(*) FROM member_devices`)
	for pass := 0; pass < 2; pass++ {
		keys := []string{}
		cursor := ""
		ended, valid, found := false, true, false
		for n := 0; n < expected+1; n++ {
			path := "/api/v1/admin/devices?limit=1"
			if cursor != "" {
				path += "&cursor=" + url.QueryEscape(cursor)
			}
			got := l.fenceCall(r, fmt.Sprintf("admin page %d/%d", pass+1, n+1), l.admin, "GET", path, nil, id, 200, "")
			var page fencePage
			var fields map[string]json.RawMessage
			if got.status != 200 || json.Unmarshal(got.body, &page) != nil || len(page.Items) > 1 {
				valid = false
				break
			}
			_ = json.Unmarshal(got.body, &fields)
			if _, ok := fields["device_limit"]; ok {
				valid = false
			}
			if _, ok := fields["authorized_count"]; ok {
				valid = false
			}
			for _, d := range page.Items {
				key := d.Member + "/" + d.ID
				if len(keys) > 0 && key <= keys[len(keys)-1] {
					valid = false
				}
				keys = append(keys, key)
				if d.Member == "" || d.Unconfirmed == nil || len(d.Since) == 0 {
					valid = false
					continue
				}
				if d.ID == id.String() {
					var since time.Time
					var stored time.Time
					err := l.pool.QueryRow(l.ctx, `SELECT fenced_at FROM device_authorization_fences WHERE device_id=$1`, id).Scan(&stored)
					found = d.Member == b.member && *d.Unconfirmed && json.Unmarshal(d.Since, &since) == nil && err == nil && since.Equal(stored)
				} else if *d.Unconfirmed || string(d.Since) != "null" {
					valid = false
				}
			}
			if page.Cursor == nil || *page.Cursor == "" {
				ended = true
				break
			}
			if cursor == *page.Cursor {
				valid = false
				break
			}
			cursor = *page.Cursor
			if firstCursor == "" {
				firstCursor = cursor
			}
		}
		r.check("D-077 stable admin keyset", "all devices exactly once ascending (member_id,device_id); required true/false/since/null; repeat same order", fmt.Sprintf("pass=%d; keys=%v; expected=%d; terminal=%t; target_found=%t", pass+1, keys, expected, ended, found), "GET /admin/devices?limit=1; follow opaque next_cursor; repeat traversal", valid && ended && found && len(keys) == expected && (pass == 0 || strings.Join(keys, ",") == strings.Join(previous, ",")))
		previous = keys
	}
	if firstCursor == "" {
		r.check("D-077 cursor available", "nonempty first cursor", "missing", "limit=1", false)
		return
	}
	l.fenceCall(r, "cursor filter mismatch", l.admin, "GET", "/api/v1/admin/devices?limit=1&authorization_unconfirmed=true&cursor="+url.QueryEscape(firstCursor), nil, id, 400, "MALFORMED_REQUEST")
	filtered := l.fenceCall(r, "admin fence filter", l.admin, "GET", "/api/v1/admin/devices?authorization_unconfirmed=true&member_id="+b.member, nil, id, 200, "")
	var page fencePage
	ok := json.Unmarshal(filtered.body, &page) == nil && len(page.Items) == 1 && page.Items[0].ID == id.String()
	r.check("D-077 fence filter contents", "only this fenced device", string(filtered.body), "filter by member_id and authorization_unconfirmed", ok)
}
func authorizePath(member string, id uuid.UUID) string {
	return "/api/v1/admin/members/" + member + "/devices/" + id.String() + "/authorize"
}
func (l *live) fenceGrant(r *record, b *browser, id uuid.UUID, service string, existing bool) {
	var oldOp uuid.UUID
	var oldGeneration string
	_ = l.pool.QueryRow(l.ctx, `SELECT id,desired_generation::text FROM operations WHERE kind='member.device_authorize' AND resource_id=$1 ORDER BY created_at DESC,id DESC LIMIT 1`, id).Scan(&oldOp, &oldGeneration)
	count := l.count(`SELECT count(*) FROM operations WHERE kind='member.device_authorize' AND resource_id=$1`, id)
	headers := commandHeaders()
	got := l.fenceCall(r, "explicit admin authorize", l.admin, "POST", authorizePath(b.member, id), headers, id, 202, "")
	var op api.Operation
	decoded := json.Unmarshal(got.body, &op) == nil
	generation, generationErr := op.DesiredGeneration.Get()
	r.check("D-077 grant transaction and operation", "member.device_authorize; fence gone; authorized restore reuses existing operation and pre-bump generation", fmt.Sprintf("operation_id=%s; previous_id=%s; desired_generation=%s; previous_generation=%s; rows_before=%d; rows_after=%d; fence_after=%d", op.OperationId, oldOp, generation, oldGeneration, count, l.count(`SELECT count(*) FROM operations WHERE kind='member.device_authorize' AND resource_id=$1`, id), l.fenceCount(id)), "POST authorize; read fence count and operation identity (never wait for a new operation)", got.status == 202 && decoded && op.Kind == api.OperationKindMemberDeviceAuthorize && l.fenceCount(id) == 0 && (!existing || (op.OperationId == oldOp && generationErr == nil && generation.String() == oldGeneration && count == l.count(`SELECT count(*) FROM operations WHERE kind='member.device_authorize' AND resource_id=$1`, id))))
	var version string
	_ = l.pool.QueryRow(l.ctx, `SELECT resource_version::text FROM member_devices WHERE id=$1`, id).Scan(&version)
	replay := l.fenceCall(r, "same-key replay", l.admin, "POST", authorizePath(b.member, id), headers, id, 202, "")
	var replayOp api.Operation
	_ = json.Unmarshal(replay.body, &replayOp)
	var afterVersion string
	_ = l.pool.QueryRow(l.ctx, `SELECT resource_version::text FROM member_devices WHERE id=$1`, id).Scan(&afterVersion)
	r.check("D-077 replay has no second grant", "same operation and device version", fmt.Sprintf("operation_id=%s; version_before=%s; version_after=%s", replayOp.OperationId, version, afterVersion), "replay identical Idempotency-Key; read device version", op.OperationId != uuid.Nil && replayOp.OperationId == op.OperationId && version == afterVersion)
	var final response
	ready := l.wait(90*time.Second, func() bool {
		final = l.api(b, "GET", "/api/v1/member/devices", nil, nil)
		return final.status == 200 && fenceMatches(final.body, id, "authorized", false, false)
	})
	r.check("D-077 worker re-grant confirmed from member list", "authorized and explicit false after worker sync", string(final.body), "poll GET /member/devices; do not poll old operation step transitions", ready)
	l.fenceList(r, b, id, "authorized", false, false)
	l.forwardingGate(r)
	counted := l.traceForwarding(r, service, id)
	r.check("D-077 re-grant counted forwarding", "origin HTTP receipt and both device counters increase", fmt.Sprintf("device_id=%s; counted=%t", id, counted), "CA-verified HTTPS through fvmb0 with named device counter attribution", counted)
}
func (l *live) reauthorization(r *record, b *browser, id uuid.UUID, service string, siblings, negatives bool) {
	l.admin = l.login("e2e-admin", "admin")
	l.forwardingGate(r)
	before := l.traceForwarding(r, service, id)
	r.check("D-077 admitted positive control", "counted forwarding before fence", fmt.Sprintf("device_id=%s; counted=%t", id, before), "same native device and controlled HTTPS origin", before)
	if !l.fenceInsert(r, id) {
		return
	}
	withdrawn := l.wait(90*time.Second, func() bool {
		return l.count(`SELECT count(*) FROM device_authorization_observations WHERE device_id=$1 AND NOT observed_authorized`, id) == 1
	})
	r.check("D-077 worker withdrawal", "fenced device observed withdrawn", fmt.Sprintf("device_id=%s; withdrawn=%t; fence_count=%d", id, withdrawn, l.fenceCount(id)), "wait running worker sync; read observation only", withdrawn)
	l.fenceList(r, b, id, "pending", true, siblings)
	if negatives {
		l.fenceAdminPages(r, b, id)
		path := authorizePath(b.member, id)
		l.fenceCall(r, "member session denied", b, "POST", path, commandHeaders(), id, 401, "UNAUTHENTICATED")
		missing := commandHeaders()
		missing["X-CSRF-Token"] = ""
		l.fenceCall(r, "missing CSRF", l.admin, "POST", path, missing, id, 403, "")
		l.fenceCall(r, "wrong member", l.admin, "POST", authorizePath(l.b.member, id), commandHeaders(), id, 404, "RESOURCE_NOT_FOUND")
		r.check("D-077 negatives retain fence", "one fence", fmt.Sprint(l.fenceCount(id)), "read fence after denied mutations", l.fenceCount(id) == 1)
	}
	// Passing the real ID deliberately retains device counter attribution even if
	// the product fails to withdraw. Classify this observation explicitly below.
	previous := len(r.Forwarding)
	l.traceForwardingExpected(r, service, id, false)
	if len(r.Forwarding) == previous {
		r.check("D-077 fenced forwarding", "no origin receipt", "probe unavailable", "bounded HTTPS", false)
	} else {
		o := r.Forwarding[len(r.Forwarding)-1]
		var transfer struct {
			Receipt    bool `json:"origin_receipt"`
			ProbeError bool `json:"probe_error"`
		}
		decoded := json.Unmarshal(o.Transfer, &transfer) == nil
		raw, _ := json.Marshal(o)
		r.check("D-077 fenced forwarding", "zero origin receipts and no counted HTTPS; wire-members owns any continued forwarding", string(raw), "same native device HTTPS after worker withdrawal; inspect fvmb0 and device counters", decoded && !transfer.ProbeError && o.OriginVerified && !transfer.Receipt && o.OriginRequests == 0 && !o.Counted)
	}
	l.write(r)
	l.fenceGrant(r, b, id, service, true)
}

// There is no HTTP grant-revoke endpoint or shared plan-downgrade export. Use
// the production domain transaction for fixture setup only; every D-077 call
// still goes through the running HTTP API. Never edit entitlement/device rows.
func (l *live) replaceFixtureGrant(r *record, b *browser, rev uuid.UUID) bool {
	svc, err := l.admission()
	if err != nil {
		return false
	}
	// Bootstrap admin credentials deliberately have no public-subject ID.
	// Resolve the actor from this authenticated admin session, never from a
	// NetBird credential or an arbitrary administrator row.
	var actor uuid.UUID
	origin, _ := url.Parse("https://portal.example.invalid:8443")
	for _, cookie := range l.admin.client.Jar.Cookies(origin) {
		if cookie.Name != auth.AdminSessionCookie {
			continue
		}
		hash := sha256.Sum256([]byte(cookie.Value))
		err = l.pool.QueryRow(l.ctx, `SELECT admin_id FROM admin_sessions WHERE token_hash=$1 AND absolute_expires_at>clock_timestamp()`, hash[:]).Scan(&actor)
		break
	}
	r.check("D-081 fixture administrator", "actor resolved from current admin session", fmt.Sprintf("resolved=%t", actor != uuid.Nil && err == nil), "hash own cookie in memory; read matching unexpired admin_sessions row; no token emitted", actor != uuid.Nil && err == nil)
	if actor == uuid.Nil || err != nil {
		return false
	}
	rows, err := l.pool.Query(l.ctx, `SELECT id FROM entitlement_grants WHERE member_id=$1 AND revoked_at IS NULL`, b.member)
	if err != nil {
		return false
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if rows.Scan(&id) != nil {
			rows.Close()
			return false
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return false
	}
	domain := memberdomain.Service{Pool: l.pool, Entitlements: svc.Entitlements}
	for _, id := range ids {
		cmd := entitlement.Command{Actor: entitlement.Actor{ID: actor, Role: "admin", RequestID: uuid.New()}, Key: uuid.New()}
		err = domain.RevokeGrant(l.ctx, cmd, uuid.MustParse(b.member), id, "TEST ONLY D-081 plan transition")
		r.check("D-081 fixture grant withdrawal", "production RevokeGrant succeeds", fmt.Sprintf("grant_id=%s; succeeded=%t", id, err == nil), "members.Service.RevokeGrant; shared export requested from devstack follow-up 15", err == nil)
		if err != nil {
			return false
		}
	}
	got := l.api(l.admin, "POST", "/api/v1/admin/members/"+b.member+"/grants", api.GrantCreateRequest{PlanRevisionId: &rev, Source: api.GrantSourceAdminGrant, Reason: "TEST ONLY D-081 plan transition", ValidUntil: time.Now().UTC().Add(time.Hour)}, commandHeaders())
	r.check("D-081 fixture grant creation", "202 through production API", string(got.body), "POST /admin/members/{id}/grants", got.status == 202)
	return got.status == 202
}
func (l *live) reauthorizationLimit(r *record, id, revoked uuid.UUID, service string) {
	l.admin = l.login("e2e-admin", "admin")
	l.fenceCall(r, "revoked device", l.admin, "POST", authorizePath(l.a.member, revoked), commandHeaders(), revoked, 409, "STATE_CONFLICT")
	var raw []byte
	if l.pool.QueryRow(l.ctx, `SELECT payload FROM plan_revisions WHERE id=$1`, l.planRev).Scan(&raw) != nil {
		r.block("D-081 plan input unavailable")
		return
	}
	var payload map[string]any
	if json.Unmarshal(raw, &payload) != nil {
		r.block("D-081 plan input decode failed")
		return
	}
	payload["device_limit"] = 2
	lower := l.activePolicy(r, "plan", payload)
	if lower == uuid.Nil {
		r.block("D-081 lower plan unavailable")
		return
	}
	if !l.replaceFixtureGrant(r, l.a, lower) {
		r.check("D-081 setup", "real plan downgrade", "failed", "production grant transition", false)
		return
	}
	pending := l.wait(60*time.Second, func() bool {
		return l.count(`SELECT count(*) FROM member_devices WHERE id=$1 AND status='pending'`, id) == 1 && l.count(`SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='authorized'`, l.a.member) == 2
	})
	r.check("D-081 limit precondition", "two authorized of limit two; replacement pending", fmt.Sprintf("ready=%t", pending), "real plan downgrade -> entitlement worker reconciliation; no row fixtures", pending)
	if !l.fenceInsert(r, id) {
		return
	}
	l.fenceCall(r, "device limit", l.admin, "POST", authorizePath(l.a.member, id), commandHeaders(), id, 409, "DEVICE_LIMIT_REACHED")
	r.check("D-081 limit denial retains fence", "one fence", fmt.Sprint(l.fenceCount(id)), "read fence count after HTTP 409", l.fenceCount(id) == 1)
	restored := l.replaceFixtureGrant(r, l.a, l.planRev)
	r.check("D-077 entitlement restoration retains fence", "original plan restored; fence stays", fmt.Sprintf("restored=%t; fence=%d", restored, l.fenceCount(id)), "production grant transition, then explicit re-authorization", restored && l.fenceCount(id) == 1)
	l.fenceGrant(r, l.a, id, service, false)
}
