//go:build integration

package httpwire_test

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestReviewD057_ConcurrentPublicationFencing(t *testing.T) {
	runtime, _, report := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
	f := setupRuntime(t, runtime)
	if _, err := runtime.Nodes.SubmitObservation(t.Context(), report.NodeId, report.NodeIncarnationId, report); err != nil {
		t.Fatal(err)
	}
	one, two := f.group("raceone"), f.group("racetwo")
	path := "/api/v1/admin/nodes/" + report.NodeId.String()
	detail := f.call("GET", path, nil, nil, nil)
	decode[api.NodeDetail](t, detail, http.StatusOK)
	etag := detail.Header().Get("ETag")
	type result struct {
		code int
		body string
	}
	results := make([]result, 2)
	var ready, done sync.WaitGroup
	ready.Add(2)
	done.Add(2)
	var start sync.WaitGroup
	start.Add(1)
	for i, id := range []uuid.UUID{one.GroupId, two.GroupId} {
		go func(i int, id uuid.UUID) {
			defer done.Done()
			ready.Done()
			start.Wait()
			ids := []uuid.UUID{id}
			body := api.PublicationRequest{PublicationIntent: api.PublicationIntentUnpublished, Reason: "TEST ONLY concurrent membership", GroupIds: &ids}
			w, err := f.callErr("PUT", path+"/publication", body, nil, map[string]string{"If-Match": etag, "Idempotency-Key": uuid.NewString()})
			if err != nil {
				results[i] = result{code: -1, body: err.Error()}
				return
			}
			results[i] = result{code: w.Code, body: w.Body.String()}
		}(i, id)
	}
	ready.Wait()
	start.Done()
	done.Wait()
	wins := 0
	for _, r := range results {
		switch r.code {
		case http.StatusAccepted:
			wins++
		case http.StatusPreconditionFailed, http.StatusConflict:
		default:
			t.Fatalf("unexpected publication result %d %s", r.code, r.body)
		}
	}
	if wins != 1 {
		t.Fatalf("If-Match fence: wins=%d results=%+v", wins, results)
	}
	got := f.exitGroupIDs(report.NodeId)
	if len(got) != 1 || (got[0] != one.GroupId && got[0] != two.GroupId) {
		t.Fatalf("partial membership remained: %v", got)
	}
}

func TestReviewD057_MembershipSurvivesUnpublishRevokeAndRepublish(t *testing.T) {
	runtime, _, report := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
	f := setupRuntime(t, runtime)
	if _, err := runtime.Nodes.SubmitObservation(t.Context(), report.NodeId, report.NodeIncarnationId, report); err != nil {
		t.Fatal(err)
	}
	group := f.group("staygrp")
	path := "/api/v1/admin/nodes/" + report.NodeId.String()
	f.putPublication(path, &[]uuid.UUID{group.GroupId}, api.PublicationIntentUnpublished)
	if n := f.groupExitCount(group.Code); n != 1 || len(f.exitGroupIDs(report.NodeId)) != 1 {
		t.Fatalf("attach count=%d ids=%v", n, f.exitGroupIDs(report.NodeId))
	}
	f.putPublication(path, nil, api.PublicationIntentUnpublished)
	if len(f.exitGroupIDs(report.NodeId)) != 1 || f.groupExitCount(group.Code) != 1 {
		t.Fatal("omitted group_ids cleared membership")
	}
	detail := f.call("GET", path, nil, nil, nil)
	decode[api.NodeDetail](t, detail, http.StatusOK)
	rev := f.call("POST", path+"/revoke", api.ReasonRequest{Reason: "TEST ONLY review revoke"}, nil, map[string]string{"If-Match": detail.Header().Get("ETag"), "Idempotency-Key": uuid.NewString()})
	if rev.Code != http.StatusAccepted {
		t.Fatalf("revoke %d %s", rev.Code, rev.Body.String())
	}
	if len(f.exitGroupIDs(report.NodeId)) != 1 || f.groupExitCount(group.Code) != 1 {
		t.Fatalf("revoke dropped membership count=%d ids=%v", f.groupExitCount(group.Code), f.exitGroupIDs(report.NodeId))
	}
	// Publishing a revoked node must fail without resurrecting or clearing rows.
	before := append([]uuid.UUID{}, f.exitGroupIDs(report.NodeId)...)
	w := f.putPublicationRaw(path, nil, api.PublicationIntentPublished)
	if w.Code == http.StatusAccepted {
		t.Fatalf("revoked node republished: %s", w.Body.String())
	}
	if ids := f.exitGroupIDs(report.NodeId); len(ids) != 1 || ids[0] != before[0] || f.groupExitCount(group.Code) != 1 {
		t.Fatalf("failed republish changed membership: %v count=%d", ids, f.groupExitCount(group.Code))
	}
	// An explicit empty list still clears, including after revocation.
	f.putPublication(path, &[]uuid.UUID{}, api.PublicationIntentDisabled)
	if len(f.exitGroupIDs(report.NodeId)) != 0 || f.groupExitCount(group.Code) != 0 {
		t.Fatal("empty group_ids left membership behind")
	}
}

