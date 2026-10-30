//go:build integration

package httpwire_test

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func (f *fixture) group(code string) api.MemberGroup {
	f.t.Helper()
	return decode[api.MemberGroup](f.t, f.call("POST", "/api/v1/admin/groups", api.MemberGroupCreate{Code: code, Name: "TEST ONLY " + code}, nil, map[string]string{"Idempotency-Key": uuid.NewString()}), http.StatusCreated)
}

func TestD057_HTTPGroupsImmutablePagingReplayAndAudit(t *testing.T) {
	f := setup(t)
	alpha := f.group("alpha")
	body := api.MemberGroupCreate{Code: "bravo", Name: "TEST ONLY bravo"}
	headers := map[string]string{"Idempotency-Key": uuid.NewString()}
	first := f.call("POST", "/api/v1/admin/groups", body, nil, headers)
	bravo := decode[api.MemberGroup](t, first, http.StatusCreated)
	replay := f.call("POST", "/api/v1/admin/groups", body, nil, headers)
	decode[api.MemberGroup](t, replay, http.StatusCreated)
	if first.Body.String() != replay.Body.String() || bravo.ExitCount != 0 {
		t.Fatal("group receipt changed on replay")
	}
	charlie := f.group("charlie")
	for _, duplicate := range []api.MemberGroupCreate{{Code: body.Code, Name: "TEST ONLY different"}, {Code: "different", Name: body.Name}} {
		decode[api.ErrorEnvelope](t, f.call("POST", "/api/v1/admin/groups", duplicate, nil, map[string]string{"Idempotency-Key": uuid.NewString()}), http.StatusConflict)
	}
	decode[api.ErrorEnvelope](t, f.call("POST", "/api/v1/admin/groups", api.MemberGroupCreate{Code: "changed", Name: "TEST ONLY changed"}, nil, headers), http.StatusConflict)
	page := decode[api.MemberGroupList](t, f.call("GET", "/api/v1/admin/groups?limit=2", nil, nil, nil), http.StatusOK)
	if !reflect.DeepEqual(page.Items, []api.MemberGroup{alpha, bravo}) {
		t.Fatalf("first page: %+v", page)
	}
	cursor := url.QueryEscape(mustValue(t, page.NextCursor))
	page = decode[api.MemberGroupList](t, f.call("GET", "/api/v1/admin/groups?limit=2&cursor="+cursor, nil, nil, nil), http.StatusOK)
	if !reflect.DeepEqual(page.Items, []api.MemberGroup{charlie}) || !page.NextCursor.IsNull() {
		t.Fatalf("last page: %+v", page)
	}
	page = decode[api.MemberGroupList](t, f.call("GET", "/api/v1/admin/groups?code=bravo", nil, nil, nil), http.StatusOK)
	if !reflect.DeepEqual(page.Items, []api.MemberGroup{bravo}) {
		t.Fatalf("code filter: %+v", page)
	}
	decode[api.ErrorEnvelope](t, f.call("GET", "/api/v1/admin/groups?code=bravo&cursor="+cursor, nil, nil, nil), http.StatusBadRequest)
	decode[api.ErrorEnvelope](t, f.call("GET", "/api/v1/admin/groups?cursor=malformed", nil, nil, nil), http.StatusBadRequest)
	store := &audit.Store{Pool: f.r.Pool}
	resource := api.ResourceTypeMemberGroup
	events, err := store.List(t.Context(), api.ListAuditEventsParams{ResourceType: &resource, ResourceId: &bravo.GroupId})
	if err != nil || len(events.Items) != 1 || store.AnomalyCount() != 0 {
		t.Fatalf("group audit: %+v %v", events, err)
	}
	event := events.Items[0]
	if event.Action != "group.create" || event.Result != api.AuditEventResultSucceeded || mustValue(t, event.Actor.Id) != f.p.ID {
		t.Fatalf("wrong group audit: %+v", event)
	}
}

func TestD057_PublicationMembershipReplaceOmitEmptyAndRollback(t *testing.T) {
	runtime, enrolled, report := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
	f := setupRuntime(t, runtime)
	if _, err := runtime.Nodes.SubmitObservation(t.Context(), report.NodeId, report.NodeIncarnationId, report); err != nil {
		t.Fatal(err)
	}
	one, two := f.group("one"), f.group("two")
	path := "/api/v1/admin/nodes/" + enrolled.Response.NodeId.String()
	for _, tc := range []struct {
		name string
		ids  *[]uuid.UUID
		want []uuid.UUID
	}{
		{"replace", &[]uuid.UUID{one.GroupId}, []uuid.UUID{one.GroupId}},
		{"omit", nil, []uuid.UUID{one.GroupId}},
		{"replace again", &[]uuid.UUID{two.GroupId}, []uuid.UUID{two.GroupId}},
		{"clear", &[]uuid.UUID{}, []uuid.UUID{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := f.publicationSnapshot(report.NodeId)
			detailResponse := f.call("GET", path, nil, nil, nil)
			decode[api.NodeDetail](t, detailResponse, http.StatusOK)
			body := api.PublicationRequest{PublicationIntent: api.PublicationIntentUnpublished, Reason: "TEST ONLY group replacement", GroupIds: tc.ids}
			headers := map[string]string{"If-Match": detailResponse.Header().Get("ETag"), "Idempotency-Key": uuid.NewString()}
			op := decode[api.Operation](t, f.call("PUT", path+"/publication", body, nil, headers), http.StatusAccepted)
			replay := decode[api.Operation](t, f.call("PUT", path+"/publication", body, nil, headers), http.StatusAccepted)
			if replay.OperationId != op.OperationId {
				t.Fatal("operation replay changed")
			}
			detail := decode[api.NodeDetail](t, f.call("GET", path, nil, nil, nil), http.StatusOK)
			if detail.GroupIds == nil || !reflect.DeepEqual(*detail.GroupIds, tc.want) || !reflect.DeepEqual(detail.MemberGroupIds, tc.want) {
				t.Fatalf("group projection: %+v", detail)
			}
			after := f.publicationSnapshot(report.NodeId)
			if after.Generation != before.Generation+1 || after.Version != before.Version+1 || after.ExitVersion != before.ExitVersion+1 {
				t.Fatalf("version advance: %+v -> %+v", before, after)
			}
			headers["Idempotency-Key"] = uuid.NewString()
			decode[api.ErrorEnvelope](t, f.call("PUT", path+"/publication", body, nil, headers), http.StatusPreconditionFailed)
			if f.publicationSnapshot(report.NodeId) != after {
				t.Fatal("stale version wrote state")
			}
		})
	}
	before := f.publicationSnapshot(report.NodeId)
	detailResponse := f.call("GET", path, nil, nil, nil)
	bad := api.PublicationRequest{PublicationIntent: api.PublicationIntentDisabled, Reason: "TEST ONLY unknown group", GroupIds: &[]uuid.UUID{one.GroupId, uuid.New()}}
	failure := decode[api.ErrorEnvelope](t, f.call("PUT", path+"/publication", bad, nil, map[string]string{"If-Match": detailResponse.Header().Get("ETag"), "Idempotency-Key": uuid.NewString()}), http.StatusUnprocessableEntity)
	if failure.Error.Code != api.ErrorCodeVALIDATIONFAILED || f.publicationSnapshot(report.NodeId) != before {
		t.Fatal("unknown group partially committed", failure)
	}
	page := decode[api.MemberGroupList](t, f.call("GET", "/api/v1/admin/groups", nil, nil, nil), http.StatusOK)
	for _, g := range page.Items {
		if g.ExitCount != 0 {
			t.Fatal("cleared exit counted", g)
		}
	}
}
