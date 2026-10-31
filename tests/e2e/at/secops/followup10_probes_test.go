//go:build e2e && integration

package secops

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

// Read the product's serving observation and the actual whole ruleset. No NAT
// workaround is applied by this harness; the archived f9 capture is the before.
func (p *liveProbe) servingGate10(admin *liveSession, node uuid.UUID) map[string]any {
	p.serving(admin, node)
	raw, err := p.docker("node", "nft", "-j", "list", "ruleset")
	if err != nil {
		p.t.Fatal("SECOPS-20 ruleset read failed")
	}
	rs, err := enforce.ParseRuleset(raw)
	if err != nil {
		p.t.Fatal("SECOPS-20 ruleset parse failed")
	}
	evidence := enforce.LoopbackEvidence(rs)
	if evidence == nil {
		evidence = []enforce.ForeignEvidence{}
	}
	obs := p.publicationObservation(node)
	nat := bytes.Contains(raw, []byte(`"family":"ip","name":"nat"`))
	var listing map[string]any
	if json.Unmarshal(raw, &listing) != nil {
		p.t.Fatal("ruleset JSON")
	}
	// Detect table independently of nft's whitespace/key order.
	for _, item := range listing["nftables"].([]any) {
		if v, ok := item.(map[string]any)["table"].(map[string]any); ok && v["family"] == "ip" && v["name"] == "nat" {
			nat = true
		}
	}
	passed := !nat && obs["member_forwarding_enabled"] == true && obs["forwarding_evidence_satisfied"] == true
	row := map[string]any{"passed": passed, "ip_nat_present": nat, "read_back_sha256": fmt.Sprintf("%x", sha256.Sum256(raw)), "read_back_listing": listing, "loopback_scoped": evidence, "product_observation": obs, "merged_changes": []string{"#240 lab ip nat deletion", "#241 D-079 classifier"}, "causal_observation": "Lab entrypoint deleted ip nat before agent start; no loopback-scoped rules remain to exercise D-079 in this run. Both changes are present; this run's serving transition is explained by #240.", "harness_network_mutations": false}
	p.metadata["SECOPS-20"] = row
	if !passed {
		p.t.Fatal("SECOPS-20 populated serving gate failed")
	}
	return row
}

func (p *liveProbe) generationState10(node uuid.UUID) json.RawMessage {
	var raw []byte
	err := p.db.QueryRow(p.t.Context(), `SELECT jsonb_build_object('desired',desired_config_generation::text,'applied',applied_config_generation::text,'max_stored',(SELECT max(generation)::text FROM desired_states WHERE node_id=$1)) FROM nodes WHERE id=$1`, node).Scan(&raw)
	if err != nil {
		p.t.Fatal("generation snapshot", err)
	}
	return json.RawMessage(raw)
}

func (p *liveProbe) restoreGenerations10(node uuid.UUID, since time.Time, restored json.RawMessage) map[string]any {
	var offers, observations []byte
	if err := p.db.QueryRow(p.t.Context(), `SELECT COALESCE(jsonb_agg(jsonb_build_object('generation',generation::text,'signed_at',signed_at) ORDER BY signed_at,generation),'[]'::jsonb) FROM desired_states WHERE node_id=$1 AND signed_at >= $2`, node, since).Scan(&offers); err != nil {
		p.t.Fatal("generation offers", err)
	}
	if err := p.db.QueryRow(p.t.Context(), `SELECT COALESCE(jsonb_agg(jsonb_build_object('applied',applied_generation::text,'received_at',received_at,'errors',report_payload->'errors') ORDER BY received_at),'[]'::jsonb) FROM node_observations WHERE node_id=$1 AND received_at >= $2`, node, since).Scan(&observations); err != nil {
		p.t.Fatal("generation observations", err)
	}
	raw, err := exec.CommandContext(p.t.Context(), "docker", "logs", "--since", since.Format(time.RFC3339Nano), livePrefix+"node").CombinedOutput()
	if err != nil {
		p.t.Fatal("generation log capture")
	}
	lines := []string{}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.Contains(line, "stale desired-state generation") || strings.Contains(line, "ErrStaleGeneration") {
			lines = append(lines, line)
		}
	}
	return map[string]any{"restored_central": restored, "retained_at_t1": p.metadata["generation_at_t1"], "offers": json.RawMessage(offers), "observations": json.RawMessage(observations), "final": p.generationState10(node), "stale_generation_lines": lines, "stale_generation_log_count": len(lines), "manual_generation_adjustment": false, "log_limit": "node/agent/sync.go silently ignores ErrStaleGeneration; zero log matches alone is not closure proof", "fix": "#245 w3-wire-policies-usage follow-up 5", "window_start": since}
}

