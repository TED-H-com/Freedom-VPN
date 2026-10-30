//go:build e2e && integration

package secops

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

// §16.3 / §19 AT29: mutate only the D-065 command, with unconditional restoration.
func (p *liveProbe) fleetMaximum(seconds int) map[string]any {
	admin := p.login("e2e-admin", "admin")
	path := "/api/v1/admin/fleet-config"
	before := p.call(admin, "GET", path, nil, nil)
	key := uuid.NewString()
	body := map[string]any{"max_offline_lease_seconds": seconds, "reason": "TEST ONLY AT29 finite lease; restore default on exit"}
	changed := p.call(admin, "PUT", path, body, map[string]string{"If-Match": before.header.Get("ETag"), "Idempotency-Key": key})
	replay := p.call(admin, "PUT", path, body, map[string]string{"If-Match": before.header.Get("ETag"), "Idempotency-Key": key})
	after := p.call(admin, "GET", path, nil, nil)
	var value map[string]any
	_ = json.Unmarshal(after.body, &value)
	return map[string]any{"utc": time.Now().UTC(), "requested_seconds": seconds, "get_status": before.status, "etag_before": before.header.Get("ETag"), "put_status": changed.status, "code": changed.code, "etag_after": changed.header.Get("ETag"), "replay_status": replay.status, "replay_equal": bytes.Equal(changed.body, replay.body), "readback": value}
}
func (p *liveProbe) deadlineMatrix(node uuid.UUID, stack *labstack.Stack) {
	out := map[string]any{"status": "partial", "wall_clock_rollback": map[string]any{"status": "needs operator run", "owner": "operator / w3-devstack-e2e", "missing_export": "disposable-guest wall-clock rollback; container shares the host clock"}, "transport_scope": map[string]any{"status": "blocked", "owner": "w3-devstack-e2e", "missing_export": "persistent established TCP and authenticated member UDP echo traffic; probe-only UDP/4443 is not a member transport"}}
	p.metadata["deadline_matrix"] = out
	out["fleet_set_60"] = p.fleetMaximum(60)
	defer func() { out["fleet_restore_86400"] = p.fleetMaximum(86400) }()
	raw, _ := os.ReadFile(filepath.Join(p.state, "config/members.json"))
	var cfg struct {
		Grace  int    `json:"grace_seconds"`
		Reason string `json:"grace_reason"`
	}
	_ = json.Unmarshal(raw, &cfg)
	out["prepared_grace"] = map[string]any{"seconds": cfg.Grace, "reason_present": cfg.Reason != "", "prepare_only": true}
	// The fake provider must express past_due before a finite grace can project.
	grace := p.login("secops-deadline-grace", "member")
	graceEnd := time.Now().UTC().Add(3 * time.Minute).Truncate(time.Second)
	if p.paidInput(grace, graceEnd) {
		out["grace"] = p.graceInput(grace, graceEnd, cfg.Grace)
	} else {
		out["grace"] = map[string]any{"status": "blocked", "reason": "paid input unavailable"}
	}
	for _, kind := range []string{"paid", "grant", "lease"} {
		admin := p.login("e2e-admin", "admin")
		subject := "secops-deadline-" + kind
		m := p.login(subject, "member")
		serviceEnd := time.Now().UTC().Add(160 * time.Second).Truncate(time.Second)
		if kind == "lease" {
			serviceEnd = time.Now().UTC().Add(20 * time.Minute).Truncate(time.Second)
		}
		row := map[string]any{"status": "blocked", "service_kind": kind, "expected_service_end": serviceEnd, "watchdog_paused": kind != "paid"}
		out[kind] = row
		if kind == "paid" {
			if !p.paidInput(m, serviceEnd) {
				row["blocker"] = "checkout/provider projection did not complete"
				continue
			}
		} else {
			p.check("finite grant", p.call(admin, "POST", "/api/v1/admin/members/"+m.member+"/grants", api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, Reason: "TEST ONLY AT29 finite deadline", PlanRevisionId: &p.planRevision, ValidUntil: serviceEnd}, nil), 202, "")
		}
		peer, ok := p.joinFlowPeer(stack, m, subject, "secops-deadline-"+kind)
		if !ok {
			row["blocker"] = "native admission unavailable"
			continue
		}
		var positive map[string]any
		for until := time.Now().Add(45 * time.Second); time.Now().Before(until); {
			positive = p.selectedFlow(peer)
			if positive["counted"] == true {
				break
			}
			time.Sleep(2 * time.Second)
		}
		row["before_fault"] = positive
		if positive["counted"] != true {
			row["blocker"] = "no counted flow"
			continue
		}
		if kind != "lease" {
			for time.Until(serviceEnd) > 35*time.Second {
				time.Sleep(time.Second)
			}
		}
		var lease []byte
		var issued, expires time.Time
		err := p.db.QueryRow(p.t.Context(), `SELECT issued_at,expires_at,jsonb_build_object('lease_id',id,'issued_at',issued_at,'expires_at',expires_at,'generation',policy_generation::text,'members',envelope->'signed'->'members') FROM serving_leases WHERE node_id=$1 ORDER BY issued_at DESC LIMIT 1`, node).Scan(&issued, &expires, &lease)
		row["lease_read_ok"] = err == nil
		if err != nil {
			row["blocker"] = "no signed lease"
			continue
		}
		row["signed_lease_bounds"] = json.RawMessage(lease)
		row["signed_node_duration_seconds"] = expires.Sub(issued).Seconds()
		if expires.Sub(issued) > 60*time.Second {
			row["blocker"] = "agent still uses a lease issued under former maximum"
			continue
		}
		end := serviceEnd
		if expires.Before(end) {
			end = expires
		}
		row["earliest_deadline"] = end
		row["before_heartbeat"] = p.heartbeatView()
		row["before_kernel_timeout"] = p.timeoutView()
		stopped := time.Now()
		row["control_stop_utc"] = stopped.UTC()
		// Pause the worker too: provider/peer removal must not confound kernel expiry.
		pause := exec.CommandContext(p.t.Context(), "docker", "pause", livePrefix+"worker").Run()
		row["worker_pause_ok"] = pause == nil
		func() {
			defer func() {
				_ = exec.CommandContext(p.t.Context(), "docker", "start", livePrefix+"control").Run()
				_, _ = p.docker("node", "pkill", "-CONT", "-f", "^fvpn-watchdog run")
				_ = stack.PauseAgent(p.t.Context(), false)
				if pause == nil {
					_ = exec.CommandContext(p.t.Context(), "docker", "unpause", livePrefix+"worker").Run()
				}
			}()
			if pause != nil {
				row["blocker"] = "cannot isolate worker reconciliation"
				return
			}
			e := exec.CommandContext(p.t.Context(), "docker", "stop", "--time", "2", livePrefix+"control").Run()
			row["control_stop_ok"] = e == nil
			if e != nil {
				return
			}
			time.Sleep(3 * time.Second)
			row["control_lost_agent_alive"] = p.rawFlow(peer)
			if kind != "paid" {
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
			late, checked := false, false
			for time.Now().Before(end.Add(8 * time.Second)) {
				tick := time.Now()
				sample := p.rawFlow(peer)
				sample["monotonic_elapsed_seconds"] = time.Since(stopped).Seconds()
				sample["request_start_after_deadline"] = !tick.Before(end)
				if !tick.Before(end) {
					checked = true
					if sample["http_200"] == true {
						late = true
					}
				}
				samples = append(samples, sample)
				time.Sleep(time.Second)
			}
			row["samples"] = samples
			row["after_kernel_timeout"] = p.timeoutView()
			row["after_heartbeat"] = p.heartbeatView()
			row["success_after_deadline"] = late
			row["deadline_sampled"] = checked
			if checked && !late {
				row["status"] = "passed repeated-HTTPS cutoff subset"
			}
			if late {
				row["status"] = "failed"
				p.t.Error("forwarding survived earliest deadline")
			}
		}()
		time.Sleep(15 * time.Second)
		row["after_control_restored"] = p.selectedFlow(peer)
		if kind != "lease" {
			admin = p.login("e2e-admin", "admin")
			r := p.call(admin, "POST", "/api/v1/admin/members/"+m.member+"/grants", api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, Reason: "TEST ONLY independent recovery authorization", PlanRevisionId: &p.planRevision, ValidUntil: time.Now().UTC().Add(20 * time.Minute)}, nil)
			row["recovery_grant_status"] = r.status
		}
		var recovery map[string]any
		for until := time.Now().Add(90 * time.Second); time.Now().Before(until); {
			recovery = p.selectedFlow(peer)
			if recovery["counted"] == true {
				break
			}
			time.Sleep(3 * time.Second)
		}
		row["recovery_flow"] = recovery
		row["recovery_ack"] = p.publicationObservation(node)
	}
}

