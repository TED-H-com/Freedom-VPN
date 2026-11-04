//go:build integration

package httpwire_test

import (
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

type conditionalWriteSnapshot struct {
	Rows                             string
	Operations, Audits, Events, Keys int
}

// Include complete node/exit and identity rows: a rejected revoke or retirement
// must not revoke credentials, alter timestamps or advance generations either.
func (f *fixture) conditionalWriteSnapshot(node uuid.UUID) conditionalWriteSnapshot {
	f.t.Helper()
	var s conditionalWriteSnapshot
	err := f.r.Pool.QueryRow(f.t.Context(), `SELECT jsonb_build_object(
		'node',to_jsonb(n), 'exit',to_jsonb(e),
		'incarnations',(SELECT jsonb_agg(to_jsonb(i) ORDER BY i.id) FROM node_incarnations i WHERE i.node_id=n.id),
		'certificates',(SELECT jsonb_agg(to_jsonb(c) ORDER BY c.id) FROM node_certificates c JOIN node_incarnations i ON i.id=c.incarnation_id WHERE i.node_id=n.id),
		'enrollments',(SELECT jsonb_agg(to_jsonb(r) ORDER BY r.id) FROM node_enrollments r WHERE r.node_id=n.id),
		'groups',(SELECT jsonb_agg(to_jsonb(g) ORDER BY g.id) FROM exit_groups g WHERE g.exit_id=e.id))::text,
		(SELECT count(*) FROM operations),
		(SELECT count(*) FROM audit_events),
		(SELECT count(*) FROM outbox_events),
		(SELECT count(*) FROM idempotency_requests)
		FROM nodes n JOIN exits e ON e.node_id=n.id WHERE n.id=$1`, node).
		Scan(&s.Rows, &s.Operations, &s.Audits, &s.Events, &s.Keys)
	if err != nil {
		f.t.Fatal(err)
	}
	return s
}

func TestAPI01_HTTPZeroIfMatchConflictAndSameKeyRetry(t *testing.T) {
	capacity := map[string]any{"down_bps": 100000000, "up_bps": 100000000, "member_down_bps_cap": nil, "member_up_bps_cap": nil}
	for _, tc := range []struct {
		name, method, suffix string
		body                 any
		kind                 api.OperationKind
		lifecycle            api.LifecycleState
		intent               api.PublicationIntent
	}{
		{"patch", http.MethodPatch, "", map[string]any{"capacity": capacity}, api.OperationKindNodeUpdate, api.LifecycleStateRegistered, api.PublicationIntentUnpublished},
		{"revoke", http.MethodPost, "/revoke", map[string]string{"reason": "TEST ONLY zero If-Match"}, api.OperationKindNodeRevoke, api.LifecycleStateRevoked, api.PublicationIntentDisabled},
		{"retire", http.MethodDelete, "", nil, api.OperationKindNodeRetire, api.LifecycleStateRetired, api.PublicationIntentDisabled},
		{"publication", http.MethodPut, "/publication", api.PublicationRequest{PublicationIntent: api.PublicationIntentDisabled, Reason: "TEST ONLY zero If-Match"}, api.OperationKindNodeDisable, api.LifecycleStateRegistered, api.PublicationIntentDisabled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := setup(t)
			_, enrolled, cert := f.enroll(f.create(nil))
			path := "/api/v1/admin/nodes/" + enrolled.NodeId.String()
			read := f.call(http.MethodGet, path, nil, nil, nil)
			initial := decode[api.NodeDetail](t, read, http.StatusOK)
			if initial.ResourceVersion < 2 {
				t.Fatal("fixture must allow both zero and a non-zero stale version")
			}
			before := f.conditionalWriteSnapshot(enrolled.NodeId)
			headers := map[string]string{"Idempotency-Key": uuid.NewString()}
			for _, stale := range []string{`"0"`, nodes.FormatETag((initial.ResourceVersion - 1).String())} {
				headers["If-Match"] = stale
				failure := decode[api.ErrorEnvelope](t, f.call(tc.method, path+tc.suffix, tc.body, nil, headers), http.StatusPreconditionFailed)
				if failure.Error.Code != api.ErrorCodeRESOURCEVERSIONCONFLICT || failure.Error.Details["current_resource_version"] != initial.ResourceVersion.String() {
					t.Fatalf("If-Match %s: wrong conflict: %+v", stale, failure)
				}
				if f.conditionalWriteSnapshot(enrolled.NodeId) != before {
					t.Fatalf("If-Match %s changed state, operations, audit, outbox or idempotency", stale)
				}
				if _, err := f.r.PKI.AuthenticateAgent(t.Context(), cert); err != nil {
					t.Fatalf("If-Match %s fenced the node identity: %v", stale, err)
				}
			}
			// Only the header changes: both failed reservations must have rolled
			// back so the original key and body can commit the requested command.
			headers["If-Match"] = read.Header().Get("ETag")
			op := decode[api.Operation](t, f.call(tc.method, path+tc.suffix, tc.body, nil, headers), http.StatusAccepted)
			if op.Kind != tc.kind || op.Status != api.OperationStatusPending || mustValue(t, op.ResourceVersion) != initial.ResourceVersion+1 {
				t.Fatalf("wrong accepted operation: %+v", op)
			}
			after := f.conditionalWriteSnapshot(enrolled.NodeId)
			if after.Operations != before.Operations+1 || after.Audits != before.Audits+1 || after.Events != before.Events+1 || after.Keys != before.Keys+1 {
				t.Fatal("same-key retry did not commit exactly one operation, audit, outbox event and idempotency record")
			}
			updated := decode[api.NodeDetail](t, f.call(http.MethodGet, path, nil, nil, nil), http.StatusOK)
			if updated.ResourceVersion != initial.ResourceVersion+1 || updated.LifecycleState != tc.lifecycle || updated.PublicationIntent != tc.intent {
				t.Fatal("same-key retry did not apply the intended version, lifecycle and publication")
			}
			if tc.method == http.MethodPatch {
				got := mustValue(t, updated.Capacity)
				if got.DownBps != 100000000 || got.UpBps != 100000000 {
					t.Fatal("same-key retry lost the requested capacity")
				}
			}
			t.Log("zero and non-zero conflicts: current version supplied, no durable effects; same-key/body retry accepted once")
		})
	}
}