type deviceRead10 struct {
	DeviceID    uuid.UUID       `json:"device_id"`
	MemberID    uuid.UUID       `json:"member_id,omitempty"`
	Status      string          `json:"status"`
	Unconfirmed *bool           `json:"authorization_unconfirmed"`
	Since       json.RawMessage `json:"authorization_unconfirmed_since,omitempty"`
}

func (p *liveProbe) deviceRead10(s *liveSession, member string, device uuid.UUID, admin bool) (int, deviceRead10) {
	path := "/api/v1/member/devices"
	if admin {
		path = "/api/v1/admin/devices?member_id=" + member
	}
	r := p.call(s, "GET", path, nil, nil)
	var page struct {
		Items []deviceRead10 `json:"items"`
	}
	if json.Unmarshal(r.body, &page) == nil {
		for _, v := range page.Items {
			if v.DeviceID == device {
				return r.status, v
			}
		}
	}
	return r.status, deviceRead10{}
}
func (p *liveProbe) fenceCount10(device uuid.UUID) int {
	var count int
	if err := p.db.QueryRow(p.t.Context(), `SELECT count(*) FROM device_authorization_fences WHERE device_id=$1`, device).Scan(&count); err != nil {
		p.t.Fatal("fence count", err)
	}
	return count
}
func operationGeneration10(op api.Operation) string {
	raw, _ := json.Marshal(op.DesiredGeneration)
	var generation string
	_ = json.Unmarshal(raw, &generation)
	return generation
}
func confirmed10(d deviceRead10) bool {
	return d.Status == "authorized" && d.Unconfirmed != nil && !*d.Unconfirmed
}
func (p *liveProbe) exerciseAdminRegrant10(m *liveSession, peer flowPeer) map[string]any {
	admin := p.login("e2e-admin", "admin")
	path := "/api/v1/admin/members/" + m.member + "/devices/" + peer.Device.String() + "/authorize"
	mbCode, mb := p.deviceRead10(m, m.member, peer.Device, false)
	adCode, ad := p.deviceRead10(admin, m.member, peer.Device, true)
	if mbCode != 200 || adCode != 200 || mb.Unconfirmed == nil || !*mb.Unconfirmed || ad.Unconfirmed == nil || !*ad.Unconfirmed || len(ad.Since) == 0 || string(ad.Since) == "null" {
		p.t.Fatal("D-077 fenced lists missing flag/timestamp")
	}
	rows := []map[string]any{{"request": "GET member/devices before", "status": mbCode, "fence_count": p.fenceCount10(peer.Device), "device": mb}, {"request": "GET admin/devices before", "status": adCode, "fence_count": p.fenceCount10(peer.Device), "device": ad}}
	wrong := p.call(m, "POST", path, nil, nil)
	p.check("D-077 member session on admin route", wrong, 401, "UNAUTHENTICATED")
	rows = append(rows, map[string]any{"request": "POST authorize with member cookie", "status": wrong.status, "error": wrong.code, "fence_count": p.fenceCount10(peer.Device), "authorization_unconfirmed": true})
	if p.fenceCount10(peer.Device) != 1 {
		p.t.Fatal("member request cleared fence")
	}
	var oldID uuid.UUID
	var oldGen string
	if err := p.db.QueryRow(p.t.Context(), `SELECT id,desired_generation::text FROM operations WHERE resource_id=$1 AND kind='member.device_authorize' ORDER BY created_at DESC LIMIT 1`, peer.Device).Scan(&oldID, &oldGen); err != nil {
		p.t.Fatal("existing authorization operation", err)
	}
	key := uuid.NewString()
	r := p.call(admin, "POST", path, nil, map[string]string{"Idempotency-Key": key})
	p.check("D-077 admin authorize fenced device", r, 202, "")
	var op api.Operation
	if json.Unmarshal(r.body, &op) != nil {
		p.t.Fatal("authorize operation decode")
	}
	fence := p.fenceCount10(peer.Device)
	rows = append(rows, map[string]any{"request": "POST authorize admin CSRF Origin fresh key no body", "status": r.status, "operation": op, "fence_count": fence})
	// Same request key must have no second resource/version/outbox effect.
	var versionBefore, versionAfter string
	var outboxBefore, outboxAfter int
	snapshot := func(version *string, count *int) {
		if err := p.db.QueryRow(p.t.Context(), `SELECT resource_version::text,(SELECT count(*) FROM outbox_events WHERE aggregate_id=$1) FROM member_devices WHERE id=$1`, peer.Device).Scan(version, count); err != nil {
			p.t.Fatal("regrant replay snapshot", err)
		}
	}
	snapshot(&versionBefore, &outboxBefore)
	replay := p.call(admin, "POST", path, nil, map[string]string{"Idempotency-Key": key})
	p.check("D-077 same-key replay", replay, 202, "")
	var replayOp api.Operation
	_ = json.Unmarshal(replay.body, &replayOp)
	snapshot(&versionAfter, &outboxAfter)
	rows = append(rows, map[string]any{"request": "POST authorize same key replay", "status": replay.status, "operation": replayOp, "fence_count": p.fenceCount10(peer.Device)})
	var after deviceRead10
	var afterCode int
	for deadline := time.Now().Add(60 * time.Second); time.Now().Before(deadline); {
		afterCode, after = p.deviceRead10(admin, m.member, peer.Device, true)
		if afterCode == 200 && confirmed10(after) {
			break
		}
		time.Sleep(time.Second)
	}
	memberAfterCode, memberAfter := p.deviceRead10(m, m.member, peer.Device, false)
	rows = append(rows, map[string]any{"request": "GET admin/devices after worker sync", "status": afterCode, "device": after, "fence_count": p.fenceCount10(peer.Device)}, map[string]any{"request": "GET member/devices after worker sync", "status": memberAfterCode, "device": memberAfter, "fence_count": p.fenceCount10(peer.Device)})
	flow := p.waitCountedFlow(peer, 60*time.Second)
	passed := r.status == 202 && fence == 0 && op.OperationId == oldID && string(op.Kind) == "member.device_authorize" && operationGeneration10(op) == oldGen && replayOp.OperationId == op.OperationId && versionBefore == versionAfter && outboxBefore == outboxAfter && afterCode == 200 && confirmed10(after) && memberAfterCode == 200 && confirmed10(memberAfter) && flow["counted"] == true
	row := map[string]any{"passed": passed, "calls": rows, "route": path, "existing_operation_id": oldID, "existing_desired_generation": oldGen, "existing_operation_returned": op.OperationId == oldID, "operation_id": op.OperationId, "operation_kind": op.Kind, "operation_desired_generation": op.DesiredGeneration, "same_key_replay_same_operation": replayOp.OperationId == op.OperationId, "replay_resource_version_unchanged": versionBefore == versionAfter, "replay_outbox_count_unchanged": outboxBefore == outboxAfter, "fence_rows_after_fresh_regrant": fence, "observed_authorized_after_sync": confirmed10(after), "member_devices_api_status": memberAfter.Status, "counted_flow_after_regrant": flow, "confirmation_method": "GET admin/devices after worker sync; no wait for a new operation or operation transition"}
	p.metadata["D-077"] = row
	if !passed {
		p.t.Error("D-077 HTTP regrant or replay evidence failed")
	}
	return row
}
func (p *liveProbe) authorizeRevoked10(m *liveSession, peer flowPeer) map[string]any {
	admin := p.login("e2e-admin", "admin")
	r := p.call(admin, "POST", "/api/v1/admin/members/"+m.member+"/devices/"+peer.Device.String()+"/authorize", nil, nil)
	p.check("D-077 revoked device cannot authorize", r, 409, "STATE_CONFLICT")
	_, d := p.deviceRead10(admin, m.member, peer.Device, true)
	return map[string]any{"request": "POST authorize revoked device", "status": r.status, "error": r.code, "fence_count": p.fenceCount10(peer.Device), "device": d}
}

