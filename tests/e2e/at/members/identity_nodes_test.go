//go:build e2e

package members

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TEST ONLY catalogue data; entitlement and publication changes go through HTTP.
// This is the documented devstack fixture seam, not a device/identity bypass.
func (l *live) seedPlan() {
	l.plan, l.planRev, l.groupA, l.groupB = uuid.New(), uuid.New(), uuid.New(), uuid.New()
	tx, err := l.pool.Begin(l.ctx)
	if err != nil {
		l.t.Fatal("plan transaction")
	}
	defer func() { _ = tx.Rollback(l.ctx) }()
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO member_groups(id,code,name) VALUES($1,$2,$3),($4,$5,$6)`, []any{l.groupA, "at-A-" + l.run, "TEST ONLY A " + l.run, l.groupB, "at-B-" + l.run, "TEST ONLY B " + l.run}},
		{`INSERT INTO plans(id,code) VALUES($1,$2)`, []any{l.plan, "at-plan-" + l.run}},
		{`INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,provider,provider_price_ref,schema_version,payload) VALUES($1,$2,1,3,$3,'stripe','e2e-price',1,'{"display_name":{"zh-TW":"TEST ONLY"}}')`, []any{l.planRev, l.plan, []uuid.UUID{l.groupA}}},
		{`UPDATE plans SET active_revision_id=$2,resource_version=resource_version+1 WHERE id=$1`, []any{l.plan, l.planRev}},
	} {
		if _, err = tx.Exec(l.ctx, q.sql, q.args...); err != nil {
			l.t.Fatal("TEST ONLY plan fixture rejected")
		}
	}
	if tx.Commit(l.ctx) != nil {
		l.t.Fatal("plan commit")
	}
}
func (l *live) grant(b *browser, until time.Time) response {
	return l.api(l.admin, "POST", "/api/v1/admin/members/"+b.member+"/grants", api.GrantCreateRequest{PlanRevisionId: &l.planRev, Source: api.GrantSourceAdminGrant, Reason: "TEST ONLY acceptance finite grant", ValidUntil: until}, commandHeaders())
}
func (l *live) patchMember(b *browser, body any) response {
	path := "/api/v1/admin/members/" + b.member
	old := l.api(l.admin, "GET", path, nil, nil)
	h := commandHeaders()
	h["If-Match"] = old.headers.Get("ETag")
	return l.api(l.admin, "PATCH", path, body, h)
}
func (l *live) policy(kind string, payload any) (response, uuid.UUID) {
	r := l.api(l.admin, "POST", "/api/v1/admin/policies", map[string]any{"type": kind, "schema_version": 1, "code": "at-" + l.run + "-" + uuid.NewString()[:8], "reason": "TEST ONLY acceptance", "payload": payload}, commandHeaders())
	var v struct {
		RevisionID uuid.UUID `json:"revision_id"`
	}
	_ = json.Unmarshal(r.body, &v)
	return r, v.RevisionID
}
func (l *live) activate(rev uuid.UUID) response {
	var id uuid.UUID
	var version string
	err := l.pool.QueryRow(l.ctx, `SELECT d.id,d.resource_version::text FROM policy_definitions d JOIN policy_revisions r ON r.definition_id=d.id WHERE r.id=$1`, rev).Scan(&id, &version)
	if err != nil {
		return response{code: "POLICY_REVISION_UNAVAILABLE"}
	}
	h := commandHeaders()
	h["If-Match"] = `"` + version + `"`
	return l.api(l.admin, "POST", "/api/v1/admin/policies/"+id.String()+"/activate", map[string]any{"revision_id": rev, "reason": "TEST ONLY acceptance"}, h)
}
func (l *live) enrollment(r *record) {
	// §6.4, AT05: this lab has only the installer node runtime. Check both
	// delivery token contracts without claiming two unused reservations booted.
	for _, delivery := range []api.EnrollmentDelivery{api.EnrollmentDeliveryInstaller, api.EnrollmentDeliveryPreinstalled} {
		name := "TEST ONLY " + string(delivery)
		body := api.EnrollmentCreateRequest{Delivery: delivery, Name: &name}
		h := commandHeaders()
		got := l.api(l.admin, "POST", "/api/v1/admin/enrollments", body, h)
		var created api.EnrollmentCreated
		err := json.Unmarshal(got.body, &created)
		token, _ := created.Token.Get()
		ok := got.status == 201 && err == nil && token != "" && created.SecretAvailable
		r.check(string(delivery)+" reservation", "201, one-time secret, no-store", fmt.Sprintf("%s; secret_present=%t; no_store=%t", detail(got), token != "", strings.Contains(got.headers.Get("Cache-Control"), "no-store")), "POST /admin/enrollments", ok && strings.Contains(got.headers.Get("Cache-Control"), "no-store"))
		again := l.api(l.admin, "POST", "/api/v1/admin/enrollments", body, h)
		var replay api.EnrollmentCreated
		_ = json.Unmarshal(again.body, &replay)
		r.check(string(delivery)+" idempotent replay", "same node, secret null", fmt.Sprintf("%s; same_node=%t; secret_available=%t", detail(again), replay.Enrollment.NodeId == created.Enrollment.NodeId, replay.SecretAvailable), "repeat POST with same Idempotency-Key", again.status == 201 && replay.Enrollment.NodeId == created.Enrollment.NodeId && !replay.SecretAvailable && replay.Token.IsNull())
		if delivery == api.EnrollmentDeliveryInstaller && ok {
			l.node = created.Enrollment.NodeId
			path := filepath.Join(l.state, "node/var/e2e/enrollment.token")
			if os.WriteFile(path, []byte(token+"\n"), 0600) != nil {
				r.block("installer token could not reach documented node entrypoint")
				continue
			}
			ready := l.wait(100*time.Second, func() bool {
				return l.pool.QueryRow(l.ctx, `SELECT current_incarnation_id FROM nodes WHERE id=$1`, l.node).Scan(&l.inc) == nil && l.inc != uuid.Nil
			})
			r.check("installer enrollment", "node-owned mTLS incarnation appears", fmt.Sprintf("incarnation_present=%t", ready), "node entrypoint: fvpn-agent enroll; SELECT current_incarnation_id", ready)
		}
	}
	if l.node != uuid.Nil {
		var intent string
		_ = l.pool.QueryRow(l.ctx, `SELECT publication_intent FROM exits WHERE node_id=$1`, l.node).Scan(&intent)
		r.check("auto-report is unpublished", "unpublished", intent, "SELECT publication_intent FROM nodes", intent == "unpublished")
		p, id := l.policy("release_support", map[string]any{"approved_release_ids": []string{}, "minimum_serving_version": map[string]any{"serving_release_ids": []string{}}, "revoked_release_ids": []string{}})
		r.check("empty release allowlist rejected", "422; at least one serving release required by contract", detail(p), "POST /admin/policies release_support (empty negative control)", p.status == 422)
		if id != uuid.Nil {
			a := l.activate(id)
			r.check("release policy activation", "202 intent recorded", detail(a), "POST /admin/policies/{id}/activate", a.status == 202)
		}
	}
	out, err := l.docker("inspect", "--format", "{{.Config.Image}}", l.name("node"))
	r.check("runtime inventory", "one actual node image", fmt.Sprintf("node_image_inspected=%t; node_containers=1", err == nil && len(out) > 0), "docker inspect fvpn-w4-members-node", err == nil)
	r.block("Running lab has one installer node runtime; preinstalled/cloud-init boot flavours and their package provenance are absent. A preinstalled reservation is not a boot test. No active release-support policy exists, and an empty serving allowlist is correctly rejected by the API. The shared runner currently inserts a non-contract empty policy directly; this suite does not bypass policy validation.")
}
func (l *live) identity(r *record) {
	again := l.login("AT-member-A-"+l.run, "member")
	r.check("same-client stable identity", "same member UUID across browser sessions", fmt.Sprintf("same_member=%t; distinct_members=%t", again.member == l.a.member, l.a.member != l.b.member), "IdP assume; /auth/login?role=member; GET /auth/session", again.member == l.a.member && l.a.member != l.b.member)
	g := l.grant(l.a, time.Now().UTC().Add(time.Hour))
	r.check("finite entitlement command", "202 accepted, not applied", detail(g), "POST /admin/members/{A}/grants", g.status == 202)
	me := l.api(l.a, "GET", "/api/v1/member/me", nil, nil)
	r.check("member self read", "200", detail(me), "GET /member/me", me.status == 200)
	// Probe the actual second-client surface, not just source configuration.
	other := l.request(nil, "GET", "https://idp.example.invalid/authorize?client_id=netbird-native&redirect_uri=https%3A%2F%2Fnetbird.example.invalid%2Fcallback&response_type=code&scope=openid&state=acceptance", nil, nil)
	r.check("second OAuth client precondition", "record actual response", detail(other), "GET test IdP /authorize with native client_id", other.status != 0)
	if other.status != 302 {
		r.block("Test IdP rejects the native OAuth client (" + detail(other) + "); only Portal login is available. Two-client stable-subject mapping and a native user peer cannot be proved.")
	}
	logout := l.api(again, "POST", "/auth/logout", nil, nil)
	session := l.api(again, "GET", "/auth/session", nil, nil)
	var s struct {
		Authenticated bool `json:"authenticated"`
	}
	_ = json.Unmarshal(session.body, &s)
	r.check("browser sign-out", "session unauthenticated", fmt.Sprintf("logout=%d; authenticated=%t", logout.status, s.Authenticated), "POST /auth/logout; GET /auth/session", logout.status >= 200 && logout.status < 400 && !s.Authenticated)
	before := l.count(`SELECT count(*) FROM member_devices WHERE member_id=$1`, l.a.member)
	r.check("native device precondition", "observe real member device count", fmt.Sprintf("member_A_devices=%d", before), "SELECT count(*) FROM member_devices WHERE member_id=A", true)
	r.block("No admitted native member device exists in this run; selected-exit forwarding, sign-out data-plane effects, network move and sleep/wake have no successful forwarding baseline. Platform script: OPERATORS.md.")
}
func (l *live) nodeMutation(method, suffix string, body any) response {
	path := "/api/v1/admin/nodes/" + l.node.String()
	old := l.api(l.admin, "GET", path, nil, nil)
	h := commandHeaders()
	h["If-Match"] = old.headers.Get("ETag")
	return l.api(l.admin, method, path+suffix, body, h)
}
func (l *live) publication(r *record) {
	if l.node == uuid.Nil {
		r.block("installer node did not enroll")
		return
	}
	patch := l.nodeMutation("PATCH", "", map[string]any{"name": "TEST ONLY renamed", "region": "TEST", "member_group_ids": []uuid.UUID{l.groupA}})
	r.check("classify and rename", "200 or 202", detail(patch), "PATCH /admin/nodes/{id} with If-Match", patch.status == 200 || patch.status == 202)
	pub := l.nodeMutation("PUT", "/publication", map[string]string{"publication_intent": "published", "reason": "TEST ONLY group A"})
	r.check("publication readiness", "record command result", detail(pub), "PUT /admin/nodes/{id}/publication", pub.status != 0)
	if pub.status != 202 {
		r.block("Publication precondition: " + detail(pub))
	}
	_ = l.wait(30*time.Second, func() bool {
		var observed string
		_ = l.pool.QueryRow(l.ctx, `SELECT publication_observed FROM exits WHERE node_id=$1`, l.node).Scan(&observed)
		return observed == "active"
	})
	var intent, observed string
	_ = l.pool.QueryRow(l.ctx, `SELECT publication_intent,publication_observed FROM exits WHERE node_id=$1`, l.node).Scan(&intent, &observed)
	r.check("desired versus observed", "both independently recorded", fmt.Sprintf("intent=%s; observed=%s", intent, observed), "GET /admin/nodes/{id}; database readback", intent != "")
	chain, chainErr := l.inside("node", "nft", "-j", "list", "table", "inet", "fvpn")
	var rules struct {
		NFTables []json.RawMessage `json:"nftables"`
	}
	_ = json.Unmarshal(chain, &rules)
	accepts := 0
	for _, line := range rules.NFTables {
		var row struct {
			Rule struct {
				Chain string           `json:"chain"`
				Expr  []map[string]any `json:"expr"`
			} `json:"rule"`
		}
		_ = json.Unmarshal(line, &row)
		if row.Rule.Chain == "forward" {
			for _, expr := range row.Rule.Expr {
				if _, ok := expr["accept"]; ok {
					accepts++
				}
			}
		}
	}
	r.check("real gateway rule readback", "record forward-chain accept count", fmt.Sprintf("nft_read_ok=%t; forward_accept_rules=%d", chainErr == nil, accepts), "docker exec node nft -j list table inet fvpn", chainErr == nil)
	routes := l.nb("GET", "/routes", nil)
	var routeList []map[string]any
	_ = json.Unmarshal(routes.body, &routeList)
	pending := l.count(`SELECT count(*) FROM operations WHERE resource_id=$1 AND status='pending'`, l.node)
	r.check("NetBird readback and operations", "actual routes and pending operations", fmt.Sprintf("routes=%d; pending_operations=%d; HTTP=%d", len(routeList), pending, routes.status), "GET NetBird /api/routes; SELECT operations", routes.status == 200)
	if observed != "active" {
		r.block("Publication cannot reach active; observed state remains " + observed + "; no applied forwarding baseline. See operation evidence; publication worker is present on this build.")
	}
	disable := l.nodeMutation("PUT", "/publication", map[string]string{"publication_intent": "disabled", "reason": "TEST ONLY disable persistence"})
	r.check("disable command", "202", detail(disable), "PUT publication=disabled", disable.status == 202)
	_, err := l.docker("restart", l.name("node"))
	r.check("node restart", "restart succeeds", fmt.Sprintf("restart_ok=%t", err == nil), "docker restart fvpn-w4-members-node", err == nil)
	persists := l.wait(20*time.Second, func() bool {
		_ = l.pool.QueryRow(l.ctx, `SELECT publication_intent FROM exits WHERE node_id=$1`, l.node).Scan(&intent)
		return intent == "disabled"
	})
	r.check("disabled persists", "disabled intent after reconnect/restart", intent, "SELECT publication_intent after node restart", persists)
	r.block("Group A/B gateway isolation and applied disable cannot be certified without admitted user peers and a previously serving exit. Restart above checks persisted administrative intent only.")
}
func (l *live) exits(r *record) {
	n := l.count(`SELECT count(*) FROM nodes WHERE current_incarnation_id IS NOT NULL`)
	out, err := l.inside("member", "netbird", "routes", "list")
	r.check("exit inventory", "record active node count and client routes", fmt.Sprintf("enrolled_nodes=%d; routes_command_ok=%t; route_output_bytes=%d", n, err == nil, len(out)), "SELECT enrolled nodes; docker exec member netbird routes list", n >= 1)
	r.block(fmt.Sprintf("Observed %d enrolled node, no admitted member binding and no two-exit forwarding baseline. Manual switch/skip_auto_apply, exit death, relay/direct, TCP/UDP/QUIC and PMTU require the missing lab topology. No bridge-only traffic is counted as exit traffic.", n))
}
