//go:build integration

package httpwire_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

type publicationSnapshot struct {
	Version, ExitVersion, Generation decstr.U64
	Intent                           api.PublicationIntent
	Operations, Audits, Events, Keys int
}

func (f *fixture) publicationSnapshot(node uuid.UUID) publicationSnapshot {
	f.t.Helper()
	var s publicationSnapshot
	err := f.r.Pool.QueryRow(f.t.Context(), `SELECT n.resource_version::text,e.resource_version::text,n.desired_config_generation::text,e.publication_intent,
		(SELECT count(*) FROM operations WHERE resource_id=n.id),
		(SELECT count(*) FROM audit_events WHERE resource='exit:'||e.id::text),
		(SELECT count(*) FROM outbox_events WHERE aggregate_id=n.id),
		(SELECT count(*) FROM idempotency_requests WHERE route LIKE '%/publication%')
		FROM nodes n JOIN exits e ON e.node_id=n.id WHERE n.id=$1`, node).Scan(&s.Version, &s.ExitVersion, &s.Generation, &s.Intent, &s.Operations, &s.Audits, &s.Events, &s.Keys)
	if err != nil {
		f.t.Fatal(err)
	}
	return s
}

// §5.2/§14.5 and AT08: acceptance is atomic intent, never observed completion.
func TestAT08_HTTPPublicationIntentsReplayAndPreconditions(t *testing.T) {
	f, enrolled, cert, member, _ := publicationFixture(t)
	path := "/api/v1/admin/nodes/" + enrolled.NodeId.String() + "/publication"
	store := &audit.Store{Pool: f.r.Pool}
	for _, change := range []struct {
		intent api.PublicationIntent
		kind   api.OperationKind
	}{
		{api.PublicationIntentPublished, api.OperationKindNodePublish},
		{api.PublicationIntentUnpublished, api.OperationKindNodeUnpublish},
		{api.PublicationIntentDisabled, api.OperationKindNodeDisable},
	} {
		t.Run(string(change.intent), func(t *testing.T) {
			before := f.publicationSnapshot(enrolled.NodeId)
			body := api.PublicationRequest{PublicationIntent: change.intent, Reason: "TEST ONLY publication command"}
			headers := map[string]string{"Idempotency-Key": uuid.NewString(), "If-Match": nodes.FormatETag(before.Version.String())}
			first := f.call("PUT", path, body, nil, headers)
			op := decode[api.Operation](t, first, http.StatusAccepted)
			if op.Kind != change.kind || op.Status != api.OperationStatusPending || op.Resource.Id != enrolled.NodeId || mustValue(t, op.DesiredGeneration) != before.Generation+1 || mustValue(t, op.ResourceVersion) != before.Version+1 {
				t.Fatalf("wrong publication operation: %+v", op)
			}
			if len(op.Steps) != 3 {
				t.Fatalf("wrong operation steps: %+v", op.Steps)
			}
			for i, name := range []string{nodes.StepAgentPolicy, nodes.StepNetBirdRouteReadback, nodes.StepDataPathProbe} {
				step := op.Steps[i]
				required, status := true, api.OperationStepStatusPending
				if name == nodes.StepDataPathProbe && change.intent != api.PublicationIntentPublished {
					required, status = false, api.OperationStepStatusNotRequired
				}
				if step.Name != name || step.Required != required || step.Status != status || !step.ObservedAt.IsNull() {
					t.Fatalf("wrong step %s: %+v", name, step)
				}
			}
			after := f.publicationSnapshot(enrolled.NodeId)
			want := publicationSnapshot{Version: before.Version + 1, ExitVersion: before.ExitVersion + 1, Generation: before.Generation + 1, Intent: change.intent, Operations: before.Operations + 1, Audits: before.Audits + 1, Events: before.Events + 1, Keys: before.Keys + 1}
			if after != want {
				t.Fatalf("publication writes: %+v want %+v", after, want)
			}
			var event, principal string
			if err := f.r.Pool.QueryRow(t.Context(), `SELECT o.event_type,p.principal FROM outbox_events o JOIN operations p ON p.id=$1 WHERE o.payload->>'operation_id'=$1::text`, op.OperationId).Scan(&event, &principal); err != nil || event != "node.publication_changed" || principal != f.p.Scope() {
				t.Fatalf("wrong event/principal: %q %q %v", event, principal, err)
			}
			page, err := store.List(t.Context(), api.ListAuditEventsParams{RequestId: &op.RequestId})
			if err != nil || len(page.Items) != 1 || store.AnomalyCount() != 0 {
				t.Fatalf("core audit unreadable: %+v %v anomalies=%d", page, err, store.AnomalyCount())
			}
			a := page.Items[0]
			if a.Action != string(change.kind) || a.Actor.Type != api.AuditEventActorTypeAdmin || mustValue(t, a.Actor.Id) != f.p.ID || a.Resource.Type != api.ResourceTypeExit || a.Result != api.AuditEventResultSucceeded || mustValue(t, a.OperationId) != op.OperationId || mustValue(t, a.RequestId) != op.RequestId || mustValue(t, a.Changes.After)["publication_intent"] != string(change.intent) {
				t.Fatalf("wrong audit: %+v", a)
			}
			replay := f.call("PUT", path, body, nil, headers)
			replayed := decode[api.Operation](t, replay, http.StatusAccepted)
			// The core store orders steps by name; the node transaction reader
			// orders them by execution order. Compare the same persisted steps.
			sort.Slice(op.Steps, func(i, j int) bool { return op.Steps[i].Name < op.Steps[j].Name })
			sort.Slice(replayed.Steps, func(i, j int) bool { return replayed.Steps[i].Name < replayed.Steps[j].Name })
			if !reflect.DeepEqual(op, replayed) || f.publicationSnapshot(enrolled.NodeId) != after {
				t.Fatalf("replay mismatch (equal=%v): first=%s replay=%s snapshot=%+v want=%+v", reflect.DeepEqual(op, replayed), first.Body.String(), replay.Body.String(), f.publicationSnapshot(enrolled.NodeId), after)
			}
			for _, precondition := range []struct {
				tag    string
				status int
				code   api.ErrorCode
			}{
				{headers["If-Match"], http.StatusPreconditionFailed, api.ErrorCodeRESOURCEVERSIONCONFLICT},
				{"", http.StatusPreconditionRequired, api.ErrorCodePRECONDITIONREQUIRED},
			} {
				failure := decode[api.ErrorEnvelope](t, f.call("PUT", path, body, nil, map[string]string{"Idempotency-Key": uuid.NewString(), "If-Match": precondition.tag}), precondition.status)
				if failure.Error.Code != precondition.code || f.publicationSnapshot(enrolled.NodeId) != after {
					t.Fatalf("precondition failure wrote state or wrong code: %+v", failure)
				}
			}
			w := f.call("GET", "/api/v1/agent/desired-state", nil, cert, nil)
			state := decode[api.DesiredStateEnvelope](t, w, http.StatusOK)
			if state.Signed.PublicationIntent != change.intent {
				t.Fatalf("intent missing from desired state: %+v", state.Signed)
			}
			if change.intent == api.PublicationIntentPublished {
				if len(state.Signed.MemberRatePolicies) != 1 || state.Signed.MemberRatePolicies[0].MemberId != member || mustValue(t, state.Signed.MemberRatePolicies[0].DownBps) != 40000000 {
					t.Fatalf("D-030/FUP member policy missing: %+v", state.Signed.MemberRatePolicies)
				}
			} else if len(state.Signed.MemberRatePolicies) != 0 {
				t.Fatal("withdrawn publication retained member policies")
			}
			trust, err := f.r.Provider.TrustBundle(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err = envelope.Verify(w.Body.Bytes(), trust, envelope.PurposeDesiredState, time.Now()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

type publicationProbe struct {
	*httpwire.Handlers
	after func(context.Context) error
}

func (h *publicationProbe) SetNodePublication(ctx context.Context, r gen.SetNodePublicationRequestObject) (gen.SetNodePublicationResponseObject, error) {
	response, err := h.Handlers.SetNodePublication(ctx, r)
	if err != nil {
		return nil, err
	}
	if err = h.after(ctx); err != nil {
		return nil, err
	}
	return response, nil
}

func TestAT08_HTTPPublicationRollsBackWithCommand(t *testing.T) {
	f, enrolled, _, _, _ := publicationFixture(t)
	before := f.publicationSnapshot(enrolled.NodeId)
	probe := &publicationProbe{Handlers: &httpwire.Handlers{Runtime: f.r}}
	probe.after = func(ctx context.Context) error {
		tx, err := httpapi.Transaction(ctx)
		if err != nil {
			return err
		}
		var intent api.PublicationIntent
		if err = tx.QueryRow(ctx, `SELECT publication_intent FROM exits WHERE node_id=$1`, enrolled.NodeId).Scan(&intent); err != nil || intent != api.PublicationIntentPublished {
			t.Errorf("publication absent inside command transaction: %s %v", intent, err)
		}
		if f.publicationSnapshot(enrolled.NodeId) != before {
			t.Error("publication committed before HTTP command")
		}
		return errors.New("TEST ONLY failure before command completion")
	}
	composite, err := httpapi.Compose(probe)
	if err != nil {
		t.Fatal(err)
	}
	c := config.Defaults()
	c.PublicOrigin = "https://portal.example.invalid"
	h, err := httpapi.New(composite, httpapi.Options{Config: c, Sessions: auth.SessionAuthenticatorFunc(func(*http.Request) (auth.Principal, error) { return f.p, nil }), Commands: &httpapi.Commands{Pool: f.r.Pool, Retention: time.Hour}})
	if err != nil {
		t.Fatal(err)
	}
	f.public = httpapi.Transport(nil, httpwire.RequestContext(h))
	path := "/api/v1/admin/nodes/" + enrolled.NodeId.String() + "/publication"
	body := api.PublicationRequest{PublicationIntent: api.PublicationIntentPublished, Reason: "TEST ONLY rollback"}
	headers := map[string]string{"Idempotency-Key": uuid.NewString(), "If-Match": nodes.FormatETag(before.Version.String())}
	if w := f.call("PUT", path, body, nil, headers); w.Code != http.StatusInternalServerError {
		t.Fatalf("injected failure: %d %s", w.Code, w.Body.String())
	}
	if got := f.publicationSnapshot(enrolled.NodeId); got != before {
		t.Fatalf("partial publication commit: %+v want %+v", got, before)
	}
	probe.after = func(context.Context) error { return nil }
	decode[api.Operation](t, f.call("PUT", path, body, nil, headers), http.StatusAccepted)
}
