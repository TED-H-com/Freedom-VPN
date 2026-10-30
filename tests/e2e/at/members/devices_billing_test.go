//go:build e2e

package members

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/billing"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/google/uuid"
)

func (l *live) nb(method, path string, body any) response {
	return l.request(nil, method, "https://netbird.example.invalid/api"+path, body, map[string]string{"Authorization": "Token " + l.secret("netbird.token")})
}
func (l *live) devices(r *record) {
	defer l.usageTrace(r)
	// AT02 first establishes what the real NetBird peer represents. A machine
	// peer is a negative control, never rewritten into a verified native user.
	groups := l.nb("GET", "/groups", nil)
	var gs []struct{ ID, Name string }
	_ = json.Unmarshal(groups.body, &gs)
	var staging string
	for _, g := range gs {
		if strings.Contains(g.Name, "staging") {
			staging = g.ID
			break
		}
	}
	if staging == "" {
		created := l.nb("POST", "/groups", map[string]any{"name": "fvpn-at-members-negative-" + l.run, "peers": []string{}})
		var group struct{ ID string }
		_ = json.Unmarshal(created.body, &group)
		staging = group.ID
		if staging == "" {
			r.block("isolated negative-control group unavailable: " + detail(created))
			return
		}
	}
	key := l.nb("POST", "/setup-keys", map[string]any{"name": "TEST ONLY admission negative control", "type": "one-off", "expires_in": 3600, "usage_limit": 1, "auto_groups": []string{staging}, "ephemeral": false, "allow_extra_dns_labels": false})
	var secret struct{ Key string }
	_ = json.Unmarshal(key.body, &secret)
	if key.status != 200 || secret.Key == "" {
		r.block("Machine-peer negative control setup unavailable: " + detail(key))
		return
	}
	if os.WriteFile(filepath.Join(l.state, "member/var/e2e/setup.key"), []byte(secret.Key+"\n"), 0600) != nil {
		r.block("member entrypoint key handoff unavailable")
		return
	}
	host, hostErr := l.docker("inspect", "--format", "{{.Config.Hostname}}", l.name("member"))
	if hostErr != nil {
		r.block("member hostname readback unavailable")
		return
	}
	hostname := strings.TrimSpace(string(host))
	var userPresent bool
	ready := l.wait(90*time.Second, func() bool {
		got := l.nb("GET", "/peers", nil)
		var peers []struct {
			ID, Name, UserID string
			User             string `json:"user_id"`
		}
		_ = json.Unmarshal(got.body, &peers)
		// Match the actual container hostname, never a guessed name or overlay IP.
		for _, p := range peers {
			if p.Name == hostname {
				l.machinePeer = p.ID
				userPresent = p.User != ""
				return true
			}
		}
		return false
	})
	r.check("real peer readback", "connected machine peer has no native user identity", fmt.Sprintf("peer_present=%t; user_id_present=%t", ready, userPresent), "GET NetBird /api/peers after documented member entrypoint", ready && !userPresent)
	if !ready {
		r.block("member container did not join NetBird")
		return
	}
	bindings := devices.NetBirdBindings{Pool: l.pool}
	client, err := netbird.New(netbird.Config{BaseURL: "https://netbird.example.invalid/api", DeploymentID: "e2e", Token: netbird.TokenFile(filepath.Join(l.state, "secrets/netbird.token")), Bindings: bindings, HTTPClient: &http.Client{Transport: l.transport}})
	if err != nil {
		r.block("real NetBird adapter configuration unavailable")
		return
	}
	store, _ := entitlement.NewStore(entitlement.Config{MaxGrantDuration: 24 * time.Hour, DefaultDeviceLimit: 3})
	svc := devices.Service{Pool: l.pool, Entitlements: store, Adapter: &devices.NetBirdAdapter{Pool: l.pool, Client: client, Bindings: bindings, DeploymentID: "e2e", OverlayScope: "e2e-overlay"}}
	start := make(chan struct{})
	out := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			<-start
			_, err := svc.ObserveAndAuthorize(l.ctx, entitlement.Command{Actor: entitlement.Actor{ID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), Role: "system", RequestID: uuid.New()}, Key: uuid.New()}, "e2e", l.machinePeer)
			out <- err
		})
	}
	close(start)
	wg.Wait()
	close(out)
	unmapped := 0
	for err := range out {
		if errors.Is(err, devices.ErrUnmapped) {
			unmapped++
		}
	}
	devicesN := l.count(`SELECT count(*) FROM member_devices`)
	bindingsN := l.count(`SELECT count(*) FROM device_bindings`)
	identitiesN := l.count(`SELECT count(*) FROM netbird_user_bindings`)
	r.check("concurrent negative admission", "both reject unverified machine identity; no fabricated bindings", fmt.Sprintf("ErrUnmapped=%d/2; member_devices=%d; device_bindings=%d; verified_NetBird_mappings=%d", unmapped, devicesN, bindingsN, identitiesN), "two barrier-released goroutines: devices.Service.ObserveAndAuthorize with real NetBirdAdapter", unmapped == 2 && devicesN == 0 && bindingsN == 0)
	// Node observation belongs to a separate namespace: it cannot manufacture a
	// member binding for usage.IngestTx (§7.3, §10.4; AT02-TRACE.md).
	nodesN := l.count(`SELECT count(*) FROM node_incarnations WHERE peer_id IS NOT NULL`)
	r.check("node/member binding separation", "observe node peer binding separately", fmt.Sprintf("node_peer_bindings=%d; member_device_bindings=%d", nodesN, bindingsN), "SELECT node_incarnations and device_bindings", true)
	list := l.api(l.a, "GET", "/api/v1/member/devices", nil, nil)
	denied := l.api(l.a, "DELETE", "/api/v1/member/devices/"+uuid.NewString(), nil, commandHeaders())
	leak := strings.Contains(string(denied.body), l.b.member) || strings.Contains(string(list.body), l.b.member)
	r.check("unowned ID denial", "403/404; no other member identifier", fmt.Sprintf("list=%d; delete=%d; member_B_leaked=%t", list.status, denied.status, leak), "GET /member/devices; DELETE unknown device UUID", list.status == 200 && (denied.status == 403 || denied.status == 404) && !leak)
	r.block("Actual member entrypoint creates a setup-key peer with empty user_id; devices.ObserveAndAuthorize returns ErrUnmapped before quota locking. There are zero member_devices/device_bindings. Last-slot winner/loser, cross-member EXISTING-device isolation, revoke-slot reuse and gateway denial need native peer admission. node.peer.observe only fills node_incarnations.peer_id; it is not the member admission path.")
}
func (l *live) webhook(id, kind string, valid bool) response {
	object := map[string]any{"id": "in_TEST_ONLY_" + l.run, "customer": "cus_TEST_ONLY_" + l.run, "parent": map[string]any{"subscription_details": map[string]string{"subscription": "sub_TEST_ONLY_" + l.run}}}
	if kind == "customer.subscription.created" {
		object = map[string]any{"id": "sub_TEST_ONLY_" + l.run, "customer": "cus_TEST_ONLY_" + l.run}
	}
	raw, _ := json.Marshal(map[string]any{"id": id, "type": kind, "created": 1, "api_version": billing.StripeAPIVersion, "livemode": false, "data": map[string]any{"object": object}})
	stamp := fmt.Sprint(time.Now().Unix())
	mac := hmac.New(sha256.New, []byte(l.secret("webhook.secret")))
	_, _ = mac.Write([]byte(stamp + "."))
	_, _ = mac.Write(raw)
	sig := hex.EncodeToString(mac.Sum(nil))
	if !valid {
		sig = strings.Repeat("0", 64)
	}
	return l.api(nil, "POST", "/api/v1/webhooks/stripe", raw, map[string]string{"Stripe-Signature": "t=" + stamp + ",v1=" + sig})
}
func (l *live) payment(r *record) {
	before := l.count(`SELECT count(*) FROM subscriptions WHERE member_id=$1`, l.c.member)
	forged := l.api(l.c, "GET", "/billing/return?success=true&paid=true&session_id=TEST_ONLY_forged", nil, nil)
	after := l.count(`SELECT count(*) FROM subscriptions WHERE member_id=$1`, l.c.member)
	r.check("forged success return", "no subscription created", fmt.Sprintf("return_HTTP=%d; before=%d; after=%d", forged.status, before, after), "GET forged billing return; SELECT member subscriptions", before == 0 && after == 0)
	if forged.status == 404 {
		r.block("The running control origin returns 404 for /billing/return; no rendered Portal return page is served. No-rights side effect is observed, but the frontend-return journey is unavailable.")
	}
	badID := "evt_TEST_ONLY_bad_" + l.run
	bad := l.webhook(badID, "invoice.paid", false)
	badRows := l.count(`SELECT count(*) FROM inbox_events WHERE external_event_id=$1`, badID)
	r.check("invalid signature", "401, no inbox row", fmt.Sprintf("%s; inbox_rows=%d", detail(bad), badRows), "POST exact webhook bytes with invalid Stripe-Signature", bad.status == 401 && badRows == 0)
	var revBefore string
	_ = l.pool.QueryRow(l.ctx, `SELECT revision::text FROM entitlements WHERE member_id=$1`, l.c.member).Scan(&revBefore)
	id := "evt_TEST_ONLY_replay_" + l.run
	accepted := 0
	for range 10 {
		w := l.webhook(id, "invoice.paid", true)
		if w.status == 202 || w.status == 200 {
			accepted++
		}
	}
	rows := l.count(`SELECT count(*) FROM inbox_events WHERE external_event_id=$1`, id)
	r.check("ten deliveries", "one durable inbox key", fmt.Sprintf("accepted=%d/10; inbox_rows=%d", accepted, rows), "POST same paid event ten times; SELECT source/external_event_id uniqueness", accepted == 10 && rows == 1)
	orders := [][3]string{{"created", "paid", "failed"}, {"created", "failed", "paid"}, {"paid", "created", "failed"}, {"paid", "failed", "created"}, {"failed", "created", "paid"}, {"failed", "paid", "created"}}
	kinds := map[string]string{"created": "customer.subscription.created", "paid": "invoice.paid", "failed": "invoice.payment_failed"}
	count := 0
	for i, order := range orders {
		for _, kind := range order {
			w := l.webhook(fmt.Sprintf("evt_TEST_ONLY_order_%s_%d_%s", l.run, i, kind), kinds[kind], true)
			if w.status == 202 || w.status == 200 {
				count++
			}
		}
	}
	r.check("all six delivery orders", "18 authenticated inbox deliveries", fmt.Sprintf("accepted=%d/18", count), "POST created/paid/failed in every permutation", count == 18)
	// The authoritative GET must work before a delivered event can mean paid.
	snapshot := l.request(nil, "GET", "https://api.stripe.com/v1/subscriptions/sub_TEST_ONLY_"+l.run, nil, map[string]string{"Authorization": "Bearer " + l.secret("stripe.key")})
	r.check("authoritative provider precondition", "record snapshot availability", detail(snapshot), "GET controlled payment provider /v1/subscriptions/{id}", snapshot.status != 0)
	var revAfter string
	_ = l.pool.QueryRow(l.ctx, `SELECT revision::text FROM entitlements WHERE member_id=$1`, l.c.member).Scan(&revAfter)
	r.check("pending entitlement revision", "unreconciled deliveries do not extend interval", fmt.Sprintf("before=%s; after=%s", revBefore, revAfter), "SELECT entitlements.revision before/after webhook sequence", revBefore == revAfter)
	state := l.entitlementState(l.c)
	r.check("unreconciled event cannot grant", "pending member has no live entitlement", state, "GET /member/me; SELECT effective entitlement", state == "pending_payment" || state == "expired" || state == "none")
	if snapshot.status == 404 {
		r.block("Controlled payment provider returns 404 for the authoritative subscription snapshot and empty invoice/refund lists. Inbox replay/order is exercised; no paid subscription can reconcile, so one paid interval, revision stability after successful reconciliation, refunds and gateway authorization remain blocked. Add mutable snapshots to fvpn-e2e-payment, not fake production entitlements.")
	}
}
func (l *live) entitlementState(b *browser) string {
	var state string
	err := l.pool.QueryRow(l.ctx, `SELECT CASE WHEN admin_suspended THEN 'suspended' WHEN state='pending_payment' THEN state WHEN effective_service_end<=clock_timestamp() THEN 'expired' ELSE state END FROM entitlements WHERE member_id=$1`, b.member).Scan(&state)
	if err != nil {
		return "none"
	}
	return state
}
func (l *live) entitlements(r *record) {
	expiry := time.Now().UTC().Add(12 * time.Second)
	g := l.grant(l.c, expiry)
	r.check("finite short entitlement", "202; active before real clock expiry", detail(g)+"; state="+l.entitlementState(l.c), "POST finite TEST ONLY grant, 12 seconds", g.status == 202 && l.entitlementState(l.c) == "active")
	expired := l.wait(20*time.Second, func() bool { return l.entitlementState(l.c) == "expired" })
	r.check("real expiry boundary", "effective interval ends without extending", fmt.Sprintf("expired=%t; elapsed_after_end_seconds=%.0f", expired, time.Since(expiry).Seconds()), "wait on real UTC clock; SELECT effective_service_end", expired)
	suspend := l.patchMember(l.a, map[string]any{"admin_suspended": true, "suspension_reason": "TEST ONLY acceptance"})
	r.check("manual suspension", "202", detail(suspend), "PATCH member admin_suspended=true with If-Match", suspend.status == 202)
	grant := l.grant(l.a, time.Now().UTC().Add(2*time.Hour))
	state := l.entitlementState(l.a)
	r.check("grant does not clear suspension", "202; remains suspended", detail(grant)+"; state="+state, "POST second finite grant; read effective entitlement", grant.status == 202 && state == "suspended")
	screen := l.api(l.a, "GET", "/api/v1/member/me", nil, nil)
	r.check("member view model", "200 with admin_suspended=true", fmt.Sprintf("HTTP=%d; suspended_flag=%t", screen.status, strings.Contains(string(screen.body), `"admin_suspended":true`)), "GET /member/me after suspend", screen.status == 200 && strings.Contains(string(screen.body), `"admin_suspended":true`))
	r.block("No successful paid snapshot (AT03) or admitted device/serving data path (AT02/AT08). A grant surviving suspension is not proof of successful-payment precedence. Existing TCP/UDP stop latency, cancellation/refund timing, every-node enforcement and rendered member screen need those prerequisites.")
}
