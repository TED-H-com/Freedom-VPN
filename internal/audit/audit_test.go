package audit

import (
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestSEC02_RedactTypedAndNestedDiffs(t *testing.T) {
	value := map[string]any{"state": map[string]string{"status": "pending", "setup_key": "TEST-ONLY-SECRET"}, "unapproved": "TEST-ONLY-NOT-ALLOWED", "authorization": "TEST-ONLY-TOKEN"}
	before, after, e := Diff(nil, value, "state", "authorization")
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(before)+string(after), "TEST-ONLY-") || !strings.Contains(string(after), "pending") {
		t.Fatal(string(before), string(after))
	}
}

func TestDecodeEvent_PurgeRetainedJSONResourceAndSystemRole(t *testing.T) {
	ev := api.AuditEvent{
		Action: "retention.purge",
		Result: api.AuditEventResultSucceeded,
	}
	ev.Actor.Type = api.AuditEventActorTypeSystem

	reason := decodeEvent(
		&ev,
		"postgres",
		`{"type":"fleet_config","id":"00000000-0000-0000-0000-000000000001"}`,
		[]byte(`{}`),
		[]byte(`{"usage_events":0}`),
	)
	if reason != "" {
		t.Fatalf("unexpected reason: %q", reason)
	}
	if !ev.Actor.Id.IsSpecified() || !ev.Actor.Id.IsNull() {
		t.Fatalf("expected actor.id = null for non-UUID system role, got %+v", ev.Actor.Id)
	}
	if ev.Resource.Type != api.ResourceType("fleet_config") {
		t.Fatalf("expected resource.type = fleet_config, got %q", ev.Resource.Type)
	}
	wantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	if ev.Resource.Id != wantID {
		t.Fatalf("expected resource.id = %s, got %s", wantID, ev.Resource.Id)
	}
}
