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

func TestENR02_TokenSha256SurvivesWriteAndReadWhileTokensStayRedacted(t *testing.T) {
	const digest = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
	redactedKeys := []string{"token", "access_token", "refresh_token", "token_hash"}

	afterInput := map[string]any{
		"reason":        "",
		"token_sha256":  digest,
		"token":         "TEST-ONLY-PLAIN-TOKEN",
		"access_token":  "TEST-ONLY-ACCESS-TOKEN",
		"refresh_token": "TEST-ONLY-REFRESH-TOKEN",
		"token_hash":    "TEST-ONLY-TOKEN-HASH",
		"nested": map[string]any{
			"token_sha256":  digest,
			"token":         "TEST-ONLY-NESTED-TOKEN",
			"access_token":  "TEST-ONLY-NESTED-ACCESS",
			"refresh_token": "TEST-ONLY-NESTED-REFRESH",
			"token_hash":    "TEST-ONLY-NESTED-HASH",
		},
	}
	allow := []string{"reason", "token_sha256", "token", "access_token", "refresh_token", "token_hash", "nested"}

	event, err := NewEvent(Entry{
		Actor:    Actor{Type: api.AuditEventActorTypeAdmin, ID: uuid.New()},
		Action:   "enrollment.created",
		Resource: api.ResourceRef{Type: api.ResourceTypeEnrollment, Id: uuid.New()},
		Result:   api.AuditEventResultSucceeded,
		After:    afterInput,
		Fields:   allow,
	})
	if err != nil {
		t.Fatal(err)
	}
	written := string(event.row.Before) + string(event.row.After)
	if strings.Contains(written, "TEST-ONLY-") {
		t.Fatalf("sensitive token fields leaked on write: %s", written)
	}
	if !strings.Contains(string(event.row.After), `"token_sha256":"`+digest+`"`) {
		t.Fatalf("token_sha256 dropped or redacted on write: %s", string(event.row.After))
	}
	for _, key := range redactedKeys {
		if sensitive(key) != true {
			t.Fatalf("expected %s to remain sensitive", key)
		}
	}
	if sensitive("token_sha256") {
		t.Fatal("expected token_sha256 to be exempt from sensitive()")
	}

	// Verify decodeEvent (read path) preserves token_sha256 from NewEvent output
	// and redacts any stored token, access_token, refresh_token, or token_hash.
	var decodedFromWrite api.AuditEvent
	decodedFromWrite.Action = event.row.Action
	decodedFromWrite.Result = api.AuditEventResult(event.row.Result)
	decodedFromWrite.Actor.Type = api.AuditEventActorType(event.row.ActorType)
	if reason := decodeEvent(&decodedFromWrite, event.row.Actor, event.row.Resource, event.row.Before, event.row.After); reason != "" {
		t.Fatalf("unexpected anomaly decoding written event: %q", reason)
	}
	afterWriteMap, err := decodedFromWrite.Changes.After.Get()
	if err != nil {
		t.Fatal(err)
	}
	if afterWriteMap["token_sha256"] != digest {
		t.Fatalf("token_sha256 did not survive write+read round-trip: got %v, want %q", afterWriteMap["token_sha256"], digest)
	}
	for _, key := range redactedKeys {
		if _, exists := afterWriteMap[key]; exists {
			t.Fatalf("expected top-level %s to be omitted by Diff, got %v", key, afterWriteMap[key])
		}
	}
	nestedWrite, ok := afterWriteMap["nested"].(map[string]any)
	if !ok || nestedWrite["token_sha256"] != digest {
		t.Fatalf("nested token_sha256 did not survive write+read: %+v", afterWriteMap["nested"])
	}
	for _, key := range redactedKeys {
		if nestedWrite[key] != "[REDACTED]" {
			t.Fatalf("nested %s not redacted: got %v", key, nestedWrite[key])
		}
	}

	// Also verify decodeEvent redacts top-level stored token fields while keeping token_sha256.
	rawStoredAfter := []byte(`{"reason":"","token_sha256":"` + digest + `","token":"TEST-ONLY-PLAIN","access_token":"TEST-ONLY-ACCESS","refresh_token":"TEST-ONLY-REFRESH","token_hash":"TEST-ONLY-HASH"}`)
	var decodedStored api.AuditEvent
	decodedStored.Action = "enrollment.created"
	decodedStored.Result = api.AuditEventResultSucceeded
	decodedStored.Actor.Type = api.AuditEventActorTypeAdmin
	if reason := decodeEvent(&decodedStored, uuid.NewString(), "enrollment:"+uuid.NewString(), []byte(`{}`), rawStoredAfter); reason != "" {
		t.Fatalf("unexpected anomaly decoding stored row: %q", reason)
	}
	storedAfterMap, err := decodedStored.Changes.After.Get()
	if err != nil {
		t.Fatal(err)
	}
	if storedAfterMap["token_sha256"] != digest {
		t.Fatalf("stored token_sha256 was redacted on read: got %v, want %q", storedAfterMap["token_sha256"], digest)
	}
	for _, key := range redactedKeys {
		if storedAfterMap[key] != "[REDACTED]" {
			t.Fatalf("stored %s was not redacted on read: got %v", key, storedAfterMap[key])
		}
	}
}
