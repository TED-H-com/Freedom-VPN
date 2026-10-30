//go:build e2e

package members

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/members/issuer"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func (l *live) grant(b *browser, until time.Time) response {
	rev := l.planRev
	if b == l.b {
		rev = l.planRevB
	}
	return l.api(l.admin, "POST", "/api/v1/admin/members/"+b.member+"/grants", api.GrantCreateRequest{PlanRevisionId: &rev, Source: api.GrantSourceAdminGrant, Reason: "TEST ONLY acceptance finite grant", ValidUntil: until}, commandHeaders())
}
func (l *live) patchMember(b *browser, body any) response {
	path := "/api/v1/admin/members/" + b.member
	old := l.api(l.admin, "GET", path, nil, nil)
	h := commandHeaders()
	h["If-Match"] = old.headers.Get("ETag")
	return l.api(l.admin, "PATCH", path, body, h)
}
func (l *live) policy(kind string, payload any) (response, uuid.UUID) {
	code := "at-" + l.run + "-" + uuid.NewString()[:8]
	// Revise the one lab DNS profile; the D-056 hook has no per-node profile selector.
	if kind == "dns_profile" {
		_ = l.pool.QueryRow(l.ctx, `SELECT code FROM policy_definitions WHERE type='dns_profile' ORDER BY id LIMIT 1`).Scan(&code)
	}
	r := l.api(l.admin, "POST", "/api/v1/admin/policies", map[string]any{"type": kind, "schema_version": 1, "code": code, "reason": "TEST ONLY acceptance", "payload": payload}, commandHeaders())
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
	if !l.prepare(r) {
		return
	}
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
	r.block("Running lab has one installer node runtime; preinstalled/cloud-init boot flavours and their package provenance are absent. A preinstalled reservation is not a boot test. The signed boot release and nonempty support policy are created through the shared PrepareServingBaseline hook; arm64 is cross-built only.")
}
func (l *live) identity(r *record) {
	again := l.login("AT-member-A-"+l.run, "member")
	r.check("same-client stable identity", "same member UUID across browser sessions", fmt.Sprintf("same_member=%t; distinct_members=%t", again.member == l.a.member, l.a.member != l.b.member), "shared Dex password form; /auth/login?role=member; GET /auth/session", again.member == l.a.member && l.a.member != l.b.member)
	l.publish(r, []uuid.UUID{l.groupA})
	g := l.grant(l.a, time.Now().UTC().Add(time.Hour))
	r.check("finite entitlement command", "202 accepted, not applied", detail(g), "POST /admin/members/{A}/grants", g.status == 202)
	me := l.api(l.a, "GET", "/api/v1/member/me", nil, nil)
	r.check("member self read", "200", detail(me), "GET /member/me", me.status == 200)
	l.nativePeer = l.native(r, l.a, "member")
	id := l.awaitDevice(l.nativePeer)
	canonicalIssuer, canonicalErr := issuer.Canonical("https://netbird.example.invalid/oauth2/")
	r.check("canonical shared issuer", "one production canonical form", fmt.Sprintf("canonicalized=%t", canonicalErr == nil), "issuer.Canonical", canonicalErr == nil)
	mapped := l.count(`SELECT count(*) FROM netbird_user_bindings n JOIN identity_bindings i ON i.id=n.identity_binding_id WHERE n.member_id=$1 AND n.external_user_id=i.subject AND i.subject=$2 AND i.issuer=$3 AND n.mapping_proof='shared_issuer_public_subject:'||$3`, l.a.member, l.a.cred.ID, canonicalIssuer)
	bindings := l.count(`SELECT count(*) FROM device_bindings WHERE member_id=$1 AND valid_from<=clock_timestamp() AND valid_until>clock_timestamp()`, l.a.member)
	r.check("cross-client production identity proof", "dashboard/CLI share public subject; verified mapping and finite device binding", fmt.Sprintf("verified_mappings=%d; device_projected=%t; live_bindings=%d", mapped, id != uuid.Nil, bindings), "shared idplogin helper; native CLI device flow; read product D-052 mapping and D-050 device projections", mapped == 1 && id != uuid.Nil && bindings == 1)
	if id == uuid.Nil {
		r.block("Native CLI joined but devices.discover did not produce a member device; identity and entitlement projection must converge before traffic tests.")
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
	l.forwardingGate(r)
	l.traceForwarding(r, "member", id)
	r.block("AT01 platform sign-out/network change/sleep-wake scripts need operator run; Linux positive counted forwarding is recorded separately.")
}
func (l *live) nodeMutation(method, suffix string, body any) response {
	path := "/api/v1/admin/nodes/" + l.node.String()
	var got response
	// Heartbeats advance resource_version. A 412 has no effect; refresh the ETag
	// and retry this setup command with a fresh key, never bypass If-Match.
	for range 3 {
		old := l.api(l.admin, "GET", path, nil, nil)
		h := commandHeaders()
		h["If-Match"] = old.headers.Get("ETag")
		got = l.api(l.admin, method, path+suffix, body, h)
		if got.status != 412 {
			return got
		}
	}
	return got
}
func (l *live) publication(r *record) {
	if l.node == uuid.Nil {
		r.block("No enrolled node")
		return
	}
	l.publish(r, []uuid.UUID{l.groupA})
	intentBefore, _ := l.publicationState()
	if intentBefore != "published" {
		l.resolverEvidence(r)
		return
	}
	routes := l.nb("GET", "/routes", nil)
	var list []map[string]any
	_ = json.Unmarshal(routes.body, &list)
	a := l.count(`SELECT count(*) FROM exit_groups g JOIN exits e ON e.id=g.exit_id WHERE e.node_id=$1 AND g.group_id=$2`, l.node, l.groupA)
	b := l.count(`SELECT count(*) FROM exit_groups g JOIN exits e ON e.id=g.exit_id WHERE e.node_id=$1 AND g.group_id=$2`, l.node, l.groupB)
	r.check("A-only publication readback", "one A attachment, zero B; NetBird routes readable", fmt.Sprintf("A=%d; B=%d; routes=%d; HTTP=%d", a, b, len(list), routes.status), "read exit_groups; GET NetBird /api/routes", a == 1 && b == 0 && routes.status == 200)
	l.forwardingGate(r)
	l.traceForwarding(r, "member", l.awaitDevice(l.nativePeer))
	l.traceForwarding(r, "member-b1", uuid.Nil)
	l.publicationCommand(r, "published", []uuid.UUID{l.groupB})
	regroup := l.count(`SELECT count(*) FROM exit_groups g JOIN exits e ON e.id=g.exit_id WHERE e.node_id=$1 AND g.group_id=$2`, l.node, l.groupB)
	r.check("regroup projection", "group B attached", fmt.Sprintf("B=%d", regroup), "PUT group_ids=[B]; read product attachment", regroup == 1)
	l.publicationCommand(r, "disabled", []uuid.UUID{l.groupB})
	l.traceForwarding(r, "member-b1", uuid.Nil)
	_, err := l.docker("restart", l.name("node"))
	r.check("node restart", "restart succeeds", fmt.Sprintf("restart_ok=%t", err == nil), "docker restart fvpn-w4-members-node", err == nil)
	l.restartReadback(r)
	intent, observed := l.publicationState()
	r.check("disabled persists", "disabled intent after node restart", fmt.Sprintf("intent=%s; observed=%s", intent, observed), "read publication after reconnect/restart", intent == "disabled")
	// Restore the API intent for independent DNS checks; no lease is forged.
	l.publicationCommand(r, "published", []uuid.UUID{l.groupA})
}
func (l *live) exits(r *record) {
	n := l.count(`SELECT count(*) FROM nodes WHERE current_incarnation_id IS NOT NULL`)
	out, err := l.inside("member", "netbird", "routes", "list")
	r.check("exit inventory", "record active node count and client routes", fmt.Sprintf("enrolled_nodes=%d; routes_command_ok=%t; route_output_bytes=%d", n, err == nil, len(out)), "SELECT enrolled nodes; docker exec member netbird routes list", n >= 1)
	l.forwardingGate(r)
	l.routeReadback(r)
	r.block(fmt.Sprintf("Observed %d enrolled node, no two-exit forwarding baseline. The second lab node stays unenrolled while the observed positive-flow gate is unresolved. Manual switch/skip_auto_apply, exit death, relay/direct, TCP/UDP/QUIC and PMTU remain unmeasured; physical second NAT needs an operator run. No bridge-only traffic is counted as exit traffic.", n))
}