func TestReviewD057_UnknownGroupAndDeleteRace(t *testing.T) {
	runtime, _, report := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
	f := setupRuntime(t, runtime)
	if _, err := runtime.Nodes.SubmitObservation(t.Context(), report.NodeId, report.NodeIncarnationId, report); err != nil {
		t.Fatal(err)
	}
	group := f.group("knownone")
	path := "/api/v1/admin/nodes/" + report.NodeId.String()
	unknown := uuid.New()
	w := f.putPublicationRaw(path, &[]uuid.UUID{group.GroupId, unknown}, api.PublicationIntentUnpublished)
	env := decode[api.ErrorEnvelope](t, w, http.StatusUnprocessableEntity)
	if env.Error.Code != api.ErrorCodeVALIDATIONFAILED || len(f.exitGroupIDs(report.NodeId)) != 0 {
		t.Fatalf("unknown id slipped or wrong code: %s ids=%v", w.Body.String(), f.exitGroupIDs(report.NodeId))
	}
	dup := []uuid.UUID{group.GroupId, group.GroupId}
	w = f.putPublicationRaw(path, &dup, api.PublicationIntentUnpublished)
	if w.Code != http.StatusUnprocessableEntity || len(f.exitGroupIDs(report.NodeId)) != 0 {
		t.Fatalf("duplicate id: %d %s", w.Code, w.Body.String())
	}
	f.putPublication(path, &[]uuid.UUID{group.GroupId}, api.PublicationIntentUnpublished)
	// The count of member_groups is not locked. A delete that commits before the
	// insert is the same FK failure this trigger raises inside the publication
	// transaction (internal/exits/groups.go:21-30).
	f.sql(`CREATE OR REPLACE FUNCTION review_d057_drop_group() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  DELETE FROM member_groups WHERE id = OLD.group_id;
  RETURN NULL;
END $$`)
	f.sql(`CREATE TRIGGER review_d057_drop AFTER DELETE ON exit_groups
FOR EACH ROW EXECUTE FUNCTION review_d057_drop_group()`)
	w = f.putPublicationRaw(path, &[]uuid.UUID{group.GroupId}, api.PublicationIntentUnpublished)
	ids := f.exitGroupIDs(report.NodeId)
	if w.Code != http.StatusUnprocessableEntity || len(ids) != 1 {
		// TODO(review): replacePublicationGroups counts member_groups, then deletes
		// and inserts, without locking the group rows. A concurrent delete makes
		// the insert's foreign key fail as a raw error (HTTP 500), not 422.
		t.Fatalf("TODO(review): group removed between count and insert: status=%d body=%s membership=%v", w.Code, w.Body.String(), ids)
	}
}