func (p *liveProbe) recordNativeStream10(stack *labstack.Stack, stream labstack.NativeStreamResult, container string) {
	rows, _ := p.metadata["protected_native_streams"].([]map[string]any)
	post, err := stack.ScanNativeJoinLog(p.t.Context(), container)
	rows = append(rows, map[string]any{"pre_delete": stream, "post_delete_scan_ok": err == nil, "post_delete_present": post.Present})
	p.metadata["protected_native_streams"] = rows
}
func (p *liveProbe) bulkProtectedStreams10(stack *labstack.Stack) {
	admin := p.login("e2e-admin", "admin")
	started := time.Now()
	result, err := stack.ScaleNativeMembers(p.t.Context(), labstack.BrowserSession{Client: admin.client, CSRF: admin.csrf}, p.planRevision, "secops10", 20, 3, func(ctx context.Context) (labstack.BrowserSession, error) {
		return stack.Login(ctx, "e2e-admin", "admin")
	})
	for _, stream := range result.Streams {
		container := strings.TrimSuffix(stream.Stream, "/protected-device-flow")
		p.recordNativeStream10(stack, stream, container)
	}
	p.metadata["bulk_native_streams"] = map[string]any{"counts": result, "error": err != nil, "elapsed_seconds": time.Since(started).Seconds(), "purpose": "D-066 final per-stream attribution only; not AT31 relay or capacity acceptance"}
	if err != nil {
		p.t.Error("SECOPS-16 bulk final stream run incomplete")
	}
}