func TestAPI01_HTTPIfMatchZeroRowMatches(t *testing.T) {
	for _, initialVersion := range []decstr.U64{0, 1} {
		t.Run(initialVersion.String(), func(t *testing.T) {
			f := setup(t)
			nodeID := uuid.New()
			// TEST ONLY explicit insert; production enrollment keeps DEFAULT 1.
			f.sql(`INSERT INTO nodes(id,resource_version) VALUES($1,$2::numeric)`, nodeID, initialVersion.String())
			f.sql(`INSERT INTO exits(node_id,display_name,region) VALUES($1,'TEST ONLY','TEST ONLY')`, nodeID)
			path := "/api/v1/admin/nodes/" + nodeID.String()
			read := f.call(http.MethodGet, path, nil, nil, nil)
			initial := decode[api.NodeDetail](t, read, http.StatusOK)
			if initial.ResourceVersion != initialVersion || read.Header().Get("ETag") != nodes.FormatETag(initialVersion.String()) {
				t.Fatal("GET did not expose the stored version")
			}
			before := f.conditionalWriteSnapshot(nodeID)
			headers := map[string]string{"Idempotency-Key": uuid.NewString(), "If-Match": read.Header().Get("ETag")}
			body := map[string]string{"notes": "TEST ONLY matching version"}
			write := f.call(http.MethodPatch, path, body, nil, headers)
			updated := decode[api.NodeDetail](t, write, http.StatusOK)
			if updated.ResourceVersion != initialVersion+1 || mustValue(t, updated.Notes) != body["notes"] || write.Header().Get("ETag") != nodes.FormatETag((initialVersion+1).String()) {
				t.Fatal("matching version failed to write notes and advance the version")
			}
			after := f.conditionalWriteSnapshot(nodeID)
			if after.Rows == before.Rows || after.Operations != before.Operations || after.Events != before.Events || after.Audits != before.Audits+1 || after.Keys != before.Keys+1 {
				t.Fatal("metadata write did not commit exactly one audit and idempotency record")
			}
			replayed := decode[api.NodeDetail](t, f.call(http.MethodPatch, path, body, nil, headers), http.StatusOK)
			if !reflect.DeepEqual(replayed, updated) || f.conditionalWriteSnapshot(nodeID) != after {
				t.Fatal("same-key replay changed the accepted write")
			}
			t.Logf("matching ETag %s advanced version to %s; replay left it unchanged", read.Header().Get("ETag"), updated.ResourceVersion)
		})
	}
}