func TestReviewD057_GroupCreateRaceAndNormalization(t *testing.T) {
	f := setup(t)
	const code = "racecode"
	const name = "TEST ONLY race name"
	f.raceCreate(8, api.MemberGroupCreate{Code: code, Name: name})
	if n := f.memberGroupRows(code, ""); n != 1 {
		t.Fatalf("duplicate code rows: %d", n)
	}
	// Distinct codes, one shared name. The unique name constraint must leave one row.
	var ready, done sync.WaitGroup
	var start sync.WaitGroup
	start.Add(1)
	const racers = 8
	ready.Add(racers)
	done.Add(racers)
	codes := make([]int, racers)
	for i := 0; i < racers; i++ {
		go func(i int) {
			defer done.Done()
			ready.Done()
			start.Wait()
			body := api.MemberGroupCreate{Code: "nm" + string(rune('a'+i)), Name: "TEST ONLY shared name"}
			w, err := f.callErr("POST", "/api/v1/admin/groups", body, nil, map[string]string{"Idempotency-Key": uuid.NewString()})
			if err != nil {
				codes[i] = -1
				return
			}
			codes[i] = w.Code
		}(i)
	}
	ready.Wait()
	start.Done()
	done.Wait()
	wins := 0
	for _, code := range codes {
		if code == http.StatusCreated {
			wins++
		} else if code != http.StatusConflict {
			t.Fatalf("name race status %d", code)
		}
	}
	if wins != 1 || f.memberGroupRows("", "TEST ONLY shared name") != 1 {
		t.Fatalf("duplicate names created: wins=%d rows=%d", wins, f.memberGroupRows("", "TEST ONLY shared name"))
	}
	upper := f.call("POST", "/api/v1/admin/groups", api.MemberGroupCreate{Code: "RaceCode", Name: "TEST ONLY other"}, nil, map[string]string{"Idempotency-Key": uuid.NewString()})
	if upper.Code != http.StatusUnprocessableEntity && upper.Code != http.StatusBadRequest {
		t.Fatalf("uppercase code accepted: %d %s", upper.Code, upper.Body.String())
	}
	folded := decode[api.MemberGroup](t, f.call("POST", "/api/v1/admin/groups", api.MemberGroupCreate{Code: "casefold", Name: "Case Fold"}, nil, map[string]string{"Idempotency-Key": uuid.NewString()}), http.StatusCreated)
	lower := decode[api.MemberGroup](t, f.call("POST", "/api/v1/admin/groups", api.MemberGroupCreate{Code: "casefold2", Name: "case fold"}, nil, map[string]string{"Idempotency-Key": uuid.NewString()}), http.StatusCreated)
	padded := decode[api.MemberGroup](t, f.call("POST", "/api/v1/admin/groups", api.MemberGroupCreate{Code: "casefold3", Name: " case fold"}, nil, map[string]string{"Idempotency-Key": uuid.NewString()}), http.StatusCreated)
	if folded.GroupId == lower.GroupId || lower.Name == padded.Name || padded.Name != " case fold" || folded.Name != "Case Fold" {
		t.Fatalf("names were normalized: %+v %+v %+v", folded, lower, padded)
	}
	page := decode[api.MemberGroupList](t, f.call("GET", "/api/v1/admin/groups?code=casefold3", nil, nil, nil), http.StatusOK)
	if len(page.Items) != 1 || page.Items[0].Name != " case fold" || page.Items[0].Code != "casefold3" {
		t.Fatalf("list name %+v", page.Items)
	}
}

func TestReviewD057_MemberResponsesDoNotCarryGroupIdentity(t *testing.T) {
	f, enrolled, cert, member, _ := servingFixture(t)
	created := f.group("leakgrp1")
	var attached uuid.UUID
	var attachedCode, attachedName string
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT g.id, g.code, g.name FROM member_groups g
		JOIN exit_groups eg ON eg.group_id=g.id JOIN exits e ON e.id=eg.exit_id WHERE e.node_id=$1`, enrolled.NodeId).Scan(&attached, &attachedCode, &attachedName); err != nil {
		t.Fatal(err)
	}
	var exitID uuid.UUID
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT id FROM exits WHERE node_id=$1`, enrolled.NodeId).Scan(&exitID); err != nil {
		t.Fatal(err)
	}
	state := f.call("GET", "/api/v1/agent/desired-state", nil, cert, nil)
	if state.Code != http.StatusOK {
		t.Fatalf("desired state %d %s", state.Code, state.Body.String())
	}
	f.p.Role = auth.Member
	f.p.ID = member
	exits := f.call("GET", "/api/v1/member/exits", nil, nil, nil)
	if exits.Code != http.StatusOK || !strings.Contains(exits.Body.String(), exitID.String()) {
		t.Fatalf("member exits %d %s", exits.Code, exits.Body.String())
	}
	bodies := map[string]string{
		"desired-state": state.Body.String(),
		"member-exits":  exits.Body.String(),
	}
	for _, p := range []string{"/api/v1/member/me", "/api/v1/member/devices", "/api/v1/member/usage"} {
		w := f.call("GET", p, nil, nil, nil)
		bodies[p] = w.Body.String()
	}
	adminKnown := f.call("GET", "/api/v1/admin/groups?code="+created.Code, nil, nil, nil)
	adminMissing := f.call("GET", "/api/v1/admin/groups?code=nosuchgrp", nil, nil, nil)
	adminNode := f.call("GET", "/api/v1/admin/nodes/"+enrolled.NodeId.String(), nil, nil, nil)
	adminPost := f.call("POST", "/api/v1/admin/groups", api.MemberGroupCreate{Code: "membertry", Name: "TEST ONLY member"}, nil, map[string]string{"Idempotency-Key": uuid.NewString()})
	for _, w := range []struct {
		name string
		code int
		body string
	}{
		{"known group", adminKnown.Code, adminKnown.Body.String()},
		{"missing group", adminMissing.Code, adminMissing.Body.String()},
		{"node", adminNode.Code, adminNode.Body.String()},
		{"create", adminPost.Code, adminPost.Body.String()},
	} {
		if w.code != http.StatusForbidden {
			t.Fatalf("member %s status %d %s", w.name, w.code, w.body)
		}
		bodies["admin "+w.name] = w.body
	}
	knownEnv := decode[api.ErrorEnvelope](t, adminKnown, http.StatusForbidden)
	missingEnv := decode[api.ErrorEnvelope](t, adminMissing, http.StatusForbidden)
	if knownEnv.Error.Code != missingEnv.Error.Code || knownEnv.Error.Message != missingEnv.Error.Message {
		t.Fatalf("group existence oracle: %+v vs %+v", knownEnv.Error, missingEnv.Error)
	}
	secrets := []string{created.GroupId.String(), created.Code, created.Name, attached.String(), attachedCode, attachedName}
	for surface, body := range bodies {
		for _, secret := range secrets {
			if strings.Contains(body, secret) {
				t.Fatalf("%s contains %q: %s", surface, secret, body)
			}
		}
	}
}

