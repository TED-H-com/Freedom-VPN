//go:build e2e

package members

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
	"github.com/google/uuid"
)

// §19.1, AT05: only API input and the shared signed-release hook seed the lab.
// D-050/D-057: catalogues and all business projections are product-owned.
func (l *live) prepare(r *record) bool {
	baseline, err := e2e.PrepareServingBaseline(l.ctx, l.admin.client, l.admin.csrf, false)
	l.baseline = baseline
	r.check("shared serving baseline", "verified tree-built amd64/arm64 boot and active nonempty release policy", fmt.Sprintf("prepared=%t; boot_present=%t; artifact_digests=%d", err == nil, baseline.Boot != uuid.Nil, len(baseline.Digests)), "e2e.PrepareServingBaseline(ctx, admin.Client, csrf, false)", err == nil)
	l.write(r)
	if err != nil {
		r.block("Shared serving baseline failed; enrollment not attempted. No fabricated release or readiness rows.")
		return false
	}
	for _, item := range []struct {
		name string
		dest *uuid.UUID
	}{{"A", &l.groupA}, {"B", &l.groupB}} {
		got := l.api(l.admin, "POST", "/api/v1/admin/groups", api.MemberGroupCreate{Code: "at-" + strings.ToLower(item.name) + "-" + l.run, Name: "TEST ONLY " + item.name}, commandHeaders())
		var group api.MemberGroup
		ok := got.status == 201 && json.Unmarshal(got.body, &group) == nil && group.GroupId != uuid.Nil
		r.check("D-057 group "+item.name, "201", detail(got), "POST /admin/groups", ok)
		if !ok {
			return false
		}
		*item.dest = group.GroupId
	}
	raw, err := os.ReadFile(filepath.Join(l.root, "contracts/examples/policies/fair-use.json"))
	if err != nil {
		return false
	}
	var fup map[string]any
	if json.Unmarshal(raw, &fup) != nil {
		return false
	}
	fup["overrides"] = []any{}
	now := time.Now().UTC()
	fup["effective_at"] = time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	fair := l.activePolicy(r, "fair_use", fup)
	rules := l.activePolicy(r, "domain_rule_set", map[string]any{"rules": []any{rule("feed-01.example.invalid", false, "allow")}})
	profile := l.activePolicy(r, "dns_profile", map[string]any{"profile_code": "at-initial", "display_name": map[string]string{"zh-TW": "TEST ONLY"}, "block_response": "nxdomain", "blocked_ttl_seconds": 5, "domain_rule_set_revision_ids": []uuid.UUID{rules}, "threat_feed_revision_ids": []string{}})
	if fair == uuid.Nil || profile == uuid.Nil {
		return false
	}
	for _, item := range []struct {
		group uuid.UUID
		rev   *uuid.UUID
	}{{l.groupA, &l.planRev}, {l.groupB, &l.planRevB}} {
		*item.rev = l.activePolicy(r, "plan", map[string]any{"display_name": map[string]string{"zh-TW": "TEST ONLY"}, "device_limit": 3, "exit_group_ids": []uuid.UUID{item.group}, "price": map[string]any{"amount_minor": 1000, "currency": "USD", "billing_interval": "month", "provider": "stripe", "provider_price_ref": "e2e-price"}, "rate_limit": map[string]any{"down_bps": 100000000, "up_bps": 100000000}, "fair_use_revision_id": fair, "dns_profile_revision_id": profile})
		if *item.rev == uuid.Nil {
			return false
		}
	}
	err = l.pool.QueryRow(l.ctx, `SELECT plan_id FROM plan_revisions WHERE id=$1`, l.planRev).Scan(&l.plan)
	r.check("projected catalogue", "two active finite plans, distinct exit groups, no SQL fixtures", fmt.Sprintf("plan_projected=%t", err == nil), "read worker plan_revisions projection after API activation", err == nil)
	return err == nil
}
func (l *live) activePolicy(r *record, kind string, payload any) uuid.UUID {
	got, id := l.policy(kind, payload)
	r.check("create "+kind, "201", detail(got), "POST /admin/policies", got.status == 201)
	if got.status != 201 {
		return uuid.Nil
	}
	act := l.activate(id)
	r.check("activate "+kind, "202 pending", detail(act), "POST /admin/policies/{id}/activate", act.status == 202)
	ready := l.wait(60*time.Second, func() bool {
		return l.count(`SELECT count(*) FROM policy_definitions WHERE active_revision_id=$1`, id) == 1
	})
	r.check("active "+kind, "active_revision_id matches issued revision", fmt.Sprintf("converged=%t", ready), "poll product policy_definitions projection", ready)
	if !ready {
		return uuid.Nil
	}
	return id
}
func (l *live) publicationState() (string, string) {
	var intent, observed string
	_ = l.pool.QueryRow(l.ctx, `SELECT publication_intent,publication_observed FROM exits WHERE node_id=$1`, l.node).Scan(&intent, &observed)
	return intent, observed
}
func (l *live) publicationCommand(r *record, intent string, groups []uuid.UUID) bool {
	got := l.nodeMutation("PUT", "/publication", api.PublicationRequest{PublicationIntent: api.PublicationIntent(intent), GroupIds: &groups, Reason: "TEST ONLY AT08"})
	r.check("publication command "+intent, "202 pending, group_ids accepted", detail(got), "PUT /admin/nodes/{id}/publication with group_ids and If-Match", got.status == 202)
	if got.status != 202 {
		r.block("Publication command rejected: " + detail(got))
		return false
	}
	var command struct {
		ID uuid.UUID `json:"operation_id"`
	}
	_ = json.Unmarshal(got.body, &command)
	var status string
	var required, observed int
	ready := l.wait(90*time.Second, func() bool {
		err := l.pool.QueryRow(l.ctx, `SELECT status FROM operations WHERE id=$1`, command.ID).Scan(&status)
		return err == nil && (status == "succeeded" || status == "failed" || status == "partially_succeeded")
	})
	// Operation completion requires read-back evidence (base F-02), not merely 202.
	_ = l.pool.QueryRow(l.ctx, `SELECT count(*) FILTER(WHERE (v->>'required')::boolean),count(*) FILTER(WHERE (v->>'required')::boolean AND v->>'status'='succeeded') FROM operations o CROSS JOIN LATERAL jsonb_each(o.steps) s(k,v) WHERE o.id=$1`, command.ID).Scan(&required, &observed)
	desired, actual := l.publicationState()
	r.check("publication operation "+intent, "record applied operation independently of intent", fmt.Sprintf("status=%s; required=%d; succeeded=%d; intent=%s; observed=%s", status, required, observed, desired, actual), "poll operations.steps and required observations; read exits projection", status != "")
	if !ready || status != "succeeded" {
		r.block("Publication operation still incomplete at bounded readback (terminal step failure or timeout): " + status)
	}
	return ready && status == "succeeded"
}
func (l *live) publish(r *record, groups []uuid.UUID) bool {
	if l.node == uuid.Nil {
		r.block("No enrolled node")
		return false
	}
	ready := l.wait(90*time.Second, func() bool {
		return l.count(`SELECT count(*) FROM nodes WHERE id=$1 AND provisioning_state='ready'`, l.node) == 1
	})
	if !ready {
		r.block("Enrolled node did not project ready after signed baseline")
		return false
	}
	patch := l.nodeMutation("PATCH", "", map[string]any{"name": "TEST ONLY renamed", "region": "TEST", "capacity": map[string]any{"down_bps": 200000000, "up_bps": 200000000, "member_down_bps_cap": 100000000, "member_up_bps_cap": 100000000}})
	r.check("finite capacity and classification", "200 or 202", detail(patch), "PATCH node capacity: TEST ONLY 200 Mbps / 100 Mbps member caps", patch.status == 200 || patch.status == 202)
	return l.publicationCommand(r, "published", groups)
}
func (l *live) routeReadback(r *record) {
	got := l.nb("GET", "/routes", nil)
	var routes []netbird.Route
	decoded := json.Unmarshal(got.body, &routes) == nil
	defaults, manual, enabled, masquerade, peers := 0, 0, 0, 0, 0
	for _, route := range routes {
		if route.Network == "0.0.0.0/0" {
			defaults++
			if route.SkipAutoApply {
				manual++
			}
			if route.Enabled {
				enabled++
			}
			if route.Masquerade {
				masquerade++
			}
			if route.Peer != "" {
				peers++
			}
		}
	}
	r.check("real default-route semantics", "explicit peer, skip_auto_apply=true and masquerade=false (node owns NAT)", fmt.Sprintf("HTTP=%d; defaults=%d; skip_auto_apply=%d; enabled=%d; masquerade=%d; peer_set=%d", got.status, defaults, manual, enabled, masquerade, peers), "GET NetBird /api/routes; retain counts, not external identifiers", got.status == 200 && decoded && defaults > 0 && defaults == manual && masquerade == 0 && defaults == peers)
}
