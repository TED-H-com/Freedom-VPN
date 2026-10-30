//go:build e2e && integration

package secops

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

func (p *liveProbe) deadlineMatrix(node uuid.UUID, stack *labstack.Stack) {
	out := map[string]any{"status": "partial", "grace": map[string]any{"status": "blocked", "owner": "w3-devstack-e2e", "reason": "prepared config/members.json grace_seconds=0; no documented finite-grace configuration hook; no projected rows or lab config rewritten"}, "short_node_lease": map[string]any{"status": "blocked", "reason": "fleet max_offline_lease_seconds has no public mutation API or exported lab hook; no fleet SQL fixture"}, "wall_clock_rollback": map[string]any{"status": "needs operator run", "reason": "disposable guest VM required; container shares host wall clock"}, "transport_scope": "repeated HTTPS connections; established TCP and UDP echo require an additional controlled lab transport hook"}
	p.metadata["deadline_matrix"] = out
	for _, kind := range []string{"paid", "grant"} {
		admin := p.login("e2e-admin", "admin")
		subject := "secops-deadline-" + kind
		m := p.login(subject, "member")
		end := time.Now().UTC().Add(180 * time.Second).Truncate(time.Second)
		row := map[string]any{"status": "blocked", "service_kind": kind, "expected_service_end": end, "watchdog_paused": kind == "grant"}
		out[kind] = row
		if kind == "paid" {
			if !p.paidInput(m, end) {
				row["blocker"] = "checkout/provider projection did not complete"
				continue
			}
		} else {
			p.check("short finite grant", p.call(admin, "POST", "/api/v1/admin/members/"+m.member+"/grants", api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, Reason: "TEST ONLY AT29 finite service deadline", PlanRevisionId: &p.planRevision, ValidUntil: end}, nil), 202, "")
		}
		peer, ok := p.joinFlowPeer(stack, m, subject, "secops-deadline-"+kind)
		if !ok {
			row["blocker"] = "finite member native admission unavailable"
			continue
		}
		var positive map[string]any
		for deadline := time.Now().Add(50 * time.Second); time.Now().Before(deadline); {
			positive = p.selectedFlow(peer)
			if positive["counted"] == true {
				break
			}
			time.Sleep(3 * time.Second)
		}
		row["before_fault"] = positive
		if positive["counted"] != true {
			row["blocker"] = "short-service member has no counted flow"
			continue
		}
		// Capture immutable server-issued lease bounds, not a fabricated short lease.
		var lease []byte
		err := p.db.QueryRow(p.t.Context(), `SELECT jsonb_build_object('lease_id',s.id,'issued_at',s.issued_at,'expires_at',s.expires_at,'generation',s.policy_generation::text,'members',s.envelope->'signed'->'members') FROM serving_leases s WHERE s.node_id=$1 ORDER BY s.issued_at DESC LIMIT 1`, node).Scan(&lease)
		row["lease_read_ok"] = err == nil
		row["signed_lease_bounds"] = json.RawMessage(lease)
		// Keep the last healthy policy current until <45 seconds remain. All waits
		// are local monotonic durations; host wall time is never modified.
		for time.Until(end) > 45*time.Second {
			time.Sleep(time.Second)
		}
		row["before_heartbeat"] = p.heartbeatView()
		row["before_kernel_timeout"] = p.timeoutView()
		stopped := time.Now()
		row["control_stop_utc"] = stopped.UTC()
		e := exec.CommandContext(p.t.Context(), "docker", "stop", "--time", "2", livePrefix+"control").Run()
		row["control_stop_ok"] = e == nil
		func() {
			defer func() {
				_ = exec.CommandContext(p.t.Context(), "docker", "start", livePrefix+"control").Run()
				if kind == "grant" {
					_, _ = p.docker("node", "pkill", "-CONT", "-f", "^fvpn-watchdog run")
				}
				_ = stack.PauseAgent(p.t.Context(), false)
			}()
			if e != nil {
				return
			}
			time.Sleep(5 * time.Second)
			row["control_lost_agent_alive"] = p.rawFlow(peer)
			if kind == "grant" {
				_, e = p.docker("node", "pkill", "-STOP", "-f", "^fvpn-watchdog run")
				row["watchdog_stop_ok"] = e == nil
				if e != nil {
					return
				}
			}
			if stack.PauseAgent(p.t.Context(), true) != nil {
				return
			}
			_, e = p.docker("node", "sh", "-c", `if [ -f /var/lib/vpn-node/e2e/agent.pid ]; then kill -KILL "$(cat /var/lib/vpn-node/e2e/agent.pid)"; fi`)
			row["agent_kill_command_ok"] = e == nil
			samples := []map[string]any{}
			afterDeadlineSuccess := false
			for time.Now().Before(end.Add(8 * time.Second)) {
				start := time.Now()
				sample := p.rawFlow(peer)
				sample["monotonic_elapsed_seconds"] = time.Since(stopped).Seconds()
				sample["request_start_after_service_end"] = !start.Before(end)
				if !start.Before(end) && sample["http_200"] == true {
					afterDeadlineSuccess = true
				}
				samples = append(samples, sample)
				time.Sleep(time.Second)
			}
			row["samples"] = samples
			row["after_kernel_timeout"] = p.timeoutView()
			row["after_heartbeat"] = p.heartbeatView()
			row["success_after_service_end"] = afterDeadlineSuccess
			row["status"] = "passed repeated-HTTPS subset"
			if afterDeadlineSuccess {
				row["status"] = "failed"
				p.t.Error("member forwarding survived finite deadline")
			}
		}()
		time.Sleep(15 * time.Second)
		row["after_control_restored_expired"] = p.selectedFlow(peer)
		admin = p.login("e2e-admin", "admin")
		response := p.call(admin, "POST", "/api/v1/admin/members/"+m.member+"/grants", api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, Reason: "TEST ONLY independently authorized recovery", PlanRevisionId: &p.planRevision, ValidUntil: time.Now().UTC().Add(20 * time.Minute)}, nil)
		row["recovery_grant_status"] = response.status
		var recovery map[string]any
		for deadline := time.Now().Add(90 * time.Second); time.Now().Before(deadline); {
			recovery = p.selectedFlow(peer)
			if recovery["counted"] == true {
				break
			}
			time.Sleep(3 * time.Second)
		}
		row["recovery_flow"] = recovery
		row["recovery_ack"] = p.publicationObservation(node)
		var raw []byte
		e = p.db.QueryRow(p.t.Context(), `SELECT jsonb_build_object('effective_service_end',e.effective_service_end,'paid_through',(SELECT paid_through FROM subscriptions WHERE member_id=e.member_id ORDER BY created_at LIMIT 1),'grants',(SELECT jsonb_agg(jsonb_build_object('valid_until',valid_until,'revoked_at',revoked_at)) FROM entitlement_grants WHERE member_id=e.member_id)) FROM entitlements e WHERE e.member_id=$1`, m.member).Scan(&raw)
		row["recovery_source_read_ok"] = e == nil
		if e == nil {
			row["recovery_source_intervals"] = json.RawMessage(raw)
		}
	}
}
func (p *liveProbe) rawFlow(peer flowPeer) map[string]any {
	origin, _ := p.originAddress()
	before, ok := p.deviceBytes(peer.Device)
	start := time.Now()
	raw, e := p.docker(peer.Service, "wget", "--server-response", "--timeout=2", "--tries=1", "--ca-certificate=/etc/vpn-node/ca.pem", "-O", "/dev/null", "https://"+origin+"/healthz")
	after, ok2 := p.deviceBytes(peer.Device)
	return map[string]any{"utc": start.UTC(), "elapsed_seconds": time.Since(start).Seconds(), "http_200": e == nil && bytes.Contains(raw, []byte("200 OK")), "counter_reads_ok": ok && ok2, "before_bytes": before, "after_bytes": after}
}
func (p *liveProbe) heartbeatView() map[string]string {
	raw, _ := p.docker("node", "cat", "/run/vpn-node/heartbeat")
	out := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && f[0] != "boot_id" {
			out[f[0]] = f[1]
		}
	}
	return out
}
func (p *liveProbe) timeoutView() json.RawMessage {
	raw, e := p.docker("node", "nft", "-j", "list", "set", "inet", "fvpn", "allowed_v4")
	if e != nil {
		return json.RawMessage(`{"available":false}`)
	}
	// Overlay addresses are synthetic RFC6598 lab peers. Kernel timeout/expiry
	// values remain intact; this captures only the product-owned member allowset.
	if !json.Valid(raw) {
		return json.RawMessage(`{"available":false}`)
	}
	return raw
}
func (p *liveProbe) paidInput(m *liveSession, end time.Time) bool {
	var plan uuid.UUID
	if p.db.QueryRow(p.t.Context(), `SELECT plan_id FROM plan_revisions WHERE id=$1`, p.planRevision).Scan(&plan) != nil {
		return false
	}
	r := p.call(m, "POST", "/api/v1/member/checkout-sessions", api.CheckoutSessionRequest{PlanId: plan, ReturnUrl: livePublic + "/billing/return"}, nil)
	var checkout api.CheckoutSession
	if r.status != 201 || json.Unmarshal(r.body, &checkout) != nil {
		return false
	}
	var external string
	if p.db.QueryRow(p.t.Context(), `SELECT external_session_id FROM billing_sessions WHERE id=$1`, checkout.CheckoutSessionId).Scan(&external) != nil {
		return false
	}
	hook := func(path string, body any) ([]byte, bool) {
		raw, _ := json.Marshal(body)
		req, e := http.NewRequestWithContext(p.t.Context(), "POST", "https://api.stripe.com/e2e/"+path, bytes.NewReader(raw))
		if e != nil {
			return nil, false
		}
		req.Header.Set("X-E2E-Secret", p.secret("payment.secret"))
		resp, e := p.client().Do(req)
		if e != nil {
			return nil, false
		}
		defer resp.Body.Close()
		out, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return out, resp.StatusCode == 200
	}
	_, ok := hook("snapshot", map[string]any{"checkout_id": external, "period_start": time.Now().UTC().Add(-time.Minute).Unix(), "period_end": end.Unix(), "amount_minor": 1000, "currency": "USD", "status": "active", "paid": true, "cancel_at_period_end": false, "refund_minor": 0})
	if !ok {
		return false
	}
	raw, ok := hook("event", map[string]string{"checkout_id": external, "type": "invoice.paid"})
	if !ok {
		return false
	}
	var event struct{ Body, Signature string }
	if json.Unmarshal(raw, &event) != nil {
		return false
	}
	p.secrets = append(p.secrets, []byte(event.Signature))
	r = p.call(&liveSession{client: p.client()}, "POST", "/api/v1/webhooks/stripe", []byte(event.Body), map[string]string{"Stripe-Signature": event.Signature})
	if r.status != 202 {
		return false
	}
	for deadline := time.Now().Add(60 * time.Second); time.Now().Before(deadline); {
		var actual time.Time
		e := p.db.QueryRow(p.t.Context(), `SELECT effective_service_end FROM entitlements WHERE member_id=$1`, m.member).Scan(&actual)
		if e == nil && actual.Equal(end) {
			return true
		}
		time.Sleep(time.Second)
	}
	return false
}