// These service entry points are also parser callers, although the HTTP revoke
// and retire handlers use enrollment and publication uses SetPublicationTx.
func TestAPI01_StandaloneZeroIfMatchConflictAndSameKeyRetry(t *testing.T) {
	for _, name := range []string{"revoke", "retire", "publication"} {
		t.Run(name, func(t *testing.T) {
			f := setup(t)
			_, enrolled, _ := f.enroll(f.create(nil))
			initial, etag, err := f.r.Nodes.GetNode(t.Context(), enrolled.NodeId)
			if err != nil {
				t.Fatal(err)
			}
			key := uuid.New()
			call := func(header string) (api.Operation, string, error) {
				switch name {
				case "revoke":
					return f.r.Nodes.RevokeNode(t.Context(), nodes.RevokeNodeParams{NodeID: enrolled.NodeId, Principal: f.p.Scope(), IdempotencyKey: key, IfMatch: header, Request: api.ReasonRequest{Reason: "TEST ONLY zero version"}})
				case "retire":
					return f.r.Nodes.RetireNode(t.Context(), nodes.RetireNodeParams{NodeID: enrolled.NodeId, Principal: f.p.Scope(), IdempotencyKey: key, IfMatch: header, Reason: "TEST ONLY zero version"})
				default:
					return exits.NewService(f.r.Pool, nil, nil).SetPublication(t.Context(), exits.SetPublicationParams{NodeID: enrolled.NodeId, Principal: f.p.Scope(), IdempotencyKey: key, IfMatch: header, Request: api.PublicationRequest{PublicationIntent: api.PublicationIntentDisabled, Reason: "TEST ONLY zero version"}})
				}
			}
			before := f.conditionalWriteSnapshot(enrolled.NodeId)
			_, _, err = call(`"0"`)
			var conflict *apierr.Error
			if !errors.As(err, &conflict) || conflict.HTTPStatus() != http.StatusPreconditionFailed || conflict.Code != apierr.ResourceVersionConflict || conflict.Details["current_resource_version"] != initial.ResourceVersion.String() {
				t.Fatalf("zero version did not reach the ordinary comparison: %v", err)
			}
			if f.conditionalWriteSnapshot(enrolled.NodeId) != before {
				t.Fatal("conflict left durable effects")
			}
			op, next, err := call(etag)
			if err != nil || op.OperationId == uuid.Nil || next != nodes.FormatETag((initial.ResourceVersion+1).String()) {
				t.Fatalf("same-key retry failed: etag=%s error=%v", next, err)
			}
			after := f.conditionalWriteSnapshot(enrolled.NodeId)
			if after.Operations != before.Operations+1 || after.Audits != before.Audits+1 || after.Events != before.Events+1 || after.Keys != before.Keys+1 {
				t.Fatal("same-key retry did not commit exactly one command")
			}
			replayed, _, err := call(`"0"`)
			if err != nil || replayed.OperationId != op.OperationId || f.conditionalWriteSnapshot(enrolled.NodeId) != after {
				t.Fatal("accepted-key replay must return the stored operation without another write", err)
			}
		})
	}
}