// This provider hook changes source truth, never subscriptions or entitlements.
func (p *liveProbe) graceInput(m *liveSession, end time.Time, seconds int) map[string]any {
	row := map[string]any{"status": "blocked", "original_paid_end": end, "configured_grace_seconds": seconds, "owner": "w3-devstack-e2e", "missing_export": "fake provider /e2e/snapshot past_due subscription state preserving the original paid invoice"}
	body, ok := p.metadata["private_paid_snapshot_"+m.member].(map[string]any)
	if !ok {
		return row
	}
	body["status"] = "past_due"
	raw, _ := json.Marshal(body)
	request, _ := http.NewRequestWithContext(p.t.Context(), "POST", "https://api.stripe.com/e2e/snapshot", bytes.NewReader(raw))
	request.Header.Set("X-E2E-Secret", p.secret("payment.secret"))
	resp, e := p.client().Do(request)
	if e == nil {
		row["snapshot_http_status"] = resp.StatusCode
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
	row["expected_snapshot_status"] = 200
	var source []byte
	e = p.db.QueryRow(p.t.Context(), `SELECT jsonb_build_object('paid_through',s.paid_through,'grace_from',s.grace_from,'grace_until',s.grace_until,'effective_service_end',e.effective_service_end) FROM subscriptions s JOIN entitlements e ON e.member_id=s.member_id WHERE s.member_id=$1`, m.member).Scan(&source)
	if e == nil {
		row["source_projection"] = json.RawMessage(source)
	}
	return row
}
func (p *liveProbe) rawFlow(peer flowPeer) map[string]any {
	origin, _ := p.originAddressAt(peer.NodeService)
	before, ok := p.deviceBytesAt(peer.NodeService, peer.Device)
	start := time.Now()
	raw, e := p.docker(peer.Service, "wget", "--server-response", "--timeout=2", "--tries=1", "--ca-certificate=/etc/vpn-node/ca.pem", "-O", "/dev/null", "https://"+origin+"/healthz")
	after, ok2 := p.deviceBytesAt(peer.NodeService, peer.Device)
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
	snapshot := map[string]any{"checkout_id": external, "period_start": time.Now().UTC().Add(-time.Minute).Unix(), "period_end": end.Unix(), "amount_minor": 1000, "currency": "USD", "status": "active", "paid": true, "cancel_at_period_end": false, "refund_minor": 0}
	p.metadata["private_paid_snapshot_"+m.member] = snapshot
	_, ok := hook("snapshot", snapshot)
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
