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

func TestPurgeRetainedRowDecodesJSONResourceRefAndSystemRole(t *testing.T) {
	ev := api.AuditEvent{
		Action: "retention.purge",
		Result: api.AuditEventResultSucceeded,
	}
	ev.Actor.Type = api.AuditEventActorTypeSystem

	reason := decodeEvent(
		&ev,
		"00000000-0000-0000-0000-000000000000",
		"fleet_config:00000000-0000-0000-0000-000000000001",
		[]byte(`{}`),
		[]byte(`{"deleted_rows":"0"}`),
	)
	if reason != "" {
		t.Fatalf("unexpected reason for canonical purge_retained row: %q", reason)
	}
	actorID, err := ev.Actor.Id.Get()
	if err != nil || uuid.UUID(actorID) != uuid.Nil {
		t.Fatalf("expected actor.id = nil UUID for canonical system role, got %v (err=%v)", actorID, err)
	}
	if ev.Resource.Type != api.ResourceTypeFleetConfig {
		t.Fatalf("expected resource.type = fleet_config, got %q", ev.Resource.Type)
	}
	wantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	if ev.Resource.Id != wantID {
		t.Fatalf("expected resource.id = %s, got %s", wantID, ev.Resource.Id)
	}

	// Single-format reader rejects JSON ResourceRef strings.
	if r := decodeEvent(&ev, "00000000-0000-0000-0000-000000000000", `{"type":"fleet_config","id":"00000000-0000-0000-0000-000000000001"}`, []byte(`{}`), []byte(`{}`)); r != "resource_type" {
		t.Fatalf("expected resource_type anomaly for JSON resource, got %q", r)
	}
}
