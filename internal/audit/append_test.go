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

// D-059 / §15.2, §16.4: groups have their own audit resource; the generated
// enum is the allowlist for both typed NewEvent and legacy Append callers.
func TestAPI01_MemberGroupAuditResource(t *testing.T) {
	groupID := uuid.New()
	entry := Entry{
		Actor:    Actor{Type: api.AuditEventActorTypeAdmin, ID: uuid.New()},
		Action:   "group.create",
		Resource: api.ResourceRef{Type: api.ResourceTypeMemberGroup, Id: groupID},
		Result:   api.AuditEventResultSucceeded,
		After:    map[string]any{"code": "standard", "name": "Standard exits"},
		Fields:   []string{"code", "name"},
	}
	event, err := NewEvent(entry)
	if err != nil {
		t.Fatal(err)
	}
	if event.row.Resource != "member_group:"+groupID.String() || event.row.Action != "group.create" || event.row.Result != "succeeded" {
		t.Fatalf("incorrect group audit metadata: %+v", event.row)
	}
	if event.row.OperationID != nil {
		t.Fatal("synchronous group creation acquired an operation")
	}
	for _, unknown := range []api.ResourceType{"group", "member_groups", "unknown"} {
		entry.Resource.Type = unknown
		if _, err := NewEvent(entry); err == nil {
			t.Fatalf("unapproved resource %q accepted", unknown)
		}
	}
}
