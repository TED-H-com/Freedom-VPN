package audit

import (
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestSEC02_SystemActorStrictConstruction(t *testing.T) {
	const id = "00000000-0000-0000-0000-000000000003"
	if SystemActor.Type != api.AuditEventActorTypeSystem || SystemActor.ID.String() != id {
		t.Fatalf("unexpected D-029 singleton: %+v", SystemActor)
	}
	event, err := NewEvent(Entry{
		Actor: SystemActor, Action: "release.verify",
		Resource: api.ResourceRef{Type: api.ResourceTypeRelease, Id: uuid.New()},
		Result:   api.AuditEventResultSucceeded,
	})
	if err != nil {
		t.Fatal(err)
	}
	if event.row.Actor != id || event.row.ActorType != string(api.AuditEventActorTypeSystem) {
		t.Fatalf("strict builder changed the system identity: %+v", event.row)
	}
}
