//go:build e2e

package members

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

func (r response) grantResp() grantHTTPResponse {
	return grantHTTPResponse{status: r.status, body: r.body, code: r.code}
}

func (r *record) jsonCheck(name, expected string, value any, command string, ok bool) {
	raw, err := json.Marshal(value)
	r.check(name, expected, string(raw), command, ok && err == nil)
}

type grantHTTP struct {
	Method string          `json:"method"`
	Path   string          `json:"path"`
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
	At     time.Time       `json:"at"`
}
type grantCapture struct {
	base http.RoundTripper
	rows []grantHTTP
}

func (c *grantCapture) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := c.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	// Only these credential-free product responses are retained. Never headers,
	// cookies, admin member details or other incidental login traffic.
	if (req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/grants")) || (req.Method == "GET" && strings.HasPrefix(req.URL.Path, "/api/v1/admin/operations/")) {
		raw, e := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if e != nil {
			return nil, e
		}
		resp.Body = io.NopCloser(bytes.NewReader(raw))
		c.rows = append(c.rows, grantHTTP{req.Method, req.URL.Path, resp.StatusCode, raw, time.Now().UTC()})
	}
	return resp, nil
}
func (l *live) replaceFiniteGrant(r *record, b *browser, plan uuid.UUID, until time.Time) bool {
	capture := &grantCapture{base: l.transport}
	client := *l.admin.client
	client.Transport = capture
	beforeAuth := l.count(`SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='authorized'`, b.member)
	beforePending := l.count(`SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='pending'`, b.member)
	l.holdSnapshot("before finite grant revocations")
	defer l.holdSnapshot("after finite grant revocations and projection")
	result, err := l.hook.ReplaceFiniteGrant(l.ctx, labstack.BrowserSession{Client: &client, CSRF: l.admin.csrf}, uuid.MustParse(b.member), plan, until)
	r.jsonCheck("finite grant production transition", "accepted, replayed and worker projected; finite deadline; export partial result on error", map[string]any{"before_authorized": beforeAuth, "before_pending": beforePending, "transition": result, "error": safeError(err), "http": capture.rows}, "Stack.ReplaceFiniteGrant: authenticated domain revoke + public grant POST + worker operation GET; no projection writes", err == nil && result.Projected && result.RevokeReplayVerified && result.CreateReplayVerified && result.Authorized == result.DeviceLimit && result.Pending == 3-result.DeviceLimit)
	for _, id := range result.Revoked {
		var audit []byte
		e := l.pool.QueryRow(l.ctx, `SELECT coalesce(jsonb_agg(jsonb_build_object('action',action,'resource',resource,'result',result,'operation_id',operation_id,'occurred_at',occurred_at)),'[]'::jsonb) FROM audit_events WHERE action='member.grant_revoke' AND resource=$1`, "entitlement_grant:"+id.String()).Scan(&audit)
		n := l.count(`SELECT count(*) FROM audit_events WHERE action='member.grant_revoke' AND resource=$1`, "entitlement_grant:"+id.String())
		r.check("finite grant audited revoke replay", "exactly one audit event per revoked grant", string(audit), "read audit_events; same-key replay must not duplicate revocation", e == nil && n == 1)
	}
	return err == nil && result.Projected
}
func safeError(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
func (l *live) invalidFiniteGrant(r *record, b *browser, plan uuid.UUID) {
	before := l.revision(b)
	live := l.count(`SELECT count(*) FROM entitlement_grants WHERE member_id=$1 AND revoked_at IS NULL AND valid_until>clock_timestamp()`, b.member)
	audits := l.count(`SELECT count(*) FROM audit_events WHERE action='member.grant_revoke'`)
	got, err := l.hook.ReplaceFiniteGrant(l.ctx, labstack.BrowserSession{Client: l.admin.client, CSRF: l.admin.csrf}, uuid.MustParse(b.member), plan, time.Now().UTC().Add(-time.Second))
	r.jsonCheck("expired finite input rejected before revoke", "error; no revoked grants, unchanged revision/live grants/audits", map[string]any{"result": got, "error": safeError(err), "revision_before": before, "revision_after": l.revision(b), "live_grants": live}, "Stack.ReplaceFiniteGrant(past deadline); read-only before/after projection and audit", err != nil && !got.Accepted && len(got.Revoked) == 0 && before == l.revision(b) && live == l.count(`SELECT count(*) FROM entitlement_grants WHERE member_id=$1 AND revoked_at IS NULL AND valid_until>clock_timestamp()`, b.member) && audits == l.count(`SELECT count(*) FROM audit_events WHERE action='member.grant_revoke'`))
}

// Expiry is a separate transition: without a lower subscription/grant the
// product expires all slots. Overlapping different plans are rejected by the
// entitlement engine, so no scheduled downgrade is fabricated by this harness.
//
// Every early return after the member's grant may have changed restores l.a
// through POST /admin/members/{id}/grants (see finiteExpirySteps in
// grant_restore.go for each path's grant state and entitlement rule).
func (l *live) finiteExpiry(r *record, id uuid.UUID, service string) {
	until := time.Now().UTC().Add(75 * time.Second).Truncate(time.Second)
	runFiniteExpiry(r, finiteExpirySteps{
		until: until,
		replaceFiniteGrant: func(u time.Time) bool {
			return l.replaceFiniteGrant(r, l.a, l.planRev, u)
		},
		forwardingGate: func() {
			l.forwardingGate(r)
		},
		traceForwarding: func() bool {
			return l.traceForwarding(r, service, id)
		},
		runNormalExpiry: func(u time.Time) {
			flow := l.startPersistent(r, service, id)
			expired := l.wait(time.Until(u)+30*time.Second, func() bool { return l.entitlementState(l.a) == "expired" })
			authorized := l.count(`SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='authorized'`, l.a.member)
			pending := l.count(`SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='pending'`, l.a.member)
			r.jsonCheck("real finite valid_until expiry", "real UTC expiry releases all three slots without a fallback plan", map[string]any{"valid_until": u, "observed_at": time.Now().UTC(), "expired": expired, "authorized": authorized, "pending": pending}, "poll production entitlement and device projection; no clock or row mutation", expired && authorized == 0 && pending == 3)
			var expiryAudit []byte
			auditErr := l.pool.QueryRow(l.ctx, `SELECT coalesce(jsonb_agg(jsonb_build_object('action',a.action,'resource',a.resource,'result',a.result,'occurred_at',a.occurred_at,'after',a.after_diff)),'[]'::jsonb) FROM audit_events a JOIN member_devices d ON a.resource='member_device:'||d.id::text WHERE d.member_id=$1 AND a.action='device.revoke' AND a.occurred_at>=$2 AND a.after_diff->>'status'='pending'`, l.a.member, u).Scan(&expiryAudit)
			auditCount := l.count(`SELECT count(*) FROM audit_events a JOIN member_devices d ON a.resource='member_device:'||d.id::text WHERE d.member_id=$1 AND a.action='device.revoke' AND a.occurred_at>=$2 AND a.after_diff->>'status'='pending'`, l.a.member, u)
			r.check("real finite expiry audited withdrawal", "three authorized-to-pending audit events", string(expiryAudit), "read expiry device.revoke audit rows; no synthetic audit input", auditErr == nil && auditCount == 3)
			l.finishPersistent(r, flow, u, l.memberHeartbeat(r))
		},
		grantRestore: func() grantHTTPResponse {
			return l.grant(l.a, time.Now().UTC().Add(time.Hour)).grantResp()
		},
		waitDevices: func() (bool, int, int) {
			var authorized, pending int
			restored := l.wait(60*time.Second, func() bool {
				authorized = l.count(`SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='authorized'`, l.a.member)
				pending = l.count(`SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='pending'`, l.a.member)
				return authorized == 3 && pending == 0
			})
			return restored, authorized, pending
		},
		waitOperation: func(opID uuid.UUID) (bool, grantHTTPResponse) {
			var operation response
			completed := l.wait(60*time.Second, func() bool {
				operation = l.api(l.admin, "GET", "/api/v1/admin/operations/"+opID.String(), nil, nil)
				return operation.status == 200 && completedOperation(operation.body)
			})
			return completed, operation.grantResp()
		},
	})
}

func (l *live) memberHeartbeat(r *record) time.Duration {
	raw, err := os.ReadFile(filepath.Join(l.state, "node/etc/agent-run.json"))
	var cfg struct {
		Seconds int `json:"heartbeat_seconds"`
	}
	decoded := json.Unmarshal(raw, &cfg) == nil
	r.jsonCheck("configured heartbeat bound", "positive prepared heartbeat interval", map[string]any{"heartbeat_seconds": cfg.Seconds}, "read prepared agent-run.json; emit interval only", err == nil && decoded && cfg.Seconds > 0)
	return time.Duration(cfg.Seconds) * time.Second
}