func (p *liveProbe) providerHook10(path string, body any) (int, []byte) {
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(p.t.Context(), "POST", "https://api.stripe.com/e2e/"+path, bytes.NewReader(raw))
	if err != nil {
		p.t.Fatal("provider hook request")
	}
	req.Header.Set("X-E2E-Secret", p.secret("payment.secret"))
	resp, err := p.client().Do(req)
	if err != nil {
		p.t.Fatal("provider hook transport")
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		p.t.Fatal("provider hook read")
	}
	return resp.StatusCode, out
}
func (p *liveProbe) entitlementRead10(m *liveSession) map[string]any {
	var raw []byte
	err := p.db.QueryRow(p.t.Context(), `SELECT jsonb_build_object('provider_status',s.provider_status,'paid_through',s.paid_through,'grace_from',s.grace_from,'grace_until',s.grace_until,'state',e.state,'effective_service_end',e.effective_service_end,'device_limit',e.device_limit,'paid_invoice_count',(SELECT count(*) FROM billing_records r WHERE r.member_id=s.member_id AND r.record_type='paid_invoice')) FROM subscriptions s JOIN entitlements e ON e.member_id=s.member_id WHERE s.member_id=$1`, m.member).Scan(&raw)
	if err != nil {
		p.t.Fatal("entitlement read", err)
	}
	var row map[string]any
	_ = json.Unmarshal(raw, &row)
	return row
}
func (p *liveProbe) pastDue10(stack *labstack.Stack, node uuid.UUID) {
	m := p.login("secops10-grace", "member")
	end := time.Now().UTC().Add(150 * time.Second).Truncate(time.Second)
	if !p.paidInput(m, end) {
		p.t.Fatal("SECOPS-17 paid setup")
	}
	peer, ok := p.joinFlowPeer(stack, m, "secops10-grace", "secops10-grace")
	if !ok {
		p.t.Fatal("SECOPS-17 native join")
	}
	before := p.waitCountedFlow(peer, 60*time.Second)
	body := p.metadata["private_paid_snapshot_"+m.member].(map[string]any)
	body["status"] = "past_due"
	status, _ := p.providerHook10("snapshot", body)
	eventStatus, raw := p.providerHook10("event", map[string]string{"checkout_id": p.checkoutExternalID(m.member), "type": "customer.subscription.updated"})
	var event struct{ Body, Signature string }
	if json.Unmarshal(raw, &event) != nil {
		p.t.Fatal("past_due event decode")
	}
	p.secrets = append(p.secrets, []byte(event.Signature))
	webhook := p.call(&liveSession{client: p.client()}, "POST", "/api/v1/webhooks/stripe", []byte(event.Body), map[string]string{"Stripe-Signature": event.Signature})
	var grace map[string]any
	for deadline := time.Now().Add(200 * time.Second); time.Now().Before(deadline); {
		grace = p.entitlementRead10(m)
		if grace["state"] == "grace" {
			break
		}
		time.Sleep(time.Second)
	}
	admin := p.login("e2e-admin", "admin")
	me := p.call(m, "GET", "/api/v1/member/me", nil, nil)
	ad := p.call(admin, "GET", "/api/v1/admin/members/"+m.member, nil, nil)
	graceFlow := p.waitCountedFlow(peer, 20*time.Second)
	until := end.Add(90 * time.Second)
	for time.Now().Before(until.Add(2 * time.Second)) {
		time.Sleep(time.Second)
	}
	samples := []map[string]any{}
	for range 6 {
		samples = append(samples, p.rawFlow(peer))
		time.Sleep(5 * time.Second)
	}
	expired := p.entitlementRead10(m)
	meExpired := p.call(m, "GET", "/api/v1/member/me", nil, nil)
	adExpired := p.call(admin, "GET", "/api/v1/admin/members/"+m.member, nil, nil)
	statusReplay, _ := p.providerHook10("snapshot", body)
	// Same past_due truth must not extend grace or restore forwarding on later sweeps.
	time.Sleep(12 * time.Second)
	final := p.entitlementRead10(m)
	finalFlow := p.rawFlow(peer)
	passed := status == 200 && eventStatus == 200 && webhook.status == 202 && before["counted"] == true && grace["state"] == "grace" && graceFlow["counted"] == true && expired["state"] == "expired" && final["state"] == "expired" && statusReplay == 200 && finalFlow["http_200"] == false
	for _, s := range samples {
		passed = passed && s["http_200"] == false
	}
	row := map[string]any{"passed": passed, "configured_grace_seconds": 90, "snapshot_status": status, "event_status": eventStatus, "webhook_status": webhook.status, "paid_end": end, "expected_grace_until": until, "before_counted_flow": before, "grace_projection": grace, "grace_counted_flow": graceFlow, "member_me_grace_status": me.status, "member_me_grace": json.RawMessage(me.body), "admin_member_grace_status": ad.status, "admin_member_grace": json.RawMessage(ad.body), "expired_projection": expired, "member_me_expired_status": meExpired.status, "member_me_expired": json.RawMessage(meExpired.body), "admin_member_expired_status": adExpired.status, "admin_member_expired": json.RawMessage(adExpired.body), "after_deadline_samples": samples, "same_snapshot_replay_status": statusReplay, "after_reconcile_projection": final, "after_reconcile_flow": finalFlow, "node_observation": p.publicationObservation(node), "portal_scope": "Portal backing member/me and admin/members APIs measured; browser rendering not exercised; established transports deferred"}
	p.metadata["SECOPS-17"] = row
	if !passed {
		p.t.Error("SECOPS-17 finite grace evidence failed")
	}
}