func (f *fixture) putPublication(path string, ids *[]uuid.UUID, intent api.PublicationIntent) {
	f.t.Helper()
	w := f.putPublicationRaw(path, ids, intent)
	if w.Code != http.StatusAccepted {
		f.t.Fatalf("publication %s: %d %s", intent, w.Code, w.Body.String())
	}
}

func (f *fixture) putPublicationRaw(path string, ids *[]uuid.UUID, intent api.PublicationIntent) *httptest.ResponseRecorder {
	f.t.Helper()
	detail := f.call("GET", path, nil, nil, nil)
	if detail.Code != http.StatusOK {
		f.t.Fatalf("node read %d %s", detail.Code, detail.Body.String())
	}
	body := api.PublicationRequest{PublicationIntent: intent, Reason: "TEST ONLY review publication", GroupIds: ids}
	return f.call("PUT", path+"/publication", body, nil, map[string]string{"If-Match": detail.Header().Get("ETag"), "Idempotency-Key": uuid.NewString()})
}

// callErr is f.call without testing.T, so concurrent requests do not race the test handle.
func (f *fixture) callErr(method, path string, body any, cert *x509.Certificate, headers map[string]string) (*httptest.ResponseRecorder, error) {
	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	r := httptest.NewRequest(method, "https://control.example.invalid"+path, bytes.NewReader(raw))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "https://portal.example.invalid")
	r.Header.Set("X-CSRF-Token", f.p.CSRFToken)
	for k, v := range headers {
		if v != "" {
			r.Header.Set(k, v)
		}
	}
	w := httptest.NewRecorder()
	if cert != nil {
		r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{cert}}}
		f.agent.ServeHTTP(w, r)
	} else {
		f.public.ServeHTTP(w, r)
	}
	return w, nil
}

func (f *fixture) exitGroupIDs(node uuid.UUID) []uuid.UUID {
	f.t.Helper()
	rows, err := f.r.Pool.Query(f.t.Context(), `SELECT eg.group_id FROM exit_groups eg JOIN exits e ON e.id=eg.exit_id WHERE e.node_id=$1 ORDER BY eg.group_id`, node)
	if err != nil {
		f.t.Fatal(err)
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err = rows.Scan(&id); err != nil {
			f.t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		f.t.Fatal(err)
	}
	return ids
}

func (f *fixture) groupExitCount(code string) int {
	f.t.Helper()
	page := decode[api.MemberGroupList](f.t, f.call("GET", "/api/v1/admin/groups?code="+code, nil, nil, nil), http.StatusOK)
	if len(page.Items) != 1 {
		f.t.Fatalf("group %s: %+v", code, page.Items)
	}
	return page.Items[0].ExitCount
}

func (f *fixture) memberGroupRows(code, name string) int {
	f.t.Helper()
	var n int
	err := f.r.Pool.QueryRow(f.t.Context(), `SELECT count(*) FROM member_groups WHERE ($1='' OR code=$1) AND ($2='' OR name=$2)`, code, name).Scan(&n)
	if err != nil {
		f.t.Fatal(err)
	}
	return n
}

func (f *fixture) raceCreate(n int, body api.MemberGroupCreate) {
	f.t.Helper()
	var ready, done sync.WaitGroup
	var start sync.WaitGroup
	start.Add(1)
	ready.Add(n)
	done.Add(n)
	codes := make([]int, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer done.Done()
			ready.Done()
			start.Wait()
			w, err := f.callErr("POST", "/api/v1/admin/groups", body, nil, map[string]string{"Idempotency-Key": uuid.NewString()})
			if err != nil {
				codes[i] = -1
				return
			}
			codes[i] = w.Code
		}(i)
	}
	ready.Wait()
	start.Done()
	done.Wait()
	wins := 0
	for _, code := range codes {
		switch code {
		case http.StatusCreated:
			wins++
		case http.StatusConflict:
		default:
			f.t.Fatalf("create race status %d", code)
		}
	}
	if wins != 1 {
		f.t.Fatalf("create race wins=%d for %+v", wins, body)
	}
}
