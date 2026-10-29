package audit

import (
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"testing"
)

func TestSEC02_TypedAuditConstruction(t *testing.T) {
	valid := Entry{Actor: Actor{Type: api.AuditEventActorTypeAdmin, ID: uuid.New()}, Action: "node.update", Resource: api.ResourceRef{Type: api.ResourceTypeNode, Id: uuid.New()}, Result: api.AuditEventResultSucceeded}
	for _, bad := range []func(*Entry){
		func(e *Entry) { e.Actor.Type = "agent" }, func(e *Entry) { e.Actor.ID = uuid.Nil },
		func(e *Entry) { e.Resource.Type = "device" }, func(e *Entry) { e.Resource.Id = uuid.Nil },
		func(e *Entry) { e.Result = "accepted" }, func(e *Entry) { e.Action = "unstructured action" },
		func(e *Entry) { id := uuid.Nil; e.RequestID = &id },
	} {
		e := valid
		bad(&e)
		if _, err := NewEvent(e); err == nil {
			t.Fatal("invalid entry accepted", e)
		}
	}
	if _, err := NewEvent(valid); err != nil {
		t.Fatal(err)
	}
	if err := AppendEvent(t.Context(), nil, Event{}); err == nil {
		t.Fatal("zero event accepted")
	}
}
