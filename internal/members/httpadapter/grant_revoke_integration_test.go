//go:build integration

package httpadapter_test

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func readGrantContract[T any](t *testing.T, path string) T {
	t.Helper()
	raw, err := os.ReadFile("../../../contracts/" + path)
	if err != nil {
		t.Fatal(err)
	}
	var out T
	if err = json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func (f *fixture) grant(admin, member browserSession, start, end time.Time) api.Operation {
	f.t.Helper()
	body := fmt.Sprintf(`{"source":"admin_grant","plan_revision_id":%q,"valid_from":%q,"valid_until":%q,"reason":"TEST ONLY withdrawal fixture"}`, f.revision, start.Format(time.RFC3339), end.Format(time.RFC3339))
	w := f.call(admin, "POST", "/api/v1/admin/members/"+member.id.String()+"/grants", body, nil)
	requireStatus(f.t, w, 202)
	return decode[api.Operation](f.t, w)
}

func grantRevokePath(member, grant uuid.UUID) string {
	return "/api/v1/admin/members/" + member.String() + "/grants/" + grant.String() + "/revoke"
}

func TestAT29_D117_RevokeFarFutureFiniteGrant(t *testing.T) {
	f := newFixture(t)
	admin := f.login("TEST-long-grant-admin", "admin")
	member := f.login("TEST-long-grant-member", "member")
	grant := f.grant(admin, member, f.now.Add(-time.Minute), f.now.Add(time.Hour))
	f.drain()
	var before entitlement.Result
	var revision decstr.U64
	if err := f.pool.QueryRow(t.Context(), `SELECT projection,revision::text FROM entitlements WHERE member_id=$1`, member.id).Scan(&before, &revision); err != nil {
		t.Fatal(err)
	}
	// TEST ONLY persisted long-lived grant. Creation still enforces its bounded
	// duration setting; withdrawal must also handle existing finite rows beyond
	// that setting, without inventing an open-ended grant representation (D-117).
	end, target := f.now.AddDate(10, 0, 0), uuid.New()
	before.EffectiveServiceEnd = &end
	before.SourceRefs = append(before.SourceRefs, target)
	raw, err := json.Marshal(before)
	if err != nil {
		t.Fatal(err)
	}
	f.exec(`INSERT INTO entitlement_grants(id,member_id,source,valid_from,valid_until,reason,created_by,plan_revision_id) VALUES($1,$2,'admin_grant',$3,$4,'TEST ONLY far-future finite grant',$5,$6)`, target, member.id, f.now.Add(-time.Minute), end, admin.id.String(), f.revision)
	f.exec(`UPDATE entitlements SET effective_service_end=$2,projection=$3,source_refs=$4,next_transition_at=$5 WHERE member_id=$1`, member.id, end, raw, before.SourceRefs, f.now.Add(time.Hour))
	if !before.Allows(f.now) || len(before.SourceRefs) != 2 || before.SourceRefs[0] != grant.Resource.Id || before.SourceRefs[1] != target {
		t.Fatal("fixture lacks an active finite grant", before)
	}
	body := `{"reason":"TEST ONLY far-future finite withdrawal"}`
	w := f.call(admin, "POST", grantRevokePath(member.id, target), body, nil)
	requireStatus(t, w, 202)
	op := decode[api.Operation](t, w)
	var after entitlement.Result
	var nextRevision decstr.U64
	var shortened bool
	remainingEnd := f.now.Add(time.Hour)
	if err = f.pool.QueryRow(t.Context(), `SELECT projection,revision::text,effective_service_end=$2 AND next_transition_at=$2 AND source_refs=ARRAY[$3::uuid] FROM entitlements WHERE member_id=$1`, member.id, remainingEnd, grant.Resource.Id).Scan(&after, &nextRevision, &shortened); err != nil {
		t.Fatal(err)
	}
	if !after.Allows(f.now) || after.EffectiveServiceEnd == nil || !after.EffectiveServiceEnd.Equal(remainingEnd) || len(after.SourceRefs) != 1 || after.SourceRefs[0] != grant.Resource.Id || nextRevision != revision+1 || !shortened {
		t.Fatalf("far-future withdrawal did not recompute projection: %+v revision %s", after, nextRevision)
	}
	f.assertGrantWithdrawalAudit(admin, member.id, op, "member.grant_revoke", body)
	f.drain()
	f.assertGrantReadback(admin, op.OperationId)
}

// Revision 14 drives statuses and codes; successful responses are also validated
// by fixture.call against the generated OpenAPI document, through composition.
func TestAT29_RevokeMemberGrantContractVectors(t *testing.T) {
	v := readGrantContract[struct {
		AuditAction string            `json:"audit_action"`
		Method      string            `json:"method"`
		OperationID string            `json:"operation_id"`
		Path        string            `json:"path"`
		Headers     map[string]string `json:"headers"`
		Request     json.RawMessage   `json:"request"`
		Response    api.Operation     `json:"response"`
		Cases       []struct {
			Name          string `json:"name"`
			HTTPStatus    int    `json:"http_status"`
			ErrorCode     string `json:"error_code"`
			SameOperation bool   `json:"same_operation"`
		} `json:"cases"`
	}](t, "vectors/member-grant-revoke.json")
	example := readGrantContract[api.Operation](t, "examples/valid/Operation.grant-withdrawal.json")
	auditExample := readGrantContract[api.AuditEventList](t, "examples/valid/AuditEventList.grant-withdrawal.json")
	if v.OperationID != "revokeMemberGrant" || v.Response.Kind != example.Kind || v.Response.Resource.Type != example.Resource.Type || v.AuditAction != auditExample.Items[0].Action {
		t.Fatal("inconsistent withdrawal contracts")
	}
	f := newFixture(t)
	admin := f.login("TEST-withdraw-admin", "admin")
	member := f.login("TEST-withdraw-member", "member")
	other := f.login("TEST-withdraw-other", "member")
	grant := f.grant(admin, member, f.now.Add(-time.Minute), f.now.Add(time.Hour))
	f.drain()
	ordinary := f.authorize(member, "TEST-withdraw-peer", "100.64.0.31")
	fenced := f.authorize(member, "TEST-withdraw-fenced", "100.64.0.32")
	since := f.fence(fenced, true)
	f.netbird.withdrawPaid("TEST-withdraw-fenced")
	grantCount := f.netbird.grantCount()
	f.now = f.now.Add(time.Second)
	path := strings.ReplaceAll(strings.ReplaceAll(v.Path, "dc01a74b-9e6f-41ac-81cd-03291d0a88c0", member.id.String()), v.Response.Resource.Id.String(), grant.Resource.Id.String())
	if path != grantRevokePath(member.id, grant.Resource.Id) {
		t.Fatal("vector path does not match the route")
	}
	var accepted api.Operation
	for _, c := range v.Cases {
		// D-117 retires this obsolete vector label: all grants are finite.
		// TestAT29_D117_RevokeFarFutureFiniteGrant covers the long-lived case.
		if c.Name == "open-ended" {
			continue
		}
		t.Run(c.Name, func(t *testing.T) {
			targets := []string{path}
			headers := map[string]string{}
			switch c.Name {
			case "finite", "same-key-replay":
				headers = v.Headers
			case "unknown-or-other-member":
				targets = []string{grantRevokePath(member.id, uuid.New()), grantRevokePath(other.id, grant.Resource.Id), grantRevokePath(uuid.New(), grant.Resource.Id)}
			case "already-revoked":
			case "expired":
				id := uuid.New()
				// Exact half-open expiry boundary, without a sweep or revoked_at.
				f.exec(`INSERT INTO entitlement_grants(id,member_id,source,valid_from,valid_until,reason,created_by,plan_revision_id) VALUES($1,$2,'admin_grant',$3,$4,'TEST ONLY expired',$5,$6)`, id, member.id, f.now.Add(-time.Hour), f.now, admin.id.String(), f.revision)
				targets = []string{grantRevokePath(member.id, id)}
			default:
				t.Fatalf("unhandled contract case %q", c.Name)
			}
			for _, target := range targets {
				before := f.grantCommandCounts()
				w := f.call(admin, v.Method, target, string(v.Request), headers)
				requireStatus(t, w, c.HTTPStatus)
				if c.ErrorCode != "" {
					if got := decode[apiErrorBody](t, w).Error.Code; got != c.ErrorCode {
						t.Fatalf("error %s want %s", got, c.ErrorCode)
					}
					if after := f.grantCommandCounts(); after != before {
						t.Fatalf("rejected command wrote effects: %v -> %v", before, after)
					}
					continue
				}
				op := decode[api.Operation](t, w)
				if op.Kind != example.Kind || op.Resource.Type != example.Resource.Type || op.Resource.Id != grant.Resource.Id || w.Header().Get("Location") != "/api/v1/admin/operations/"+op.OperationId.String() {
					t.Fatalf("withdrawal receipt: %+v", op)
				}
				if c.SameOperation {
					if op.OperationId != accepted.OperationId || op.RequestId != accepted.RequestId || !op.CreatedAt.Equal(accepted.CreatedAt) || f.grantCommandCounts() != before {
						t.Fatal("replay changed the receipt or duplicated effects")
					}
					continue
				}
				accepted = op
				if op.Status != api.OperationStatusPending || !op.CompletedAt.IsNull() {
					t.Fatal("acceptance reported completion")
				}
				for _, step := range op.Steps {
					if step.Status != api.OperationStepStatusPending || !step.ObservedAt.IsNull() {
						t.Fatal("acceptance fabricated read-back", step)
					}
				}
				var p entitlement.Result
				var revision decstr.U64
				if err := f.pool.QueryRow(t.Context(), `SELECT projection,revision::text FROM entitlements WHERE member_id=$1`, member.id).Scan(&p, &revision); err != nil {
					t.Fatal(err)
				}
				prior, _ := grant.DesiredGeneration.Get()
				if p.Allows(f.now) || len(p.SourceRefs) != 0 || revision <= prior {
					t.Fatalf("withdrawal did not recompute rights: %+v revision %s", p, revision)
				}
				f.assertGrantWithdrawalAudit(admin, member.id, accepted, v.AuditAction, string(v.Request))
				f.drain()
				f.assertGrantReadback(admin, accepted.OperationId)
			}
		})
	}
	if accepted.OperationId == uuid.Nil {
		t.Fatal("vector omitted finite acceptance")
	}
	requireFlag(t, f.memberDevices(member), ordinary, api.DeviceStatusPending, false)
	requireFlag(t, f.memberDevices(member), fenced, api.DeviceStatusPending, true)
	var retained time.Time
	if err := f.pool.QueryRow(t.Context(), `SELECT fenced_at FROM device_authorization_fences WHERE device_id=$1`, fenced).Scan(&retained); err != nil || !retained.Equal(since) {
		t.Fatal("withdrawal changed the authorization fence", err)
	}
	if f.netbird.inPaidGroup("TEST-withdraw-peer") || f.netbird.inPaidGroup("TEST-withdraw-fenced") || f.netbird.grantCount() != grantCount {
		t.Fatal("withdrawal retained or re-granted access")
	}
	// Entitlement restoration still cannot clear the fence or re-grant that peer.
	f.grant(admin, member, f.now.Add(-time.Minute), f.now.Add(time.Hour))
	f.drain()
	requireFlag(t, f.memberDevices(member), fenced, api.DeviceStatusPending, true)
	if f.fences(fenced) != 1 || f.netbird.inPaidGroup("TEST-withdraw-fenced") {
		t.Fatal("restoration cleared the fence")
	}
}

func (f *fixture) grantCommandCounts() [4]int {
	f.t.Helper()
	return [4]int{f.count(`SELECT count(*) FROM operations`), f.count(`SELECT count(*) FROM audit_events`), f.count(`SELECT count(*) FROM outbox_events`), f.count(`SELECT count(*) FROM idempotency_requests`)}
}

func (f *fixture) assertGrantWithdrawalAudit(admin browserSession, member uuid.UUID, op api.Operation, action, body string) {
	f.t.Helper()
	w := f.call(admin, "GET", "/api/v1/admin/audit-events?resource_type=entitlement_grant&resource_id="+op.Resource.Id.String(), "", nil)
	requireStatus(f.t, w, 200)
	page := decode[api.AuditEventList](f.t, w)
	var request api.ReasonRequest
	if err := json.Unmarshal([]byte(body), &request); err != nil {
		f.t.Fatal(err)
	}
	n := 0
	for _, event := range page.Items {
		if event.Action != action {
			continue
		}
		n++
		actor, _ := event.Actor.Id.Get()
		operation, _ := event.OperationId.Get()
		requestID, _ := event.RequestId.Get()
		if actor != admin.id || event.Actor.Type != api.AuditEventActorTypeAdmin || operation != op.OperationId || requestID != op.RequestId || event.Resource != op.Resource || event.Result != api.AuditEventResultSucceeded {
			f.t.Fatal("withdrawal audit correlation", event)
		}
		raw, err := json.Marshal(event)
		if err != nil {
			f.t.Fatal(err)
		}
		var decoded struct {
			Changes struct{ Before, After map[string]any }
		}
		if err = json.Unmarshal(raw, &decoded); err != nil {
			f.t.Fatal(err)
		}
		if !reflect.DeepEqual(decoded.Changes.Before, map[string]any{"revoked_at": nil, "member_id": nil, "grant_id": nil, "reason": nil}) || !reflect.DeepEqual(decoded.Changes.After, map[string]any{"revoked_at": f.now.Format(time.RFC3339), "reason": request.Reason, "member_id": member.String(), "grant_id": op.Resource.Id.String()}) {
			f.t.Fatalf("withdrawal audit diff: %s", raw)
		}
	}
	if n != 1 || f.count(`SELECT count(*) FROM entitlement_grants WHERE id=$1 AND revoked_at=$2 AND revocation_reason=$3`, op.Resource.Id, f.now, request.Reason) != 1 {
		f.t.Fatal("withdrawal/audit missing or duplicated")
	}
}

func (f *fixture) assertGrantReadback(admin browserSession, id uuid.UUID) {
	f.t.Helper()
	w := f.call(admin, "GET", "/api/v1/admin/operations/"+id.String(), "", nil)
	requireStatus(f.t, w, 200)
	op := decode[api.Operation](f.t, w)
	if len(op.Steps) != 2 || !op.CompletedAt.IsNull() || op.Status == api.OperationStatusSucceeded {
		f.t.Fatal("withdrawal completed without node enforcement evidence", op)
	}
	for _, step := range op.Steps {
		switch step.Name {
		case "netbird":
			if step.Status != api.OperationStepStatusSucceeded || step.ObservedAt.IsNull() {
				f.t.Fatal("withdrawal lacks authorization read-back", step)
			}
		case "nodes":
			if step.Status != api.OperationStepStatusPending || !step.ObservedAt.IsNull() {
				f.t.Fatal("compilation fabricated enforcement evidence", step)
			}
		default:
			f.t.Fatal("unexpected operation step", step)
		}
	}
}

func TestAT29_RevokeFutureGrantAndHTTPGuards(t *testing.T) {
	f := newFixture(t)
	admin := f.login("TEST-future-admin", "admin")
	member := f.login("TEST-future-member", "member")
	grant := f.grant(admin, member, f.now.Add(time.Hour), f.now.Add(2*time.Hour))
	f.drain()
	path := grantRevokePath(member.id, grant.Resource.Id)
	body := `{"reason":"TEST ONLY future withdrawal"}`
	for _, c := range []struct {
		name, body, header, value, code string
		session                         browserSession
		status                          int
	}{
		{"anonymous", body, "", "", "UNAUTHENTICATED", browserSession{}, 401},
		{"member", body, "", "", "UNAUTHENTICATED", member, 401},
		{"csrf", body, "X-CSRF-Token", "", "CSRF_TOKEN_INVALID", admin, 403},
		{"idempotency", body, "Idempotency-Key", "", "MALFORMED_REQUEST", admin, 400},
		{"reason", `{"reason":" "}`, "", "", "VALIDATION_FAILED", admin, 422},
	} {
		t.Run(c.name, func(t *testing.T) {
			before := f.grantCommandCounts()
			w := f.call(c.session, "POST", path, c.body, map[string]string{c.header: c.value})
			requireStatus(t, w, c.status)
			if decode[apiErrorBody](t, w).Error.Code != c.code || f.grantCommandCounts() != before {
				t.Fatalf("guard response/effects: %s", w.Body.String())
			}
		})
	}
	key := uuid.NewString()
	// An audit failure must roll back withdrawal, its operation, work and key.
	f.exec(`ALTER TABLE audit_events ADD CONSTRAINT test_withdrawal_reject CHECK(action <> 'member.grant_revoke')`)
	beforeFailure := f.grantCommandCounts()
	w := f.call(admin, "POST", path, body, map[string]string{"Idempotency-Key": key})
	requireStatus(t, w, 500)
	if f.grantCommandCounts() != beforeFailure || f.count(`SELECT count(*) FROM entitlement_grants WHERE id=$1 AND revoked_at IS NULL`, grant.Resource.Id) != 1 {
		t.Fatal("audit failure committed withdrawal effects")
	}
	f.exec(`ALTER TABLE audit_events DROP CONSTRAINT test_withdrawal_reject`)
	w = f.call(admin, "POST", path, body, map[string]string{"Idempotency-Key": key})
	requireStatus(t, w, 202)
	op := decode[api.Operation](t, w)
	if op.DesiredGeneration != grant.DesiredGeneration {
		t.Fatal("future withdrawal changed current entitlement revision")
	}
	if f.count(`SELECT count(*) FROM outbox_events WHERE aggregate_id=$1 AND event_type='member.grant_revoked'`, member.id) != 1 || f.count(`SELECT count(*) FROM operations WHERE id=$1 AND status='superseded'`, grant.OperationId) != 1 {
		t.Fatal("future withdrawal did not supersede its grant and schedule read-back")
	}
	f.drain()
	f.assertGrantReadback(admin, op.OperationId)
	before := f.grantCommandCounts()
	w = f.call(admin, "POST", path, `{"reason":"TEST ONLY different payload"}`, map[string]string{"Idempotency-Key": key})
	requireStatus(t, w, 409)
	if decode[apiErrorBody](t, w).Error.Code != "IDEMPOTENCY_CONFLICT" || f.grantCommandCounts() != before {
		t.Fatal("changed-payload replay was accepted or wrote effects")
	}
}
